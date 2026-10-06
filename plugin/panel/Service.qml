import QtQuick
import Quickshell
import Quickshell.Io
import "compat"
import "compat/PluginIds.js" as PluginIds
import "../media/Model.js" as Model

Item {
    id: root

    property var shell: null
    property var player: ({})
    property bool openedOnStart: false

    onShellChanged: {
        if (!shell || openedOnStart)
            return
        openedOnStart = true
        Qt.callLater(function() {
            if (shell && typeof shell.summon === "function")
                shell.summon(PluginIds.pluginId, "{}")
        })
    }

    Loader {
        id: mediaLoader
        source: Qt.resolvedUrl("../media/Service.qml")
        onLoaded: item.shell = Qt.binding(function() { return root.shell })
    }

    readonly property var media: mediaLoader.item
    readonly property var activePlayer: media ? media.activePlayer : null
    readonly property bool bravePlaying: !!(media && media.bravePlaying)

    function runAction(action, showFeedback, targetKey) {
        if (!media || typeof media.runAction !== "function")
            return false
        return media.runAction(action, showFeedback, targetKey)
    }
    property var vizLevels: []
    property int vizSequence: 0
    property int vizGeneration: 0
    property string announcedPath: ""

    readonly property string home: Quickshell.env("HOME") || ""
    readonly property string socketPath: {
        var dir = Quickshell.env("XDG_RUNTIME_DIR") || ""
        if (!dir) return ""
        return dir + "/evoplayer.sock"
    }

    function playerCmd(args) {
        return Util.evoplayerCommand(home, args || [])
    }

    property int ipcNextReq: 1
    property bool ipcSubscribed: false
    property var ipcPending: ({})
    property var ipcWaitQueue: []
    property var ipcTimeouts: ({})

    readonly property int ipcTimeoutMs: 15000
    readonly property int ipcWaitQueueMax: 64
    readonly property int connectRetryMinMs: 150
    readonly property int connectRetryMaxMs: 5000

    property bool scanNoticeShown: false
    readonly property int scanNotifyId: 42421
    readonly property int notifyNormalMs: 5000
    readonly property int notifyLowMs: 3000

    function notify(opts) {
        var o = opts || {}
        var args = ["evo", "notify", "send", "--app-name", PluginIds.pluginId]
        if (o.urgency)
            args.push("-u", String(o.urgency))
        if (o.timeoutMs !== undefined)
            args.push("-t", String(o.timeoutMs))
        var image = String(o.image || "").trim()
        if (image)
            args.push("--image", image)
        args.push(Model.plain(o.summary || "Evoplayer", 180))
        var body = Model.plain(o.body || "", 180)
        if (body)
            args.push(body)
        if (o.replaceId !== undefined)
            args.push("--", "-r", String(o.replaceId))
        Quickshell.execDetached(args)
    }

    signal daemonJobUpdated(var data)

    readonly property bool ipcReady: playerSocket.connected
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
        if (!path)
            announcedPath = ""
        mergePlayer(patch)
    }

    function ipcCall(method, params, onDone) {
        if (!playerSocket.connected) {
            ensurePlayer()
            queueIpc(method, params, onDone)
            return
        }
        var id = ipcWrite(method, params)
        if (id < 0) {
            if (onDone)
                onDone(false, null)
            return
        }
        if (onDone) {
            ipcPending[id] = onDone
            ipcTimeouts[id] = Date.now() + ipcTimeoutMs
        }
    }

    function queueIpc(method, params, onDone) {
        if (ipcWaitQueue.length >= ipcWaitQueueMax) {
            var dropped = ipcWaitQueue.shift()
            if (dropped.onDone)
                dropped.onDone(false, null)
        }
        ipcWaitQueue.push({ method: method, params: params, onDone: onDone || null })
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
    }

    function mergePlayer(patch) {
        var prevPath = String(player.path || "")
        var prevState = String(player.state || "")
        var next = Object.assign({}, player, patch)
        var patchPath = String(patch.path || "")
        var samePath = patchPath !== "" && patchPath === prevPath
        if (samePath) {
            var metaKeys = ["title", "artist", "album", "art", "genre", "year", "label", "waveform", "liked", "queue_revision"]
            for (var i = 0; i < metaKeys.length; i++) {
                var key = metaKeys[i]
                if (key === "liked") {
                    if (typeof patch.liked !== "boolean")
                        next.liked = !!player.liked
                    continue
                }
                if (!String(patch[key] || "") && String(player[key] || ""))
                    next[key] = player[key]
            }
        }
        var newPath = String(next.path || "")
        player = next
        var state = String(player.state || "")
        if (newPath && state === "playing" && (newPath !== prevPath || state !== prevState))
            notifyNowPlaying()
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
            showScanNotice(msg, notifyNormalMs)
        }
    }

    function showScanNotice(body, timeoutMs) {
        var urgent = timeoutMs === 0
        notify({
            summary: "Evoplayer",
            body: String(body || ""),
            replaceId: scanNotifyId,
            urgency: urgent ? "critical" : "normal",
            timeoutMs: urgent ? 0 : Math.max(1, timeoutMs || notifyNormalMs)
        })
    }

    function applyWarmPayload(data) {
        if (!data || !data.path)
            return
        var path = String(data.path)
        if (String(player.path || "") === path)
            requestEnrich(path)
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
        if (!p)
            return
        enrichBusy = true
        enrichCurrentPath = p
        ipcCall("library.meta", { path: p }, function(ok, msg) {
            enrichBusy = false
            var requested = String(enrichCurrentPath || "")
            if (ok && msg && msg.data && requested && requested === String(root.enrichPath || "")) {
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

    function maybeWarmTrack() {
        var path = String(player.path || "")
        if (!path)
            return
        var hasArt = String(player.art || "").trim() !== ""
        if (hasArt)
            return
        ipcCallVoid("library.warm", { path: path })
    }

    function notifyDisplayArtReady(trackPath, artPath) {
        if (!shell || !shell.panelLoaders)
            return
        var loader = shell.panelLoaders[PluginIds.pluginId]
        if (loader && loader.item && typeof loader.item.applyDisplayArt === "function")
            loader.item.applyDisplayArt(trackPath, artPath)
    }

    function runTransport(action) {
        if (action === "toggle")
            ipcCallVoid("playback.toggle")
        else if (action === "next")
            ipcCallVoid("playback.next")
        else if (action === "prev" || action === "previous")
            ipcCallVoid("playback.prev")
        else if (action === "stop")
            ipcCallVoid("playback.stop")
        else if (action === "play") {
            if (String(player.state || "") !== "playing")
                ipcCallVoid("playback.toggle")
        } else if (action === "pause") {
            if (String(player.state || "") === "playing")
                ipcCallVoid("playback.toggle")
        }
    }

    function cacheDisplayArt(path) {
        if (!notifyArtProc.running) {
            notifyArtProc.requestedPath = path
            notifyArtProc.command = playerCmd(["art", "notify-cache", path])
            notifyArtProc.running = true
        } else {
            notifyArtProc.pendingPath = path
        }
    }

    function notifyNowPlaying() {
        if (!shell)
            return
        var path = String(player.path || "")
        if (!path || path === announcedPath)
            return
        announcedPath = path
        maybeWarmTrack()
        if (String(player.art || "") !== "")
            cacheDisplayArt(path)
    }

    function ensureEvoplayerConnect() {
        if (playerSocket.connected)
            return
        playerSocket.connected = true
        if (!connectRetryTimer.running)
            connectRetryTimer.start()
    }

    function ensurePlayer() {
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
        var applyState = function(ok, msg) {
            if (ok && msg && msg.data)
                root.applyStatePayload(msg.data)
        }
        ipcCall("subscribe", undefined, applyState)
        ipcCall("state.get", undefined, applyState)
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
                connectRetryTimer.interval = root.connectRetryMinMs
                root.ipcSubscribed = false
                root.ipcSynced = false
                Qt.callLater(root.subscribePlayer)
                Qt.callLater(root.flushIpcWaitQueue)
            } else {
                root.ipcSubscribed = false
                root.ipcSynced = false
                root.player = { state: "stopped", volume: 100 }
                root.failPendingIPC()
                Qt.callLater(root.ensurePlayer)
            }
        }
    }

    Timer {
        id: connectRetryTimer
        interval: root.connectRetryMinMs
        repeat: true
        onTriggered: {
            interval = Math.min(root.connectRetryMaxMs, interval * 2)
            root.ensureEvoplayerConnect()
        }
    }

    Process {
        id: startPlayerProc
        command: root.playerCmd(["start"])
        onExited: Qt.callLater(root.ensureEvoplayerConnect)
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
        onStarted: stdoutBuf = ""

        property string stdoutBuf: ""
        property string requestedPath: ""
        property string pendingPath: ""

        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                notifyArtProc.stdoutBuf += chunk
                if (notifyArtProc.stdoutBuf.length > 262144) {
                    notifyArtProc.signal(15)
                    notifyArtProc.stdoutBuf = ""
                }
            }
        }
        onExited: function(exitCode) {
            var requested = String(notifyArtProc.requestedPath || "")
            notifyArtProc.requestedPath = ""
            var cached = String(notifyArtProc.stdoutBuf || "").trim()
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

    Component.onCompleted: ensurePlayer()

    IpcHandler {
        target: "evoplayer"

        function status(): string {
            var p = root.player || {}
            return JSON.stringify({
                state: String(p.state || ""),
                title: String(p.title || "").slice(0, 120),
                artist: String(p.artist || "").slice(0, 120),
                album: String(p.album || "").slice(0, 120),
                playing: String(p.state || "") === "playing",
                position: Number(p.position) || 0,
                duration: Number(p.duration) || 0
            })
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
