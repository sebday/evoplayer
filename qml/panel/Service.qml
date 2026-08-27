import QtQuick
import Quickshell
import Quickshell.Io
import "compat"
import "compat/PluginIds.js" as PluginIds
import "compat/CliampBackend.js" as CliampBackend

Item {
    id: root

    property var shell: null
    property var player: ({})
    property var vizLevels: []
    property int vizRevision: 0
    property int vizSequence: 0
    property int vizGeneration: 0
    property string lastNotifiedPath: ""
    property bool lastNotifiedHadArt: false
    property string lastNotifiedArt: ""
    property string lastNotifiedGenre: ""
    property string lastNotifiedYear: ""
    property bool lastNotifiedLiked: false
    property string scrobblePath: ""
    property bool scrobbleSubmitted: false
    property real scrobbleStartPos: -1
    property int scrobbleStartedAt: 0
    property string lastNowPlayingScrobblePath: ""
    property int lastNowPlayingScrobbleAt: 0

    readonly property string home: Quickshell.env("HOME") || ""
    readonly property string socketPath: (Quickshell.env("XDG_RUNTIME_DIR") || "/tmp") + "/evoplayer.sock"
    readonly property bool useCliampBackend: CliampBackend.enabled(Quickshell.env("EVOPLAYER_BACKEND"))

    function playerCmd(args) {
        return Util.evoplayerCommand(home, args || [])
    }

    property int ipcNextReq: 1
    property bool ipcSubscribed: false
    property var ipcPending: ({})
    property var ipcWaitQueue: []
    property var ipcTimeouts: ({})

    readonly property int ipcTimeoutMs: 15000

    CliampBridge {
        id: cliampBridge
        service: root
        active: root.useCliampBackend
    }

    property bool scanNoticeShown: false
    readonly property int scanNotifyId: 42421

    signal daemonJobUpdated(var data)

    readonly property bool ipcReady: useCliampBackend ? cliampBridge.ipcReady : playerSocket.connected
    property bool ipcSynced: false
    property string enrichPath: ""
    property string enrichQueuedPath: ""
    property bool enrichBusy: false
    property string enrichCurrentPath: ""

    function formatTime(sec) {
        var total = Math.max(0, Math.floor(Number(sec) || 0))
        var min = Math.floor(total / 60)
        var s = total % 60
        return min + ":" + (s < 10 ? "0" : "") + s
    }

    function ipcWrite(method, params) {
        if (!playerSocket.connected)
            return -1
        var id = ipcNextReq++
        var payload = { id: id, method: method }
        if (params !== undefined)
            payload.params = params
        playerSocket.write(JSON.stringify(payload) + "\n")
        playerSocket.flush()
        return id
    }

    function applyStatePayload(data) {
        if (!data || typeof data !== "object")
            return
        ipcSynced = true
        var patch = Object.assign({}, data)
        if (patch.position !== undefined)
            patch.position_label = formatTime(patch.position)
        if (patch.duration !== undefined)
            patch.duration_label = formatTime(patch.duration)
        var path = String(patch.path || "")
        if (path && path !== enrichPath) {
            enrichPath = path
            requestEnrich(path)
        }
        if (!path) {
            lastNotifiedPath = ""
            lastNotifiedHadArt = false
            lastNotifiedArt = ""
            lastNotifiedGenre = ""
            lastNotifiedYear = ""
            lastNotifiedLiked = false
            resetScrobbleSession()
        }
        mergePlayer(patch)
    }

    function ipcCall(method, params, onDone) {
        if (useCliampBackend) {
            cliampBridge.ipcCall(method, params, onDone)
            return
        }
        if (!playerSocket.connected) {
            ensurePlayer()
            ipcWaitQueue.push({ method: method, params: params, onDone: onDone || null })
            return
        }
        var id = ipcWrite(method, params)
        if (id < 0) {
            if (onDone)
                onDone(false, null)
            return
        }
        if (onDone)
            ipcPending[id] = onDone
        ipcTimeouts[id] = Date.now() + ipcTimeoutMs
    }

    function clearIpcTimeout(id) {
        if (ipcTimeouts[id] !== undefined)
            delete ipcTimeouts[id]
    }

    function sweepIpcTimeouts() {
        var now = Date.now()
        for (var id in ipcPending) {
            if (ipcTimeouts[id] !== undefined && ipcTimeouts[id] < now) {
                var cb = ipcPending[id]
                delete ipcPending[id]
                delete ipcTimeouts[id]
                if (cb)
                    cb(false, null)
            }
        }
    }

    function flushIpcWaitQueue() {
        if (!playerSocket.connected || ipcWaitQueue.length === 0)
            return
        var pending = ipcWaitQueue.slice()
        ipcWaitQueue = []
        for (var i = 0; i < pending.length; i++) {
            var job = pending[i]
            ipcCall(job.method, job.params, job.onDone)
        }
    }

    function callEvoplayerIPC(method, params, onDone) {
        if (!playerSocket.connected) {
            ensureEvoplayerConnect()
            ipcWaitQueue.push({ method: method, params: params, onDone: onDone || null })
            return
        }
        var id = ipcWrite(method, params)
        if (id < 0) {
            if (onDone)
                onDone(false, null)
            return
        }
        if (onDone)
            ipcPending[id] = onDone
        ipcTimeouts[id] = Date.now() + ipcTimeoutMs
    }

    function ipcCallVoid(method, params) {
        ipcCall(method, params, null)
    }

    function failPendingIPC() {
        var pending = ipcPending
        ipcPending = {}
        ipcTimeouts = {}
        for (var id in pending) {
            if (pending[id])
                pending[id](false, null)
        }
    }

    function handleIpcLine(line) {
        var msg
        try {
            msg = JSON.parse(String(line || ""))
        } catch (e) {
            return
        }
        if (msg.id && ipcPending[msg.id]) {
            var cb = ipcPending[msg.id]
            delete ipcPending[msg.id]
            root.clearIpcTimeout(msg.id)
            cb(!!msg.ok, msg)
            return
        }
        if (msg.event === "state")
            applyStatePayload(msg.data)
        else if (msg.event === "viz")
            applyVizPayload(msg.data)
        else if (msg.event === "job")
            applyJobPayload(msg.data)
        else if (msg.event === "warm")
            applyWarmPayload(msg.data)
        else if (msg.ok && msg.data)
            applyStatePayload(msg.data)
    }

    function mergePlayer(patch) {
        var prevPath = String(player.path || "")
        var prevState = String(player.state || "")
        var prevArt = String(player.art || "")
        var prevGenre = String(player.genre || "").trim()
        var prevYear = String(player.year || "").trim()
        var prevLiked = !!player.liked
        var next = Object.assign({}, player, patch)
        var patchPath = String(patch.path || "")
        var samePath = patchPath !== "" && patchPath === prevPath
        var prevPos = Number(player.position) || 0
        if (samePath && patch.position !== undefined) {
            var seekPos = Number(patch.position) || 0
            if (seekPos < prevPos - 5)
                beginScrobbleSession()
        }
        if (samePath) {
            var metaKeys = ["title", "artist", "album", "art", "genre", "year", "label", "waveform", "liked", "queue_revision"]
            for (var i = 0; i < metaKeys.length; i++) {
                var key = metaKeys[i]
                if (!String(patch[key] || "") && String(player[key] || ""))
                    next[key] = player[key]
            }
        }
        var newPath = String(next.path || "")
        if (prevPath && newPath !== prevPath
                && scrobblePath === prevPath
                && !scrobbleSubmitted
                && prevState === "playing")
            maybeSubmitScrobble()
        player = next
        var state = String(player.state || "")
        var pathChanged = newPath !== prevPath
        var artChanged = String(next.art || "") !== prevArt
        var metaChanged = newPath !== "" && newPath === prevPath
            && (String(next.genre || "").trim() !== prevGenre
                || String(next.year || "").trim() !== prevYear
                || !!next.liked !== prevLiked)
        if (newPath && state === "playing") {
            if (pathChanged || state !== prevState || artChanged || metaChanged)
                notifyNowPlaying()
            maybeSubmitScrobble()
        }
    }

    function applyStatusPayload(text) {
        var parsed
        try {
            parsed = JSON.parse(String(text || "{}"))
        } catch (e) {
            parsed = {}
        }
        applyStatePayload(parsed)
    }

    function applyVizPayload(data) {
        if (!data || !Array.isArray(data.levels))
            return
        var seq = Number(data.sequence) || 0
        var gen = Number(data.generation) || 0
        if (seq > 0 && seq <= vizSequence && gen === vizGeneration)
            return
        if (gen > 0)
            vizGeneration = gen
        if (seq > 0)
            vizSequence = seq
        vizLevels = data.levels
        vizRevision++
    }

    function applyJobPayload(data) {
        if (!data)
            return
        daemonJobUpdated(data)
        root.applyScanNotice(data)
    }

    function applyScanNotice(data) {
        var name = String(data.name || "")
        var status = String(data.status || "")
        if (name !== "scan")
            return
        if (status === "running") {
            if (scanNoticeShown)
                return
            scanNoticeShown = true
            showScanNotice("Scanning library…", 0)
            return
        }
        if (status === "done" || status === "error") {
            if (!scanNoticeShown && status !== "error")
                return
            scanNoticeShown = false
            var msg = "Library ready"
            if (status === "error") {
                var errText = String(data.error || "").toLowerCase()
                msg = errText.indexOf("cancel") >= 0 ? "Scan stopped" : "Library scan failed"
            }
            showScanNotice(msg, 3000)
        }
    }

    function showScanNotice(body, timeoutMs) {
        var summary = "Evoplayer"
        var text = String(body || "")
        var args = ["notify-send", "-a", PluginIds.pluginId, "-r", String(scanNotifyId)]
        if (timeoutMs === 0) {
            args.push("-u", "critical")
            args.push("-t", "0")
        } else {
            args.push("-t", String(Math.max(1, timeoutMs || 3000)))
        }
        args.push(summary, text)
        Quickshell.execDetached(args)
    }

    function applyWarmPayload(data) {
        if (!data || !data.path)
            return
        daemonWarmEvent(data)
        var path = String(data.path)
        if (String(player.path || "") === path)
            requestEnrich(path)
    }

    function queueAppendFolder(folderPath, onDone) {
        ipcCall("queue.append_folder", { path: String(folderPath || "") }, onDone || null)
    }

    function currentSave(paths, onDone) {
        ipcCall("library.current.save", { paths: paths || [] }, onDone || null)
    }

    function currentLoad(onDone) {
        ipcCall("library.current.load", {}, onDone || null)
    }

    function browseQueue(relPath, onDone) {
        ipcCall("library.browse", {
            path: String(relPath || ""),
            queue: true,
            queue_paths_only: true
        }, onDone || null)
    }

    function warmBatch(paths, art, onDone) {
        ipcCall("library.warm.batch", {
            paths: paths || [],
            workers: 8,
            art: !!art
        }, onDone || null)
    }

    function favoriteToggle(path, onDone) {
        ipcCall("library.favorite.toggle", { path: String(path || "") }, onDone || null)
    }

    function requestEnrich(path) {
        var p = String(path || "")
        if (!p) {
            enrichQueuedPath = ""
            return
        }
        enrichQueuedPath = p
        if (enrichBusy)
            return
        pumpEnrich()
    }

    function pumpEnrich() {
        var p = String(enrichQueuedPath || "")
        enrichQueuedPath = ""
        if (!p) {
            if (enrichQueuedPath)
                pumpEnrich()
            return
        }
        enrichBusy = true
        enrichCurrentPath = p
        ipcCall("library.meta", { path: p }, function(ok, msg) {
            enrichBusy = false
            var requested = String(enrichCurrentPath || "")
            if (!requested || requested !== String(root.enrichPath || ""))
                return
            if (ok && msg && msg.data) {
                var parsed = Object.assign({}, msg.data)
                delete parsed.state
                delete parsed.position
                delete parsed.position_label
                delete parsed.duration
                delete parsed.duration_label
                delete parsed.volume
                delete parsed.shuffle
                root.mergePlayer(parsed)
            }
            if (String(root.enrichQueuedPath || ""))
                root.pumpEnrich()
        })
    }

    function beginScrobbleSession() {
        var path = String(player.path || "")
        if (!path)
            return
        scrobblePath = path
        scrobbleStartPos = Number(player.position) || 0
        scrobbleStartedAt = Math.floor(Date.now() / 1000 - scrobbleStartPos)
        scrobbleSubmitted = false
    }

    function resetScrobbleSession() {
        scrobblePath = ""
        scrobbleStartPos = -1
        scrobbleStartedAt = 0
        scrobbleSubmitted = false
    }

    function maybeWarmTrack() {
        var path = String(player.path || "")
        if (!path)
            return
        var hasArt = String(player.art || "").trim() !== ""
        if (hasArt)
            return
        ipcCallVoid("library.warm", { path: path })
    }

    function showBrief(title, body, durationMs) {
        var summary = String(title || "Evoplayer")
        var text = String(body || "")
        var timeout = Math.max(1, Math.round((durationMs || 3000) / 1000))
        Quickshell.execDetached(["notify-send", "-t", String(timeout), summary, text])
    }

    function showMedia(opts) {
        var o = opts || {}
        var title = String(o.title || "Unknown")
        var artist = String(o.artist || "")
        var body = artist ? (title + " — " + artist) : title
        var args = ["notify-send", "-a", PluginIds.pluginId, title, body]
        var art = String(o.art || "").trim()
        if (art)
            args = ["notify-send", "-a", PluginIds.pluginId, "-i", art, title, body]
        Quickshell.execDetached(args)
    }

    function notifyDisplayArtReady(trackPath, artPath) {
        if (!shell || !shell.panelLoaders)
            return
        var loader = shell.panelLoaders[PluginIds.pluginId]
        if (loader && loader.item && typeof loader.item.applyDisplayArt === "function")
            loader.item.applyDisplayArt(trackPath, artPath)
    }

    function runTransport(action) {
        if (useCliampBackend) {
            cliampBridge.runTransport(action)
            return
        }
        if (action === "toggle")
            ipcCallVoid("playback.toggle")
        else if (action === "next")
            ipcCallVoid("playback.next")
        else if (action === "prev" || action === "previous")
            ipcCallVoid("playback.prev")
        else if (action === "play")
            ipcCallVoid("playback.toggle")
        else if (action === "pause") {
            if (String(player.state || "") === "playing")
                ipcCallVoid("playback.toggle")
        }
    }

    function pushMediaNotification(artPath) {
        var path = String(player.path || "")
        if (!path)
            return
        var art = String(artPath || "")
        var genre = String(player.genre || "").trim()
        var year = String(player.year || "").trim()
        var liked = !!player.liked
        if (path === lastNotifiedPath && art === lastNotifiedArt
                && genre === lastNotifiedGenre && year === lastNotifiedYear
                && liked === lastNotifiedLiked)
            return
        var needsArtUpdate = path === lastNotifiedPath && art !== "" && !lastNotifiedHadArt
        showMedia({
            app: PluginIds.pluginId,
            title: String(player.title || "Unknown"),
            artist: String(player.artist || ""),
            art: art,
            path: path,
            genre: genre,
            year: year,
            liked: liked
        })
        if (needsArtUpdate)
            runScrobble(["touch"])
        lastNotifiedPath = path
        lastNotifiedHadArt = art !== ""
        lastNotifiedArt = art
        lastNotifiedGenre = genre
        lastNotifiedYear = year
        lastNotifiedLiked = liked
    }

    function notifyNowPlaying() {
        if (!shell)
            return
        var path = String(player.path || "")
        if (!path)
            return
        var hasArt = String(player.art || "") !== ""
        var genre = String(player.genre || "").trim()
        var year = String(player.year || "").trim()
        var liked = !!player.liked
        var scrobbleTrackChanged = path !== scrobblePath
        var pathChanged = path !== lastNotifiedPath
        var needsArtUpdate = path === lastNotifiedPath && hasArt && !lastNotifiedHadArt
        var needsMetaUpdate = path === lastNotifiedPath
            && (genre !== lastNotifiedGenre || year !== lastNotifiedYear || liked !== lastNotifiedLiked)

        if (scrobbleTrackChanged) {
            var now = Date.now()
            if (path !== lastNowPlayingScrobblePath || now - lastNowPlayingScrobbleAt >= 2000) {
                runScrobble(["nowplaying"])
                lastNowPlayingScrobblePath = path
                lastNowPlayingScrobbleAt = now
            }
            beginScrobbleSession()
            maybeWarmTrack()
        }

        if (!pathChanged && !needsArtUpdate && !needsMetaUpdate)
            return

        if (pathChanged) {
            lastNotifiedPath = path
            lastNotifiedHadArt = false
            lastNotifiedArt = ""
            lastNotifiedGenre = ""
            lastNotifiedYear = ""
            lastNotifiedLiked = false
        }

        if (hasArt) {
            if (pathChanged || needsArtUpdate) {
                if (!notifyArtProc.running) {
                    notifyArtProc.requestedPath = path
                    notifyArtProc.command = playerCmd(["art", "notify-cache", path])
                    notifyArtProc.running = true
                } else {
                    notifyArtProc.pendingPath = path
                }
                return
            }
            if (needsMetaUpdate) {
                pushMediaNotification(String(player.art || ""))
                return
            }
            return
        }
        pushMediaNotification("")
    }

    // Last.fm: track > 30s; listen min(half duration, 4 minutes).
    function scrobbleListenThreshold(durationSec) {
        var dur = Number(durationSec) || 0
        if (dur <= 30)
            return -1
        return Math.min(dur * 0.5, 240)
    }

    function maybeSubmitScrobble() {
        if (!player.path || scrobbleSubmitted || String(player.path) !== scrobblePath)
            return
        if (String(player.state || "") !== "playing")
            return
        if (scrobbleStartPos < 0)
            return
        var dur = Number(player.duration) || 0
        var threshold = scrobbleListenThreshold(dur)
        if (threshold < 0)
            return
        if (dur <= 0)
            return
        var pos = Number(player.position) || 0
        if (pos < scrobbleStartPos)
            scrobbleStartPos = pos
        var listened = pos - scrobbleStartPos
        if (listened < threshold)
            return
        scrobbleSubmitted = true
        var started = scrobbleStartedAt > 0
            ? scrobbleStartedAt
            : Math.floor(Date.now() / 1000 - listened)
        runScrobble(["submit", "--started", String(started)])
    }

    function runScrobble(args) {
        args = args || []
        if (args[0] === "nowplaying") {
            ipcCallVoid("scrobble.nowplaying")
            return
        }
        if (args[0] === "submit") {
            var started = 0
            for (var i = 1; i < args.length; i++) {
                if (args[i] === "--started" && i + 1 < args.length)
                    started = Number(args[++i]) || 0
            }
            ipcCallVoid("scrobble.submit", { started: started })
            return
        }
        if (scrobbleProc.running)
            return
        scrobbleProc.command = playerCmd(["scrobble"].concat(args))
        scrobbleProc.running = true
    }

    function ensureEvoplayerConnect() {
        if (playerSocket.connected)
            return
        playerSocket.connected = true
        if (!playerSocket.connected)
            connectRetryTimer.restart()
    }

    function ensurePlayerConnect() {
        ensureEvoplayerConnect()
    }

    function ensurePlayer() {
        if (useCliampBackend) {
            cliampBridge.ensurePlayer()
            return
        }
        if (playerSocket.connected)
            return
        if (!startPlayerProc.running)
            startPlayerProc.running = true
        else
            ensureEvoplayerConnect()
    }

    function subscribePlayer() {
        if (ipcSubscribed)
            return
        ipcSubscribed = true
        ipcWrite("subscribe")
        ipcWrite("state.get")
        ipcCall("job.status", {}, function(ok, msg) {
            if (!ok || !msg || !msg.data)
                return
            root.applyScanNotice(msg.data)
        })
    }

    Socket {
        id: playerSocket
        path: root.socketPath
        connected: false

        parser: SplitParser {
            onRead: line => root.handleIpcLine(line)
        }

        onConnectedChanged: {
            if (connected) {
                connectRetryTimer.stop()
                if (!root.useCliampBackend) {
                    root.ipcSubscribed = false
                    root.ipcSynced = false
                    Qt.callLater(root.subscribePlayer)
                }
                Qt.callLater(root.flushIpcWaitQueue)
            } else {
                root.ipcSubscribed = false
                if (!root.useCliampBackend) {
                    root.ipcSynced = false
                    root.player = { state: "stopped", volume: 100 }
                }
                root.failPendingIPC()
                if (!root.useCliampBackend)
                    Qt.callLater(root.ensurePlayer)
            }
        }
    }

    Timer {
        id: connectRetryTimer
        interval: 150
        repeat: true
        onTriggered: root.ensurePlayerConnect()
    }

    Process {
        id: startPlayerProc
        command: root.playerCmd(["start"])
        onExited: Qt.callLater(root.ensurePlayerConnect)
    }

    Timer {
        id: ipcTimeoutTimer
        interval: 500
        repeat: true
        running: root.ipcReady
        onTriggered: root.sweepIpcTimeouts()
    }

    Process {
        id: notifyArtProc
        property string requestedPath: ""
        property string pendingPath: ""

        stdout: StdioCollector {
            onStreamFinished: {
                var requested = String(notifyArtProc.requestedPath || "")
                notifyArtProc.requestedPath = ""
                var cached = String(text || "").trim()
                var currentPath = String(root.player.path || "")
                if (requested && currentPath === requested)
                    root.pushMediaNotification(cached)
                if (cached && requested)
                    root.notifyDisplayArtReady(requested, cached)
                var pending = String(notifyArtProc.pendingPath || "")
                if (pending) {
                    notifyArtProc.pendingPath = ""
                    notifyArtProc.requestedPath = pending
                    notifyArtProc.command = root.playerCmd(["art", "notify-cache", pending])
                    notifyArtProc.running = true
                }
            }
        }
    }

    Process {
        id: scrobbleProc
    }

    Component.onCompleted: {
        if (useCliampBackend)
            cliampBridge.ensurePlayer()
        else
            ensurePlayer()
    }

    IpcHandler {
        target: "evoplayer"

        function status(): string {
            return JSON.stringify(root.player || {})
        }

        function toggle(): string {
            root.runTransport("toggle")
            return "ok"
        }

        function playPause(): string {
            root.runTransport("toggle")
            return "ok"
        }

        function next(): string {
            root.runTransport("next")
            return "ok"
        }

        function previous(): string {
            root.runTransport("previous")
            return "ok"
        }

        function play(): string {
            root.runTransport("play")
            return "ok"
        }

        function pause(): string {
            root.runTransport("pause")
            return "ok"
        }
    }
}
