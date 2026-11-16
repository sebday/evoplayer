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
    onServiceChanged: {
        if (service && service.ipcReady && active) {
            libraryTries = 0
            ensureLibrary()
        }
    }
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
    property int libraryTries: 0
    property string pane: "playlist"
    property string mode: "queue"
    property bool textCapture: false
    property string err: ""
    property bool scanRunning: false

    property string browsePath: ""
    property string libraryView: ""
    property int playlistColumns: 4
    property bool asGrid: false
    property string browseParent: ""
    property var browseEntries: []
    property var playlists: []
    property var playlistCovers: ({})
    property int playlistCoverRev: 0
    property var likesCoverChoices: []
    property int playlistCoverGen: 0
    property var sidebar: []
    property int browseIdx: 0
    property int browsePage: 12

    property var queue: []
    property int playlistIdx: 0
    property int playingReveal: 0
    property int playlistAnchor: 0
    property var playlistPicks: []
    property bool tagMany: false
    property int reorderPending: 0
    property real reorderScroll: -1
    property real playlistScroll: 0
    property var scrollMemory: ({})
    property bool holdScroll: false
    property int scrollEpoch: 0
    property int viewChangeDepth: 0
    property bool scrollPending: false
    property bool playlistLikedOnly: false
    property string shownPlaylist: ""
    property var shownTracks: []
    property int shownGen: 0
    property var discoverTracks: []
    property var relatedTracks: []
    property int relatedShown: 0
    property bool relatedBlockArmed: true
    property var adoptedRelated: []
    property string adoptedFor: ""
    property string relatedFor: ""
    property string relatedPending: ""
    property string relatedNote: ""
    property int relatedGen: 0
    property string relatedFollowKey: ""
    property int discoverIdx: 0
    property string discoverNote: ""
    property string discoverSeedTitle: ""
    property var discoverSeed: ({})
    property string downloadUrl: ""
    property string downloadQuery: ""
    property var downloadHits: []
    property int downloadHit: -1
    property string downloadPreviewId: ""
    property string downloadPreviewURL: ""
    property string downloadPreviewArt: ""
    property int downloadIdx: 0
    property string downloadNote: ""
    property bool downloadBusy: false
    property string downloadLog: ""
    property var downloadFiles: []
    property bool scSearchBusy: false
    property int playlistPage: 16
    property int savedPlaylistIdx: 0
    property string savedFocus: "playlist"

    property string searchQuery: ""
    property var searchHits: []
    property var searchArtists: []
    property var searchAlbums: []
    property int searchGen: 0

    // An open artist, album, genre, or year page, or the artists / genres directory.
    property var page: null
    property var pageData: null
    property var pageTracks: []
    property var pageStack: []
    property int pageGen: 0
    property bool pageBusy: false
    property var indexCache: ({})
    property var indexRows: []
    property string indexShown: ""
    property string indexFilter: ""
    property int indexIdx: 0
    property int indexColumns: 1

    property bool vizOn: true
    property bool vizSubscribed: false
    property string eqPanel: "eq"
    property var libraryJob: null
    property var incomingFiles: []
    property var incomingFolders: []
    property int incomingIdx: 0
    property string incomingGenrePath: ""
    property string modeReturn: ""
    property bool eqEnabled: true
    property real eqPreamp: 0
    property var eqBands: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]
    property int eqIdx: 0
    property var eqPresets: [null, null, null]
    property bool eqSaveArmed: false
    property bool eqKeys: false
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
    property string artTarget: ""
    property string artQuery: ""
    property bool artQueryFocus: false
    property bool artPicking: false
    property bool artRestart: false
    property string artNextQuery: ""
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
    property string tagTags: ""
    property string tagTagsInitial: ""
    property string tagField: ""
    property string tagFieldValue: ""
    property var tagPaths: []
    property string tagScope: ""

    property string musicRoot: ""

    property string _waveBuf: ""
    property bool _waveOverflow: false
    property string _artBuf: ""
    property bool _artOverflow: false
    property string _applyBuf: ""
    property bool _applyOverflow: false

    onQueueChanged: scheduleRelated()
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
        adoptPlayedRelated()
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
            ensureLibrary()
            loadQueue()
            loadRoot()
            if (service)
                service.ipcCall("job.status", {}, function(ok, msg) {
                    if (ok && msg && msg.data)
                        root.scanRunning = String(msg.data.name || "") === "scan" && String(msg.data.status || "") === "running"
                        var jobName = String(msg.data.name || "")
                        if ((jobName === "download" || jobName === "download-url") && String(msg.data.status || "") === "running")
                            root.noteDownloadJob(msg.data)
                })
        }
        syncViz()
        loadWave()
        loadEq()
        scheduleRelated()
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
            if (root.err === "request failed")
                root.err = ""
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

    function soundcloudArtURL(value) {
        var s = safeArtURL(value)
        var lower = s.toLowerCase()
        if (lower.indexOf(".sndcdn.com/") >= 0 && lower.indexOf("/artworks-") >= 0 && lower.slice(-4) === ".png")
            return s.slice(0, -4) + ".jpg"
        return s
    }

    function discoverPreviewId(path) {
        path = String(path || "")
        var mark = "/discover/"
        var at = path.lastIndexOf(mark)
        if (at < 0 || path.slice(-4) !== ".mp3")
            return ""
        var id = path.slice(at + mark.length, path.length - 4)
        if (!id || id.replace(/[0-9]/g, "") !== "")
            return ""
        return id
    }

    function previewRowForPath(path) {
        var id = discoverPreviewId(path)
        if (!id)
            return null
        var lists = [adoptedRelated, relatedTracks, discoverTracks, downloadHits]
        for (var n = 0; n < lists.length; n++) {
            var rows = lists[n] || []
            for (var i = 0; i < rows.length; i++) {
                if (rows[i] && String(rows[i].id) === id)
                    return rows[i]
            }
        }
        return null
    }

    function playingPreviewArt() {
        var id = discoverPreviewId(trackPath)
        if (!id)
            return ""
        if (String(downloadPreviewId) === id && downloadPreviewArt)
            return downloadPreviewArt
        var row = previewRowForPath(trackPath)
        return row ? soundcloudArtURL(row.artwork) : ""
    }

    function artSource() {
        var url = ""
        if (artPreview)
            url = artPreview
        else if (mode === "tags" && tagPath && tagPath !== trackPath)
            url = Util.fileUrl(rowArt(tagPath))
        else if (artOverride && artOverridePath === trackPath)
            url = Util.fileUrl(artOverride)
        else
            url = playingPreviewArt() || Util.fileUrl(String(player.art || ""))
        if (url.indexOf("https://") === 0 || url.indexOf("http://") === 0)
            return url
        if (url && artEpoch)
            url += "#" + artEpoch
        return url
    }

    function rowArt(path) {
        var lists = [queue, shownTracks, searchHits, pageTracks]
        for (var n = 0; n < lists.length; n++) {
            var rows = lists[n] || []
            for (var i = 0; i < rows.length; i++) {
                if (rows[i] && String(rows[i].path || "") === path)
                    return String(rows[i].thumb || rows[i].art || "")
            }
        }
        return ""
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

    function playlistLabel(name) {
        if (name === "all")
            return "Likes"
        if (name === "current")
            return "Now Playing"
        if (name === "mixes")
            return "Mixes"
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
        var current = (queue || []).length
        var others = 0
        var items = playlists || []
        for (var i = 0; i < items.length; i++) {
            var name = String(items[i].name || "")
            var count = Number(items[i].count) || 0
            if (name === "current")
                current = count || current
            else if (name)
                others++
        }
        var rows = []
        rows.push({ kind: "playlist", type: "", label: "Now Playing", path: "", count: current, id: "current", depth: 0 })
        rows.push({ kind: "lists", type: "", label: "Playlists", path: "", count: others, id: "playlists", depth: 0 })
        rows.push({ kind: "page", type: "", label: "Genres", path: "", count: indexCount("genre"), id: "genres", depth: 0 })
        sidebar = rows
        var tab = tabIndexForView()
        if (tab >= 0)
            browseIdx = tab
    }

    function tabIndexForView() {
        var id = "current"
        if (libraryView === "lists")
            id = "playlists"
        else if (page && page.kind === "genres")
            id = page.kind
        else if (shownPlaylist)
            id = shownPlaylist
        var rows = sidebar || []
        for (var i = 0; i < rows.length; i++) {
            if (rows[i] && rows[i].id === id)
                return i
        }
        return -1
    }

    function menuTabs() {
        var rows = sidebar || []
        var out = []
        for (var i = 0; i < rows.length; i++) {
            if (rows[i] && rows[i].kind !== "tree" && rows[i].kind !== "rule")
                out.push({ index: i, row: rows[i] })
        }
        return out
    }

    function showBrowseTab(row, fromKey) {
        if (!row)
            return
        beginViewChange()
        if (row.id === "playlists") {
            openPlaylists()
            endViewChange()
            return
        }
        if (row.id === "current") {
            showNowPlaying()
            endViewChange()
            return
        }
        if (row.kind === "playlist") {
            libraryView = ""
            leaveOverlays()
            closePages()
            showPlaylist(row.id)
            playlistIdx = 0
            endViewChange()
            return
        }
        if (row.kind === "page") {
            libraryView = ""
            leaveOverlays()
            openIndex(row.id)
        }
        endViewChange()
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
        downloadBusy = true
        ipc("library.download", { url: url, import: true }, function() {
            root.err = ""
            if (fromBox) {
                root.downloadUrl = ""
                root.downloadQuery = ""
            }
        }, function() {
            root.downloadNote = ""
            root.downloadBusy = false
        })
    }

    function stopDownload() {
        ipc("job.cancel", {}, function() {
            root.downloadBusy = false
            root.err = ""
            root.downloadNote = "stopped"
        })
    }

    function isDownloadLink(text) {
        var t = String(text || "").toLowerCase()
        return t.indexOf("://") >= 0 || t.indexOf("www.") === 0 || t.indexOf("youtu.be/") >= 0 || t.indexOf("youtube.com/") >= 0 || t.indexOf("soundcloud.com/") >= 0
    }

    function runDownloadBox() {
        var text = String(downloadQuery || "").replace(/^\s+|\s+$/g, "")
        if (!text) {
            err = "type a search or paste a link"
            return
        }
        if (isDownloadLink(text)) {
            downloadUrl = text
            submitDownload()
            return
        }
        searchSoundCloud()
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

    function soundcloudPage(url) {
        url = String(url || "").replace(/^\s+|\s+$/g, "")
        if (url.indexOf("\n") >= 0 || url.indexOf("\r") >= 0)
            return ""
        var lower = url.toLowerCase()
        if (lower.indexOf("https://soundcloud.com/") !== 0 && lower.indexOf("https://m.soundcloud.com/") !== 0 && lower.indexOf("https://on.soundcloud.com/") !== 0)
            return ""
        return url
    }

    function openInBrave(url) {
        url = soundcloudPage(url)
        if (!url)
            return false
        Quickshell.execDetached(["brave", url])
        return true
    }

    function playDRMInBrave(message) {
        if (String(message || "").toLowerCase().indexOf("drm") < 0)
            return false
        var url = soundcloudPage(downloadPreviewURL)
        if (!url && mode === "discover") {
            var row = currentDiscover()
            url = soundcloudPage(row && row.permalink)
        }
        if (!openInBrave(url))
            return false
        err = ""
        if (mode === "download")
            downloadNote = "opened in brave"
        else
            discoverNote = "opened in brave"
        return true
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
        downloadPreviewArt = soundcloudArtURL(row.artwork)
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
        downloadBusy = true
        ipc("library.soundcloud.download", { import: false }, null, function() {
            root.downloadNote = ""
            root.downloadBusy = false
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
            downloadBusy = true
            downloadNote = phase || "downloading…"
            return
        }
        downloadBusy = false
        if (status === "error") {
            var message = String(data.error || "download failed")
            if (message.indexOf("cancel") >= 0) {
                downloadNote = "stopped"
                err = ""
                return
            }
            downloadNote = ""
            err = message
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
            data = data || {}
            root.downloadFiles = data.files || []
            root.incomingFiles = data.files || []
            root.incomingFolders = data.folders || data.genres || []
            if (root.incomingIdx >= root.incomingFiles.length)
                root.incomingIdx = Math.max(0, root.incomingFiles.length - 1)
        })
    }

    function toggleLibrary() {
        if (mode === "library") {
            closeMode()
            return
        }
        eqSaveArmed = false
        eqKeys = false
        incomingGenrePath = ""
        pushOverlay("library")
        loadIncoming()
        loadEq()
        refreshLibraryJob()
    }

    function playIncoming(index) {
        var rows = incomingFiles || []
        if (index < 0 || index >= rows.length)
            return
        incomingIdx = index
        eqKeys = false
        var path = String(rows[index].path || "")
        if (!path)
            return
        playTrack(path)
    }

    function toggleIncomingGenre(path) {
        path = String(path || "")
        incomingGenrePath = incomingGenrePath === path ? "" : path
    }

    function setIncomingGenre(path, folder) {
        ipc("library.incoming.set_genre", { path: path, folder: folder }, function() {
            root.incomingGenrePath = ""
            root.err = ""
            root.loadIncoming()
        })
    }

    function editIncoming(index) {
        var row = (incomingFiles || [])[index]
        if (!row || !row.path)
            return
        incomingIdx = index
        modeReturn = "library"
        beginTagEdit([String(row.path)], String(row.path))
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

    function stepOffTool() {
        var rows = sidebar || []
        if (!(rows[browseIdx] && rows[browseIdx].kind === "tool"))
            return
        var n = browseIdx - 1
        while (n >= 0 && rows[n] && rows[n].kind === "rule")
            n--
        if (n >= 0)
            browseIdx = n
    }

    function closeTool() {
        stepOffTool()
        mode = "queue"
        pane = "playlist"
        syncShownPlaylist()
        err = ""
    }

    function dismissDownload() {
        if (mode !== "download")
            return
        stepOffTool()
        mode = "queue"
        downloadIdx = 0
        err = ""
        syncShownPlaylist()
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
            discoverSeed = ({})
            discoverNote = ""
            err = "nothing playing"
            return
        }
        var seedArt = ""
        if (id) {
            var from = currentDiscover() || selectedDownloadHit()
            if (from && Number(from.id) === Number(id))
                seedArt = soundcloudArtURL(from.artwork)
        } else {
            seedArt = artSource()
        }
        discoverCall("discover.similar", params, function(data) {
            if (root.mode !== "discover")
                return
            data = data || {}
            root.discoverTracks = data.tracks || []
            var artist = String(data.seed_artist || "").replace(/^\s+|\s+$/g, "")
            var title = String(data.seed_title || "").replace(/^\s+|\s+$/g, "")
            root.discoverSeedTitle = artist && title ? artist + " — " + title : (artist || title)
            root.discoverSeed = { artist: artist, title: title, art: seedArt }
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
        if (isPreviewing(row.id))
            return ">"
        return ""
    }

    function isPreviewing(id) {
        id = String(id || "")
        return id !== "" && trackPath.indexOf("/discover/" + id + ".mp3") >= 0
    }

    function discoverStats() {
        var rows = discoverTracks || []
        var kept = 0
        for (var i = 0; i < rows.length; i++) {
            if (rows[i] && rows[i].kept)
                kept++
        }
        var parts = [counted(rows.length, "new track")]
        if (kept)
            parts.push(kept + " kept")
        if (discoverNote && discoverNote !== discoverSeedTitle)
            parts.push(discoverNote)
        return parts.join("  ·  ")
    }

    function discoverPlay() {
        var row = currentDiscover()
        if (row && isPreviewing(row.id))
            togglePlay()
        else
            previewDiscover()
    }

    function discoverAction(index, action) {
        clickDiscover(index)
        if (action === "keep")
            keepDiscover()
        else if (action === "dismiss")
            dismissDiscover()
        else if (action === "more")
            moreDiscover()
        else if (action === "play")
            discoverPlay()
    }

    function downloadAction(index, action) {
        downloadHit = index
        focusPane("playlist")
        var row = selectedDownloadHit()
        if (action === "download")
            downloadSoundCloudHit()
        else if (action === "play" && row && isPreviewing(row.id))
            togglePlay()
        else if (action === "play")
            playSoundCloudHit()
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

    function queuePathsFrom(rows) {
        var list = rows || []
        var paths = []
        var seen = {}
        for (var i = 0; i < list.length; i++) {
            var path = String(list[i] && list[i].path || "")
            if (!path || path.indexOf("/discover/") >= 0 || seen[path])
                continue
            seen[path] = true
            paths.push(path)
        }
        return paths
    }

    function queuePaths() {
        return queuePathsFrom(queue)
    }

    function libraryQueueKey(rows) {
        return queuePathsFrom(rows).join("\n")
    }

    function relatedAsQueueItem(row, path) {
        var art = soundcloudArtURL(row && row.artwork)
        var dur = Number(row && row.duration) || 0
        return {
            id: row && row.id,
            path: path,
            title: String(row && row.title || ""),
            artist: String(row && row.artist || ""),
            genre: String(row && row.genre || ""),
            year: row && row.year ? String(row.year) : "",
            duration: dur,
            duration_label: clock(dur),
            art: art,
            thumb: art,
            artwork: row && row.artwork || ""
        }
    }

    function rememberAdopted(item) {
        var path = String(item && item.path || "")
        if (!path)
            return
        if (!adoptedFor)
            adoptedFor = libraryQueueKey(queue)
        var next = []
        var rows = adoptedRelated || []
        var found = false
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i] && rows[i].path || "") === path) {
                next.push(item)
                found = true
            } else
                next.push(rows[i])
        }
        if (!found)
            next.push(item)
        adoptedRelated = next
    }

    function withAdopted(items) {
        var key = libraryQueueKey(items)
        if (adoptedFor && key !== adoptedFor) {
            adoptedRelated = []
            adoptedFor = key
            return items || []
        }
        if (!adoptedFor)
            adoptedFor = key
        var next = (items || []).slice()
        var seen = {}
        for (var i = 0; i < next.length; i++)
            seen[String(next[i] && next[i].path || "")] = true
        var extra = adoptedRelated || []
        for (var j = 0; j < extra.length; j++) {
            var path = String(extra[j] && extra[j].path || "")
            if (!path || seen[path])
                continue
            seen[path] = true
            next.push(extra[j])
        }
        return next
    }

    function adoptPlayedRelated() {
        var path = String(trackPath || "")
        var id = discoverPreviewId(path)
        if (!id)
            return
        var known = null
        var kept = adoptedRelated || []
        for (var n = 0; n < kept.length; n++) {
            if (kept[n] && String(kept[n].id) === id) {
                known = kept[n]
                break
            }
        }
        if (!known) {
            var count = Number(player.playlist_count) || 0
            var lib = queuePaths().length
            if (count <= lib + kept.length)
                return
            known = relatedRowById(id) || { id: Number(id) }
        }
        var item = relatedAsQueueItem(known, path)
        rememberAdopted(item)
        var rows = (queue || []).slice()
        var found = false
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i] && rows[i].path || "") === path) {
                rows[i] = Object.assign({}, rows[i], item)
                found = true
                break
            }
        }
        if (!found)
            rows.push(item)
        queue = rows
        if (relatedRowById(id))
            removeRelated(id)
    }

    function relatedRowById(id) {
        var rows = relatedTracks || []
        for (var i = 0; i < rows.length; i++) {
            if (rows[i] && String(rows[i].id) === String(id))
                return rows[i]
        }
        return null
    }

    function scheduleRelated() {
        if (!active)
            return
        var key = queuePaths().join("\n")
        if (key === relatedFor && relatedNote === "ready")
            return
        if (key === relatedPending && relatedNote === "finding similar…")
            return
        relatedPending = key
        if (key !== relatedFor) {
            relatedTracks = []
            relatedShown = 0
            relatedBlockArmed = true
            relatedFor = ""
            publishRelated()
        }
        if (!key) {
            relatedNote = ""
            return
        }
        relatedNote = "finding similar…"
        relatedTimer.restart()
    }

    function loadRelated() {
        var paths = queuePaths()
        var key = paths.join("\n")
        if (!active || !paths.length)
            return
        if (key === relatedFor && relatedNote === "ready")
            return
        if (!service || !service.ipcCall)
            return
        var gen = ++relatedGen
        service.ipcCall("discover.similar", { paths: paths }, function(ok, msg) {
            if (gen !== root.relatedGen || root.queuePaths().join("\n") !== key)
                return
            if (!ok) {
                root.relatedNote = ""
                root.relatedTracks = []
                root.publishRelated()
                return
            }
            var data = msg && msg.data ? msg.data : {}
            var src = data.tracks || []
            var next = []
            var cap = Math.min(20, src.length)
            for (var i = 0; i < cap; i++)
                next.push(src[i])
            root.relatedFor = key
            root.relatedPending = key
            root.relatedTracks = next
            root.relatedShown = Math.min(5, next.length)
            root.relatedBlockArmed = true
            root.relatedNote = next.length ? "ready" : ""
            root.publishRelated()
        })
    }

    function relatedPage() {
        var rows = relatedTracks || []
        var n = relatedShown
        if (n > rows.length)
            n = rows.length
        if (n < 0)
            n = 0
        var out = []
        for (var i = 0; i < n; i++)
            out.push(rows[i])
        return out
    }

    function revealRelated() {
        var rows = relatedTracks || []
        if (relatedShown >= rows.length)
            return
        relatedShown = Math.min(relatedShown + 5, rows.length)
        relatedBlockArmed = false
        publishRelated()
    }

    function noteRelatedScroll(y, h, contentH) {
        if (listSource() !== "queue" || relatedShown >= (relatedTracks || []).length)
            return
        var near = contentH > h && y + h >= contentH - rowGap()
        if (!near) {
            relatedBlockArmed = true
            return
        }
        if (!relatedBlockArmed)
            return
        revealRelated()
    }

    function rowGap() {
        return 48
    }

    function publishRelated() {
        var ids = []
        var rows = relatedPage()
        for (var i = 0; i < rows.length; i++) {
            var id = Number(rows[i] && rows[i].id)
            if (id)
                ids.push(id)
        }
        var key = ids.join(",")
        if (key === relatedFollowKey)
            return
        relatedFollowKey = key
        if (!service || !service.ipcCall)
            return
        service.ipcCall("playback.follow", { ids: ids }, function() {})
    }

    function relatedRow(index) {
        var rows = relatedTracks || []
        if (index < 0 || index >= rows.length)
            return null
        return rows[index]
    }

    function previewRelated(index) {
        var row = relatedRow(index)
        if (!row || !row.id)
            return
        if (isPreviewing(row.id)) {
            togglePlay()
            return
        }
        ipc("discover.preview", { id: Number(row.id) }, null)
    }

    function keepRelated(index) {
        var row = relatedRow(index)
        if (!row || !row.id || row.kept)
            return
        var id = row.id
        ipc("discover.keep", { id: Number(id) }, function(data) {
            data = data || {}
            if (data.path || String(data.status || "") === "done")
                root.markRelatedKept(id)
        })
    }

    function dismissRelated(index) {
        var row = relatedRow(index)
        if (!row || !row.id)
            return
        var id = row.id
        ipc("discover.dismiss", { id: Number(id) }, function() {
            root.removeRelated(id)
        })
    }

    function moreRelated(index) {
        var row = relatedRow(index)
        if (row && row.id)
            loadDiscover(row.id)
    }

    function markRelatedKept(id) {
        relatedTracks = patchRelated(relatedTracks, id, true)
    }

    function removeRelated(id) {
        relatedTracks = patchRelated(relatedTracks, id, false)
        publishRelated()
    }

    function patchRelated(rows, id, keep) {
        var next = []
        var list = rows || []
        for (var i = 0; i < list.length; i++) {
            var row = list[i]
            if (Number(row && row.id) !== Number(id)) {
                next.push(row)
                continue
            }
            if (keep)
                next.push(Object.assign({}, row, { kept: true }))
        }
        return next
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
        markRelatedKept(id)
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
        removeRelated(id)
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

    function ensureLibrary() {
        if ((sidebar || []).length > 0)
            return
        if (!service || !service.ipcCall)
            return
        if (libraryTries >= 8)
            return
        libraryTries++
        loadBrowse(browsePath || "")
        loadPlaylistIndex()
        loadPlaylistFilter()
        prefetchIndexes()
    }

    function noteLibraryMiss() {
        if (!active || (sidebar || []).length > 0 || libraryTries >= 8)
            return
        libraryRetry.restart()
    }

    function loadBrowse(rel, stayIfLeaf) {
        ipc("library.browse", { path: String(rel || ""), offset: 0, limit: 2000 }, function(data) {
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
            libraryTries = 0
            playlistIdx = 0
            rebuildSidebar()
        }, function() {
            root.noteLibraryMiss()
        })
    }

    function loadPlaylistFilter() {
        ipc("library.playlist.filter", {}, function(data) {
            root.playlistLikedOnly = !!(data && data.liked_only)
        })
    }

    function togglePlaylistFilter() {
        ipc("library.playlist.filter", { liked_only: !playlistLikedOnly }, function(data) {
            root.playlistLikedOnly = !!(data && data.liked_only)
            root.loadPlaylistIndex()
            root.reloadShownPlaylist()
        })
    }

    function loadPlaylistIndex() {
        ipc("library.playlist.list", undefined, function(data) {
            playlists = Array.isArray(data) ? data : []
            root.libraryTries = 0
            rebuildSidebar()
            root.loadPlaylistCovers()
        }, function() {
            root.noteLibraryMiss()
        })
    }

    function applyQueue(items, keep) {
        queue = withAdopted(items || [])
        var idx = 0
        if (keep) {
            for (var i = 0; i < queue.length; i++) {
                if (queue[i].path === keep) {
                    idx = i
                    break
                }
            }
        }
        if (shownPlaylist || page || libraryView)
            return
        var playing = discoverPreviewId(trackPath) ? queueIndexOf(trackPath) : -1
        if (playing >= 0)
            idx = playing
        playlistIdx = idx
        playlistAnchor = idx
        playlistPicks = queue[idx] && queue[idx].path ? [String(queue[idx].path)] : []
        revealPlaying()
    }

    function queueIndexOf(path) {
        path = String(path || "")
        if (!path)
            return -1
        var rows = queue || []
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i] && rows[i].path || "") === path)
                return i
        }
        return -1
    }

    function loadQueue() {
        var keep = queue[playlistIdx] ? String(queue[playlistIdx].path || "") : ""
        loadPlaylistItems("current", function(items) {
            root.applyQueue(items, keep)
        })
    }

    function revealPlaying() {
        if (shownPlaylist || page)
            return
        var at = queueIndexOf(trackPath)
        if (at < 0)
            return
        var onQueue = pane === "playlist" && mode === "queue"
        if (onQueue && !discoverPreviewId(trackPath))
            return
        playlistIdx = at
        playlistAnchor = at
        if (onQueue)
            playingReveal++
    }

    function focusPlayingRow() {
        if (shownPlaylist || page)
            return
        if (String(searchQuery || "").replace(/^\s+|\s+$/g, "") !== "")
            return
        var rows = queue || []
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i].path || "") === trackPath) {
                selectOnly(i)
                playingReveal++
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
        showBrowseTab(sidebar[i], true)
    }

    function syncPageRow() {
        var row = sidebar[browseIdx]
        if (row && row.kind === "page") {
            openIndex(row.id)
            return
        }
        if (row && row.kind !== "rule")
            closePages()
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
        else if (mode === "art") {
            if (delta < 0 && artIdx <= 0) {
                artQueryFocus = true
                return
            }
            stepIndex("art", (artHits || []).length, delta)
        }
        else if (pane === "playlist" && mode === "queue") {
            stepIndex("playlist", playlistRows().length, delta)
            selectOnly(playlistIdx)
        }
        else
            stepBrowse(delta)
    }

    function currentRow() {
        return sidebar[browseIdx] || null
    }

    function enterFolder() {
    }

    function leaveFolder() {
        if (libraryView !== "files" || !browsePath)
            return
        loadBrowse(browseParent)
    }

    function scrollKey() {
        if (mode !== "queue")
            return ""
        if (indexPage())
            return "index:" + page.kind
        if (entityPage())
            return "page:" + page.kind + ":" + page.name + (asGrid ? ":grid" : ":list")
        if (searchActive())
            return "search:" + (asGrid ? "grid" : "list")
        if (listSource() === "lists")
            return "playlists"
        if (shownPlaylist === "all")
            return asGrid ? "likes-grid" : "likes-list"
        if (shownPlaylist)
            return "pl:" + shownPlaylist + (asGrid ? ":grid" : ":list")
        if (listSource() === "queue")
            return "queue"
        return ""
    }

    function rememberScroll(key, y) {
        if (!key || y < 0)
            return
        var next = Object.assign({}, scrollMemory)
        next[key] = y
        scrollMemory = next
    }

    function savedScroll(key) {
        if (!scrollMemory || scrollMemory[key] === undefined)
            return -1
        return Number(scrollMemory[key]) || 0
    }

    function beginViewChange() {
        if (viewChangeDepth === 0) {
            if (playlist && playlist.captureScroll)
                playlist.captureScroll()
            holdScroll = true
        }
        viewChangeDepth++
    }

    function endViewChange() {
        if (viewChangeDepth <= 0)
            return
        viewChangeDepth--
        if (viewChangeDepth > 0)
            return
        scrollPending = true
        scrollEpoch++
        Qt.callLater(function() { holdScroll = false })
    }

    function setLayout(grid) {
        if (asGrid === !!grid)
            return
        beginViewChange()
        asGrid = !!grid
        endViewChange()
    }

    function openPlaylists() {
        beginViewChange()
        leaveOverlays()
        libraryView = "lists"
        closePages()
        clearShownPlaylist()
        playlistIdx = 0
        endViewChange()
    }

    function backToPlaylists() {
        beginViewChange()
        clearShownPlaylist()
        libraryView = "lists"
        playlistIdx = 0
        endViewChange()
    }

    function openBrowsePath(path) {
        playFolder(String(path || ""))
    }

    function openListedPlaylist(name) {
        name = String(name || "")
        if (!name)
            return
        beginViewChange()
        libraryView = "lists"
        showPlaylist(name)
        playlistIdx = 0
        endViewChange()
    }

    function fileRows() {
        var entries = browseEntries || []
        var out = []
        for (var i = 0; i < entries.length; i++) {
            var en = entries[i] || {}
            if (String(en.type || "") === "dir") {
                out.push({
                    type: "dir",
                    title: String(en.name || en.path || ""),
                    path: String(en.path || ""),
                    count: Number(en.count) || 0
                })
            } else {
                out.push(en)
            }
        }
        return out
    }

    function playlistIndexRows() {
        var items = orderedPlaylists()
        var out = []
        for (var i = 0; i < items.length; i++) {
            var name = String(items[i].name || "")
            if (!name || name === "current")
                continue
            var cover = playlistCovers[name] || {}
            var row = {
                type: "playlist",
                id: name,
                title: playlistLabel(name),
                path: "",
                count: Number(items[i].count) || 0,
                art: String(cover.art || ""),
                thumb: String(cover.thumb || "")
            }
            if (name === "all")
                out.unshift(row)
            else
                out.push(row)
        }
        return out
    }

    function openFileSelection() {
        var rows = fileRows()
        var row = rows[playlistIdx]
        if (!row)
            return
        if (String(row.type || "") === "dir") {
            openBrowsePath(row.path)
            return
        }
        var paths = []
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i].type || "") !== "dir" && rows[i].path)
                paths.push(String(rows[i].path))
        }
        playPaths(paths, String(row.path || ""))
    }

    function queueHas(path) {
        path = String(path || "")
        if (!path)
            return false
        var queued = queue || []
        for (var i = 0; i < queued.length; i++) {
            if (String(queued[i] && queued[i].path || "") === path)
                return true
        }
        return false
    }

    function playTrack(path) {
        path = String(path || "")
        if (!path)
            return
        if (queueHas(path)) {
            ipc("queue.play_path", { path: path }, null)
            return
        }
        ipc("queue.append", { paths: [path] }, function() {
            root.err = ""
            root.loadQueue()
            root.ipc("queue.play_path", { path: path }, null)
        })
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

    function addFolder(rel) {
        rel = String(rel || "")
        if (!rel)
            return
        ipc("queue.append_folder", { path: rel }, function() {
            root.err = ""
            root.loadQueue()
            root.loadPlaylistIndex()
        })
    }

    // Leave whatever list was open and land on the playing row of the current queue.
    function showNowPlaying() {
        beginViewChange()
        libraryView = ""
        if (String(searchQuery || "").replace(/^\s+|\s+$/g, "") !== "") {
            searchQuery = ""
            searchHits = []
            searchArtists = []
            searchAlbums = []
            searchGen++
            textCapture = false
        }
        closePages()
        var rows = sidebar || []
        var found = false
        for (var i = 0; i < rows.length; i++) {
            if (rows[i] && rows[i].kind === "playlist" && rows[i].id === "current") {
                browseIdx = i
                found = true
                break
            }
        }
        pane = "playlist"
        textCapture = false
        if (found)
            syncShownPlaylist()
        else
            clearShownPlaylist()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
        loadPlaylistItems("current", function(items) {
            root.applyQueue(items, root.trackPath)
            Qt.callLater(function() {
                if (root.savedScroll("queue") < 0)
                    root.focusPlayingRow()
            })
        })
        endViewChange()
    }

    function searchActive() {
        return String(searchQuery || "").replace(/^\s+|\s+$/g, "") !== ""
    }

    function indexPage() {
        return !!page && page.kind === "genres"
    }

    function entityPage() {
        return !!page && !indexPage()
    }

    // Which rows the main list shows, most specific first.
    function listSource() {
        if (entityPage())
            return "page"
        if (indexPage())
            return "index"
        if (searchActive())
            return "search"
        if (libraryView === "lists" && !shownPlaylist)
            return "lists"
        if (shownPlaylist)
            return "shown"
        return "queue"
    }

    function leaveOverlays() {
        if (mode === "discover")
            closeDiscover()
        else if (mode === "download")
            closeDownload()
        else if (mode !== "queue")
            closeMode()
    }

    function openPage(kind, name, dir) {
        name = String(name || "").replace(/^\s+|\s+$/g, "")
        if (!name)
            return
        var spec = { kind: kind, name: name, dir: String(dir || "") }
        if (page && page.kind === spec.kind && page.dir === spec.dir && String(page.name).toLowerCase() === name.toLowerCase())
            return
        leaveOverlays()
        pushPage()
        showPage(spec, null)
        focusPane("playlist")
    }

    function openArtist(name) { openPage("artist", name, "") }

    function togglePageFav() {
        if (!page || (page.kind !== "artist" && page.kind !== "album"))
            return
        var name = String(page.name || "")
        if (!name)
            return
        var album = page.kind === "album"
        ipc(album ? "library.album.favorite" : "library.artist.favorite", { name: name }, function(data) {
            var starred = !!(data && data.starred)
            if (root.page && root.page.kind === (album ? "album" : "artist") && String(root.page.name || "") === name && root.pageData)
                root.pageData = Object.assign({}, root.pageData, { starred: starred })
            root.placeIndexStar(album ? "album" : "artist", album ? "albums" : "artists", name, starred)
            root.loadPlaylistIndex()
        })
    }

    function placeIndexStar(cacheKind, pageKind, name, starred) {
        var rows = (indexCache && indexCache[cacheKind]) || []
        if (!rows.length)
            return
        var key = String(name || "").toLowerCase()
        var fav = []
        var rest = []
        for (var i = 0; i < rows.length; i++) {
            var row = rows[i]
            if (String(row && row.name || "").toLowerCase() === key)
                row = Object.assign({}, row, { starred: !!starred })
            if (row && row.starred)
                fav.push(row)
            else
                rest.push(row)
        }
        var next = Object.assign({}, indexCache)
        next[cacheKind] = fav.concat(rest)
        indexCache = next
        indexShown = ""
        if (indexPage() && page && page.kind === pageKind)
            applyIndexFilter()
    }
    function openGenre(name) { openPage("genre", name, "") }
    function openYear(name) { openPage("year", name, "") }

    function openAlbum(name, path) {
        var p = String(path || "")
        var slash = p.lastIndexOf("/")
        openPage("album", name, slash > 0 ? p.slice(0, slash) : "")
    }

    function gotoSelected(kind) {
        var t = mode === "queue" && pane === "playlist" ? selectedListTrack() : null
        var name = String((t && t[kind]) || "").replace(/^\s+|\s+$/g, "")
        if (!name)
            name = String(player[kind] || "")
        openPage(kind, name, "")
    }

    function pushPage() {
        pageStack = pageStack.concat([{
            page: page,
            data: pageData,
            rows: page ? pageTracks : null,
            idx: playlistIdx,
            scroll: playlistScroll,
            index: indexIdx,
            filter: indexFilter
        }])
    }

    function showPage(spec, saved) {
        beginViewChange()
        pageGen++
        playlistPicks = []
        textCapture = false
        pageBusy = false
        if (spec.kind === "genres") {
            pageData = null
            pageTracks = []
            page = spec
            indexFilter = saved ? String(saved.filter || "") : ""
            indexIdx = saved ? saved.index : 0
            loadIndex(indexKind())
            endViewChange()
            return
        }
        if (saved && saved.data) {
            reorderScroll = saved.scroll
            pageData = saved.data
            pageTracks = saved.rows || saved.data.tracks || []
            page = spec
            playlistIdx = saved.idx
            endViewChange()
            return
        }
        pageData = null
        pageTracks = []
        page = spec
        playlistIdx = 0
        pageBusy = true
        endViewChange()
        var gen = pageGen
        ipc("library.entity", { kind: spec.kind, name: spec.name, dir: spec.dir }, function(data) {
            if (gen !== root.pageGen)
                return
            root.pageBusy = false
            root.pageData = data || {}
            root.pageTracks = (data && data.tracks) || []
            root.playlistIdx = 0
        }, function() {
            if (gen === root.pageGen)
                root.pageBusy = false
        })
    }

    function reloadPage() {
        if (!entityPage())
            return
        var spec = page
        var keep = playlistIdx
        var scroll = playlistScroll
        var gen = ++pageGen
        ipc("library.entity", { kind: spec.kind, name: spec.name, dir: spec.dir }, function(data) {
            if (gen !== root.pageGen)
                return
            root.reorderScroll = scroll
            root.pageData = data || {}
            root.pageTracks = (data && data.tracks) || []
            root.playlistIdx = Math.max(0, Math.min(keep, root.pageTracks.length - 1))
        })
    }

    function pageBack() {
        if (!page)
            return false
        beginViewChange()
        var moved = finishPageBack()
        endViewChange()
        return moved
    }

    function finishPageBack() {
        if (!page)
            return false
        var stack = pageStack.slice()
        var prev = stack.length ? stack.pop() : null
        pageStack = stack
        if (prev && prev.page) {
            showPage(prev.page, prev)
            return true
        }
        pageGen++
        pageBusy = false
        if (prev)
            reorderScroll = prev.scroll
        page = null
        pageData = null
        pageTracks = []
        playlistPicks = []
        if (prev)
            playlistIdx = prev.idx
        return true
    }

    function closePages() {
        if (!page && !pageStack.length)
            return
        pageGen++
        pageBusy = false
        pageStack = []
        page = null
        pageData = null
        pageTracks = []
    }

    function afterLibraryEdit() {
        indexCache = ({})
        indexShown = ""
        prefetchIndexes()
        if (mode === "library") {
            loadIncoming()
            return
        }
        if (entityPage())
            reloadPage()
        else
            showNowPlaying()
        loadPlaylistIndex()
    }

    function indexCount(kind) {
        var rows = indexCache[kind]
        return rows ? rows.length : 0
    }

    function prefetchIndexes() {
        var kinds = ["artist", "genre", "album"]
        for (var i = 0; i < kinds.length; i++)
            prefetchIndex(kinds[i])
    }

    function prefetchIndex(kind) {
        if (indexCache[kind])
            return
        ipc("library.index", { kind: kind }, function(data) {
            var next = Object.assign({}, root.indexCache)
            next[kind] = Array.isArray(data) ? data : []
            root.indexCache = next
            root.rebuildSidebar()
            if (kind === "artist" && root.searchActive())
                root.refreshSearchArtists()
            if (kind === "album" && root.searchActive())
                root.refreshSearchAlbums()
            if (root.indexPage() && root.indexKind() === kind)
                root.applyIndexFilter()
        })
    }

    function indexKind() {
        if (page && page.kind === "genres")
            return "genre"
        if (page && page.kind === "albums")
            return "album"
        return "artist"
    }

    function openIndex(kind) {
        if (page && page.kind === kind && libraryView === "")
            return
        beginViewChange()
        libraryView = ""
        leaveOverlays()
        if (!indexPage())
            pushPage()
        showPage({ kind: kind, name: kind, dir: "" }, null)
        endViewChange()
    }

    function loadIndex(kind) {
        if (indexCache[kind]) {
            var key = kind + "\n" + String(indexFilter || "")
            if (indexShown === key && (indexRows || []).length)
                return
            applyIndexFilter()
            return
        }
        indexRows = []
        pageBusy = true
        var gen = pageGen
        ipc("library.index", { kind: kind }, function(data) {
            var next = Object.assign({}, root.indexCache)
            next[kind] = Array.isArray(data) ? data : []
            root.indexCache = next
            if (gen !== root.pageGen)
                return
            root.pageBusy = false
            root.applyIndexFilter()
        }, function() {
            if (gen === root.pageGen)
                root.pageBusy = false
        })
    }

    function applyIndexFilter() {
        var all = indexCache[indexKind()] || []
        var q = String(indexFilter || "").replace(/^\s+|\s+$/g, "").toLowerCase()
        if (!q) {
            indexRows = all
        } else {
            var out = []
            for (var i = 0; i < all.length; i++) {
                if (String(all[i].name || "").toLowerCase().indexOf(q) >= 0)
                    out.push(all[i])
            }
            indexRows = out
        }
        if (indexIdx >= indexRows.length)
            indexIdx = Math.max(0, indexRows.length - 1)
        indexShown = indexKind() + "\n" + String(indexFilter || "")
    }

    function rowThumbUrl(row) {
        return Util.fileUrl(String(row && (row.thumb || row.art) || ""))
    }

    function setIndexFilter(text) {
        if (indexFilter === text)
            return
        indexFilter = String(text || "")
        indexIdx = 0
        indexFilterTimer.restart()
    }

    function openIndexAt(i) {
        var row = (indexRows || [])[i]
        if (!row)
            return
        indexIdx = i
        openPage(indexKind(), row.name, "")
    }

    function loadPlaylistCovers() {
        var items = orderedPlaylists()
        var gen = ++playlistCoverGen
        for (var i = 0; i < items.length; i++) {
            var name = String(items[i].name || "")
            if (!name || name === "current")
                continue
            fetchPlaylistCover(name, gen)
        }
    }

    function coverKeys(cover) {
        var keys = []
        var art = String(cover && cover.art || "")
        var thumb = String(cover && cover.thumb || "")
        if (art)
            keys.push(art)
        if (thumb && thumb !== art)
            keys.push(thumb)
        return keys
    }

    function usedCoverKeys(exceptName) {
        var used = {}
        var covers = playlistCovers || {}
        var names = Object.keys(covers)
        for (var i = 0; i < names.length; i++) {
            if (names[i] === exceptName)
                continue
            var keys = coverKeys(covers[names[i]])
            for (var k = 0; k < keys.length; k++)
                used[keys[k]] = true
        }
        return used
    }

    function chooseCover(rows, exceptName) {
        var used = exceptName ? usedCoverKeys(exceptName) : {}
        var fallback = { art: "", thumb: "" }
        var list = rows || []
        for (var i = 0; i < list.length; i++) {
            var row = list[i] || {}
            if (!row.art && !row.thumb)
                continue
            var cover = { art: String(row.art || ""), thumb: String(row.thumb || "") }
            if (!fallback.art && !fallback.thumb)
                fallback = cover
            var keys = coverKeys(cover)
            var taken = false
            for (var k = 0; k < keys.length; k++) {
                if (used[keys[k]])
                    taken = true
            }
            if (!taken)
                return cover
        }
        return fallback
    }

    function playlistCoverUrl(name, row) {
        var bump = playlistCoverRev
        var cover = (playlistCovers || {})[String(name || "")] || {}
        return rowCoverUrl({
            art: String((bump >= 0 && cover.art) || (row && row.art) || ""),
            thumb: String(cover.thumb || (row && row.thumb) || "")
        })
    }

    function storePlaylistCover(name, cover) {
        var next = Object.assign({}, playlistCovers)
        next[name] = cover
        playlistCovers = next
        playlistCoverRev = playlistCoverRev + 1
        if (name !== "all")
            refreshLikesCover()
    }

    function refreshLikesCover() {
        if (!(likesCoverChoices || []).length)
            return
        var pick = chooseCover(likesCoverChoices, "all")
        var cur = (playlistCovers || {})["all"] || {}
        if (coverKeys(cur).join("\n") === coverKeys(pick).join("\n"))
            return
        var next = Object.assign({}, playlistCovers)
        next["all"] = pick
        playlistCovers = next
        playlistCoverRev = playlistCoverRev + 1
    }

    function fetchPlaylistCover(name, gen) {
        ipc("library.playlist.tracks", { name: name, offset: 0, limit: name === "all" ? 200 : 24 }, function(data) {
            if (gen !== root.playlistCoverGen)
                return
            var rows = (data && data.items) || []
            if (name === "all")
                root.likesCoverChoices = rows
            root.storePlaylistCover(name, root.chooseCover(rows, name === "all" ? "all" : ""))
        })
    }

    function stepTrackGrid(delta) {
        var n = playlistRows().length
        if (!n || !delta)
            return
        playlistIdx = Math.max(0, Math.min(n - 1, playlistIdx + delta))
    }

    function stepPlaylistGrid(delta) {
        var n = playlistIndexRows().length
        if (!n || !delta)
            return
        playlistIdx = Math.max(0, Math.min(n - 1, playlistIdx + delta))
    }

    function stepIndexGrid(delta) {
        var n = (indexRows || []).length
        if (!n || !delta)
            return
        indexIdx = Math.max(0, Math.min(n - 1, indexIdx + delta))
    }

    function releaseText() {
        textCapture = false
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function longClock(sec) {
        var m = Math.round((Number(sec) || 0) / 60)
        if (m < 60)
            return m + " min"
        return Math.floor(m / 60) + " hr " + (m % 60) + " min"
    }

    function counted(n, word) {
        n = Number(n) || 0
        return n + " " + word + (n === 1 ? "" : "s")
    }

    function rowsDuration(rows) {
        var total = 0
        var list = rows || []
        for (var i = 0; i < list.length; i++)
            total += Number(list[i] && list[i].duration) || 0
        return total
    }

    function rowArtUrl(row) {
        return Util.fileUrl(String(row && (row.thumb || row.art) || ""))
    }

    function rowCoverUrl(row) {
        return Util.fileUrl(String(row && (row.art || row.thumb) || ""))
    }

    function mosaicOf(rows) {
        var out = []
        var seen = {}
        var list = rows || []
        for (var i = 0; i < list.length && out.length < 4; i++) {
            var url = rowArtUrl(list[i])
            if (url && !seen[url]) {
                seen[url] = true
                out.push(url)
            }
        }
        return out
    }

    function headerInfo() {
        var src = listSource()
        var rows = playlistRows()
        if (src === "page") {
            var d = pageData || {}
            var stats = []
            if (pageData) {
                stats.push(counted(d.total, "track"))
                if (d.duration)
                    stats.push(longClock(d.duration))
                if (d.years)
                    stats.push(String(d.years))
                var releases = (d.releases || []).length
                if (page.kind !== "album" && releases)
                    stats.push(releases >= 60 ? "60+ releases" : counted(releases, "release"))
                if (d.liked)
                    stats.push(d.liked + " liked")
            } else if (pageBusy) {
                stats.push("loading…")
            }
            var first = (d.releases || [])[0] || {}
            var album = page.kind === "album"
            return {
                kind: page.kind,
                caption: page.kind,
                title: String(d.name || page.name || ""),
                art: Util.fileUrl(String(d.art || "")),
                blur: Util.fileUrl(String(d.thumb || d.art || "")),
                round: page.kind === "artist",
                mosaic: [],
                artist: album ? String(first.artist || "") : "",
                album: "",
                label: album ? String(first.label || "") : "",
                year: album ? String(first.year || "") : "",
                stats: stats.join("  ·  "),
                path: "",
                starred: !!d.starred
            }
        }
        if (src === "search" || src === "shown") {
            var mosaic = mosaicOf(rows)
            return {
                kind: src === "search" ? "search" : "playlist",
                caption: src === "search" ? "search" : "playlist",
                title: src === "search" ? "“" + String(searchQuery).replace(/^\s+|\s+$/g, "") + "”" : playlistLabel(shownPlaylist),
                art: mosaic.length === 1 ? mosaic[0] : "",
                blur: mosaic[0] || "",
                round: false,
                mosaic: mosaic.length > 1 ? mosaic : [],
                artist: "",
                album: "",
                label: "",
                year: "",
                stats: [counted(rows.length, src === "search" ? "result" : "track"), longClock(rowsDuration(rows))].join("  ·  "),
                path: "",
                starred: false
            }
        }
        var art = artSource()
        var title = String(player.title || "").replace(/^\s+|\s+$/g, "")
        if (!title && trackPath)
            title = trackPath.slice(trackPath.lastIndexOf("/") + 1)
        var artist = String(player.artist || "")
        var preview = previewRowForPath(trackPath)
        if (preview) {
            var previewTitle = String(preview.title || "").replace(/^\s+|\s+$/g, "")
            var previewArtist = String(preview.artist || "").replace(/^\s+|\s+$/g, "")
            if (previewTitle)
                title = previewTitle
            if (previewArtist)
                artist = previewArtist
        }
        return {
            kind: "queue",
            caption: "now playing",
            title: title || "nothing playing",
            art: art,
            blur: art,
            round: false,
            mosaic: [],
            artist: artist,
            album: String(player.album || ""),
            label: String(player.label || ""),
            year: String(player.year || ""),
            stats: ["queue  " + counted(rows.length, "track"), longClock(rowsDuration(rows))].join("  ·  "),
            path: trackPath,
            starred: false
        }
    }

    function headerPlay() {
        if (listSource() === "queue") {
            togglePlay()
            return
        }
        playTrackList(playlistRows(), "")
    }

    function headerAdd() {
        var items = playlistRows()
        var paths = []
        var queued = {}
        var existing = queue || []
        for (var i = 0; i < existing.length; i++) {
            var have = String(existing[i] && existing[i].path || "")
            if (have)
                queued[have] = true
        }
        for (var j = 0; j < items.length; j++) {
            var path = String(items[j] && items[j].path || "")
            if (!path || queued[path])
                continue
            queued[path] = true
            paths.push(path)
        }
        if (!paths.length)
            return
        ipc("queue.append", { paths: paths }, function() {
            root.err = ""
            root.loadQueue()
            root.loadPlaylistIndex()
        })
    }

    function headerShuffle() {
        if (listSource() === "queue") {
            toggleShuffle()
            return
        }
        var rows = playlistRows()
        if (!rows.length)
            return
        var start = rows[Math.floor(Math.random() * rows.length)]
        ipc("playback.shuffle", { on: true }, function() {
            root.playTrackList(rows, start && start.path)
        })
    }

    function showPlaylist(name) {
        name = String(name || "")
        if (!name)
            return
        if (shownPlaylist === name)
            return
        beginViewChange()
        shownPlaylist = name
        shownTracks = []
        playlistIdx = 0
        var gen = ++shownGen
        loadPlaylistItems(name, function(items) {
            if (gen !== root.shownGen || root.shownPlaylist !== name)
                return
            root.shownTracks = items
            if (playlist && playlist.restoreActiveScroll)
                playlist.restoreActiveScroll()
        })
        endViewChange()
    }

    function clearShownPlaylist() {
        if (!shownPlaylist)
            return
        beginViewChange()
        shownPlaylist = ""
        shownTracks = []
        shownGen++
        endViewChange()
    }

    function syncShownPlaylist() {
        if (libraryView === "lists")
            return
        var row = sidebar[browseIdx]
        if (row && row.kind === "playlist" && row.id && row.id !== "current")
            showPlaylist(row.id)
        else
            clearShownPlaylist()
    }

    function patchPlaylistCount(name, count) {
        name = String(name || "")
        count = Number(count) || 0
        if (!name)
            return
        var lists = playlists || []
        var nextLists = []
        var changed = false
        for (var i = 0; i < lists.length; i++) {
            var it = lists[i]
            if (it && String(it.name || "") === name && (Number(it.count) || 0) !== count) {
                nextLists.push(Object.assign({}, it, { count: count }))
                changed = true
            } else
                nextLists.push(it)
        }
        if (changed)
            playlists = nextLists
        var rows = sidebar || []
        var nextRows = []
        var rowChanged = false
        for (var r = 0; r < rows.length; r++) {
            var row = rows[r]
            if (row && row.kind === "playlist" && String(row.id || "") === name && (Number(row.count) || 0) !== count) {
                nextRows.push(Object.assign({}, row, { count: count }))
                rowChanged = true
            } else
                nextRows.push(row)
        }
        if (rowChanged)
            sidebar = nextRows
    }

    function loadPlaylistItems(name, done) {
        ipc("library.playlist.tracks", { name: name, offset: 0, limit: 500 }, function(data) {
            data = data || {}
            var items = data.items || []
            var total = Number(data.total) || items.length
            if (total > items.length && total <= 8000) {
                root.ipc("library.playlist.tracks", { name: name, offset: 0, limit: total }, function(full) {
                    var rows = (full && full.items) || []
                    root.patchPlaylistCount(name, Number(full && full.total) || rows.length || total)
                    done(rows)
                })
                return
            }
            root.patchPlaylistCount(name, total)
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
        if (!row || row.kind === "tool" || row.kind === "rule" || row.kind === "page")
            return
        if (row.kind === "playlist") {
            playPlaylist(row.id)
            return
        }
        if (row.type === "dir") {
            playFolder(row.path)
            return
        }
        if (row.type === "track")
            playTrack(row.path)
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
            else
                runDownloadBox()
            return
        }
        if (mode === "tags")
            return
        if (listSource() === "files") {
            openFileSelection()
            return
        }
        if (listSource() === "lists") {
            var listed = playlistIndexRows()[playlistIdx]
            if (listed)
                openListedPlaylist(listed.id)
            return
        }
        if (pane === "playlist" && indexPage()) {
            openIndexAt(indexIdx)
            return
        }
        if (pane === "playlist" && entityPage()) {
            var pageRow = selectedListTrack()
            if (pageRow && pageRow.path)
                playTrack(pageRow.path)
            return
        }
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        if (q && (pane === "playlist" || pane === "search")) {
            var hit = (searchHits || [])[playlistIdx]
            if (hit && hit.path)
                playTrack(hit.path)
            return
        }
        if (pane === "playlist") {
            if (shownPlaylist) {
                var shown = selectedListTrack()
                if (shown && shown.path)
                    playTrack(shown.path)
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
        if (row.kind === "page") {
            openIndex(row.id)
            focusPane("playlist")
            return
        }
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
        if (row.type === "dir") {
            playFolder(row.path)
            return
        }
        if (row.type === "track")
            playTrack(row.path)
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

    function pluginFile(rel) {
        var u = Qt.resolvedUrl(rel).toString()
        if (u.indexOf("file://") !== 0)
            return ""
        var path = decodeURIComponent(u.slice("file://".length))
        if (!path || path.charAt(0) !== "/" || path.indexOf("\n") >= 0 || path.indexOf("\r") >= 0)
            return ""
        return path
    }

    function openInFiles() {
        var path = String(playlistArtPath() || "").replace(/^\s+|\s+$/g, "")
        if (!path || path.charAt(0) !== "/" || path.indexOf("\n") >= 0 || path.indexOf("\r") >= 0)
            return
        var script = pluginFile("bin/open-files")
        if (!script)
            return
        Quickshell.execDetached(["/usr/bin/bash", script, path])
    }

    function patchLiked(path, likedNow) {
        if (reorderPending === 0)
            reorderScroll = playlistScroll
        queue = withLiked(queue, path, likedNow)
        if (shownPlaylist)
            shownTracks = withLiked(shownTracks, path, likedNow)
        if (searchHits && searchHits.length)
            searchHits = withLiked(searchHits, path, likedNow)
        if (pageTracks && pageTracks.length)
            pageTracks = withLiked(pageTracks, path, likedNow)
        if (service && String(player.path || "") === path && service.mergePlayer)
            service.mergePlayer({ path: path, liked: likedNow })
    }

    function toggleLike(path) {
        if (!path)
            return
        var id = discoverPreviewId(path)
        if (id) {
            var row = soundCloudRow(path)
            likeSoundCloud(id, path, row && row.artist, row && row.title, !!(row && row.liked))
            return
        }
        ipc("library.favorite.toggle", { path: path }, function(data) {
            root.err = ""
            root.patchLiked(path, !!(data && data.liked))
            root.refreshPlaylistCounts()
        })
    }

    function soundCloudRow(path) {
        var row = previewRowForPath(path)
        if (row)
            return row
        var lists = [queue, shownTracks, searchHits, pageTracks, downloadHits]
        for (var n = 0; n < lists.length; n++) {
            var rows = lists[n] || []
            for (var i = 0; i < rows.length; i++) {
                if (rows[i] && String(rows[i].path || "") === path)
                    return rows[i]
            }
        }
        return null
    }

    function likeSoundCloud(id, previewPath, artist, title, likedNow) {
        id = String(id || "")
        if (!id || id === "0")
            return
        ipc("library.soundcloud.like", {
            id: Number(id),
            like: !likedNow,
            artist: String(artist || ""),
            title: String(title || "")
        }, function(data) {
            if (!data)
                return
            if (String(data.status || "") === "running") {
                root.err = ""
                return
            }
            root.applySoundCloudLike(previewPath, data)
        })
    }

    function applySoundCloudLike(previewPath, data) {
        var likedNow = !!(data && data.liked)
        var path = String(data && data.path || "")
        var id = String(data && data.id || "")
        if (path)
            patchLiked(path, likedNow)
        if (previewPath && previewPath !== path)
            patchLiked(previewPath, likedNow)
        if (id)
            patchLikedPreview(id, likedNow)
        refreshPlaylistCounts()
    }

    function patchLikedPreview(id, likedNow) {
        id = String(id || "")
        if (!id)
            return
        function mark(rows) {
            var list = rows || []
            var next = []
            var changed = false
            for (var i = 0; i < list.length; i++) {
                var row = list[i]
                if (row && (String(row.id || "") === id || discoverPreviewId(row.path) === id)) {
                    next.push(Object.assign({}, row, { liked: likedNow }))
                    changed = true
                } else
                    next.push(row)
            }
            return changed ? next : rows
        }
        queue = mark(queue)
        if (shownPlaylist)
            shownTracks = mark(shownTracks)
        if (searchHits && searchHits.length)
            searchHits = mark(searchHits)
        if (pageTracks && pageTracks.length)
            pageTracks = mark(pageTracks)
        if (relatedTracks && relatedTracks.length)
            relatedTracks = mark(relatedTracks)
        if (adoptedRelated && adoptedRelated.length)
            adoptedRelated = mark(adoptedRelated)
    }

    function like() {
        if (mode === "queue" && (pane === "playlist" || pane === "search")) {
            likeNext(pickedPaths(), 0)
            return
        }
        likePlaying()
    }

    function likeNext(paths, i) {
        if (!paths || i >= paths.length) {
            if (i > 0)
                root.refreshPlaylistCounts()
            return
        }
        var path = paths[i]
        ipc("library.favorite.toggle", { path: path }, function(data) {
            root.err = ""
            root.patchLiked(path, !!(data && data.liked))
            root.likeNext(paths, i + 1)
        }, function() {
            root.refreshPlaylistCounts()
        })
    }

    function refreshPlaylistCounts() {
        loadPlaylistIndex()
        reloadShownPlaylist()
    }

    function reloadShownPlaylist() {
        var name = shownPlaylist
        if (!name)
            return
        var gen = ++shownGen
        var keep = ""
        var row = selectedListTrack()
        if (row && row.path)
            keep = String(row.path)
        var scroll = playlistScroll
        var idx = playlistIdx
        loadPlaylistItems(name, function(items) {
            if (gen !== root.shownGen || root.shownPlaylist !== name)
                return
            items = items || []
            var next = idx
            if (keep) {
                for (var i = 0; i < items.length; i++) {
                    if (String(items[i].path || "") === keep) {
                        next = i
                        break
                    }
                }
            }
            if (next >= items.length)
                next = Math.max(0, items.length - 1)
            root.reorderScroll = scroll
            root.shownTracks = items
            root.playlistIdx = next
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

    function playlistArtPath() {
        var row = selectedListTrack()
        var path = row && String(row.path || "")
        if (path)
            return path
        return trackPath
    }

    function artEditPath() {
        if (mode === "tags" && tagPath)
            return tagPath
        return trackPath
    }

    function patchListArt(rows, path, artPath) {
        return patchListArtPaths(rows, [path], artPath)
    }

    function patchListArtPaths(rows, paths, artPath) {
        var want = {}
        var listPaths = paths || []
        for (var n = 0; n < listPaths.length; n++)
            if (listPaths[n])
                want[String(listPaths[n])] = true
        var list = rows || []
        var next = []
        var changed = false
        for (var i = 0; i < list.length; i++) {
            var row = list[i]
            if (row && want[String(row.path || "")]) {
                next.push(Object.assign({}, row, { art: artPath, thumb: "" }))
                changed = true
            } else
                next.push(row)
        }
        return changed ? next : rows
    }

    function playlistRows() {
        switch (listSource()) {
        case "page":
            return pageTracks || []
        case "index":
            return []
        case "search":
            return searchHits || []
        case "files":
            return fileRows()
        case "lists":
            return playlistIndexRows()
        case "shown":
            return shownTracks || []
        }
        return queue || []
    }

    function playlistRowSelected(index) {
        var rows = playlistRows()
        if (index < 0 || index >= rows.length)
            return false
        var path = String((rows[index] && rows[index].path) || "")
        if (path && trackPicked(path))
            return true
        return !(playlistPicks && playlistPicks.length) && index === playlistIdx
    }

    function unselectPlaylist(index) {
        var rows = playlistRows()
        var path = String((rows[index] && rows[index].path) || "")
        var picks = (playlistPicks || []).slice()
        var next = []
        for (var i = 0; i < picks.length; i++) {
            if (picks[i] !== path)
                next.push(picks[i])
        }
        playlistPicks = next
        if (!next.length)
            playlistIdx = -1
    }

    function selectOnly(index) {
        var rows = playlistRows()
        if (index < 0)
            index = 0
        if (rows.length && index >= rows.length)
            index = rows.length - 1
        playlistIdx = index
        playlistAnchor = index
        var row = rows[index]
        playlistPicks = row && row.path ? [String(row.path)] : []
    }

    function togglePlaylist(index) {
        var rows = playlistRows()
        if (!rows.length || index < 0 || index >= rows.length)
            return
        var path = rows[index] && String(rows[index].path || "")
        if (!path)
            return
        var picks = (playlistPicks || []).slice()
        if (!picks.length) {
            var cur = rows[playlistIdx]
            if (cur && cur.path)
                picks = [String(cur.path)]
        }
        var next = []
        var found = false
        for (var i = 0; i < picks.length; i++) {
            if (picks[i] === path)
                found = true
            else
                next.push(picks[i])
        }
        if (!found)
            next.push(path)
        playlistIdx = index
        playlistAnchor = index
        playlistPicks = next
    }

    function extendPlaylist(index) {
        var rows = playlistRows()
        if (!rows.length)
            return
        if (index < 0)
            index = 0
        if (index >= rows.length)
            index = rows.length - 1
        var anchor = playlistAnchor
        if (anchor < 0 || anchor >= rows.length)
            anchor = playlistIdx
        var lo = Math.min(anchor, index)
        var hi = Math.max(anchor, index)
        var picks = []
        for (var i = lo; i <= hi; i++) {
            if (rows[i] && rows[i].path)
                picks.push(String(rows[i].path))
        }
        playlistIdx = index
        playlistPicks = picks
    }

    function trackPicked(path) {
        var picks = playlistPicks || []
        path = String(path || "")
        for (var i = 0; i < picks.length; i++) {
            if (picks[i] === path)
                return true
        }
        return false
    }

    function pickedPaths() {
        var rows = playlistRows()
        var want = {}
        var picks = playlistPicks || []
        for (var i = 0; i < picks.length; i++)
            want[picks[i]] = true
        var out = []
        for (var j = 0; j < rows.length; j++) {
            var path = rows[j] && String(rows[j].path || "")
            if (path && want[path])
                out.push(path)
        }
        if (!out.length) {
            var one = selectedListTrack()
            if (one && one.path)
                out.push(String(one.path))
        }
        return out
    }

    function patchSharedTags(paths) {
        var want = {}
        for (var i = 0; i < paths.length; i++)
            want[paths[i]] = true
        function patch(rows) {
            var list = rows || []
            var next = []
            var changed = false
            for (var n = 0; n < list.length; n++) {
                var row = list[n]
                if (row && want[row.path]) {
                    next.push(Object.assign({}, row, {
                        album: root.tagAlbum,
                        year: root.tagYear,
                        label: root.tagLabel
                    }))
                    changed = true
                } else
                    next.push(row)
            }
            return changed ? next : rows
        }
        queue = patch(queue)
        shownTracks = patch(shownTracks)
        searchHits = patch(searchHits)
        pageTracks = patch(pageTracks)
    }

    function selectedListTrack() {
        var rows = playlistRows()
        if (!rows || playlistIdx < 0 || playlistIdx >= rows.length)
            return null
        return rows[playlistIdx]
    }

    function likePlaying() {
        if (mode === "download" && downloadHit >= 0) {
            var hit = selectedDownloadHit()
            if (hit && hit.id) {
                likeSoundCloud(hit.id, "", hit.artist, hit.title, !!hit.liked)
                return
            }
        }
        var id = discoverPreviewId(trackPath)
        if (id) {
            var row = soundCloudRow(trackPath)
            likeSoundCloud(id, trackPath, row && row.artist, row && row.title, liked)
            return
        }
        toggleLike(trackPath)
    }

    function pushOverlay(next) {
        if (mode === "queue") {
            savedPlaylistIdx = playlistIdx
            savedFocus = pane === "search" ? "playlist" : pane
        }
        mode = next
        pane = "playlist"
        textCapture = false
    }

    function closeMode() {
        var back = savedFocus || "playlist"
        var returnMode = modeReturn
        modeReturn = ""
        eqKeys = false
        mode = returnMode === "library" ? "library" : "queue"
        artPreview = ""
        artHits = []
        artBusy = false
        artRestart = false
        artQueryFocus = false
        artPicking = false
        tagMany = false
        tagField = ""
        tagPaths = []
        tagScope = ""
        moveBusy = false
        tagBusy = false
        playlistIdx = savedPlaylistIdx
        pane = returnMode === "library" ? "playlist" : back
        textCapture = false
        err = ""
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
        rebuildSidebar()
    }

    function openSearch() {
        dismissDownload()
        if (mode !== "queue")
            closeMode()
        pane = "search"
    }

    function clearSearch() {
        searchQuery = ""
        searchHits = []
        searchArtists = []
        searchAlbums = []
        searchGen++
        pane = "playlist"
        textCapture = false
        rebuildSidebar()
        if (host && host.forceKeyFocus)
            host.forceKeyFocus()
    }

    function setSearch(text) {
        dismissDownload()
        closePages()
        searchQuery = String(text || "")
        searchTimer.restart()
    }

    function runSearch() {
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "")
        searchGen++
        var gen = searchGen
        if (!q) {
            searchHits = []
            searchArtists = []
            searchAlbums = []
            return
        }
        refreshSearchArtists()
        refreshSearchAlbums()
        prefetchIndex("artist")
        prefetchIndex("album")
        ipc("library.search", { mode: "search", query: q }, function(data) {
            if (gen !== root.searchGen)
                return
            var items = Array.isArray(data) ? data : []
            if (items.length > 400)
                items = items.slice(0, 400)
            root.searchHits = items
            root.selectOnly(0)
        })
    }

    function refreshSearchArtists() {
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "").toLowerCase()
        if (!q) {
            searchArtists = []
            return
        }
        var all = (indexCache && indexCache.artist) || []
        var out = []
        for (var i = 0; i < all.length; i++) {
            var name = String(all[i] && all[i].name || "")
            if (name.toLowerCase().indexOf(q) < 0)
                continue
            out.push(all[i])
        }
        out.sort(function(a, b) {
            var an = String(a && a.name || "").toLowerCase()
            var bn = String(b && b.name || "").toLowerCase()
            var as = an.indexOf(q) === 0 ? 0 : 1
            var bs = bn.indexOf(q) === 0 ? 0 : 1
            if (as !== bs)
                return as - bs
            return (Number(b && b.count) || 0) - (Number(a && a.count) || 0)
        })
        if (out.length > 8)
            out = out.slice(0, 8)
        searchArtists = out
    }

    function refreshSearchAlbums() {
        var q = String(searchQuery || "").replace(/^\s+|\s+$/g, "").toLowerCase()
        if (!q) {
            searchAlbums = []
            return
        }
        var all = (indexCache && indexCache.album) || []
        var out = []
        for (var i = 0; i < all.length; i++) {
            var name = String(all[i] && all[i].name || "")
            if (name.toLowerCase().indexOf(q) < 0)
                continue
            out.push(all[i])
        }
        out.sort(function(a, b) {
            var an = String(a && a.name || "").toLowerCase()
            var bn = String(b && b.name || "").toLowerCase()
            var as = an.indexOf(q) === 0 ? 0 : 1
            var bs = bn.indexOf(q) === 0 ? 0 : 1
            if (as !== bs)
                return as - bs
            return (Number(b && b.count) || 0) - (Number(a && a.count) || 0)
        })
        if (out.length > 8)
            out = out.slice(0, 8)
        searchAlbums = out
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
            root.reloadPage()
        })
    }

    function openTags() {
        if ((pane !== "playlist" && pane !== "search") || mode !== "queue")
            return
        var paths = pickedPaths()
        if (!paths.length)
            return
        var track = selectedListTrack()
        beginTagEdit(paths, String((track && track.path) || paths[0]))
    }

    function editTrack(path) {
        path = String(path || "")
        if (!path || mode !== "queue")
            return
        beginTagEdit([path], path)
    }

    function beginTagEdit(paths, path) {
        tagField = ""
        tagFieldValue = ""
        tagScope = ""
        tagMany = paths.length > 1
        tagPaths = paths
        tagPath = path
        tagBusy = !tagMany
        tagFocus = 0
        tagTitle = ""
        tagArtist = ""
        tagAlbum = ""
        tagYear = ""
        tagGenre = ""
        tagLabel = ""
        tagTags = ""
        pushOverlay("tags")
        if (tagMany) {
            artTarget = ""
            artPreview = ""
            artHits = []
            artPicking = false
            return
        }
        artTarget = path
        artQuery = ""
        artQueryFocus = false
        artPicking = false
        artPreview = ""
        artHits = []
        artIdx = 0
        runArtSearch("")
        ipc("library.track.tags.get", { path: tagPath }, function(data) {
            data = data || {}
            root.tagBusy = false
            root.tagTitle = String(data.title || "")
            root.tagArtist = String(data.artist || "")
            root.tagAlbum = String(data.album || "")
            root.tagYear = String(data.year || "")
            root.tagGenre = String(data.genre || "")
            root.tagLabel = String(data.label || "")
            root.tagTags = String(data.tags || "")
        }, function() {
            root.tagBusy = false
        })
    }

    function openFieldTags(field, value) {
        field = String(field || "")
        value = String(value || "").replace(/^\s+|\s+$/g, "")
        if (!field || !value || mode !== "queue")
            return
        ipc("library.track.tags.paths", { field: field, value: value }, function(data) {
            var paths = (data && data.paths) || []
            if (!paths.length) {
                root.err = "no tracks"
                return
            }
            root.tagField = field
            root.tagFieldValue = value
            root.tagMany = false
            root.tagPaths = paths
            root.tagScope = paths.length + " tracks · " + value
            root.tagPath = String(paths[0] || "")
            root.tagTags = ""
            if ((field === "artist" || field === "album") && data.tags_uniform)
                root.tagTags = String(data.tags || "")
            root.tagTagsInitial = root.tagTags
            root.tagBusy = false
            root.tagFocus = 0
            root.pushOverlay("tags")
        })
    }

    function tagLimit() {
        if (tagField === "artist" || tagField === "album")
            return 2
        if (tagField)
            return 1
        return tagMany ? 2 : 7
    }

    function moveTag(delta) {
        var limit = tagLimit()
        var next = tagFocus + delta
        if (next >= limit) {
            saveTags()
            return
        }
        if (next < 0)
            next = limit - 1
        tagFocus = next
    }

    function saveTags() {
        if (tagBusy)
            return
        if (tagField) {
            var fieldPaths = tagPaths || []
            if (!fieldPaths.length)
                return
            tagBusy = true
            var params = {
                field: tagField,
                value: tagFieldValue,
                paths: fieldPaths
            }
            if ((tagField === "artist" || tagField === "album") && tagTags !== tagTagsInitial)
                params.tags = tagTags
            ipc("library.track.tags.set_field", params, function() {
                root.tagBusy = false
                root.err = ""
                root.closeMode()
                root.afterLibraryEdit()
            }, function() {
                root.tagBusy = false
            })
            return
        }
        if (tagMany) {
            var paths = tagPaths || []
            if (!paths.length)
                return
            tagBusy = true
            ipc("library.track.tags.set_many", {
                paths: paths,
                album: tagAlbum,
                year: tagYear
            }, function() {
                root.tagBusy = false
                root.err = ""
                root.patchSharedTags(paths)
                root.closeMode()
                root.afterLibraryEdit()
            }, function() {
                root.tagBusy = false
            })
            return
        }
        if (!tagPath)
            return
        tagBusy = true
        ipc("library.track.tags.set", {
            path: tagPath,
            title: tagTitle,
            artist: tagArtist,
            album: tagAlbum,
            year: tagYear,
            genre: tagGenre,
            label: tagLabel,
            tags: tagTags
        }, function() {
            root.tagBusy = false
            root.err = ""
            root.closeMode()
            root.afterLibraryEdit()
        }, function() {
            root.tagBusy = false
        })
    }

    function openArt(path) {
        var target = String(path || "")
        if (!target)
            target = trackPath
        if (!target)
            return
        if (mode !== "queue" && mode !== "tags")
            return
        if (mode === "tags" && !tagMany && tagField === "" && tagPath === target)
            return
        beginTagEdit([target], target)
    }

    function searchDiscogs() {
        runArtSearch(String(artQuery || "").replace(/^\s+|\s+$/g, ""))
    }

    function runArtSearch(query) {
        var q = String(query || "").replace(/^\s+|\s+$/g, "")
        artBusy = true
        err = ""
        if (artSearchProc.running) {
            artRestart = true
            artNextQuery = q
            artSearchProc.signal(15)
            return
        }
        launchArtSearch(q)
    }

    function launchArtSearch(query) {
        var q = String(query || "").replace(/^\s+|\s+$/g, "")
        artHits = []
        artIdx = 0
        artPreview = ""
        _artBuf = ""
        _artOverflow = false
        var bin = Util.evoplayerBinPath(service ? service.home : "")
        if (q)
            artSearchProc.command = [bin, "art", "search", "--query", q, "--json"]
        else if (artTarget)
            artSearchProc.command = [bin, "art", "search", artTarget, "--json"]
        else {
            artBusy = false
            return
        }
        artSearchProc.running = true
    }

    function showArtPreview() {
        var hit = artHits[artIdx]
        if (!hit) {
            artPreview = ""
            return
        }
        artPreview = safeArtURL(hit.url) || safeArtURL(hit.thumb)
    }

    function applyArt(scope) {
        if (artBusy || mode !== "tags" || tagMany || tagField)
            return
        var hit = artHits[artIdx]
        var url = hit ? safeArtURL(hit.url) || safeArtURL(hit.thumb) : ""
        var target = artEditPath()
        if (!url || !target || artApplyProc.running)
            return
        artBusy = true
        artFromDrop = false
        var cmd = [Util.evoplayerBinPath(service ? service.home : ""), "art", "apply", target, url, "--json"]
        if (scope === "album")
            cmd.splice(cmd.length - 1, 0, "--album")
        _applyBuf = ""
        _applyOverflow = false
        artApplyProc.command = cmd
        artApplyProc.running = true
    }

    function dropArt(raw) {
        var target = artEditPath()
        if (!target) {
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
            cmd = [bin, "art", "set", target, local, "--json"]
        else if (safeArtURL(value))
            cmd = [bin, "art", "apply", target, value, "--json"]
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

    function finishArt(ok, artPath, paths) {
        artFromDrop = false
        artBusy = false
        if (!ok) {
            err = "art apply failed"
            return
        }
        var target = artEditPath()
        var targets = (paths && paths.length) ? paths : (target ? [target] : [])
        if (artPath && targets.length) {
            artEpoch++
            for (var i = 0; i < targets.length; i++) {
                if (targets[i] === trackPath) {
                    artOverride = artPath
                    artOverridePath = trackPath
                    break
                }
            }
            reorderScroll = playlistScroll
            queue = patchListArtPaths(queue, targets, artPath)
            shownTracks = patchListArtPaths(shownTracks, targets, artPath)
            searchHits = patchListArtPaths(searchHits, targets, artPath)
            pageTracks = patchListArtPaths(pageTracks, targets, artPath)
        }
        err = ""
        if (service && service.requestEnrich)
            service.requestEnrich(target)
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

    function cycleShuffleRepeat() {
        var mode = (shuffle ? 1 : 0) + (repeatOn ? 2 : 0)
        var next = (mode + 1) % 4
        var wantShuffle = next === 1 || next === 3
        var wantRepeat = next === 2 || next === 3
        if (wantShuffle !== shuffle)
            ipc("playback.shuffle", { on: wantShuffle }, null)
        if (wantRepeat !== repeatOn)
            ipc("playback.repeat", { on: wantRepeat }, null)
    }

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

    function eqHot() {
        if (!eqEnabled)
            return false
        if (Math.abs(eqPreamp) >= 0.05)
            return true
        var bands = eqBands || []
        for (var i = 0; i < bands.length; i++) {
            if (Math.abs(Number(bands[i]) || 0) >= 0.05)
                return true
        }
        return false
    }

    function loadEq() {
        ipc("eq.get", {}, function(data) {
            root.applyEq(data)
        })
    }

    function applyEq(data) {
        if (!data)
            return
        eqEnabled = data.enabled !== false
        eqPreamp = Number(data.preamp) || 0
        var next = []
        var src = data.bands || []
        for (var i = 0; i < 10; i++) {
            var row = src[i] || {}
            next.push(Number(row.gain) || 0)
        }
        eqBands = next
        eqPresets = data.presets || [null, null, null]
    }

    function refreshLibraryJob() {
        ipc("job.status", {}, function(data) {
            root.libraryJob = data || { status: "idle" }
        })
    }

    function startLibraryJob(method, params) {
        ipc(method, params || {}, function(data) {
            root.libraryJob = data || { status: "running" }
            root.err = ""
        })
    }

    function restartPlayer() {
        if (!service)
            return
        err = ""
        Quickshell.execDetached(service.playerCmd(["restart"]))
    }

    function cancelLibraryJob() {
        ipc("job.cancel", {}, function() {
            root.refreshLibraryJob()
        })
    }

    function eqPresetActive(slot) {
        var p = (eqPresets || [])[slot - 1]
        if (!p || !eqEnabled)
            return false
        if (Math.abs((Number(p[0]) || 0) - eqPreamp) >= 0.05)
            return false
        for (var i = 0; i < 10; i++) {
            if (Math.abs((Number(p[i + 1]) || 0) - (Number((eqBands || [])[i]) || 0)) >= 0.05)
                return false
        }
        return true
    }

    function pickEqPreset(slot) {
        eqSend.stop()
        if (eqSaveArmed) {
            eqSaveArmed = false
            ipc("eq.set", { enabled: eqEnabled, preamp: eqPreamp, bands: eqBands }, function() {
                root.ipc("eq.preset.save", { slot: slot }, function(data) {
                    root.applyEq(data)
                })
            })
            return
        }
        if (!(eqPresets || [])[slot - 1])
            return
        ipc("eq.preset.load", { slot: slot }, function(data) {
            root.applyEq(data)
        })
    }

    function toggleEqEnabled() {
        eqEnabled = !eqEnabled
        eqSend.restart()
    }

    function setEqGain(index, db) {
        db = Math.round(db)
        if (db < -12)
            db = -12
        if (db > 12)
            db = 12
        if (index <= 0)
            eqPreamp = db
        else {
            var bands = (eqBands || []).slice()
            while (bands.length < 10)
                bands.push(0)
            bands[index - 1] = db
            eqBands = bands
        }
        eqSend.restart()
    }

    function nudgeEq(delta) {
        var cur = eqIdx <= 0 ? eqPreamp : Number((eqBands || [])[eqIdx - 1]) || 0
        setEqGain(eqIdx, cur + delta)
    }

    function resetEq() {
        eqSend.stop()
        eqPreamp = 0
        eqBands = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]
        ipc("eq.reset", {}, function(data) {
            root.applyEq(data)
        })
    }

    function sendEq() {
        ipc("eq.set", { enabled: eqEnabled, preamp: eqPreamp, bands: eqBands }, null)
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
        if (shuffle || pane !== "playlist" || mode !== "queue" || shownPlaylist || page || reorderPending)
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
        if (mode === "eq" || mode === "library") {
            eqSaveArmed = false
            incomingGenrePath = ""
            closeMode()
        }
        browseIdx = index
        showBrowseTab(sidebar[index], false)
        focusPane("playlist")
    }

    function clickPlaylist(index, extend, toggle) {
        if (toggle)
            togglePlaylist(index)
        else if (extend)
            extendPlaylist(index)
        else
            selectOnly(index)
        focusPane("playlist")
    }

    function clickMove(index) {
        moveIdx = index
        focusPane("playlist")
    }

    function clickArt(index) {
        artIdx = index
        artQueryFocus = false
        artPicking = true
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
        if (page && pane !== "search") {
            pageBack()
            return
        }
        if (libraryView === "lists" && shownPlaylist) {
            backToPlaylists()
            return
        }
        if (pane === "search")
            clearSearch()
    }

    function dispatch(event) {
        var key = event.key
        var text = event.text || ""
        var shift = (event.modifiers & Qt.ShiftModifier) !== 0
        var ctrl = (event.modifiers & Qt.ControlModifier) !== 0

        if (mode === "library") {
            if (key === Qt.Key_Escape && eqSaveArmed) { eqSaveArmed = false; return true }
            if (key === Qt.Key_Escape && eqKeys) { eqKeys = false; return true }
            if (key === Qt.Key_Escape && incomingGenrePath) { incomingGenrePath = ""; return true }
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (key === Qt.Key_Up) { incomingIdx = Math.max(0, incomingIdx - 1); return true }
            if (key === Qt.Key_Down) {
                incomingIdx = Math.min(Math.max(0, (incomingFiles || []).length - 1), incomingIdx + 1)
                return true
            }
            if (text === "g" || text === "G") {
                var genreRow = (incomingFiles || [])[incomingIdx]
                if (genreRow && genreRow.path)
                    toggleIncomingGenre(String(genreRow.path))
                return true
            }
            if (text === "h" || text === "H") { toggleLibrary(); return true }
            return false
        }
        if (mode === "tags") {
            if (key === Qt.Key_Escape) { closeMode(); return true }
            if (artQueryFocus && !tagMany && tagField === "") {
                if (key === Qt.Key_Down) {
                    artQueryFocus = false
                    artPicking = true
                    textCapture = false
                    if (host && host.forceKeyFocus)
                        host.forceKeyFocus()
                    return true
                }
                if (key === Qt.Key_Up)
                    return true
                return false
            }
            if (artPicking && !tagMany && tagField === "") {
                if (key === Qt.Key_Up) {
                    if (artIdx <= 0) {
                        artPicking = false
                        artQueryFocus = true
                        return true
                    }
                    stepIndex("art", (artHits || []).length, -1)
                    return true
                }
                if (key === Qt.Key_Down) {
                    stepIndex("art", (artHits || []).length, 1)
                    return true
                }
            }
            if (key === Qt.Key_Up) { moveTag(-1); return true }
            if (key === Qt.Key_Down) { moveTag(1); return true }
            return false
        }
        if (mode === "art" && textCapture) {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (key === Qt.Key_Down) {
                artQueryFocus = false
                textCapture = false
                if (host && host.forceKeyFocus)
                    host.forceKeyFocus()
                return true
            }
            if (key === Qt.Key_Up) { return true }
            return false
        }
        if (mode === "download" && textCapture) {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (key === Qt.Key_Down) {
                var rows = downloadHits || []
                if (!rows.length)
                    return true
                textCapture = false
                if (downloadHit < 0)
                    downloadHit = 0
                else if (downloadHit < rows.length - 1)
                    downloadHit += 1
                if (host && host.forceKeyFocus)
                    host.forceKeyFocus()
                return true
            }
            if (key === Qt.Key_Up) {
                focusPane("browse")
                return true
            }
            return false
        }
        if (mode === "download" && pane === "playlist" && downloadHit >= 0) {
            if (key === Qt.Key_Escape) { onEsc(); return true }
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
            if (text === "d" || text === "D") { downloadSoundCloudHit(); return true }
            if (text === "l" || text === "L") { likePlaying(); return true }
            return true
        }
        if (mode === "download") {
            if (key === Qt.Key_Escape) { onEsc(); return true }
            if (text === "l" || text === "L") { likePlaying(); return true }
        }
        if (mode === "queue" && asGrid && pane === "playlist"
                && (listSource() === "shown" || listSource() === "page" || listSource() === "search")) {
            if (key === Qt.Key_Left) { stepTrackGrid(-1); return true }
            if (key === Qt.Key_Right) { stepTrackGrid(1); return true }
            if (key === Qt.Key_Up) { stepTrackGrid(-playlistColumns); return true }
            if (key === Qt.Key_Down) { stepTrackGrid(playlistColumns); return true }
            if (key === Qt.Key_PageUp) { stepTrackGrid(-playlistColumns * 4); return true }
            if (key === Qt.Key_PageDown) { stepTrackGrid(playlistColumns * 4); return true }
        }
        if (mode === "queue" && listSource() === "lists" && pane === "playlist") {
            if (key === Qt.Key_Left) { stepPlaylistGrid(-1); return true }
            if (key === Qt.Key_Right) { stepPlaylistGrid(1); return true }
            if (key === Qt.Key_Up) { stepPlaylistGrid(-playlistColumns); return true }
            if (key === Qt.Key_Down) { stepPlaylistGrid(playlistColumns); return true }
        }
        if (mode === "queue" && indexPage() && pane === "playlist") {
            if (textCapture) {
                if (key === Qt.Key_Escape) {
                    if (indexFilter)
                        setIndexFilter("")
                    else
                        releaseText()
                    return true
                }
                if (key === Qt.Key_Down) { releaseText(); return true }
                return false
            }
            if (key === Qt.Key_Left) { stepIndexGrid(-1); return true }
            if (key === Qt.Key_Right) { stepIndexGrid(1); return true }
            if (key === Qt.Key_Up) { stepIndexGrid(-indexColumns); return true }
            if (key === Qt.Key_Down) { stepIndexGrid(indexColumns); return true }
            if (key === Qt.Key_PageUp) { stepIndexGrid(-indexColumns * 4); return true }
            if (key === Qt.Key_PageDown) { stepIndexGrid(indexColumns * 4); return true }
            if (key === Qt.Key_Backspace) { pageBack(); return true }
            if (text === "/") { openSearch(); return true }
        }
        if (pane === "search" && textCapture) {
            if (key === Qt.Key_Escape) { clearSearch(); return true }
            if (key === Qt.Key_Up) {
                stepIndex("playlist", (searchHits || []).length, -1)
                selectOnly(playlistIdx)
                return true
            }
            if (key === Qt.Key_Down) {
                stepIndex("playlist", (searchHits || []).length, 1)
                selectOnly(playlistIdx)
                return true
            }
            if (key === Qt.Key_PageUp) { stepIndex("playlist", (searchHits || []).length, -playlistPage); selectOnly(playlistIdx); return true }
            if (key === Qt.Key_PageDown) { stepIndex("playlist", (searchHits || []).length, playlistPage); selectOnly(playlistIdx); return true }
            return false
        }

        if (mode === "discover" && pane === "playlist") {
            if (key === Qt.Key_Up) { stepDiscover(-1); return true }
            if (key === Qt.Key_Down) { stepDiscover(1); return true }
            if (key === Qt.Key_PageUp) { stepDiscover(-playlistPage); return true }
            if (key === Qt.Key_PageDown) { stepDiscover(playlistPage); return true }
        }

        if ((ctrl && key === Qt.Key_C) || (key === Qt.Key_Q && !shift && !ctrl)) {
            if (host && host.requestClose)
                host.requestClose()
            return true
        }
        if (text === "h" || text === "H") { toggleLibrary(); return true }
        if (key === Qt.Key_Escape) { onEsc(); return true }
        if (text === "/") { openSearch(); return true }
        if (key === Qt.Key_Space) { togglePlay(); return true }
        if (text === ",") { transport("prev"); return true }
        if (text === ".") { transport("next"); return true }
        if (text === "<") { seekBy(-10); return true }
        if (text === ">") { seekBy(10); return true }
        if (text === "-" || text === "_") { volumeDelta(-5); return true }
        if (text === "=" || text === "+") { volumeDelta(5); return true }
        if (text === "l" || text === "L") { like(); return true }
        if (text === "e" || text === "E") { openTags(); return true }
        if ((key === Qt.Key_Backspace || key === Qt.Key_Left) && libraryView === "lists" && shownPlaylist && !page) {
            backToPlaylists()
            return true
        }
        if ((key === Qt.Key_Backspace || key === Qt.Key_Left) && page && pane !== "browse") {
            pageBack()
            return true
        }
        if (key === Qt.Key_Right && listSource() === "lists") {
            var listed = playlistIndexRows()[playlistIdx]
            if (listed)
                openListedPlaylist(listed.id)
            return true
        }
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
        id: libraryRetry
        interval: 400
        onTriggered: root.ensureLibrary()
    }

    Timer {
        id: searchTimer
        interval: 180
        onTriggered: root.runSearch()
    }

    Timer {
        id: indexFilterTimer
        interval: 120
        onTriggered: root.applyIndexFilter()
    }

    Timer {
        id: eqSend
        interval: 80
        onTriggered: root.sendEq()
    }

    Timer {
        id: relatedTimer
        interval: 400
        onTriggered: root.loadRelated()
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
        function onIpcReadyChanged() {
            if (!root.service || !root.service.ipcReady || !root.active)
                return
            root.libraryTries = 0
            root.ensureLibrary()
            root.loadPlaylistIndex()
            root.loadPlaylistFilter()
        }
        function onDaemonJobUpdated(data) {
            if (!data)
                return
            var name = String(data.name || "")
            if (name === "scan" || name === "cache" || name === "import" || name === "art-maintain") {
                var running = String(data.status || "") === "running"
                if (name === "scan")
                    root.scanRunning = running
                else
                    root.libraryJob = data
                if (!running) {
                    root.indexCache = ({})
                    root.indexShown = ""
                    root.prefetchIndexes()
                    root.loadBrowse(root.browsePath)
                    if (name === "import")
                        root.loadIncoming()
                    if (root.entityPage())
                        root.reloadPage()
                }
                return
            }
            if (name === "download" || name === "download-url") {
                root.libraryJob = data
                root.noteDownloadJob(data)
                if (String(data.status || "") === "done" && data.result && data.result.path)
                    root.applySoundCloudLike("", data.result)
                if (name === "download" && String(data.status || "") === "done")
                    root.refreshPlaylistCounts()
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
                    var previewErr = root.jobError(data)
                    if (root.playDRMInBrave(previewErr))
                        return
                    root.downloadNote = ""
                    root.err = previewErr
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
                var discoverErr = root.jobError(data)
                if (name === "discover-preview" && root.playDRMInBrave(discoverErr))
                    return
                root.discoverNote = ""
                root.err = discoverErr
                return
            }
            if (status === "done") {
                root.err = ""
                var id = data.result && data.result.id
                if (name === "discover-keep" && id) {
                    root.markDiscoverKept(id)
                    root.markRelatedKept(id)
                }
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
            if (root.artRestart && root.mode === "tags" && !root.tagMany && root.tagField === "") {
                root.artRestart = false
                var next = root.artNextQuery
                Qt.callLater(function() { root.launchArtSearch(next) })
                return
            }
            root.artRestart = false
            root.artBusy = false
            if (root.mode !== "tags" || root.tagMany || root.tagField !== "") {
                root._artBuf = ""
                return
            }
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
            root.artPreview = ""
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
            var paths = []
            if (code === 0 && !root._applyOverflow) {
                try {
                    var parsed = JSON.parse(root._applyBuf || "{}")
                    artPath = String(parsed.art || "")
                    paths = parsed.paths || []
                } catch (e) {
                    artPath = ""
                }
            }
            root._applyBuf = ""
            root.finishArt(code === 0 && !root._applyOverflow, artPath, paths)
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

    Item {
        id: panes
        anchors.fill: parent
        anchors.margins: root.paneGap

            PlaylistPane {
                id: playlist
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                anchors.topMargin: browse.barHeight + 12
                sideMargin: 0
                view: root
            }

            BrowsePane {
                id: browse
                anchors.top: parent.top
                anchors.left: parent.left
                anchors.right: parent.right
                height: barHeight
                z: 5
                view: root
            }
    }

    TapHandler {
        acceptedButtons: Qt.BackButton | Qt.ForwardButton | Qt.ExtraButton1 | Qt.ExtraButton2 | Qt.ExtraButton3 | Qt.ExtraButton4
        acceptedDevices: PointerDevice.Mouse | PointerDevice.TouchPad
        gesturePolicy: TapHandler.WithinBounds
        onTapped: function(eventPoint, button) {
            var back = button === Qt.BackButton || button === Qt.ExtraButton1 || button === Qt.ExtraButton4
            if (back && root.page)
                root.pageBack()
            else if (back && root.libraryView === "lists" && root.shownPlaylist)
                root.backToPlaylists()
        }
    }
}
