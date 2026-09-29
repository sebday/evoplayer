import QtQuick
import Quickshell
import Quickshell.Io
import "compat"
import "panes"

Item {
    id: root

    property var shell: null
    property var host: null

    readonly property var service: shell && shell.serviceFor ? shell.serviceFor("evo.player") : null
    readonly property var player: service && service.player ? service.player : ({})
    readonly property string trackPath: String(player.path || "")
    readonly property bool playing: String(player.state || "") === "playing"
    readonly property bool shuffle: !!player.shuffle
    readonly property bool repeatOn: !!player.repeat
    readonly property bool liked: !!player.liked
    property int channels: 0
    readonly property int queueRevision: Number(player.queue_revision) || 0
    readonly property string waveFile: String(player.waveform || "")

    property bool active: false
    property bool loaded: false
    property string pane: "browse"
    property string mode: "queue"
    property bool textCapture: false
    property string err: ""
    property bool scanRunning: false

    property string browsePath: ""
    property bool browseFiles: false
    property string browseParent: ""
    property var browseEntries: []
    property var playlists: []
    property var sidebar: []
    property int browseIdx: 0
    property int browsePage: 12

    property var queue: []
    property int playlistIdx: 0
    property int reorderPending: 0
    property real reorderScroll: -1
    property real playlistScroll: 0
    property string shownPlaylist: ""
    property var shownTracks: []
    property int shownGen: 0
    property var discoverTracks: []
    property int discoverIdx: 0
    property string discoverNote: ""
    property string discoverSeedTitle: ""
    property string downloadUrl: ""
    property string downloadQuery: ""
    property var downloadHits: []
    property int downloadHit: -1
    property string downloadPreviewId: ""
    property string downloadPreviewURL: ""
    property int downloadIdx: 0
    property string downloadNote: ""
    property string downloadLog: ""
    property var downloadFiles: []
    property bool scSearchBusy: false
    property int playlistPage: 16
    property int savedPlaylistIdx: 0
    property string savedFocus: "browse"

    property string searchQuery: ""
    property var searchHits: []
    property int searchGen: 0

    property bool vizOn: true
    property bool vizSubscribed: false
    property var peaks: []
    property real seekPreview: -1

    onPlayerChanged: {
        if (seekPreview < 0 || seekTimer.running)
            return
        var pos = Number(player.position) || 0
        if (Math.abs(pos - seekPreview) < 1.25)
            seekPreview = -1
    }
    readonly property real shownPosition: seekPreview >= 0 ? seekPreview : (Number(player.position) || 0)

    property string artOverride: ""
    property string artOverridePath: ""
    property int artEpoch: 0
    property var artHits: []
    property int artIdx: 0
    property bool artBusy: false
    property bool artFromDrop: false
    property string artPreview: ""

    property var moveFolders: []
    property int moveIdx: 0
    property bool moveBusy: false
    property string movePath: ""

    property string tagPath: ""
    property bool tagBusy: false
    property int tagFocus: 0
    property string tagTitle: ""
    property string tagArtist: ""
    property string tagAlbum: ""
    property string tagYear: ""
    property string tagGenre: ""
    property string tagLabel: ""

    property string musicRoot: ""

    property string _waveBuf: ""
    property bool _waveOverflow: false
    property string _artBuf: ""
    property bool _artOverflow: false
    property string _applyBuf: ""
    property bool _applyOverflow: false

    onQueueRevisionChanged: {
        if (active && reorderPending === 0)
            loadQueue()
    }
    onTrackPathChanged: {
        artOverride = ""
        artOverridePath = ""
        seekPreview = -1
        loadWave()
        loadChannels()
        revealPlaying()
    }
    onWaveFileChanged: loadWave()
    onPlayingChanged: syncViz()
    onVizOnChanged: syncViz()
    onArtIdxChanged: showArtPreview()

    function activate() {
        active = true
        loadChannels()
        if (!loaded) {
            loaded = true
            loadBrowse("")
            loadPlaylistIndex()
            loadQueue()
            loadRoot()
            if (service)
                service.ipcCall("job.status", {}, function(ok, msg) {
                    if (ok && msg && msg.data)
                        root.scanRunning = String(msg.data.name || "") === "scan" && String(msg.data.status || "") === "running"
                })
        }
        syncViz()
        loadWave()
    }

    function deactivate() {
        active = false
        syncViz()
    }

    function ipc(method, params, done, failed) {
        if (!service || !service.ipcCall) {
            err = "not connected"
            if (failed)
                failed()
            return
        }
        service.ipcCall(method, params === null ? undefined : params, function(ok, msg) {
            if (!ok) {
                root.err = msg && msg.error ? String(msg.error) : "request failed"
                if (failed)
                    failed()
                return
            }
            if (done)
                done(msg ? msg.data : null)
        })
    }

    function clock(sec) {
        var total = Math.max(0, Math.floor(Number(sec) || 0))
        if (total <= 0)
            return ""
        var s = total % 60
        return Math.floor(total / 60) + ":" + (s < 10 ? "0" : "") + s
    }

    function trackClock(t) {
        var label = String(t && t.duration_label || "")
        if (label && label !== "0:00")
            return label
        return clock(t ? t.duration : 0)
    }

    function trackLabel(t) {
        var title = String(t && t.title || "").replace(/^\s+|\s+$/g, "")
        if (!title)
            title = String(t && t.path || "")
        var artist = String(t && t.artist || "").replace(/^\s+|\s+$/g, "")
        if (!artist)
            return title
        return artist + " — " + title
    }

    function nowTitle() {
        if (!trackPath)
            return "NOTHING PLAYING"
        var title = String(player.title || trackPath)
        var artist = String(player.artist || "").replace(/^\s+|\s+$/g, "")
        var msg = artist ? artist + " - " + title : title
        return msg.toUpperCase()
    }

    function nowRelease() {
        var album = String(player.album || "").replace(/^\s+|\s+$/g, "")
        var label = String(player.label || "").replace(/^\s+|\s+$/g, "")
        var year = String(player.year || "").replace(/^\s+|\s+$/g, "")
        var parts = []
        if (album)
            parts.push(album)
        if (label && label.toLowerCase() !== album.toLowerCase())
            parts.push(label)
        if (year)
            parts.push(year)
        return parts.join("  ").toUpperCase()
    }

    function progressFrac() {
        var dur = Number(player.duration) || 0
        if (dur <= 0)
            return 0
        var p = shownPosition / dur
        if (p < 0)
            return 0
        if (p > 1)
            return 1
        return p
    }

    function volumeFrac() {
        var v = Number(player.volume)
        if (!isFinite(v))
            v = 100
        if (v < 0)
            v = 0
        if (v > 100)
            v = 100
        return v / 100
    }

    function playerState() {
        return String(player.state || "")
    }

    function artLegend() {
        var year = String(player.year || "").replace(/^\s+|\s+$/g, "")
        return year || "artwork"
    }

    function artSource() {
        var url = ""
        if (mode === "art" && artPreview)
            url = artPreview
        else if (artOverride && artOverridePath === trackPath)
            url = Util.fileUrl(artOverride)
        else
            url = Util.fileUrl(String(player.art || ""))
        if (url && artEpoch)
            url += "#" + artEpoch
        return url
    }

    function artHitLabel(hit) {
        return String(hit && (hit.label || hit.url) || "")
    }

    function safeArtURL(value) {
        var s = String(value || "")
        if (s.indexOf("https://") === 0 || s.indexOf("http://") === 0)
            return s
        return ""
    }

    function nowHints() {
        return [
            { key: "space", label: playing ? "pause" : "play" },
            { key: "/", label: "find" },
            { key: "h", label: "help" }
        ]
    }

    function playlistLegend() {
        if (mode === "tags")
            return "edit tags"
        if (mode === "move")
            return "move"
        if (mode === "art")
            return "cover"
        if (mode === "download")
            return "download"
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        if (q)
            return "search (" + ((searchHits && searchHits.length) || 0) + ")"
        if (mode === "help")
            return "help"
        if (mode === "discover")
            return "discover (" + ((discoverTracks && discoverTracks.length) || 0) + ")"
        if (shownPlaylist)
            return playlistLabel(shownPlaylist) + " (" + ((shownTracks && shownTracks.length) || 0) + ")"
        return "playlist (" + (queue ? queue.length : 0) + ")"
    }

    function playlistHints() {
        if (mode === "move")
            return [{ key: "⏎", label: "move" }]
        if (mode === "art")
            return [{ key: "⏎", label: "set" }]
        if (mode === "download") {
            var hints = []
            if (downloadHit >= 0)
                hints = [{ key: "⏎", label: "play" }, { key: "d", label: "download" }, { key: "l", label: "like" }, { key: "↑", label: "search" }]
            else if (downloadIdx === 1)
                hints = [{ key: "⏎", label: "search" }, { key: "↓", label: "results" }]
            else
                return [{ key: "⏎", label: "download" }, { key: "s", label: "sync" }]
            var withSync = []
            for (var i = 0; i < hints.length; i++) {
                withSync.push(hints[i])
                if (hints[i].label === "search")
                    withSync.push({ key: "s", label: "sync" })
            }
            return withSync
        }
        if (mode === "help" || mode === "tags")
            return []
        if (mode === "discover")
            return [
                { key: "⏎", label: "preview" },
                { key: "k", label: "keep" },
                { key: "x", label: "dismiss" },
                { key: "n", label: "more" }
            ]
        var hints = [
            { key: "l", label: "like playing" },
            { key: "m", label: "move" },
            { key: "e", label: "edit" }
        ]
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        if (!q && !shuffle && pane === "playlist" && !shownPlaylist)
            hints.push({ key: "⇧↑↓", label: "reorder" })
        return hints
    }

    function playlistLabel(name) {
        if (name === "all")
            return "likes"
        if (name === "current")
            return "now playing"
        return String(name || "")
    }

    function orderedPlaylists() {
        var items = playlists || []
        var current = null
        var rest = []
        for (var i = 0; i < items.length; i++) {
            if (items[i].name === "current")
                current = items[i]
            else
                rest.push(items[i])
        }
        if (!current)
            return rest
        return [current].concat(rest)
    }

    function rebuildSidebar() {
        var rows = []
        if (!browseFiles && !browsePath) {
                rows.push({
                    kind: "entry",
                    type: "dir",
                    label: "filesystem",
                    path: "",
                    count: 0,
                    id: "filesystem"
                })
            } else {
                var entries = browseEntries || []
                for (var e = 0; e < entries.length; e++) {
                    var en = entries[e]
                    if (String(en.type || "") !== "dir")
                        continue
                    var name = String(en.name || en.path || "")
                    if (name && name.charAt(name.length - 1) !== "/")
                        name += "/"
                    rows.push({
                        kind: "entry",
                        type: "dir",
                        label: name,
                        path: String(en.path || ""),
                        count: Number(en.count) || 0,
                        id: ""
                    })
                }
            }
            var lists = orderedPlaylists()
            if (rows.length && lists.length)
                rows.push({ kind: "rule", type: "", label: "", path: "", count: 0, id: "" })
            for (var p = 0; p < lists.length; p++) {
                var it = lists[p]
                rows.push({
                    kind: "playlist",
                    type: "",
                    label: playlistLabel(it.name),
                    path: "",
                    count: Number(it.count) || 0,
                    id: String(it.name || "")
                })
            }
            if (rows.length)
                rows.push({ kind: "rule", type: "", label: "", path: "", count: 0, id: "" })
            rows.push({ kind: "tool", type: "", label: "download", path: "", count: 0, id: "download" })
            rows.push({ kind: "tool", type: "", label: "discover", path: "", count: 0, id: "discover" })
        sidebar = rows
        if (browseIdx >= rows.length)
            browseIdx = Math.max(0, rows.length - 1)
        syncSettingsMode()
        syncShownPlaylist()
    }

    function syncSettingsMode() {
        var row = sidebar[browseIdx]
        var tool = row && row.kind === "tool" ? String(row.id || "") : ""
        if (tool === "discover") {
            if (mode !== "discover")
                loadDiscover(0)
            return
        }
        if (tool === "download") {
            if (mode !== "download")
                openDownload()
            return
        }
        if (mode === "discover" || mode === "download")
            mode = "queue"
    }

    function openDownload() {
        mode = "download"
        downloadIdx = 0
        err = ""
        loadIncoming()
    }

    function submitDownload(explicitUrl) {
        var fromBox = explicitUrl === undefined || explicitUrl === null
        var url = String(fromBox ? downloadUrl : explicitUrl).replace(/^\s+|\s+$/g, "")
        if (!url) {
            err = fromBox ? "paste a url" : "no soundcloud link"
            return
        }
        err = ""
        downloadNote = "downloading…"
        ipc("library.download", { url: url, import: true }, function() {
            root.err = ""
            if (fromBox)
                root.downloadUrl = ""
        }, function() {
            root.downloadNote = ""
        })
    }

    function searchSoundCloud() {
        var q = String(downloadQuery || "").replace(/^\s+|\s+$/g, "")
        if (!q) {
            err = "type a search"
            return
        }
        if (scSearchBusy)
            return
        err = ""
        downloadNote = "searching…"
        scSearchBusy = true
        ipc("library.soundcloud.search", { query: q, limit: 12 }, function(data) {
            root.scSearchBusy = false
            root.applySoundCloudSearch(data)
        }, function() {
            root.scSearchBusy = false
            root.downloadNote = ""
        })
    }

    function applySoundCloudSearch(data) {
        downloadNote = ""
        err = ""
        downloadHits = (data && Array.isArray(data.tracks)) ? data.tracks : []
        if (downloadHits.length) {
            downloadHit = 0
            downloadNote = ""
        } else {
            downloadHit = -1
            downloadNote = "nothing on soundcloud"
        }
    }

    function selectedDownloadHit() {
        var rows = downloadHits || []
        if (downloadHit < 0 || downloadHit >= rows.length)
            return null
        return rows[downloadHit]
    }

    function playSoundCloudHit() {
        var row = selectedDownloadHit()
        if (!row || !row.id) {
            err = "nothing to play"
            return
        }
        err = ""
        downloadNote = "playing…"
        downloadPreviewId = String(row.id)
        downloadPreviewURL = String(row.permalink || "")
        ipc("discover.preview", { id: Number(downloadPreviewId) }, function(data) {
            if (data && data.path)
                root.downloadNote = ""
        }, function() {
            root.downloadNote = ""
        })
    }

    function downloadSoundCloudHit() {
        var row = selectedDownloadHit()
        submitDownload(row && row.permalink ? String(row.permalink) : "")
    }

    function importPlayingPreview() {
        if (!downloadPreviewId || !downloadPreviewURL)
            return false
        if (trackPath.indexOf("/discover/" + downloadPreviewId + ".mp3") < 0)
            return false
        submitDownload(downloadPreviewURL)
        return true
    }

    function syncLikes() {
        err = ""
        downloadNote = "syncing likes…"
        ipc("library.soundcloud.download", { import: false }, null, function() {
            root.downloadNote = ""
        })
    }

    function jobError(data) {
        var shown = String(data && data.error || "failed")
        if (shown.indexOf("worker exited") < 0)
            return shown
        var lines = String(data && data.log || "").split("\n")
        for (var i = lines.length - 1; i >= 0; i--) {
            var line = String(lines[i] || "").replace(/^\s+|\s+$/g, "").replace(/^·\s*/, "")
            if (!line || line.indexOf("preview ") === 0)
                continue
            return line
        }
        return shown
    }

    function noteDownloadJob(data) {
        var status = String(data.status || "")
        downloadLog = tailJobLog(data.log)
        var phase = data.progress && data.progress.phase ? String(data.progress.phase) : ""
        if (status === "running") {
            downloadNote = phase || "downloading…"
            return
        }
        if (status === "error") {
            downloadNote = ""
            err = String(data.error || "download failed")
            return
        }
        if (status === "done") {
            downloadNote = "done"
            err = ""
            if (data.result && data.result.files)
                downloadFiles = data.result.files
            else
                loadIncoming()
        }
    }

    function loadIncoming() {
        ipc("library.incoming.list", {}, function(data) {
            root.downloadFiles = (data && data.files) || []
        })
    }

    function tailJobLog(log) {
        var lines = String(log || "").split("\n")
        var out = []
        for (var i = 0; i < lines.length; i++) {
            var line = String(lines[i] || "").replace(/^\s+|\s+$/g, "")
            if (line)
                out.push(line)
        }
        if (out.length > 12)
            out = out.slice(out.length - 12)
        return out.join("\n")
    }

    function closeTool() {
        var rows = sidebar || []
        if (rows[browseIdx] && rows[browseIdx].kind === "tool") {
            var n = browseIdx - 1
            while (n >= 0 && rows[n] && rows[n].kind === "rule")
                n--
            if (n >= 0)
                browseIdx = n
        }
        mode = "queue"
        pane = "browse"
        syncShownPlaylist()
        err = ""
    }

    function closeDownload() {
        closeTool()
        downloadIdx = 0
        textCapture = false
    }

    function loadDiscover(id) {
        mode = "discover"
        discoverNote = "finding similar…"
        err = ""
        var params = {}
        if (id)
            params.id = Number(id)
        else if (trackPath)
            params.path = trackPath
        else {
            discoverTracks = []
            discoverSeedTitle = ""
            discoverNote = ""
            err = "nothing playing"
            return
        }
        discoverCall("discover.similar", params, function(data) {
            if (root.mode !== "discover")
                return
            data = data || {}
            root.discoverTracks = data.tracks || []
            var artist = String(data.seed_artist || "").replace(/^\s+|\s+$/g, "")
            var title = String(data.seed_title || "").replace(/^\s+|\s+$/g, "")
            root.discoverSeedTitle = artist && title ? artist + " — " + title : (artist || title)
            root.discoverIdx = 0
            root.showDiscoverSeed()
        })
    }

    function showDiscoverSeed() {
        if (mode !== "discover")
            return
        if (discoverSeedTitle)
            discoverNote = discoverSeedTitle
        else if (!(discoverTracks && discoverTracks.length))
            discoverNote = "nothing new"
        else
            discoverNote = ""
    }

    function discoverCall(method, params, done) {
        ipc(method, params, function(data) {
            root.err = ""
            if (done)
                done(data)
        }, function() {
            root.discoverNote = ""
        })
    }

    function currentDiscover() {
        var rows = discoverTracks || []
        if (discoverIdx < 0 || discoverIdx >= rows.length)
            return null
        return rows[discoverIdx]
    }

    function stepDiscover(delta) {
        var len = (discoverTracks || []).length
        if (len < 1 || !delta)
            return
        var step = delta < 0 ? -1 : 1
        var left = Math.abs(delta)
        var i = discoverIdx
        while (left > 0) {
            var n = i + step
            if (n < 0 || n >= len)
                break
            i = n
            left--
        }
        discoverIdx = i
    }

    function discoverMark(row) {
        if (!row)
            return ""
        if (row.kept)
            return "ok"
        var id = String(row.id || "")
        if (id && trackPath.indexOf("/discover/" + id + ".mp3") >= 0)
            return ">"
        return ""
    }

    function previewDiscover() {
        var row = currentDiscover()
        if (!row || !row.id)
            return
        discoverNote = "previewing…"
        discoverCall("discover.preview", { id: Number(row.id) }, function(data) {
            data = data || {}
            if (data.path)
                root.showDiscoverSeed()
        })
    }

    function keepDiscover() {
        var row = currentDiscover()
        if (!row || !row.id || row.kept)
            return
        var id = row.id
        discoverNote = "keeping…"
        discoverCall("discover.keep", { id: Number(id) }, function(data) {
            data = data || {}
            if (data.path || String(data.status || "") === "done") {
                root.markDiscoverKept(id)
                root.showDiscoverSeed()
                return
            }
            if (String(data.status || "") !== "running")
                root.showDiscoverSeed()
        })
    }

    function dismissDiscover() {
        var row = currentDiscover()
        if (!row || !row.id)
            return
        var id = row.id
        discoverCall("discover.dismiss", { id: Number(id) }, function() {
            root.removeDiscover(id)
        })
    }

    function moreDiscover() {
        var row = currentDiscover()
        if (!row || !row.id)
            return
        loadDiscover(row.id)
    }

    function markDiscoverKept(id) {
        var next = []
        var rows = discoverTracks || []
        for (var i = 0; i < rows.length; i++) {
            var row = rows[i]
            if (Number(row.id) === Number(id))
                row = Object.assign({}, row, { kept: true })
            next.push(row)
        }
        discoverTracks = next
    }

    function removeDiscover(id) {
        var next = []
        var rows = discoverTracks || []
        for (var i = 0; i < rows.length; i++) {
            if (Number(rows[i].id) !== Number(id))
                next.push(rows[i])
        }
        discoverTracks = next
        if (discoverIdx >= next.length)
            discoverIdx = Math.max(0, next.length - 1)
        showDiscoverSeed()
    }

    function closeDiscover() {
        closeTool()
        discoverNote = ""
    }

    function clickDiscover(index) {
        discoverIdx = index
        focusPane("playlist")
    }

    function loadBrowse(rel, stayIfLeaf) {
        ipc("library.browse", { path: String(rel || ""), offset: 0, limit: 400 }, function(data) {
            data = data || {}
            var entries = data.entries || []
            if (stayIfLeaf) {
                var dirs = 0
                for (var i = 0; i < entries.length; i++) {
                    if (String(entries[i].type || "") === "dir")
                        dirs++
                }
                if (dirs === 0)
                    return
            }
            browsePath = String(data.path || "")
            browseParent = data.parent == null ? "" : String(data.parent)
            browseEntries = entries
            browseIdx = 0
            rebuildSidebar()
        })
    }

    function loadPlaylistIndex() {
        ipc("library.playlist.list", undefined, function(data) {
            playlists = Array.isArray(data) ? data : []
            rebuildSidebar()
        })
    }

    function applyQueue(items, keep) {
        queue = items || []
        var idx = 0
        if (keep) {
            for (var i = 0; i < queue.length; i++) {
                if (queue[i].path === keep) {
                    idx = i
                    break
                }
            }
        }
        if (shownPlaylist)
            return
        playlistIdx = idx
        revealPlaying()
    }

    function loadQueue() {
        var keep = queue[playlistIdx] ? String(queue[playlistIdx].path || "") : ""
        loadPlaylistItems("current", function(items) {
            root.applyQueue(items, keep)
        })
    }

    function revealPlaying() {
        if (shownPlaylist)
            return
        if (pane === "playlist" && mode === "queue")
            return
        for (var i = 0; i < queue.length; i++) {
            if (String(queue[i].path || "") === trackPath) {
                playlistIdx = i
                return
            }
        }
    }

    function stepBrowse(delta) {
        var rows = sidebar || []
        if (!rows.length || !delta)
            return
        var step = delta < 0 ? -1 : 1
        var left = Math.abs(delta)
        var i = browseIdx
        while (left > 0) {
            var n = i + step
            if (n < 0 || n >= rows.length)
                break
            i = n
            if (!rows[i] || rows[i].kind !== "rule")
                left--
        }
        browseIdx = i
        syncSettingsMode()
        syncShownPlaylist()
    }

    function stepIndex(name, len, delta) {
        if (len < 1 || !delta)
            return
        var cur = name === "playlist" ? playlistIdx : (name === "move" ? moveIdx : artIdx)
        var step = delta < 0 ? -1 : 1
        var left = Math.abs(delta)
        var i = cur
        while (left > 0) {
            var n = i + step
            if (n < 0 || n >= len)
                break
            i = n
            left--
        }
        if (name === "playlist")
            playlistIdx = i
        else if (name === "move")
            moveIdx = i
        else
            artIdx = i
    }

    function moveActive(delta) {
        if (mode === "move")
            stepIndex("move", (moveFolders || []).length, delta)
        else if (mode === "art")
            stepIndex("art", (artHits || []).length, delta)
        else if (pane === "playlist" && mode === "queue") {
            var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
            var rows = q ? (searchHits || []) : (shownPlaylist ? shownTracks : queue)
            stepIndex("playlist", rows.length, delta)
        }
        else
            stepBrowse(delta)
    }

    function currentRow() {
        return sidebar[browseIdx] || null
    }

    function enterFolder() {
        var row = currentRow()
        if (!row || row.kind !== "entry" || row.type !== "dir")
            return
        if (row.id === "filesystem") {
            browseFiles = true
            browseIdx = 0
            rebuildSidebar()
            return
        }
        loadBrowse(row.path, true)
    }

    function leaveFolder() {
        if (!browsePath) {
            if (!browseFiles)
                return
            browseFiles = false
            browseIdx = 0
            rebuildSidebar()
            return
        }
        loadBrowse(browseParent)
    }

    function playPaths(paths, start) {
        var list = []
        for (var i = 0; i < paths.length; i++) {
            if (paths[i])
                list.push(String(paths[i]))
        }
        if (!list.length) {
            err = "no tracks"
            return
        }
        ipc("queue.replace", { paths: list, start_path: start || list[0] }, function() {
            root.err = ""
            root.loadQueue()
            root.loadPlaylistIndex()
        })
    }

    function playFolder(rel) {
        if (!rel)
            return
        ipc("library.browse", { path: String(rel), queue: true, queue_paths_only: true }, function(data) {
            var paths = (data && data.paths) || []
            root.playPaths(paths, paths[0] || "")
        })
    }

    function showPlaylist(name) {
        name = String(name || "")
        if (!name)
            return
        if (shownPlaylist === name)
            return
        shownPlaylist = name
        shownTracks = []
        playlistIdx = 0
        var gen = ++shownGen
        loadPlaylistItems(name, function(items) {
            if (gen !== root.shownGen || root.shownPlaylist !== name)
                return
            root.shownTracks = items
        })
    }

    function clearShownPlaylist() {
        if (!shownPlaylist)
            return
        shownPlaylist = ""
        shownTracks = []
        shownGen++
    }

    function syncShownPlaylist() {
        var row = sidebar[browseIdx]
        if (row && row.kind === "playlist" && row.id && row.id !== "current")
            showPlaylist(row.id)
        else
            clearShownPlaylist()
    }

    function loadPlaylistItems(name, done) {
        ipc("library.playlist.tracks", { name: name, offset: 0, limit: 500 }, function(data) {
            data = data || {}
            var items = data.items || []
            var total = Number(data.total) || items.length
            if (total > items.length && total <= 8000) {
                root.ipc("library.playlist.tracks", { name: name, offset: 0, limit: total }, function(full) {
                    done((full && full.items) || [])
                })
                return
            }
            done(items)
        })
    }

    function openBrowseAt(index) {
        browseIdx = index
        var row = sidebar[index]
        if (!row)
            return
        if (row.kind === "entry" && row.type === "dir") {
            enterFolder()
            return
        }
        playBrowseAt(index)
    }

    function playBrowseAt(index) {
        browseIdx = index
        var row = sidebar[index]
        if (!row || row.kind === "tool" || row.kind === "rule")
            return
        if (row.kind === "playlist") {
            playPlaylist(row.id)
            return
        }
        if (row.id === "filesystem") {
            enterFolder()
            return
        }
        if (row.type === "dir") {
            playFolder(row.path)
            return
        }
        if (row.type === "track") {
            var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
            if (q)
                playTrackList(searchHits || [], row.path)
            else
                playPaths([row.path], row.path)
        }
    }

    function playPlaylist(name) {
        if (!name)
            return
        loadPlaylistItems(name, function(items) {
            root.playTrackList(items, "")
        })
    }

    function playTrackList(items, start) {
        var paths = []
        for (var i = 0; i < items.length; i++) {
            if (items[i] && items[i].path)
                paths.push(items[i].path)
        }
        playPaths(paths, start || (paths[0] || ""))
    }

    function playSelected() {
        if (mode === "art") {
            applyArt("album")
            return
        }
        if (mode === "move") {
            applyMove()
            return
        }
        if (mode === "download") {
            if (downloadHit >= 0)
                playSoundCloudHit()
            else if (downloadIdx === 1)
                searchSoundCloud()
            else
                submitDownload()
            return
        }
        if (mode === "help" || mode === "tags")
            return
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        if (q && (pane === "playlist" || pane === "search")) {
            var hit = (searchHits || [])[playlistIdx]
            if (hit && hit.path)
                playTrackList(searchHits, hit.path)
            return
        }
        if (pane === "playlist") {
            if (shownPlaylist) {
                var shown = selectedListTrack()
                if (shown && shown.path)
                    playTrackList(shownTracks, shown.path)
                return
            }
            var track = queue[playlistIdx]
            if (track && track.path)
                ipc("queue.play_path", { path: track.path }, null)
            return
        }
        var row = currentRow()
        if (!row)
            return
        if (row.kind === "tool") {
            if (row.id === "discover") {
                if (mode !== "discover")
                    loadDiscover(0)
                pane = "playlist"
                return
            }
            if (row.id === "download") {
                if (mode !== "download")
                    openDownload()
                pane = "playlist"
                return
            }
            return
        }
        if (row.kind === "playlist") {
            playPlaylist(row.id)
            return
        }
        if (row.id === "filesystem") {
            enterFolder()
            return
        }
        if (row.type === "dir") {
            playFolder(row.path)
            return
        }
        if (row.type === "track") {
            var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
            if (q)
                playTrackList(searchHits || [], row.path)
            else
                playPaths([row.path], row.path)
        }
    }

    function addDir() {
        if (pane === "playlist")
            return
        var row = currentRow()
        if (!row || row.kind !== "entry")
            return
        var rel = row.type === "dir" ? row.path : browsePath
        if (!rel)
            return
        ipc("queue.append_folder", { path: rel }, function() {
            root.err = ""
            root.loadQueue()
            root.loadPlaylistIndex()
        })
    }

    function absPath(rel) {
        var p = String(rel || "")
        if (!p)
            return ""
        if (p.charAt(0) === "/")
            return p
        if (!musicRoot)
            return ""
        return musicRoot.replace(/\/$/, "") + "/" + p
    }

    function folderTarget() {
        var row = currentRow()
        if (!row)
            return ""
        if (row.kind === "entry" && row.type === "dir")
            return absPath(row.path)
        if (row.kind === "entry" && row.type === "track" && row.path) {
            var slash = row.path.lastIndexOf("/")
            return slash > 0 ? row.path.slice(0, slash) : ""
        }
        return ""
    }

    function openFolder() {
        var target = folderTarget()
        if (!target || openProc.running)
            return
        openProc.command = ["xdg-open", target]
        openProc.running = true
    }

    function patchLiked(path, likedNow) {
        if (reorderPending === 0)
            reorderScroll = playlistScroll
        queue = withLiked(queue, path, likedNow)
        if (shownPlaylist)
            shownTracks = withLiked(shownTracks, path, likedNow)
        if (service && String(player.path || "") === path && service.mergePlayer)
            service.mergePlayer({ path: path, liked: likedNow })
    }

    function toggleLike(path) {
        if (!path)
            return
        ipc("library.favorite.toggle", { path: path }, function(data) {
            root.err = ""
            root.patchLiked(path, !!(data && data.liked))
        })
    }

    function withLiked(rows, path, likedNow) {
        var next = []
        var list = rows || []
        for (var i = 0; i < list.length; i++) {
            var row = list[i]
            if (row.path === path)
                next.push(Object.assign({}, row, { liked: likedNow }))
            else
                next.push(row)
        }
        return next
    }

    function selectedListTrack() {
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        var rows = q ? (searchHits || []) : (shownPlaylist ? shownTracks : queue)
        if (!rows || playlistIdx < 0 || playlistIdx >= rows.length)
            return null
        return rows[playlistIdx]
    }

    function likePlaying() {
        if (mode === "download" && downloadHit >= 0) {
            downloadSoundCloudHit()
            return
        }
        if (importPlayingPreview())
            return
        toggleLike(trackPath)
    }

    function pushOverlay(next) {
        if (mode === "queue") {
            savedPlaylistIdx = playlistIdx
            savedFocus = pane === "search" ? "browse" : pane
        }
        mode = next
        pane = "playlist"
        textCapture = false
    }

    function closeMode() {
        var back = savedFocus || "browse"
        mode = "queue"
        artPreview = ""
        artHits = []
        artBusy = false
        moveBusy = false
        tagBusy = false
        playlistIdx = savedPlaylistIdx
        pane = back
        textCapture = false
        err = ""
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
        rebuildSidebar()
    }

    function toggleHelp() {
        if (mode === "help")
            closeMode()
        else
            pushOverlay("help")
    }

    function openSearch() {
        if (mode !== "queue")
            closeMode()
        pane = "search"
    }

    function clearSearch() {
        searchQuery = ""
        searchHits = []
        searchGen++
        pane = "browse"
        textCapture = false
        rebuildSidebar()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function setSearch(text) {
        searchQuery = String(text || "")
        searchTimer.restart()
    }

    function runSearch() {
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        searchGen++
        var gen = searchGen
        if (!q) {
            searchHits = []
            return
        }
        ipc("library.search", { mode: "search", query: q }, function(data) {
            if (gen !== root.searchGen)
                return
            var items = Array.isArray(data) ? data : []
            if (items.length > 400)
                items = items.slice(0, 400)
            root.searchHits = items
            root.playlistIdx = 0
        })
    }

    function openMove() {
        if (pane !== "playlist" || mode !== "queue")
            return
        var track = selectedListTrack()
        if (!track || !track.path)
            return
        movePath = track.path
        moveIdx = 0
        moveBusy = false
        pushOverlay("move")
        if (!browsePath && (browseEntries || []).length) {
            moveFolders = dirNames(browseEntries)
            return
        }
        ipc("library.browse", { path: "", offset: 0, limit: 400 }, function(data) {
            root.moveFolders = root.dirNames((data && data.entries) || [])
        })
    }

    function dirNames(entries) {
        var out = []
        for (var i = 0; i < entries.length; i++) {
            if (String(entries[i].type || "") === "dir")
                out.push(String(entries[i].name || entries[i].path || ""))
        }
        return out
    }

    function applyMove() {
        if (moveBusy || moveIdx < 0 || moveIdx >= moveFolders.length || !movePath)
            return
        moveBusy = true
        var folder = moveFolders[moveIdx]
        ipc("library.track.move", { path: movePath, folder: folder }, function() {
            root.moveBusy = false
            root.err = ""
            root.closeMode()
            root.loadBrowse(root.browsePath)
            root.loadQueue()
        })
    }

    function openTags() {
        if (pane !== "playlist" || mode !== "queue")
            return
        var track = selectedListTrack()
        if (!track || !track.path)
            return
        tagPath = track.path
        tagBusy = true
        tagFocus = 0
        tagTitle = ""
        tagArtist = ""
        tagAlbum = ""
        tagYear = ""
        tagGenre = ""
        tagLabel = ""
        pushOverlay("tags")
        ipc("library.track.tags.get", { path: tagPath }, function(data) {
            data = data || {}
            root.tagBusy = false
            root.tagTitle = String(data.title || "")
            root.tagArtist = String(data.artist || "")
            root.tagAlbum = String(data.album || "")
            root.tagYear = String(data.year || "")
            root.tagGenre = String(data.genre || "")
            root.tagLabel = String(data.label || "")
        })
    }

    function moveTag(delta) {
        var next = tagFocus + delta
        if (next >= 6) {
            saveTags()
            return
        }
        if (next < 0)
            next = 5
        tagFocus = next
    }

    function saveTags() {
        if (!tagPath || tagBusy)
            return
        tagBusy = true
        ipc("library.track.tags.set", {
            path: tagPath,
            title: tagTitle,
            artist: tagArtist,
            album: tagAlbum,
            year: tagYear,
            genre: tagGenre,
            label: tagLabel
        }, function() {
            root.tagBusy = false
            root.err = ""
            root.closeMode()
            root.loadQueue()
        })
    }

    function openArt() {
        if (mode === "help" || !trackPath)
            return
        if (artSearchProc.running)
            return
        artHits = []
        artIdx = 0
        artPreview = ""
        artBusy = true
        pushOverlay("art")
        _artBuf = ""
        _artOverflow = false
        artSearchProc.command = [Util.evoplayerBinPath(service ? service.home : ""), "art", "search", trackPath, "--json"]
        artSearchProc.running = true
    }

    function showArtPreview() {
        var hit = artHits[artIdx]
        if (!hit) {
            artPreview = ""
            return
        }
        artPreview = safeArtURL(hit.thumb) || safeArtURL(hit.url)
    }

    function applyArt(scope) {
        if (artBusy || mode !== "art")
            return
        var hit = artHits[artIdx]
        var url = hit ? safeArtURL(hit.url) || safeArtURL(hit.thumb) : ""
        if (!url || !trackPath || artApplyProc.running)
            return
        artBusy = true
        artFromDrop = false
        var cmd = [Util.evoplayerBinPath(service ? service.home : ""), "art", "apply", trackPath, url, "--json"]
        if (scope === "album")
            cmd.splice(cmd.length - 1, 0, "--album")
        _applyBuf = ""
        _applyOverflow = false
        artApplyProc.command = cmd
        artApplyProc.running = true
    }

    function dropArt(raw) {
        if (!trackPath) {
            err = "nothing playing"
            return false
        }
        if (artBusy || artApplyProc.running)
            return false
        var value = String(raw || "").replace(/\r/g, "").split("\n")[0].replace(/^\s+|\s+$/g, "")
        if (!value)
            return false
        var local = ""
        if (value.indexOf("file://") === 0) {
            local = value.slice("file://".length)
            try {
                local = decodeURIComponent(local)
            } catch (e) {
            }
        } else if (value.charAt(0) === "/") {
            local = value
        }
        var cmd
        var bin = Util.evoplayerBinPath(service ? service.home : "")
        if (local)
            cmd = [bin, "art", "set", trackPath, local, "--json"]
        else if (safeArtURL(value))
            cmd = [bin, "art", "apply", trackPath, value, "--json"]
        else {
            err = "not an image"
            return false
        }
        artFromDrop = true
        artBusy = true
        err = ""
        _applyBuf = ""
        _applyOverflow = false
        artApplyProc.command = cmd
        artApplyProc.running = true
        return true
    }

    function finishArt(ok, artPath) {
        var fromDrop = artFromDrop
        artFromDrop = false
        artBusy = false
        if (!ok) {
            err = "art apply failed"
            return
        }
        if (artPath) {
            artOverride = artPath
            artOverridePath = trackPath
            artEpoch++
        }
        err = ""
        if (!fromDrop)
            closeMode()
        if (service && service.requestEnrich)
            service.requestEnrich(trackPath)
    }

    function applyLibrary(path) {
        path = String(path || "").replace(/^\s+|\s+$/g, "").replace(/\/$/, "")
        if (!path)
            return
        musicRoot = path
        ipc("config.set", { section: "paths", key: "root", value: path }, function() {
            root.err = ""
            root.loadBrowse("")
            root.loadPlaylistIndex()
        })
    }

    function pickLibrary() {
        if (musicRoot || pickProc.running)
            return
        pickProc.stdoutBuf = ""
        pickProc.command = [Util.evoplayerBinPath(service ? service.home : ""), "config", "pick"]
        pickProc.running = true
    }

    function loadRoot() {
        if (rootProc.running)
            return
        rootProc.stdoutBuf = ""
        rootProc.command = [Util.evoplayerBinPath(""), "config", "toml-get", "paths", "root"]
        rootProc.running = true
    }

    function transport(action) {
        if (service && service.runTransport)
            service.runTransport(action)
    }

    function togglePlay() { transport("toggle") }
    function toggleShuffle() { ipc("playback.shuffle", { on: !shuffle }, null) }
    function toggleRepeat() { ipc("playback.repeat", { on: !repeatOn }, null) }

    function seekTo(sec) {
        var dur = Number(player.duration) || 0
        var t = Number(sec) || 0
        if (t < 0)
            t = 0
        if (dur > 0 && t > dur)
            t = dur
        seekPreview = t
        seekTimer.restart()
        seekGiveUp.restart()
    }

    function seekBy(delta) {
        var pos = seekPreview >= 0 ? seekPreview : (Number(player.position) || 0)
        seekTo(pos + delta)
    }

    function setVolume(value) {
        var v = Math.round(Number(value) || 0)
        if (v < 0)
            v = 0
        if (v > 100)
            v = 100
        ipc("playback.volume.set", { volume: v }, null)
    }

    function volumeDelta(delta) {
        ipc("playback.volume.delta", { delta: delta }, null)
    }

    function toggleViz() {
        vizOn = !vizOn
        syncViz()
    }

    function syncViz() {
        if (!service || !service.ipcCall)
            return
        if (!service.ipcReady) {
            vizSubscribed = false
            return
        }
        var want = active && vizOn && playing
        if (want && !vizSubscribed) {
            vizSubscribed = true
            service.ipcCall("viz.subscribe", undefined, null)
        } else if (!want && vizSubscribed) {
            vizSubscribed = false
            service.ipcCall("viz.unsubscribe", undefined, null)
        }
    }

    function loadChannels() {
        if (!trackPath || trackPath.indexOf("\n") >= 0 || trackPath.indexOf("\r") >= 0) {
            channels = 0
            return
        }
        if (!active)
            return
        if (channelProc.running) {
            if (channelProc.forPath !== trackPath) {
                channels = 0
                channelProc.signal(15)
            }
            return
        }
        channels = 0
        channelProc.forPath = trackPath
        channelProc.stdoutBuf = ""
        channelProc.command = ["ffprobe", "-v", "quiet", "-select_streams", "a:0", "-show_entries", "stream=channels", "-of", "csv=p=0", trackPath]
        channelProc.running = true
    }

    function loadWave() {
        if (!active)
            return
        if (waveFile) {
            readWave(waveFile)
            return
        }
        if (!trackPath)
            return
        ipc("library.warm.waveform", { path: trackPath }, function(data) {
            var file = data && data.waveform ? String(data.waveform) : ""
            if (file)
                root.readWave(file)
        })
    }

    function readWave(file) {
        if (!file || file.indexOf("\n") >= 0 || waveProc.running)
            return
        _waveBuf = ""
        _waveOverflow = false
        waveProc.command = ["dd", "if=" + file, "iflag=nofollow,nonblock,count_bytes,fullblock", "bs=1", "count=65537", "status=none"]
        waveProc.running = true
    }

    function applyWave(text) {
        var parsed
        try {
            parsed = JSON.parse(String(text || ""))
        } catch (e) {
            return
        }
        var data = parsed && parsed.data ? parsed.data : []
        var channels = Number(parsed && parsed.channels) || 1
        var src = []
        if (channels >= 2) {
            for (var i = 0; i < data.length; i += 2) {
                var v = Number(data[i]) || 0
                if (i + 1 < data.length && Number(data[i + 1]) > v)
                    v = Number(data[i + 1])
                src.push(v)
            }
        } else {
            for (var j = 0; j < data.length; j++)
                src.push(Number(data[j]) || 0)
        }
        var max = 1
        for (var k = 0; k < src.length; k++) {
            if (src[k] > max)
                max = src[k]
        }
        var out = []
        for (var n = 0; n < src.length; n++)
            out.push(src[n] / max)
        peaks = out
    }

    function reorder(delta) {
        reorderFrom(playlistIdx, delta, -1)
    }

    // The daemon only accepts a move of one slot, so a longer drag is a chain of those.
    function reorderFrom(index, delta, scrollY) {
        if (String(searchQuery || "").replace(/^\s+|\s+$/g, "") !== "")
            return
        if (shuffle || pane !== "playlist" || mode !== "queue" || shownPlaylist || reorderPending)
            return
        if (!delta || index < 0 || index >= queue.length)
            return
        var dest = index + delta
        if (dest < 0)
            dest = 0
        if (dest >= queue.length)
            dest = queue.length - 1
        delta = dest - index
        if (!delta)
            return
        var items = queue.slice()
        var moved = items.splice(index, 1)[0]
        items.splice(dest, 0, moved)
        var step = delta > 0 ? 1 : -1
        var left = Math.abs(delta)
        var at = index
        reorderPending = left
        if (scrollY >= 0)
            reorderScroll = scrollY
        playlistIdx = dest
        queue = items
        function stepOnce() {
            root.ipc("queue.move", { index: at, delta: step }, function() {
                at += step
                left--
                root.reorderPending--
                if (left > 0)
                    stepOnce()
                else if (root.reorderPending === 0)
                    root.loadQueue()
            }, function() {
                root.reorderPending = 0
                root.loadQueue()
            })
        }
        stepOnce()
    }

    function cycleFocus(dir) {
        if (pane === "search")
            pane = dir > 0 ? "playlist" : "browse"
        else if (pane === "browse")
            pane = "playlist"
        else
            pane = "browse"
        textCapture = false
        syncSettingsMode()
        if (pane === "playlist")
            revealPlaying()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function focusPane(which) {
        pane = which
        textCapture = false
        syncSettingsMode()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function clickBrowse(index) {
        browseIdx = index
        pane = searchQuery ? "search" : "browse"
        syncSettingsMode()
        syncShownPlaylist()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function clickPlaylist(index) {
        playlistIdx = index
        focusPane("playlist")
    }

    function clickMove(index) {
        moveIdx = index
        focusPane("playlist")
    }

    function clickArt(index) {
        artIdx = index
        focusPane("playlist")
        showArtPreview()
    }

    function applyDisplayArt(path, artPath) {
        if (String(path || "") !== trackPath)
            return
        artOverride = String(artPath || "")
        artOverridePath = trackPath
    }

    function onEsc() {
        if (mode === "discover") {
            closeDiscover()
            return
        }
        if (mode === "download") {
            closeDownload()
            return
        }
        if (mode !== "queue") {
            closeMode()
            return
        }
        if (pane === "search") {
            clearSearch()
            return
        }
        if (browsePath)
            leaveFolder()
    }

    function dispatch(event) {
        var key = event.key
        var text = event.text || ""
        if (key === Qt.Key_Back) { leaveFolder(); return true }
        if (key === Qt.Key_Forward) { enterFolder(); return true }
        var shift = (event.modifiers & Qt.ShiftModifier) !== 0
        var ctrl = (event.modifiers & Qt.ControlModifier) !== 0

        if (mode === "tags") {
            if (key === Qt.Key_Escape) { closeMode(); return true }
            if (key === Qt.Key_Up || key === Qt.Key_Backtab || (key === Qt.Key_Tab && shift)) { moveTag(-1); return true }
            if (key === Qt.Key_Down || (key === Qt.Key_Tab && !shift)) { moveTag(1); return true }
            if (key === Qt.Key_Return || key === Qt.Key_Enter) { moveTag(1); return true }
            return false
        }
        if (mode === "download" && textCapture) {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (key === Qt.Key_Return || key === Qt.Key_Enter) {
                if (downloadIdx === 1)
                    searchSoundCloud()
                else
                    submitDownload()
                return true
            }
            if (key === Qt.Key_Down) {
                if (downloadIdx === 0) {
                    downloadIdx = 1
                    return true
                }
                if ((downloadHits || []).length) {
                    downloadHit = 0
                    return true
                }
                return true
            }
            if (key === Qt.Key_Up || key === Qt.Key_Backtab) {
                if (downloadIdx === 1) {
                    downloadIdx = 0
                    return true
                }
                focusPane("browse")
                return true
            }
            return false
        }
        if (mode === "download" && downloadHit >= 0) {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (text === "s" || text === "S") { syncLikes(); return true }
            if (key === Qt.Key_Up) {
                if (downloadHit > 0)
                    downloadHit -= 1
                else
                    downloadHit = -1
                return true
            }
            if (key === Qt.Key_Down) {
                if (downloadHit < (downloadHits || []).length - 1)
                    downloadHit += 1
                return true
            }
            if (key === Qt.Key_Return || key === Qt.Key_Enter) { playSoundCloudHit(); return true }
            if (text === "d" || text === "D" || text === "l" || text === "L") { downloadSoundCloudHit(); return true }
            return true
        }
        if (mode === "download") {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (text === "s" || text === "S") { syncLikes(); return true }
            if ((text === "l" || text === "L") && importPlayingPreview())
                return true
        }
        if (pane === "search" && textCapture) {
            if (key === Qt.Key_Escape) { clearSearch(); return true }
            if (key === Qt.Key_Return || key === Qt.Key_Enter) { playSelected(); return true }
            if (key === Qt.Key_Up) { stepIndex("playlist", (searchHits || []).length, -1); return true }
            if (key === Qt.Key_Down) { stepIndex("playlist", (searchHits || []).length, 1); return true }
            if (key === Qt.Key_PageUp) { stepIndex("playlist", (searchHits || []).length, -playlistPage); return true }
            if (key === Qt.Key_PageDown) { stepIndex("playlist", (searchHits || []).length, playlistPage); return true }
            return false
        }

        if (mode === "discover" && pane === "playlist") {
            if (key === Qt.Key_Up) { stepDiscover(-1); return true }
            if (key === Qt.Key_Down) { stepDiscover(1); return true }
            if (key === Qt.Key_PageUp) { stepDiscover(-playlistPage); return true }
            if (key === Qt.Key_PageDown) { stepDiscover(playlistPage); return true }
            if (key === Qt.Key_Return || key === Qt.Key_Enter) { previewDiscover(); return true }
        }
        if (mode === "discover") {
            if (text === "k" || text === "K") { keepDiscover(); return true }
            if (text === "x" || text === "X") { dismissDiscover(); return true }
            if (text === "n" || text === "N") { moreDiscover(); return true }
        }

        if ((ctrl && key === Qt.Key_C) || (key === Qt.Key_Q && !shift && !ctrl)) {
            if (host && host.requestClose)
                host.requestClose()
            return true
        }
        if (text === "h" || text === "H") { toggleHelp(); return true }
        if (key === Qt.Key_Escape) { onEsc(); return true }
        if (key === Qt.Key_Backtab || (key === Qt.Key_Tab && shift)) { cycleFocus(-1); return true }
        if (key === Qt.Key_Tab) { cycleFocus(1); return true }
        if (text === "/") { openSearch(); return true }
        if (key === Qt.Key_Space) { togglePlay(); return true }
        if (text === ",") { transport("prev"); return true }
        if (text === ".") { transport("next"); return true }
        if (text === "<") { seekBy(-10); return true }
        if (text === ">") { seekBy(10); return true }
        if (text === "-" || text === "_") { volumeDelta(-5); return true }
        if (text === "=" || text === "+") { volumeDelta(5); return true }
        if (text === "d" || text === "D") { addDir(); return true }
        if (text === "f" || text === "F") { openFolder(); return true }
        if (text === "l") { likePlaying(); return true }
        if (text === "m" || text === "M") { openMove(); return true }
        if (text === "e" || text === "E") { openTags(); return true }
        if (text === "a" || text === "A") { openArt(); return true }
        if (text === "v" || text === "V") { toggleViz(); return true }
        if (key === Qt.Key_Backspace) { leaveFolder(); return true }
        if (key === Qt.Key_Left) {
            leaveFolder()
            return true
        }
        if (key === Qt.Key_Right) {
            var row = currentRow()
            if (pane !== "playlist" && row && row.kind === "tool") { focusPane("playlist"); return true }
            if (pane !== "playlist")
                enterFolder()
            return true
        }
        if (key === Qt.Key_Return || key === Qt.Key_Enter) { playSelected(); return true }
        if (shift && key === Qt.Key_Up) { reorder(-1); return true }
        if (shift && key === Qt.Key_Down) { reorder(1); return true }
        if (key === Qt.Key_Up) { moveActive(-1); return true }
        if (key === Qt.Key_Down) { moveActive(1); return true }
        if (key === Qt.Key_PageUp) { moveActive(pane === "playlist" ? -playlistPage : -browsePage); return true }
        if (key === Qt.Key_PageDown) { moveActive(pane === "playlist" ? playlistPage : browsePage); return true }
        return false
    }

    function handleKey(event) {
        if (dispatch(event))
            event.accepted = true
    }

    Timer {
        id: searchTimer
        interval: 180
        onTriggered: root.runSearch()
    }

    Timer {
        id: seekTimer
        interval: 200
        onTriggered: root.ipc("playback.seek", { seconds: root.seekPreview }, null)
    }

    Timer {
        id: seekGiveUp
        interval: 1200
        onTriggered: root.seekPreview = -1
    }

    Connections {
        target: root.service
        function onDaemonJobUpdated(data) {
            if (!data)
                return
            var name = String(data.name || "")
            if (name === "scan") {
                var running = String(data.status || "") === "running"
                root.scanRunning = running
                if (!running)
                    root.loadBrowse(root.browsePath)
                return
            }
            if (name === "download" || name === "download-url") {
                root.noteDownloadJob(data)
                return
            }
            if (name === "discover-preview" && root.mode === "download") {
                var previewStatus = String(data.status || "")
                if (previewStatus === "running") {
                    var previewPhase = data.progress && data.progress.phase ? String(data.progress.phase) : ""
                    root.downloadNote = previewPhase || "playing…"
                    return
                }
                if (previewStatus === "error") {
                    root.downloadNote = ""
                    root.err = root.jobError(data)
                    return
                }
                if (previewStatus === "done") {
                    root.downloadNote = ""
                    root.err = ""
                    return
                }
            }
            if (name !== "discover-preview" && name !== "discover-keep")
                return
            var status = String(data.status || "")
            if (status === "running") {
                var phase = data.progress && data.progress.phase ? String(data.progress.phase) : ""
                root.discoverNote = phase || (name === "discover-keep" ? "keeping…" : "previewing…")
                return
            }
            if (status === "error") {
                root.discoverNote = ""
                root.err = root.jobError(data)
                return
            }
            if (status === "done") {
                root.err = ""
                var id = data.result && data.result.id
                if (name === "discover-keep" && id)
                    root.markDiscoverKept(id)
                root.showDiscoverSeed()
            }
        }
    }

    Process { id: openProc }

    Process {
        id: waveProc
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                if (root._waveOverflow)
                    return
                root._waveBuf += String(chunk || "")
                if (root._waveBuf.length > 65536) {
                    root._waveOverflow = true
                    root._waveBuf = ""
                    waveProc.signal(15)
                }
            }
        }
        onExited: {
            if (!root._waveOverflow)
                root.applyWave(root._waveBuf)
            root._waveBuf = ""
        }
    }

    Process {
        id: artSearchProc
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                if (root._artOverflow)
                    return
                root._artBuf += String(chunk || "")
                if (root._artBuf.length > 65536) {
                    root._artOverflow = true
                    root._artBuf = ""
                    artSearchProc.signal(15)
                }
            }
        }
        onExited: function(code) {
            root.artBusy = false
            if (root._artOverflow || code !== 0) {
                root.err = "art search failed"
                root._artBuf = ""
                return
            }
            var parsed
            try {
                parsed = JSON.parse(root._artBuf || "{}")
            } catch (e) {
                root.err = "art search failed"
                root._artBuf = ""
                return
            }
            root._artBuf = ""
            root.err = ""
            root.artHits = parsed.results || []
            root.artIdx = 0
            root.showArtPreview()
        }
    }

    Process {
        id: artApplyProc
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                if (root._applyOverflow)
                    return
                root._applyBuf += String(chunk || "")
                if (root._applyBuf.length > 65536) {
                    root._applyOverflow = true
                    root._applyBuf = ""
                    artApplyProc.signal(15)
                }
            }
        }
        onExited: function(code) {
            var artPath = ""
            if (code === 0 && !root._applyOverflow) {
                try {
                    var parsed = JSON.parse(root._applyBuf || "{}")
                    artPath = String(parsed.art || "")
                } catch (e) {
                    artPath = ""
                }
            }
            root._applyBuf = ""
            root.finishArt(code === 0 && !root._applyOverflow, artPath)
        }
    }

    Process {
        id: channelProc
        property string forPath: ""
        property string stdoutBuf: ""
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                channelProc.stdoutBuf += String(chunk || "")
                if (channelProc.stdoutBuf.length > 64) {
                    channelProc.stdoutBuf = ""
                    channelProc.signal(15)
                }
            }
        }
        onExited: function(code) {
            var raw = String(channelProc.stdoutBuf || "").replace(/^\s+|\s+$/g, "")
            channelProc.stdoutBuf = ""
            if (channelProc.forPath !== root.trackPath) {
                Qt.callLater(root.loadChannels)
                return
            }
            if (code !== 0 || !raw)
                return
            var n = parseInt(raw, 10)
            root.channels = n === 1 ? 1 : (n >= 2 ? n : 0)
        }
    }

    Process {
        id: rootProc
        property string stdoutBuf: ""
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                rootProc.stdoutBuf += String(chunk || "")
                if (rootProc.stdoutBuf.length > 4096) {
                    rootProc.stdoutBuf = ""
                    rootProc.signal(15)
                }
            }
        }
        onExited: {
            var path = String(rootProc.stdoutBuf || "").replace(/^\s+|\s+$/g, "")
            rootProc.stdoutBuf = ""
            if (path)
                root.musicRoot = path
            else
                root.pickLibrary()
        }
    }

    Process {
        id: pickProc
        property string stdoutBuf: ""
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                pickProc.stdoutBuf += String(chunk || "")
                if (pickProc.stdoutBuf.length > 65536) {
                    pickProc.stdoutBuf = ""
                    pickProc.signal(15)
                }
            }
        }
        onExited: {
            var raw = String(pickProc.stdoutBuf || "").replace(/^\s+|\s+$/g, "")
            pickProc.stdoutBuf = ""
            if (!raw)
                return
            var parsed
            try {
                parsed = JSON.parse(raw)
            } catch (e) {
                return
            }
            var path = parsed && parsed.paths && parsed.paths.root ? String(parsed.paths.root) : ""
            if (path)
                root.applyLibrary(path)
        }
    }

    readonly property int paneGap: 16

    Column {
        anchors.fill: parent
        anchors.margins: root.paneGap
        spacing: root.paneGap

        NowPlayingPane {
            id: nowPane
            width: parent.width
            height: implicitHeight
            view: root
        }

        Item {
            width: parent.width
            height: parent.height - nowPane.height - root.paneGap

            BrowsePane {
                id: browsePane
                width: Math.round(Math.max(180, Math.min(280, parent.width * 0.24)) * 0.6)
                height: parent.height
                view: root
            }

            PlaylistPane {
                anchors.left: browsePane.right
                anchors.leftMargin: root.paneGap
                anchors.right: artPane.left
                anchors.rightMargin: root.paneGap
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                view: root
            }

            ArtPane {
                id: artPane
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                width: Math.min(parent.height, Math.max(160, parent.width - browsePane.width - root.paneGap * 2 - 420))
                view: root
            }
        }
    }

    TapHandler {
        acceptedButtons: Qt.BackButton | Qt.ForwardButton | Qt.ExtraButton1 | Qt.ExtraButton2 | Qt.ExtraButton3 | Qt.ExtraButton4
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        gesturePolicy: TapHandler.WithinBounds
        onTapped: function(eventPoint, button) {
            var back = button === Qt.BackButton || button === Qt.ExtraButton1 || button === Qt.ExtraButton4
            if (back)
                root.leaveFolder()
            else
                root.enterFolder()
        }
    }
}
