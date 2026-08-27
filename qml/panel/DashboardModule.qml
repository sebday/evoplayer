import Quickshell
import Quickshell.Io
import QtQuick
import QtQuick.Layouts
import "compat"
import "compat/PluginIds.js" as PluginIds
import "widgets"
import "panels"
import "settings"
import "filetree/FiletreeLogic.js" as FiletreeLogic

Item {
    id: root

    property var shell: null
    property var host: null

    readonly property bool active: host && host.opened === true

    readonly property string home: Quickshell.env("HOME") || ""
    readonly property var playerMonitor: shell ? shell.serviceFor(PluginIds.pluginId) : null

    function ipcTraceEnabled() {
        var v = String(Quickshell.env("EVOPLAYER_TRACE_IPC") || "")
        return v !== "" && v !== "0" && v.toLowerCase() !== "false"
    }

    function agentDebugEnabled() {
        var v = String(Quickshell.env("EVOPLAYER_AGENT_DEBUG") || "")
        return v !== "" && v !== "0" && v.toLowerCase() !== "false"
    }

    // #region agent log
    readonly property string agentDebugLogPath: (home || "") + "/Projects/evoplayer/.cursor/debug-f38fb2.log"

    function agentDebug(hypothesisId, location, message, data) {
        if (!agentDebugEnabled())
            return
        var payload = JSON.stringify({
            sessionId: "f38fb2",
            runId: "pre-fix",
            hypothesisId: hypothesisId,
            location: location,
            message: message,
            data: data || {},
            timestamp: Date.now()
        })
        agentDebugLogProc.enqueue(payload)
    }

    function pumpAgentDebugLog() {
        if (agentDebugLogProc.running || !agentDebugLogProc._queue.length)
            return
        var line = agentDebugLogProc._queue.shift()
        agentDebugLogProc._line = line
        agentDebugLogProc.command = [
            "bash", "-c",
            "printf '%s\\n' " + JSON.stringify(line) + " >> " + JSON.stringify(agentDebugLogPath)
        ]
        agentDebugLogProc.running = true
    }
    // #endregion

    function traceIPC(msg) {
        if (ipcTraceEnabled())
            console.warn("[evoplayer-ipc] " + msg)
    }

    function playerCmd(args) {
        return Util.evoplayerCommand(home, args || [])
    }
    readonly property int columnPad: Theme.hoverPanelMargin
    readonly property int columnTopPad: Theme.hoverPanelTopPad
    readonly property int contentPad: Theme.hoverPanelContentPad
    readonly property int playerSectionGap: Theme.hoverPanelSectionSpacing
    readonly property int bodyFont: Theme.fontSize3xl
    readonly property int hintFont: Theme.fontSizeL
    readonly property int listFont: hintFont
    readonly property int titleFont: Theme.fontSize7xl
    readonly property int nowplayingMinBioWidth: 300
    readonly property int nowplayingInlineArtSize: 112
    readonly property int albumartWidth: {
        if (root.compactLayout)
            return 0
        var w = nowplayingSection.width
        var h = nowplayingSection.height
        if (w <= 0 || h <= 0)
            return nowplayingInlineArtSize
        var maxSide = Math.min(h, w - columnPad - nowplayingMinBioWidth)
        if (maxSide < nowplayingInlineArtSize)
            return 0
        return maxSide
    }
    readonly property bool nowplayingCompact: root.compactLayout
        || (nowplayingSection.width > 0
            && nowplayingSection.height > 0
            && albumartWidth <= 0)

    onNowplayingCompactChanged: {
        if (!nowplayingCompact)
            Qt.callLater(root.refreshNowPlayingArtDisplay)
    }
    readonly property int genreTabHeight: 34
    readonly property int controlsHeight: genreTabHeight
    readonly property int transportBtnSize: genreTabHeight
    readonly property int nowplayingTitleFont: nowplayingCompact
        ? Theme.fontSize6xl
        : Theme.fontSize9xl
    readonly property int nowplayingFieldsetMinHeight: nowplayingTitleFont * 2
        + Theme.fontSizeXl
        + Theme.fontSizeS
        + Theme.spacingM * 2
        + nowplayingWaveformMinHeight
        + columnPad
        + contentPad * 2
    readonly property int nowplayingMinBodyHeight: 200
    readonly property int nowplayingWaveformMinHeight: 56
    property var waveformSamples: []
    readonly property int transportIconFont: Theme.fontSize7xl
    readonly property int transportSecondaryIconFont: Theme.fontSize7xl
    readonly property int libraryFont: Theme.fontSizeS
    readonly property int sectionLabelFont: Theme.fontSizeL
    property bool trackTransitionPending: false
    property bool scannerHoldPending: false
    property string transportTargetPath: ""
    property bool panelTransitionGuard: false
    property int queueRevision: 0
    property bool scannerReverse: false
    readonly property bool nowplayingContentVisible: !trackTransitionPending
    readonly property var transportSnap: playerMonitor && playerMonitor.player
        ? playerMonitor.player
        : player
    readonly property bool playerPlaying: String(transportSnap.state || "") === "playing"
    readonly property real progress: Number(transportSnap.duration) > 0
        ? Math.max(0, Math.min(1, Number(transportSnap.position) / Number(transportSnap.duration)))
        : 0

    property var tracks: []
    property int selectedTrackIndex: -1
    property var player: ({})
    readonly property string playerPathForViz: String(player.path || "")
    property var libraryStats: ({ tracks: 0, genres: 0 })
    property string musicRoot: ""
    property bool jobBusy: false
    property bool libraryJobSawRunning: false
    property string jobLabel: ""
    property var downloadFiles: []
    property var downloadFolders: []
    property bool externalJobBusy: false
    property string externalJobLabel: ""
    property bool scanJobRunning: false
    property var scanJobProgress: ({ phase: "", folder: "", done: 0, total: 0 })
    property int ffmpegCpuPercent: 0
    property int evoplayerCpuPercent: 0
    readonly property bool libraryJobBusy: jobBusy || externalJobBusy
    readonly property string libraryJobActiveLabel: jobBusy ? jobLabel : externalJobLabel
    property string activeLibraryJobKey: ""
    readonly property bool libraryActivityBusy: libraryJobBusy || sortProc.running
    property bool tracksLoading: false
    property bool filetreeLoading: false
    property string filetreeForPath: ""
    property var filetreeExpanded: ({})
    property var filetreeChildren: ({})
    property var filetreeRows: []
    property bool filetreeTreeLoading: false
    property real filetreeScrollY: 0
    property real filetreeRestoreY: -1
    property real filetreeHoldY: -1
    property bool filetreeArtOnlyPatch: false
    property var _filetreeArtPatchQueue: ({})
    property bool _filetreeArtPatchFlushPending: false
    property bool _filetreeRefreshPending: false
    property bool filetreeReflowHidden: false
    property bool filetreeScrollLocked: false
    property bool filetreeUserScrolling: false
    property bool filetreeDeferVisibleArtWarm: false
    property bool _vizSubscribed: false
    property var filetreeScrollByKey: ({})
    property var filetreeListView: null
    property var playlistLibraryListView: null
    property var filetreeFolderMeta: ({})
    property bool filetreeLoadingMore: false
    property string selectedTrackPath: ""
    property bool playlistFocusNowPlaying: false
    property string selectedFiletreeFolderPath: ""
    property var selectedTrackCache: ({})
    property var playlists: []
    property var libraryPlaylists: []
    property string selectedPlaylist: ""
    readonly property string currentPlaylistId: "current"
    property bool currentPlaylistActive: false
    property string currentPlaylistPath: ""
    property var currentPlaylistTracks: []
    property bool playlistsLoading: false
    property int playlistTrackTotal: 0
    property int playlistTrackOffset: 0
    property bool playlistTracksLoadingMore: false
    property var playlistTrackList: null
    property var playlistViewByKey: ({})
    property real playlistRestoreY: -1
    property bool playlistScrollLocked: false
    property bool playlistUserScrolling: false
    property int tracksRevision: 0
    readonly property int playlistPageSize: 50
    property string resumePlaylist: ""
    property string playerScreen: "nowplaying"
    property string tabSearchText: ""
    property bool filetreeQueueBusy: false
    property bool filetreePanelOpen: false
    property bool playlistsPanelOpen: false
    property bool settingsPanelOpen: false
    property bool downloadsPanelOpen: false
    property bool statsPanelOpen: false
    property string playlistPanelMode: "library"

    onSettingsPanelOpenChanged: {
        if (settingsPanelOpen)
            loadPlayerSettings()
    }
    readonly property bool sidePanelOpen: filetreePanelOpen || playlistsPanelOpen || settingsPanelOpen || downloadsPanelOpen || statsPanelOpen
    readonly property bool splitSidePanelMode: filetreePanelOpen || playlistsPanelOpen
        || settingsPanelOpen || downloadsPanelOpen || statsPanelOpen || playerScreen === "filter"
    readonly property int sideContentStackIndex: {
        if (playerScreen === "filter")
            return 6
        if (downloadsPanelOpen)
            return 5
        if (statsPanelOpen)
            return 4
        if (settingsPanelOpen)
            return 3
        if (filetreePanelOpen)
            return 1
        if (playlistsPanelOpen)
            return 2
        return 0
    }
    property var queueUpNextTracks: []
    property string _queueUpNextSyncKey: ""
    readonly property var upNextTracks: queueUpNextTracks
    property var statsReport: ({})
    property bool statsLoading: false
    property string selectedLibraryPlaylist: ""
    readonly property bool nowplayingTabActive: !filetreePanelOpen && !playlistsPanelOpen
        && !settingsPanelOpen && !downloadsPanelOpen && !statsPanelOpen && playerScreen === "nowplaying"
    readonly property bool showBottomTransport: !compactLayout || splitSidePanelMode
    readonly property bool showInlineTransport: compactLayout && !splitSidePanelMode
    readonly property bool artPreviewActive: (filetreePanelOpen || playlistsPanelOpen)
        && String(selectedTrackPath || "") !== ""
        && String(selectedTrackPath) !== String(player.path || "")
    readonly property string artTargetPath: {
        if ((filetreePanelOpen || playlistsPanelOpen) && String(selectedTrackPath || "") !== "")
            return String(selectedTrackPath)
        return String((player && player.path) || "")
    }
    readonly property var selectedTrackInfo: {
        var path = String(selectedTrackPath || "")
        if (!path)
            return ({})
        return trackForPath(path) || ({})
    }
    readonly property var artTargetTrack: {
        var path = artTargetPath
        if (!path)
            return ({})
        if (path === String((player && player.path) || ""))
            return player
        return selectedTrackInfo
    }
    readonly property string nowplayingArt: {
        var p = String((player && player.path) || "")
        if (!p)
            return ""
        var _tracksRev = root.tracksRevision
        if (resolvedArtPath === p && resolvedArt)
            return resolvedArt
        var fromPlayer = String((player && player.art) || "")
        if (fromPlayer && fromPlayer.charAt(0) === "/")
            return fromPlayer
        return artForTrackPath(p)
    }
    readonly property string nowplayingArtPath: {
        var preview = artPreviewActive ? String(artTargetPath || "") : ""
        if (preview) {
            if (resolvedArtPath === preview && resolvedArt)
                return resolvedArt
            var previewInfo = selectedTrackInfo
            if (previewInfo && previewInfo.art && String(previewInfo.art).charAt(0) === "/")
                return String(previewInfo.art)
            return artForTrackPath(preview)
        }
        var p = String((player && player.path) || "")
        if (!p)
            return ""
        var _tracksRev = root.tracksRevision
        if (resolvedArtPath === p && resolvedArt)
            return resolvedArt
        var fromPlayer = String((player && player.art) || "")
        if (fromPlayer && fromPlayer.charAt(0) === "/")
            return fromPlayer
        return artForTrackPath(p)
    }
    readonly property string nowplayingArtUrl: nowplayingArtPath
        ? artUrl(nowplayingArtPath, false)
        : ""
    readonly property string displayedArt: {
        if (artPreviewActive)
            return String((selectedTrackInfo && selectedTrackInfo.art) || "")
        return nowplayingArt
    }
    property string resolvedArt: ""
    property string resolvedArtPath: ""
    property string sideArtHeldSource: ""
    property string sideArtIncomingSource: ""
    property string sideArtPendingSource: ""
    property bool sideArtLayoutReady: false
    property bool sideArtLoaded: false
    readonly property string albumartDisplayUrl: {
        var path = nowplayingArtPath
        if (!path)
            return ""
        root.artRevision
        return artUrl(path, true)
    }
    property bool queueExtendBusy: false
    property string filterKind: ""
    property string filterLabel: ""
    property var filterTracks: []
    property bool filterLoading: false
    property int volumeApplyTarget: 100
    property bool volumeApplyPending: false
    property bool transportApplyPending: false
    property var transportApplyTarget: null
    property string transportPreviewPath: ""
    property bool favoriteApplyPending: false
    property string favoriteApplyPath: ""
    property bool favoriteApplyLiked: false
    property var likedByPath: ({})
    property var waveformCache: ({})
    property var waveformPathByTrack: ({})
    property bool waveformLoading: false
    property var prefetchArtSources: []
    property var rowArtWarmQueue: []
    property bool rowArtWarmRunning: false
    property int rowArtWarmInflight: 0
    readonly property int rowArtWarmMaxInflight: 4
    property bool filetreeMoving: false
    property int filetreeArtRevision: 0
    property var neighborWaveformJobs: []
    property int neighborWaveformJobIndex: 0
    property real seekApplyTarget: 0
    property bool seekApplyPending: false
    property bool playbackStatePending: false
    property string playbackStateTarget: ""
    property string _lastSavedCurrentPathsKey: ""
    property string waveformRecheckPath: ""
    property bool jobStopRequested: false
    property bool sortStopRequested: false
    property bool menubarHidden: false
    property bool compactMode: false
    readonly property int compactLayoutBreakpoint: 768
    readonly property bool compactLayout: compactMode || width <= compactLayoutBreakpoint
    property string statusNote: ""
    readonly property string playerStatusText: {
        if (libraryActivityBusy)
            return jobLogInline()
        return statusNote
    }
    property string trashConfirmPath: ""
    property string trashConfirmTitle: ""
    readonly property bool trashConfirmOpen: trashConfirmPath !== ""
    readonly property bool keyShortcutsBlocked: tabSearchInput.activeFocus
        || trashConfirmOpen || artPickerSearchFocused || settingsFieldFocused
    property var volumeTransportBtn: null
    property string settingsMusicLibrary: ""
        property string settingsScUser: ""
        property string settingsScOAuthSource: ""
    property bool settingsReady: false
    property int settingsVizFrameRate: 45
    property int settingsVizSensitivity: 145
    property int settingsVizAutosens: 2
    property int settingsVizNoiseReduction: 34
    property real settingsVizMonstercat: 1
    property int settingsVizLowCutoff: 50
    property int settingsVizHighCutoff: 10000
    readonly property bool settingsInputsEnabled: !settingsPickProc.running
        && !settingsSetProc.running
        && !applyVizProc.running
    property bool settingsFieldFocused: false
    readonly property var libraryActions: [
        {
            key: "import",
            icon: "󰉍",
            label: "import incoming",
            button: "Import incoming",
            hint: "import incoming",
            args: ["import"]
        }
    ]
    property int artRevision: 0
    property bool artPickerOpen: false
    property bool artPickerLoading: false
    property string artPickerQuery: ""
    property var artPickerResults: []
    property string artPickerSearchText: ""
    property bool artPickerSearchFocused: false
    property string artApplyScope: "track"
    property string artPendingDropPath: ""
    readonly property bool artApplyAlbumAvailable: {
        var t = artTargetTrack
        var album = String((t && t.album) || "").trim()
        var title = String((t && t.title) || "").trim()
        return album !== "" && album !== title
    }
    readonly property var nowplayingMetaChips: {
        var chips = []
        var source = nowplayingTrack
        var label = String(source.label || "").trim()
        if (label !== "")
            chips.push({ label: label, kind: "label", value: label, tint: Theme.chartPalette[3] })
        var genre = String(source.genre || "").trim()
        if (genre !== "")
            chips.push({ label: genre, kind: "genre", value: genre, tint: Theme.accent })
        var year = String(source.year || "").trim()
        if (year !== "")
            chips.push({ label: year, kind: "year", value: year })
        var durationLabel = String(source.duration_label || "").trim()
        if (durationLabel !== "" && Number(source.duration || 0) > 0)
            chips.push({ label: durationLabel, kind: "duration", value: durationLabel })
        return chips
    }
    readonly property var nowplayingTrack: artPreviewActive ? artTargetTrack : player
    readonly property string nowplayingTitle: {
        var title = String(nowplayingTrack.title || "").trim()
        return title !== "" ? title : "No track"
    }
    readonly property string nowplayingArtist: String(nowplayingTrack.artist || "").trim()
    readonly property string nowplayingAlbum: {
        var album = String(nowplayingTrack.album || "").trim()
        var title = String(nowplayingTrack.title || "").trim()
        if (album === "" || album === title)
            return ""
        return album
    }
    readonly property bool nowplayingBylineVisible: nowplayingArtist !== "" || nowplayingAlbum !== ""
    readonly property string artworkLegendText: {
        var t = artTargetTrack || {}
        var album = String(t.album || "").trim()
        var title = String(t.title || "").trim()
        if (album !== "" && album !== title)
            return album
        return "Artwork"
    }

    function artUrl(path, withRevision) {
        if (!path)
            return ""
        var value = String(path).trim()
        if (!value)
            return ""
        var base = value.indexOf("file://") === 0 ? value : Util.fileUrl(value)
        if (!withRevision)
            return base
        var sep = base.indexOf("?") >= 0 ? "&" : "?"
        return base + sep + "rev=" + artRevision
    }

    function localPathFromUrl(url) {
        if (!url)
            return ""
        var s = url.toString ? url.toString() : String(url)
        if (s.indexOf("file://") === 0) {
            var p = s.replace(/^file:\/\//, "")
            if (p.indexOf("localhost/") === 0)
                p = p.substring("localhost/".length)
            return decodeURIComponent(p)
        }
        return s
    }

    function isImagePath(path) {
        var p = String(path || "").toLowerCase()
        return /\.(jpe?g|png|webp|gif|bmp)$/.test(p)
    }

    function bumpArtRevision() {
        artRevision++
        syncSideArtImageSource()
        kickSideArtUntilLoaded()
    }

    function kickSideArtUntilLoaded() {
        if (!sideArtLoaded && nowplayingArtPath && !nowplayingCompact)
            sideArtKickTimer.restart()
    }

    function refreshNowPlayingArtDisplay() {
        sideArtLoaded = false
        sideArtIncomingSource = ""
        syncSideArtImageSource()
        kickSideArtUntilLoaded()
    }

    function syncSideArtImageSource() {
        var path = nowplayingArtPath
        if (!path || nowplayingCompact) {
            sideArtHeldSource = ""
            sideArtIncomingSource = ""
            sideArtPendingSource = ""
            sideArtLoaded = false
            return
        }
        var next = artUrl(path, true)
        if (!sideArtLayoutReady) {
            sideArtPendingSource = next
            return
        }
        sideArtPendingSource = ""
        if (next === sideArtIncomingSource)
            return
        if (next === sideArtHeldSource && !sideArtIncomingSource) {
            if (!sideArtLoaded)
                kickSideArtUntilLoaded()
            return
        }
        if (!sideArtHeldSource || !sideArtLoaded) {
            sideArtIncomingSource = ""
            if (next !== sideArtHeldSource) {
                sideArtHeldSource = next
                sideArtLoaded = false
            }
            kickSideArtUntilLoaded()
            return
        }
        sideArtIncomingSource = next
    }

    function finishSideArtIncoming() {
        if (!sideArtIncomingSource)
            return
        sideArtIncomingSource = ""
        sideArtLoaded = true
        sideArtKickTimer.stop()
    }

    function beginSideArtHeldPromote(incomingUrl) {
        incomingUrl = String(incomingUrl || "")
        if (!incomingUrl || sideArtHeldSource === incomingUrl)
            return
        sideArtHeldSource = incomingUrl
    }

    function onAlbumArtUpdated(scope) {
        bumpArtRevision()
        refreshStatus()
        if (scope === "album")
            notify("album art updated", 2500)
        else if (scope === "track")
            notify("track art updated", 2500)
        else
            notify("art cleared", 2500)
    }

    function defaultArtApplyScope() {
        return artApplyAlbumAvailable ? "album" : "track"
    }

    function artScopeArgs() {
        return artApplyScope === "album" ? ["--album"] : ["--track"]
    }

    function patchTrackArtInLists(trackPath, art, thumb) {
        trackPath = String(trackPath || "")
        art = String(art || "")
        thumb = thumb !== undefined ? String(thumb || "") : ""
        if (!trackPath)
            return
        function patch(arr) {
            var next = []
            var i, entry, fields
            for (i = 0; i < arr.length; i++) {
                entry = arr[i]
                if (entry && String(entry.path) === trackPath) {
                    fields = {}
                    if (art)
                        fields.art = art
                    if (thumb)
                        fields.thumb = thumb
                    if (Object.keys(fields).length)
                        next.push(Object.assign({}, entry, fields))
                    else
                        next.push(entry)
                } else {
                    next.push(entry)
                }
            }
            return next
        }
        assignTracks(patch(tracks))
        currentPlaylistTracks = patch(currentPlaylistTracks)
        filterTracks = patch(filterTracks)
        patchFiletreeArt(trackPath, art, filetreePanelOpen && !playlistsPanelOpen, thumb)
        if (playlistsPanelOpen)
            tracksRevision++
        if (trackPath === String(player.path || "") && art.charAt(0) === "/") {
            if (String(player.art || "") !== art)
                player = Object.assign({}, player, { art: art })
            bumpArtRevision()
        }
    }

    function patchTracksArtBatch(updates) {
        if (!updates || typeof updates !== "object")
            return
        function patch(arr) {
            var next = []
            var i, entry, fields
            for (i = 0; i < arr.length; i++) {
                entry = arr[i]
                if (!entry || !entry.path) {
                    next.push(entry)
                    continue
                }
                fields = updates[String(entry.path)]
                if (fields && Object.keys(fields).length)
                    next.push(Object.assign({}, entry, fields))
                else
                    next.push(entry)
            }
            return next
        }
        assignTracks(patch(tracks))
        currentPlaylistTracks = patch(currentPlaylistTracks)
        filterTracks = patch(filterTracks)
        tracksRevision++
    }

    function warmPlaylistPage(entries) {
        if (!entries || !entries.length)
            return
        var paths = []
        var i, entry, art, thumb
        for (i = 0; i < entries.length && paths.length < playlistPageSize; i++) {
            entry = entries[i]
            if (!entry || !entry.path)
                continue
            art = String(entry.art || "")
            thumb = String(entry.thumb || "")
            if (art.charAt(0) === "/" && thumb.charAt(0) !== "/")
                paths.push(String(entry.path))
        }
        if (!paths.length)
            return
        if (playerMonitor && typeof playerMonitor.ipcCallVoid === "function")
            playerMonitor.ipcCallVoid("library.warm.batch", { paths: paths, workers: 8, art: true })
    }

    function patchAlbumArtInLists(trackPath, art) {
        trackPath = String(trackPath || "")
        art = String(art || "")
        var slash = trackPath.lastIndexOf("/")
        var dir = slash >= 0 ? trackPath.substring(0, slash + 1) : ""
        if (!dir)
            return patchTrackArtInLists(trackPath, art)
        function sameAlbum(path) {
            path = String(path || "")
            if (path.indexOf(dir) !== 0)
                return false
            return path.indexOf("/", dir.length) < 0
        }
        function patch(arr) {
            var next = []
            var i, entry
            for (i = 0; i < arr.length; i++) {
                entry = arr[i]
                if (entry && sameAlbum(entry.path)) {
                    entry.art = art
                    next.push(Object.assign({}, entry, { art: art }))
                } else {
                    next.push(entry)
                }
            }
            return next
        }
        assignTracks(patch(tracks))
        currentPlaylistTracks = patch(currentPlaylistTracks)
        filterTracks = patch(filterTracks)
        patchFiletreeArt(trackPath, art, true)
        tracksRevision++
    }

    function patchWaveformInLists(trackPath, waveform) {
        trackPath = String(trackPath || "")
        waveform = String(waveform || "")
        if (!trackPath || !waveform)
            return
        function patch(arr) {
            var next = []
            var i, entry
            for (i = 0; i < arr.length; i++) {
                entry = arr[i]
                if (entry && String(entry.path) === trackPath)
                    next.push(Object.assign({}, entry, { waveform: waveform }))
                else
                    next.push(entry)
            }
            return next
        }
        assignTracks(patch(tracks))
        currentPlaylistTracks = patch(currentPlaylistTracks)
        filterTracks = patch(filterTracks)
        patchFiletreeWaveform(trackPath, waveform)
        rememberWaveformPath(trackPath, waveform)
        tracksRevision++
        if (trackPath === String(player.path || ""))
            player = Object.assign({}, player, { waveform: waveform })
    }

    function tracksShareAlbumFolder(a, b) {
        a = String(a || "")
        b = String(b || "")
        var aslash = a.lastIndexOf("/")
        var bslash = b.lastIndexOf("/")
        if (aslash < 0 || bslash < 0)
            return false
        return a.substring(0, aslash + 1) === b.substring(0, bslash + 1)
    }

    function applyArtCommandResult(text, scope, trackPath) {
        var data = JSON.parse(String(text || "{}"))
        if (!data || (data.art === undefined && data.ok !== true))
            throw new Error("art failed")
        trackPath = String(trackPath || artTargetPath || "")
        var art = data.art !== undefined && data.art !== null ? String(data.art) : ""
        var playingPath = String(player.path || "")
        if (playingPath && (playingPath === trackPath
                || (scope === "album" && tracksShareAlbumFolder(playingPath, trackPath))))
            player = Object.assign({}, player, { art: art })
        if (!art)
            invalidateResolvedArt(trackPath)
        else if (playingPath === trackPath)
            applyPlayerArt(trackPath, art)
        if (scope === "album")
            patchAlbumArtInLists(trackPath, art)
        else
            patchTrackArtInLists(trackPath, art)
        if (String(selectedTrackCache.path || "") === trackPath)
            selectedTrackCache = Object.assign({}, selectedTrackCache, { art: art })
        onAlbumArtUpdated(scope)
    }

    function setAlbumArtFromFile(imagePath) {
        var track = String(artTargetPath || "")
        if (!track || !imagePath)
            return
        var scope = artApplyScope
        runMusic(["art", "set", track, imagePath].concat(artScopeArgs()).concat(["--json"]), function(text) {
            try {
                root.applyArtCommandResult(text, scope, track)
            } catch (e) {
                root.notify("could not update art", 3000)
            }
        })
    }

    function selectArtApplyScope(scope) {
        artApplyScope = scope === "album" ? "album" : "track"
        if (!artPendingDropPath)
            return
        var imagePath = artPendingDropPath
        artPendingDropPath = ""
        setAlbumArtFromFile(imagePath)
        artPickerOpen = false
    }

    function openArtPickerForDrop(imagePath) {
        var track = String(artTargetPath || "")
        imagePath = String(imagePath || "")
        if (!track || !imagePath)
            return
        artPendingDropPath = imagePath
        artApplyScope = defaultArtApplyScope()
        artPickerOpen = true
        artPickerLoading = false
        artPickerResults = []
        artPickerQuery = ""
    }

    function applyAlbumArtFromUrl(url) {
        var track = String(artTargetPath || "")
        if (!track || !url)
            return
        var scope = artApplyScope
        artPickerLoading = true
        runMusic(["art", "apply", track, url].concat(artScopeArgs()).concat(["--json"]), function(text) {
            artPickerLoading = false
            try {
                root.applyArtCommandResult(text, scope, track)
                artPickerOpen = false
            } catch (e) {
                root.notify("could not update art", 3000)
            }
        })
    }

    function toggleMenubar() {
        menubarHidden = !menubarHidden
    }

    function toggleCompactMode() {
        compactMode = !compactMode
        menubarHidden = compactMode
        Qt.callLater(function() {
            if (!root.nowplayingCompact)
                root.refreshNowPlayingArtDisplay()
            else
                root.syncSideArtImageSource()
        })
    }

    function copyTitleToClipboard() {
        var title = String(player.title || "").trim()
        if (!title)
            return
        Quickshell.execDetached(["wl-copy", "--", title])
        notify("copied title", 1500)
    }

    function copyTrackArtistTitle(track) {
        track = track || {}
        var artist = String(track.artist || "").trim()
        var title = String(track.title || "").trim()
        var text = artist && title ? (artist + " - " + title) : (artist || title)
        if (!text)
            return
        Quickshell.execDetached(["wl-copy", "--", text])
        notify("copied artist - title", 1500)
    }

    function searchArtPicker(query) {
        artSearchDebounce.stop()
        query = String(query || "").trim()
        var track = String(artTargetPath || "")
        artPendingDropPath = ""
        artPickerLoading = true
        var args
        if (query)
            args = ["art", "search", "--query", query, "--json"]
        else if (track)
            args = ["art", "search", track, "--json"]
        else {
            artPickerLoading = false
            return
        }
        if (runQuery(args, function(text) {
            root.artPickerLoading = false
            try {
                var data = JSON.parse(String(text || "{}"))
                root.artPickerQuery = String(data.query || query)
                root.artPickerResults = data.results || []
            } catch (e) {
                root.artPickerResults = []
            }
        }))
            return
        Qt.callLater(function() {
            if (root.artPickerOpen)
                root.searchArtPicker(query)
        })
    }

    function openArtPicker() {
        var track = String(artTargetPath || "")
        if (!track)
            return
        artPendingDropPath = ""
        artApplyScope = defaultArtApplyScope()
        artPickerOpen = true
        artPickerLoading = true
        artPickerResults = []
        artPickerQuery = ""
        artPickerSearchText = ""
        searchArtPicker("")
    }

    function closeArtPicker() {
        artPendingDropPath = ""
        artPickerOpen = false
        artPickerSearchFocused = false
        artSearchDebounce.stop()
    }

    function queueArtSearch(text) {
        artPickerSearchText = String(text || "")
        if (String(text || "").trim() !== "")
            artSearchDebounce.restart()
        else
            artSearchDebounce.stop()
    }

    function showNowplaying() {
        // #region agent log
        agentDebug("H5", "DashboardModule.qml:showNowplaying", "tab switch start", {
            playlistsOpen: playlistsPanelOpen,
            playing: playerPlaying,
            path: String(transportSnap.path || ""),
            guard: panelTransitionGuard,
            transportPending: transportApplyPending,
            playbackPending: playbackStatePending
        })
        // #endregion
        traceIPC("showNowplaying playlistsOpen=" + playlistsPanelOpen
            + " playing=" + playerPlaying + " path=" + String(transportSnap.path || ""))
        savePlaylistView(selectedPlaylist)
        panelTransitionGuard = true
        panelTransitionTimer.restart()
        // Defer layout swap so this click cannot land on transport after reflow.
        Qt.callLater(function() {
            filetreePanelOpen = false
            playlistsPanelOpen = false
            settingsPanelOpen = false
            downloadsPanelOpen = false
            statsPanelOpen = false
            playerScreen = "nowplaying"
            forceRevealNowPlaying()
            syncVizSubscription()
            // #region agent log
            agentDebug("H5", "DashboardModule.qml:showNowplaying", "tab switch deferred", {
                nowplayingActive: root.nowplayingTabActive,
                playing: root.playerPlaying,
                path: String(root.transportSnap.path || "")
            })
            // #endregion
        })
    }

    function syncPlayerFromMonitor() {
        var snap = playerMonitor && playerMonitor.player
        if (!snap || typeof snap !== "object")
            return
        transportTargetPath = ""
        applyStatus(JSON.stringify(snap), true)
    }

    function queueAlreadyPlaying(startPath, pathList) {
        var snap = playerMonitor && playerMonitor.player
        if (!snap)
            return false
        if (String(snap.path || "") !== String(startPath || ""))
            return false
        if (Number(snap.playlist_count) !== pathList.length)
            return false
        return String(snap.state || "") === "playing"
    }

    function toggleFiletreePanel() {
        savePlaylistView(selectedPlaylist)
        filetreePanelOpen = true
        playlistsPanelOpen = false
        settingsPanelOpen = false
        downloadsPanelOpen = false
        statsPanelOpen = false
        playerScreen = "nowplaying"
        kickSidePanels()
    }

    function togglePlaylistsPanel() {
        filetreePanelOpen = false
        settingsPanelOpen = false
        downloadsPanelOpen = false
        statsPanelOpen = false
        playlistsPanelOpen = true
        playlistPanelMode = "library"
        playerScreen = "nowplaying"
        refreshPlaylistsPanelView()
        kickSidePanels()
    }

    function kickSidePanels() {
        sidePanelKickTimer.restart()
    }

    function refreshFiletreeListView() {
        if (!filetreePanelOpen)
            return true
        if (!filetreeRows.length && !filetreeTreeLoading)
            loadFiletreeRoot()
        if (!filetreeListView)
            return false
        var list = filetreeListView
        if (list.width <= 8 || list.height <= 8)
            return false
        syncFiletreePanel()
        if (filetreeTreeLoading)
            return false
        return filetreeRows.length > 0
    }

    function refreshPlaylistListView() {
        if (!playlistsPanelOpen)
            return true
        if (playlistPanelMode !== "library")
            return true
        if (!libraryPlaylists.length && !playlistsLoading)
            loadPlaylists(true)
        if (!playlistLibraryListView)
            return false
        var list = playlistLibraryListView
        if (list.width <= 8 || list.height <= 8)
            return false
        if (playlistsLoading)
            return false
        return libraryPlaylists.length > 0
    }

    function refreshFiletreePanelView() {
        syncFiletreePanel()
    }

    function syncFiletreePanel() {
        filetreeReflowHidden = false
        filetreeHoldY = -1
        if (!filetreeRows.length && !filetreeTreeLoading)
            loadFiletreeRoot()
        if (!filetreeListView)
            return
        rebuildFiletreeRows(true)
        restoreFiletreeScroll()
        if (filetreeRestoreY >= 0)
            filetreeKickScrollTimer.restart()
    }

    function attachFiletreeList(listView) {
        filetreeListView = listView
        kickSidePanels()
    }

    function detachFiletreeList() {
        filetreeListView = null
    }

    function refreshPlaylistsPanelView() {
        if (!libraryPlaylists.length && !playlistsLoading)
            loadPlaylists(true)
        else
            Qt.callLater(root.ensureLibraryData)
        if (playlistPanelMode === "tracks" && selectedPlaylist)
            loadPlaylistTracks(selectedPlaylist, true)
        else if (selectedPlaylist === currentPlaylistId)
            refreshCurrentPlaylistView()
    }

    function toggleSettingsPanel() {
        savePlaylistView(selectedPlaylist)
        filetreePanelOpen = false
        playlistsPanelOpen = false
        downloadsPanelOpen = false
        statsPanelOpen = false
        settingsPanelOpen = true
        playerScreen = "nowplaying"
        loadPlayerSettings()
    }

    function toggleDownloadsPanel() {
        savePlaylistView(selectedPlaylist)
        filetreePanelOpen = false
        playlistsPanelOpen = false
        settingsPanelOpen = false
        statsPanelOpen = false
        downloadsPanelOpen = true
        playerScreen = "nowplaying"
        loadIncomingDownloads()
        kickSidePanels()
    }

    function toggleStatsPanel() {
        savePlaylistView(selectedPlaylist)
        filetreePanelOpen = false
        playlistsPanelOpen = false
        settingsPanelOpen = false
        downloadsPanelOpen = false
        statsPanelOpen = true
        playerScreen = "nowplaying"
        loadStatsReport()
        kickSidePanels()
    }

    function loadStatsReport() {
        statsLoading = true
        runQuery(["history", "report", "--json", "--limit", "12"], function(text) {
            statsLoading = false
            try {
                statsReport = JSON.parse(String(text || "{}"))
            } catch (e) {
                statsReport = {}
            }
        })
    }

    function statsRecapStat(kind) {
        var stat = ((root.statsReport.recap || {})[kind]) || {}
        var top = stat.top || {}
        return {
            count: Number(stat.count) || 0,
            deltaPct: Number(stat.delta_pct) || 0,
            trendUp: stat.up !== false,
            peak: !!stat.peak,
            allTime: !!stat.all_time || root.statsReport.period === "overall",
            topName: String(top.name || ""),
            topCount: Number(top.count) || 0,
            topArtist: String(top.artist || ""),
            topTitle: String(top.title || "")
        }
    }

    function statsTopTracks() {
        return (root.statsReport.top_tracks || []).slice(0, 12)
    }

    function statsTopTrackMaxCount() {
        var rows = root.statsTopTracks()
        var max = 1
        for (var i = 0; i < rows.length; i++)
            max = Math.max(max, Number(rows[i].count) || 0)
        return max
    }

    function statsTrackField(value) {
        if (value === undefined || value === null)
            return ""
        if (typeof value === "object")
            return String(value["#text"] || value.name || value.text || "").trim()
        var text = String(value).trim()
        if (text.indexOf("'#text'") >= 0 || text.indexOf("\"#text\"") >= 0) {
            var match = text.match(/['"]#text['"]\s*:\s*['"]([^'"]*)['"]/)
            if (match && match[1])
                return match[1].trim()
        }
        return text
    }

    function statsTrackArtist(row) {
        return root.statsTrackField(row && row.artist)
    }

    function statsTrackTitle(row) {
        return root.statsTrackField(row && (row.title || row.name))
    }

    function openStatsTrackFilter(row) {
        var artist = root.statsTrackArtist(row)
        var title = root.statsTrackTitle(row)
        if (!title)
            return
        var label = artist ? artist + " — " + title : title
        root.openFilter("track", title, label)
    }

    function createUserPlaylist(name) {
        name = String(name || "").trim()
        if (!name)
            return
        runPlaylistQuery(["playlist", "create", name, "--json"], function() {
            loadPlaylists(true)
            notify("created playlist " + name, 2500)
        })
    }

    function renameUserPlaylist(oldName, newName) {
        oldName = String(oldName || "").trim()
        newName = String(newName || "").trim()
        if (!oldName || !newName)
            return
        runPlaylistQuery(["playlist", "rename", oldName, newName, "--json"], function() {
            if (selectedLibraryPlaylist === oldName)
                selectedLibraryPlaylist = newName
            if (selectedPlaylist === oldName)
                selectedPlaylist = newName
            loadPlaylists(true)
            notify("renamed playlist", 2500)
        })
    }

    function deleteUserPlaylist(name) {
        name = String(name || "").trim()
        if (!name)
            return
        runPlaylistQuery(["playlist", "delete", name, "--json"], function() {
            if (selectedLibraryPlaylist === name)
                selectedLibraryPlaylist = ""
            loadPlaylists(true)
            notify("deleted playlist", 2500)
        })
    }

    function seekBy(delta) {
        var pos = Number(player.position) || 0
        var dur = Number(player.duration) || 0
        if (dur <= 0)
            return
        queueSeek(pos + Number(delta || 0))
    }

    function focusUpNextInPlaylist() {
        var entry = upNextTracks.length > 0 ? upNextTracks[0] : null
        if (!entry || !entry.path)
            return
        focusTrackInPlaylist(String(entry.path))
    }

    function focusTrackInPlaylist(path) {
        path = String(path || "")
        if (!path)
            return
        playerScreen = "nowplaying"
        filetreePanelOpen = false
        settingsPanelOpen = false
        statsPanelOpen = false
        playlistsPanelOpen = true
        playlistPanelMode = "tracks"
        if (selectedPlaylist !== currentPlaylistId) {
            if (selectedPlaylist)
                savePlaylistView(selectedPlaylist)
            selectedPlaylist = currentPlaylistId
            syncPlaylistTabPosition()
            playlistFocusNowPlaying = false
            loadPlaylistTracks(currentPlaylistId)
        }
        loadPlaylistUntilTrack(path)
    }

    function queueUpNextSyncKey() {
        return String(player.path || "") + "|" + String(player.playlist_pos || 0)
            + "|" + String(player.playlist_count || 0) + "|" + (player.shuffle ? "1" : "0")
    }

    function maybeRefreshQueueUpNext() {
        var key = queueUpNextSyncKey()
        if (key === _queueUpNextSyncKey)
            return
        _queueUpNextSyncKey = key
        refreshQueueUpNext()
    }

    function refreshQueueUpNext() {
        var path = String(player.path || "")
        var count = Number(player.playlist_count) || 0
        if (!path || count < 2) {
            queueUpNextTracks = []
            return
        }
        if (playerMonitor && typeof playerMonitor.ipcCall === "function") {
            playerMonitor.ipcCall("queue.up_next", { limit: 3 }, function(ok, msg) {
                if (!ok || !msg || !msg.data) {
                    root.queueUpNextTracks = []
                    return
                }
                root.queueUpNextTracks = Array.isArray(msg.data) ? msg.data : []
            })
            return
        }
        if (!runQuery(["queue", "up-next", "--json", "--limit", "3"], function(text) {
            try {
                root.queueUpNextTracks = JSON.parse(String(text || "[]"))
            } catch (e) {
                root.queueUpNextTracks = []
            }
        }))
            root.queueUpNextTracks = []
    }

    function upNextLineText(tracks) {
        var list = tracks || root.upNextTracks
        return list.map(function(t) {
            var artist = String(t.artist || "").trim()
            var title = String(t.title || "").trim()
                || String(t.path || "").split("/").pop()
            return artist ? artist + " — " + title : title
        }).join("   ·   ")
    }

    function libraryActivityBannerLabel() {
        var key = String(activeLibraryJobKey || libraryJobActiveLabel || "").trim()
        if (sortProc.running)
            return "Sorting"
        if (key === "scan")
            return "Scanning library"
        if (key === "import")
            return "Importing"
        if (key === "download" || key === "download-url")
            return "Downloading"
        if (key === "cache")
            return "Caching"
        if (key === "art.maintain")
            return "Fixing art"
        if (key)
            return key
        return "Processing"
    }

    function libraryCpuLoadLabel(proc, percent) {
        var n = Number(percent) || 0
        if (n <= 0)
            return ""
        return String(proc || "") + " " + n + "% cpu load"
    }

    function libraryCpuLoadParts() {
        var parts = []
        var ffmpegLoad = libraryCpuLoadLabel("ffmpeg", ffmpegCpuPercent)
        if (ffmpegLoad)
            parts.push(ffmpegLoad)
        var evoLoad = libraryCpuLoadLabel("evoplayer", evoplayerCpuPercent)
        if (evoLoad)
            parts.push(evoLoad)
        return parts
    }

    function libraryCpuLoadLine() {
        return libraryCpuLoadParts().join("   ·   ")
    }

    function compactScanFolder(folder) {
        folder = String(folder || "").trim().replace(/\\/g, "/")
        if (!folder)
            return ""
        var parts = folder.split("/").filter(function(p) { return p })
        if (parts.length > 2)
            return parts.slice(-2).join("/")
        return folder
    }

    function libraryActivityLineText() {
        return libraryCpuLoadLine()
    }

    function showPlaylistLibrary() {
        savePlaylistView(selectedPlaylist)
        playlistPanelMode = "library"
    }

    function checkAutoExtendQueue() {
        if (queueExtendBusy || !currentPlaylistTracks.length)
            return
        var path = String(player.path || "")
        if (!path)
            return
        var idx = -1
        for (var i = 0; i < currentPlaylistTracks.length; i++) {
            if (currentPlaylistTracks[i].path === path) {
                idx = i
                break
            }
        }
        if (idx < 0 || idx < currentPlaylistTracks.length - 1)
            return
        var dur = Number(player.duration) || 0
        var pos = Number(player.position) || 0
        if (dur > 0 && String(player.state || "") === "playing" && pos < dur - 1.5)
            return
        autoExtendQueue()
    }

    function autoExtendQueue() {
        if (queueExtendBusy)
            return
        queueExtendBusy = true
        runMusic(["queue", "extend", "--json"], function(text) {
            queueExtendBusy = false
            try {
                var result = JSON.parse(String(text || "{}"))
                if (Number(result.added || 0) > 0) {
                    root.loadCurrentPlaylist(function() {
                        if (root.playlistsPanelOpen && root.selectedPlaylist === root.currentPlaylistId)
                            root.loadPlaylistTracks(root.currentPlaylistId)
                    })
                    root.notify("+" + result.added + " from " + String(result.folder || "").split("/").pop(), 2500)
                }
            } catch (e) {
            }
            root.refreshStatus()
        }, queuePlayProc)
    }

    function notify(body, durationMs) {
        var text = String(body || "").trim()
        if (text) {
            statusNote = text
            statusNoteTimer.interval = durationMs || 3000
            statusNoteTimer.restart()
        }
        if (!shell) return
        if (playerMonitor && typeof playerMonitor.showBrief === "function")
            playerMonitor.showBrief("Evoplayer", String(body || ""), durationMs || 3000)
    }

    function clearAlbumArt() {
        var track = String(artTargetPath || "")
        if (!track)
            return
        runMusic(["art", "clear", track, "--json"], function(text) {
            try {
                root.applyArtCommandResult(text, "", track)
                root.artPickerOpen = false
            } catch (e) {
                root.notify("could not update art", 3000)
            }
        })
    }

    function formatJobLog(text) {
        if (!text)
            return ""
        return String(text).replace(/\r\n/g, "\n").replace(/\r/g, "\n")
    }

    function jobLogBriefNote(text) {
        var lines = formatJobLog(text).split("\n")
        for (var i = lines.length - 1; i >= 0; i--) {
            var line = String(lines[i] || "").trim()
            if (!line)
                continue
            line = line.replace(/^evoplayer:\s*/i, "")
            line = line.replace(/\s+at\s+\d{4}-\d{2}-\d{2}T[0-9:+.-]+/g, "")
            line = line.replace(/\d{4}-\d{2}-\d{2}T[0-9:+.-]+/g, "")
            line = line.replace(/\s+/g, " ").trim()
            if (line)
                return line
        }
        return ""
    }

    function jobLogInline() {
        var text = jobLogBriefNote(jobLog)
        if (text)
            return text
        if (libraryJobBusy)
            return libraryJobActiveLabel + "…"
        if (sortProc.running)
            return String(sortProc._label || "sort") + "…"
        return ""
    }

    function syncJobLog() {
        var parts = []
        var err = jobErr.text ? formatJobLog(jobErr.text) : ""
        var out = jobOut.text ? formatJobLog(jobOut.text) : ""
        if (err)
            parts.push(err)
        if (out)
            parts.push(out)
        if (parts.length)
            jobLog = parts.join("\n")
    }

    function runLibraryAction(action) {
        if (!action)
            return
        var stay = downloadsPanelOpen ? { stayOnScreen: true } : {}
        var key = String(action.key || "")
        if (key === "import") {
            runDaemonLibraryJob("library.import", {}, action.label || "import", Object.assign({ key: "import" }, stay))
            return
        }
        runJob(action.args || [], action.label || "library task", Object.assign({ key: key }, stay))
    }

    function runDaemonLibraryJob(method, params, label, options) {
        if (libraryJobBusy) {
            notify("busy — " + libraryJobActiveLabel, 2000)
            return
        }
        jobStopRequested = false
        jobBusy = true
        libraryJobSawRunning = false
        jobLabel = label
        activeLibraryJobKey = (options && options.key) ? String(options.key) : ""
        jobLog = label + "…\n"
        if (method === "library.download")
            downloadFiles = []
        daemonJobPollTimer.restart()
        if (!(options && options.stayOnScreen) && playerScreen !== "filter" && !settingsPanelOpen && !downloadsPanelOpen)
            playerScreen = "nowplaying"
        notify(label + "…", 2000)
        function onStart(text) {
            try {
                var st = JSON.parse(String(text || "{}"))
                if (st && st.error && st.status !== "running") {
                    jobLog = String(jobLog || "") + "\n" + String(st.error)
                    onJobFinished(1)
                    return
                }
                if (st && st.status === "done") {
                    applyJobResult(st)
                    onJobFinished(0)
                    return
                }
                if (st && st.status === "error") {
                    jobLog = String(jobLog || "") + "\n" + String(st.error || "job failed")
                    onJobFinished(1)
                    return
                }
            } catch (e) {
            }
            libraryJobSawRunning = true
            daemonJobPollTimer.restart()
        }
        function fallback() {
            var args = (options && options.args) ? options.args : []
            if (!args.length) {
                if (method === "library.import")
                    args = ["import"]
                else if (method === "library.soundcloud.download")
                    args = ["download"]
                else if (method === "library.download") {
                    args = ["download", "url"]
                    var u = String((params && params.url) || "").trim()
                    if (u)
                        args.push(u)
                }
                else if (method === "library.art.maintain")
                    args = ["art", "maintain"]
                else if (method === "library.cache")
                    args = ["cache", "--force"]
            }
            jobBusy = false
            jobLabel = ""
            activeLibraryJobKey = ""
            runJob(args, label, options || {})
        }
        if (!runDaemonJSON(method, params || {}, onStart, fallback))
            fallback()
    }

    function syncScanJobProgress(st) {
        if (!st || !st.progress)
            return
        var p = st.progress
        scanJobProgress = {
            phase: String(p.phase || ""),
            folder: String(p.folder || ""),
            done: Number(p.done) || 0,
            total: Number(p.total) || 0
        }
    }

    function syncScanJobRunning(st) {
        if (!st) {
            scanJobRunning = false
            scanJobProgress = { phase: "", folder: "", done: 0, total: 0 }
            return
        }
        var name = String(st.name || "")
        var status = String(st.status || "")
        if (!name && (status === "idle" || status === "done")) {
            scanJobRunning = false
            scanJobProgress = { phase: "", folder: "", done: 0, total: 0 }
            return
        }
        if (name !== "scan")
            return
        syncScanJobProgress(st)
        if (status === "running")
            scanJobRunning = true
        else if (status === "done" || status === "idle" || status === "error") {
            scanJobRunning = false
            scanJobProgress = { phase: "", folder: "", done: 0, total: 0 }
        }
    }

    function libraryScanPhaseText() {
        var p = scanJobProgress || {}
        var parts = []
        var phase = String(p.phase || "")
        if (phase === "index")
            parts.push("indexing")
        else if (phase === "warm")
            parts.push("caching art & waveforms")
        var total = Number(p.total) || 0
        var done = Number(p.done) || 0
        if (total > 0)
            parts.push(done + "/" + total)
        if (!parts.length)
            return "scanning…"
        return parts.join("   ·   ")
    }

    function libraryScanFolderText() {
        return compactScanFolder((scanJobProgress || {}).folder)
    }

    function stopScanJob() {
        if (!scanJobRunning)
            return
        runDaemonJSON("job.cancel", {}, function(text) {
            try {
                var st = JSON.parse(String(text || "{}"))
                if (st && st.cancelled)
                    notify("Scan stopped", 2000)
            } catch (e) {
            }
        })
    }

    function applyScanStatusNote(st) {
        if (!st || String(st.name || "") !== "scan")
            return
        if (st.status === "running") {
            statusNote = "Scanning library…"
            statusNoteTimer.stop()
            return
        }
        if (st.status === "done") {
            statusNote = "Library ready"
            statusNoteTimer.interval = 3000
            statusNoteTimer.restart()
            return
        }
        if (st.status === "error") {
            var errText = String(st.error || "").toLowerCase()
            if (errText.indexOf("cancel") >= 0) {
                statusNote = "Scan stopped"
                statusNoteTimer.interval = 3000
                statusNoteTimer.restart()
                return
            }
            statusNote = "Library scan failed"
            statusNoteTimer.interval = 3000
            statusNoteTimer.restart()
        }
    }

    function applyJobProgress(st) {
        if (!st || String(st.status || "") !== "running")
            return
        var name = String(st.name || "")
        if (name !== "download-url" && name !== "download")
            return
        var p = st.progress || {}
        var phase = String(p.phase || name)
        var line = phase
        var done = Number(p.done) || 0
        var total = Number(p.total) || 0
        if (total > 0)
            line += " " + Math.min(100, done) + "%"
        jobLog = line + "…\n"
    }

    function applyJobResult(st) {
        if (!st)
            return
        var name = String(st.name || "")
        var result = st.result || {}
        var folders = result.folders || result.genres
        if (folders && folders.length)
            downloadFolders = folders
        if (name === "import") {
            downloadFiles = result.files || []
            return
        }
        if (name !== "download-url")
            return
        var files = result.files
        if (files && files.length)
            downloadFiles = files
    }

    function loadIncomingDownloads() {
        runDaemonJSON("library.incoming.list", {}, function(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                downloadFiles = data.files || []
                var folders = data.folders || data.genres
                if (folders && folders.length)
                    downloadFolders = folders
            } catch (e) {
                downloadFiles = []
            }
        }, function() {
        })
    }

    function setIncomingFolder(path, folder) {
        path = String(path || "").trim()
        folder = String(folder || "").trim()
        if (!path || !folder)
            return
        runDaemonJSON("library.incoming.set_genre", { path: path, folder: folder, genre: folder }, function(text) {
            try {
                var file = JSON.parse(String(text || "{}"))
                if (!file || !file.path) {
                    notify("could not set folder", 3000)
                    return
                }
                var next = []
                var found = false
                for (var i = 0; i < downloadFiles.length; i++) {
                    if (String(downloadFiles[i].path || "") === String(file.path)) {
                        next.push(file)
                        found = true
                    } else {
                        next.push(downloadFiles[i])
                    }
                }
                if (!found)
                    next.push(file)
                downloadFiles = next
            } catch (e) {
                notify("could not set folder", 3000)
            }
        }, function() {
            notify("could not set folder", 3000)
        })
    }

    function onDaemonJobEvent(st) {
        if (!st)
            return
        syncScanJobRunning(st)
        root.applyScanStatusNote(st)
        root.applyJobResult(st)
        root.applyJobProgress(st)
        if (st.status === "idle" || st.status === "done") {
            daemonJobPollTimer.stop()
            if (jobBusy)
                onJobFinished(0)
            else {
                externalJobBusy = false
                externalJobLabel = ""
                activeLibraryJobKey = ""
            }
            return
        }
        if (st.status === "error") {
            daemonJobPollTimer.stop()
            if (jobBusy) {
                jobLog = String(jobLog || "") + "\n" + String(st.error || "job failed")
                onJobFinished(1)
            } else {
                externalJobBusy = false
                externalJobLabel = ""
                activeLibraryJobKey = ""
                jobLog = String(jobLog || "") + "\n" + String(st.error || "job failed")
            }
            return
        }
        if (st.name) {
            var label = String(st.name)
            if (label === "scan")
                label = "Scanning library"
            jobLabel = label
            if (!jobBusy) {
                externalJobBusy = true
                externalJobLabel = label
                activeLibraryJobKey = String(st.name)
            }
        }
    }

    function pollDaemonJobStatus() {
        if (!jobBusy && !externalJobBusy)
            return
        runDaemonJSON("job.status", {}, function(text) {
            try {
                var st = JSON.parse(String(text || "{}"))
                root.syncScanJobRunning(st)
                root.applyScanStatusNote(st)
                root.applyJobResult(st)
                root.applyJobProgress(st)
                var status = String((st && st.status) || "idle")
                if (status === "running") {
                    libraryJobSawRunning = true
                    return
                }
                if (status === "idle" && jobBusy && !libraryJobSawRunning)
                    return
                daemonJobPollTimer.stop()
                if (status === "error") {
                    if (jobBusy) {
                        jobLog = String(jobLog || "") + "\n" + String(st.error || "job failed")
                        onJobFinished(1)
                    } else {
                        externalJobBusy = false
                        externalJobLabel = ""
                        activeLibraryJobKey = ""
                        jobLog = String(jobLog || "") + "\n" + String(st.error || "job failed")
                    }
                    return
                }
                if (jobBusy)
                    onJobFinished(0)
                else {
                    externalJobBusy = false
                    externalJobLabel = ""
                    activeLibraryJobKey = ""
                }
            } catch (e) {
            }
        }, function() {
            syncExternalJobStatus()
        })
    }

    function runJob(args, label, options) {
        if (libraryJobBusy) {
            notify("busy — " + libraryJobActiveLabel, 2000)
            return
        }
        jobStopRequested = false
        jobBusy = true
        jobLabel = label
        activeLibraryJobKey = (options && options.key) ? String(options.key) : ""
        jobLog = label + "…\n"
        if (!(options && options.stayOnScreen) && playerScreen !== "filter" && !settingsPanelOpen && !downloadsPanelOpen)
            playerScreen = "nowplaying"
        jobProc.command = playerCmd(args || [])
        notify(label + "…", 2000)
        jobProc.running = true
    }

    function syncExternalJobStatus() {
        if (jobProc.running)
            return
        runQuery(["job", "status", "--json"], function(text) {
            try {
                var st = JSON.parse(String(text || "{}"))
                var running = st && String(st.status || "") === "running"
                if (running && !root.jobBusy) {
                    root.syncScanJobRunning(st)
                    root.externalJobBusy = true
                    root.externalJobLabel = String(st.name || "library task")
                    root.activeLibraryJobKey = String(st.name || "")
                    if (!root.jobLog || root.jobLog.indexOf("running") < 0)
                        root.jobLog = root.externalJobLabel + " running…\n"
                    if (!daemonJobPollTimer.running)
                        daemonJobPollTimer.start()
                } else if (!root.jobBusy) {
                    root.externalJobBusy = false
                    root.externalJobLabel = ""
                    root.activeLibraryJobKey = ""
                }
            } catch (e) {
                if (!root.jobBusy) {
                    root.externalJobBusy = false
                    root.externalJobLabel = ""
                    root.activeLibraryJobKey = ""
                }
            }
        })
    }

    function stopLibraryJob() {
        var key = String(activeLibraryJobKey || "").trim()
        if (key === "scan") {
            stopScanJob()
            return
        }
        if (!libraryJobBusy && !sortProc.running)
            return
        var label = libraryJobActiveLabel || (sortProc.running ? "sort" : "library task")
        jobStopRequested = true
        if (sortProc.running) {
            sortStopRequested = true
            sortProc.running = false
        }
        if (jobBusy && jobProc.running)
            jobProc.running = false
        runDaemonJSON("job.cancel", {}, function(text) {
            daemonJobPollTimer.stop()
            var stopped = false
            try {
                stopped = !!JSON.parse(String(text || "{}")).cancelled
            } catch (e) {}
            if (!stopped)
                runQuery(["job", "stop", "--json"], function() {})
            jobBusy = false
            jobLabel = ""
            activeLibraryJobKey = ""
            externalJobBusy = false
            externalJobLabel = ""
            jobStopRequested = false
            syncJobLog()
            jobLog = String(jobLog || "") + "\n" + label + " stopped"
            notify((stopped ? label : "job") + " stopped", 3000)
        }, function() {
            runQuery(["job", "stop", "--json"], function(text) {
                var stopped = false
                try {
                    stopped = !!JSON.parse(String(text || "{}")).stopped
                } catch (e) {}
                jobBusy = false
                jobLabel = ""
                activeLibraryJobKey = ""
                externalJobBusy = false
                externalJobLabel = ""
                jobStopRequested = false
                syncJobLog()
                jobLog = String(jobLog || "") + "\n" + label + " stopped"
                notify((stopped ? label : "job") + " stopped", 3000)
            })
        })
    }

    function onJobFinished(exitCode) {
        syncJobLog()
        var label = jobLabel
        jobBusy = false
        libraryJobSawRunning = false
        jobLabel = ""
        activeLibraryJobKey = ""
        if (jobStopRequested) {
            jobStopRequested = false
            if (label)
                jobLog = String(jobLog || "") + "\n" + label + " stopped"
            syncExternalJobStatus()
            return
        }
        if (exitCode === 0) {
            notify(label + " complete", 4000)
            loadIncomingDownloads()
            loadLibraryStats()
            loadPlaylists(true)
            if (filetreePanelOpen || filetreeRows.length > 0) {
                _filetreeRefreshPending = true
                if (!filetreeTreeLoading)
                    reloadFiletreeView()
            }
            if (playlistsPanelOpen && selectedPlaylist)
                loadPlaylistTracks(selectedPlaylist, true)
            jobLog = jobLog + "\n\n" + label + " complete"
        } else if (exitCode === 2) {
            var busy = jobErr.text ? String(jobErr.text).trim().split("\n").pop() : ""
            notify(busy || (label + " — already running"), 4000)
            syncExternalJobStatus()
        } else {
            var err = jobErr.text ? String(jobErr.text).trim() : ""
            if (err)
                err = err.split("\n").pop()
            notify(label + " failed" + (err ? " — " + err : ""), 5000)
        }
        if (exitCode !== 2)
            syncExternalJobStatus()
    }

    function loadLibraryStats() {
        runQuery(["stats", "--json"], function(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                libraryStats = data
                if (data.root)
                    musicRoot = String(data.root)
            } catch (e) {
                libraryStats = { tracks: 0, genres: 0 }
            }
        })
    }

    function parsePlayerSettings(raw) {
        var prevRoot = String(settingsMusicLibrary || "")
        var trimmed = String(raw || "").trim()
        if (!trimmed) {
            return
        }
        try {
            var data = JSON.parse(trimmed)
            var sc = data.soundcloud || {}
            var paths = data.paths || {}
            var viz = data.viz || {}
            settingsScUser = String(sc.user || "")
            settingsScOAuthSource = String(sc.oauth_source || "")
            settingsMusicLibrary = String(paths.root || "")
            settingsVizFrameRate = parseInt(viz.frame_rate, 10) || 45
            settingsVizSensitivity = parseInt(viz.sensitivity, 10) || 145
            settingsVizAutosens = parseInt(viz.autosens, 10) || 2
            settingsVizNoiseReduction = parseInt(viz.noise_reduction, 10) || 34
            settingsVizMonstercat = parseFloat(viz.monstercat) || 1
            settingsVizLowCutoff = parseInt(viz.low_cutoff, 10) || 50
            settingsVizHighCutoff = parseInt(viz.high_cutoff, 10) || 10000
            if (settingsMusicLibrary)
                musicRoot = settingsMusicLibrary
            settingsReady = true
        } catch (e) {
            settingsReady = false
        }
        if (prevRoot && settingsMusicLibrary && settingsMusicLibrary !== prevRoot) {
            loadLibraryStats()
            if (filetreePanelOpen || filetreeRows.length > 0)
                reloadFiletreeView()
        }
    }

    function loadPlayerSettings() {
        if (settingsLoadProc.running)
            return
        settingsLoadProc.running = true
    }

    function setPlayerSetting(key, value) {
        if (!settingsReady || settingsSetProc.running || settingsPickProc.running)
            return
        settingsSetProc.key = String(key || "")
        settingsSetProc.value = String(value || "")
        settingsSetProc.running = true
    }

    function setMusicLibrary(path) {
        setPlayerSetting("paths.root", path)
    }

    function setVizSetting(field, value) {
        setPlayerSetting("viz." + String(field || ""), String(value))
    }

    function pickMusicLibrary() {
        if (settingsPickProc.running || settingsSetProc.running)
            return
        settingsPickProc.running = true
    }

    function runMusic(args, onDone, proc) {
        return runMusicSoon(args, onDone, proc, 0)
    }

    function runMusicSoon(args, onDone, proc, attempt) {
        var runner = proc || cmdProc
        if (runner.running) {
            if ((attempt || 0) >= 24)
                return false
            Qt.callLater(function() {
                root.runMusicSoon(args, onDone, proc, (attempt || 0) + 1)
            })
            return true
        }
        runner.command = playerCmd(args || [])
        runner._onDone = onDone || null
        runner.running = true
        return true
    }

    function queueIpcParams(params) {
        var p = Object.assign({}, params || {})
        if (queueRevision > 0)
            p.if_revision = queueRevision
        return p
    }

    function applyQueueRevisionFromMsg(msg) {
        if (!msg)
            return
        if (msg.data && msg.data.queue_revision !== undefined)
            queueRevision = Number(msg.data.queue_revision) || 0
        if (!msg.ok && msg.code === "conflict") {
            if (msg.data && msg.data.queue_revision !== undefined)
                queueRevision = Number(msg.data.queue_revision) || 0
            mergeMonitorStatus()
        }
    }

    function runDaemonJSON(method, params, onDone, onFallback) {
        if (!playerMonitor || typeof playerMonitor.ipcCall !== "function") {
            if (onFallback)
                onFallback()
            else if (onDone)
                onDone("")
            return false
        }
        if (!playerMonitor.ipcReady) {
            if (onFallback)
                onFallback()
            else if (onDone)
                onDone("")
            return false
        }
        var payload = String(method).indexOf("queue.") === 0 ? queueIpcParams(params) : params
        playerMonitor.ipcCall(method, payload, function(ok, msg) {
            applyQueueRevisionFromMsg(msg)
            if (ok && msg && msg.data !== undefined) {
                if (onDone)
                    onDone(JSON.stringify(msg.data))
                return
            }
            if (onFallback)
                onFallback()
            else if (onDone)
                onDone("")
        })
        return true
    }

    function runDaemonVoid(method, params, onDone, onFallback) {
        if (!playerMonitor || typeof playerMonitor.ipcCall !== "function" || !playerMonitor.ipcReady) {
            if (onFallback)
                onFallback()
            else if (onDone)
                onDone(false)
            return false
        }
        var payload = String(method).indexOf("queue.") === 0 ? queueIpcParams(params) : params
        playerMonitor.ipcCall(method, payload, function(ok, msg) {
            applyQueueRevisionFromMsg(msg)
            if (onDone)
                onDone(!!ok)
        })
        return true
    }

    function daemonJSON(method, params, onDone, cliFallback) {
        if (!runDaemonJSON(method, params, onDone, cliFallback) && cliFallback)
            cliFallback()
    }

    function daemonVoid(method, params, onDone, cliFallback) {
        if (!runDaemonVoid(method, params, onDone, cliFallback) && cliFallback)
            cliFallback()
    }

    function parsePlaylistArgs(args) {
        var out = {
            name: "",
            offset: -1,
            limit: 0,
            list: false,
            star: false,
            starName: ""
        }
        var i = 0
        if (args.length && args[0] === "playlist")
            i = 1
        for (; i < args.length; i++) {
            var a = String(args[i])
            if (a === "--json")
                continue
            if (a === "--offset" && i + 1 < args.length) {
                out.offset = Number(args[++i]) || 0
                continue
            }
            if (a === "--limit" && i + 1 < args.length) {
                out.limit = Number(args[++i]) || 0
                continue
            }
            if (a === "star" && i + 1 < args.length && args[i + 1] === "toggle") {
                out.star = true
                i += 2
                if (i < args.length && args[i] !== "--json")
                    out.starName = String(args[i])
                continue
            }
            if (!out.name && a !== "star")
                out.name = a
        }
        out.list = !out.name && !out.star
        return out
    }

    function tryPlaylistIPC(args, onDone, onFallback) {
        var parsed = parsePlaylistArgs(args || [])
        if (parsed.list)
            return runDaemonJSON("library.playlist.list", {}, onDone, onFallback)
        if (parsed.star && parsed.starName)
            return runDaemonJSON("library.playlist.star", { name: parsed.starName }, onDone, onFallback)
        if (parsed.name && parsed.offset >= 0) {
            var limit = parsed.limit > 0 ? parsed.limit : playlistPageSize
            return runDaemonJSON("library.playlist.tracks", {
                name: parsed.name,
                offset: parsed.offset,
                limit: limit
            }, onDone, onFallback)
        }
        return false
    }

    function monitorTransport(method, params) {
        if (!method || !playerMonitor || typeof playerMonitor.ipcCallVoid !== "function")
            return false
        if (panelTransitionGuard) {
            var m = String(method)
            if (m === "playback.toggle" || m === "playback.stop"
                    || m === "playback.next" || m === "playback.prev"
                    || m === "playback.seek"
                    || m.indexOf("playback.volume.") === 0
                    || m.indexOf("queue.") === 0) {
                traceIPC("blocked " + m + " (panelTransitionGuard)")
                // #region agent log
                agentDebug(m.indexOf("queue.") === 0 ? "H2" : "H1",
                    "DashboardModule.qml:monitorTransport", "blocked by guard", { method: m })
                // #endregion
                return false
            }
        }
        traceIPC("ipc " + method + (params ? " " + JSON.stringify(params) : ""))
        // #region agent log
        var hyp = "H1"
        if (String(method).indexOf("queue.") === 0)
            hyp = "H2"
        else if (method === "viz.subscribe" || method === "viz.unsubscribe")
            hyp = "H4"
        agentDebug(hyp, "DashboardModule.qml:monitorTransport", "ipc send", {
            method: method,
            params: params || null
        })
        // #endregion
        var payload = String(method).indexOf("queue.") === 0 ? queueIpcParams(params) : params
        playerMonitor.ipcCallVoid(method, payload)
        return true
    }

    function transportMethodForArgs(args) {
        if (!args || !args.length)
            return ""
        switch (String(args[0])) {
        case "toggle": return "playback.toggle"
        case "stop": return "playback.stop"
        case "next": return "playback.next"
        case "prev": return "playback.prev"
        case "seek": return "playback.seek"
        case "volume":
            if (args.length >= 3 && String(args[1]) === "set")
                return "playback.volume.set"
            return "playback.volume.delta"
        default: return ""
        }
    }

    function transportParamsForArgs(args) {
        if (!args || !args.length)
            return undefined
        switch (String(args[0])) {
        case "seek":
            return { seconds: Number(args[1]) || 0 }
        case "volume":
            if (args.length >= 3 && String(args[1]) === "set")
                return { volume: Number(args[2]) || 0 }
            return { delta: Number(args[1]) || 0 }
        default:
            return undefined
        }
    }

    function runPlayer(args, onDone, proc) {
        if (panelTransitionGuard) {
            var method = transportMethodForArgs(args)
            if (method)
                return false
        }
        if (monitorTransport(transportMethodForArgs(args), transportParamsForArgs(args))) {
            if (onDone)
                onDone("")
            return true
        }
        var runner = proc || cmdProc
        if (runner.running) return false
        runner.command = playerCmd(args || [])
        runner._onDone = onDone || null
        runner.running = true
        return true
    }

    function runQuery(args, onDone) {
        if (queryProc.running) return false
        queryProc.command = playerCmd(args || [])
        queryProc._onDone = onDone || null
        queryProc.running = true
        return true
    }

    function runPlayerQuery(args, onDone) {
        if (playerQueryProc.running) return false
        playerQueryProc.command = playerCmd(args || [])
        playerQueryProc._onDone = onDone || null
        playerQueryProc.running = true
        return true
    }

    property var _playlistQueryQueue: []
    property var _filetreeQueryQueue: []
    property bool _startupBootstrapOpenDone: false
    property bool _startupBootstrapCurrentDone: false

    function pumpPlaylistQuery() {
        if (playlistQueryProc.running || _playlistQueryQueue.length === 0)
            return
        var job = _playlistQueryQueue[0]
        playlistQueryProc.command = playerCmd(job.args || [])
        playlistQueryProc._onDone = function(text) {
            _playlistQueryQueue.shift()
            if (job.onDone)
                job.onDone(text)
            Qt.callLater(pumpPlaylistQuery)
        }
        playlistQueryProc.running = true
    }

    function runPlaylistQuery(args, onDone) {
        function runCLI() {
            _playlistQueryQueue.push({
                args: (args || []).slice(),
                onDone: onDone || null
            })
            pumpPlaylistQuery()
        }
        if (libraryActivityBusy) {
            runCLI()
            return
        }
        if (tryPlaylistIPC(args, onDone, runCLI))
            return
        runCLI()
    }

    function finishStartupBootstrap() {
        if (!_startupBootstrapOpenDone || !_startupBootstrapCurrentDone)
            return
        ensureNowPlayingFromCurrentPlaylist()
        var bootPath = String(player.path || "")
        if (bootPath && (resolvedArtPath !== bootPath || !resolvedArt))
            applyDisplayArtForPath(bootPath)
        syncSideArtImageSource()
        kickSideArtUntilLoaded()
        refreshCurrentPlaylistView()
        Qt.callLater(function() {
            root.ensureLibraryData()
        })
    }

    function ensureLibraryData() {
        if (!libraryPlaylists.length && !playlistsLoading)
            loadPlaylists()
        if (filetreePanelOpen && !filetreeRows.length && !filetreeTreeLoading)
            loadFiletreeRoot()
    }

    function applyPlaylists(list) {
        var all = []
        var starred = []
        for (var j = 0; j < list.length; j++) {
            var item = list[j]
            all.push(item)
            if (item.starred === true)
                starred.push(item)
        }
        all.sort(function(a, b) {
            return String(a.name || "").localeCompare(String(b.name || ""))
        })
        libraryPlaylists = all
        playlists = starred
        rebuildPlaylistTabs()
    }

    function rebuildPlaylistTabs() {
        playlistTabModel.clear()
        playlistTabModel.append({
            name: currentPlaylistId,
            count: currentPlaylistActive ? currentPlaylistTracks.length : 0
        })
        for (var k = 0; k < playlists.length; k++) {
            var tabName = String(playlists[k].name || "")
            if (!tabName || tabName === currentPlaylistId)
                continue
            playlistTabModel.append({
                name: tabName,
                count: playlists[k].count || 0
            })
        }
        refreshCurrentPlaylistView()
    }

    function playlistCanStar(name) {
        var n = String(name || "")
        return n !== "" && n !== currentPlaylistId
    }

    function playlistIsUserEditable(name) {
        var n = String(name || "")
        if (!n || n === currentPlaylistId || n === "all" || n === "favorites" || n === "mixes")
            return false
        for (var i = 0; i < libraryPlaylists.length; i++) {
            if (libraryPlaylists[i].name === n)
                return String(libraryPlaylists[i].kind || "") === "other"
        }
        return false
    }

    function refreshCurrentPlaylistView() {
        if (!currentPlaylistActive)
            return
        if (playlistsPanelOpen && selectedPlaylist === currentPlaylistId)
            loadPlaylistTracks(currentPlaylistId)
    }

    function commitCurrentPlaylist(opts) {
        opts = opts || {}
        rebuildPlaylistTabs()
        syncPlaylistTabPosition()
        if (selectedPlaylist === currentPlaylistId && playlistsPanelOpen) {
            if (opts.appendCount > 0)
                appendCurrentPlaylistTracksView(opts.appendCount)
            else
                loadPlaylistTracks(currentPlaylistId)
        }
        if (!opts.skipPersist)
            persistCurrentPlaylist()
        if (!opts.skipWarm)
            scheduleCurrentTracksEnrich()
    }

    function scheduleCurrentTracksEnrich() {
        if (!currentPlaylistActive || !currentPlaylistTracks.length)
            return
        currentEnrichTimer.restart()
    }

    function enrichCurrentTracksBatch() {
        if (!currentPlaylistActive || !currentPlaylistTracks.length)
            return
        var paths = []
        var i
        for (i = 0; i < currentPlaylistTracks.length; i++) {
            var t = currentPlaylistTracks[i]
            if (!t || !t.path)
                continue
            if (!t.artist && !t.album && !t.art)
                paths.push(String(t.path))
            if (paths.length >= 64)
                break
        }
        if (!paths.length)
            return
        if (playerMonitor && typeof playerMonitor.ipcCallVoid === "function") {
            playerMonitor.ipcCallVoid("library.warm.batch", {
                paths: paths,
                workers: 8,
                art: true
            })
        }
    }

    function appendCurrentPlaylistTracksView(added) {
        added = Number(added) || 0
        playlistTrackTotal = currentPlaylistTracks.length
        if (added <= 0)
            return
        var prevTotal = playlistTrackTotal - added
        if (tracks.length >= prevTotal && tracks.length < playlistTrackTotal) {
            var sliceEnd = Math.min(tracks.length + added, playlistTrackTotal)
            var newItems = currentPlaylistTracks.slice(tracks.length, sliceEnd)
            if (newItems.length) {
                tracks = tracks.concat(newItems)
                playlistTrackOffset = tracks.length
                tracksRevision++
                warmPlaylistPage(newItems)
                scheduleVisibleArtWarm()
            }
            return
        }
        if (tracks.length > 0 && tracks.length < prevTotal)
            return
        loadPlaylistTracks(currentPlaylistId)
    }

    function persistCurrentPlaylist() {
        persistCurrentTimer.restart()
    }

    function flushPersistCurrentPlaylist() {
        if (!currentPlaylistActive || !currentPlaylistTracks.length) {
            if (_lastSavedCurrentPathsKey !== "") {
                _lastSavedCurrentPathsKey = ""
                runMusic(["current", "clear"], null, saveCurrentProc)
            }
            return
        }
        var paths = pathsFromTracks(currentPlaylistTracks)
        var key = paths.join("\n")
        if (key === _lastSavedCurrentPathsKey)
            return
        _lastSavedCurrentPathsKey = key
        function runCLI() {
            var args = ["current", "save"]
            for (var i = 0; i < paths.length; i++)
                args.push(paths[i])
            runMusic(args, null, saveCurrentProc)
        }
        daemonVoid("library.current.save", { paths: paths }, function(ok) {
            if (ok && root.currentPlaylistNeedsMeta())
                root.loadCurrentPlaylist()
        }, runCLI)
    }

    function folderArtFromSiblings(trackPath) {
        return FiletreeLogic.folderArtFromSiblings(filetreeChildren, trackPath)
    }

    function artForTrackPath(path) {
        var t = trackForPath(path)
        if (t && t.art)
            return String(t.art)
        return ""
    }

    function trackArtForRow(track) {
        var _rev = tracksRevision + filetreeArtRevision
        var _children = filetreeChildren
        var thumb = String((track && track.thumb) || "")
        if (thumb.charAt(0) === "/")
            return thumb
        var path = String((track && track.path) || "")
        var art = String((track && track.art) || "")
        if (art.charAt(0) === "/")
            return art
        var resolved = artForTrackPath(path)
        if (resolved)
            return resolved
        return folderArtFromSiblings(path)
    }

    function rowArtUrl(path, trackPath) {
        var base = artUrl(path, false)
        if (!base)
            return ""
        var key = String(trackPath || path || "")
        if (!key)
            return base
        var sep = base.indexOf("?") >= 0 ? "&" : "?"
        return base + sep + "row=" + encodeURIComponent(key)
    }

    function queueRowArtWarm(path) {
        path = String(path || "")
        if (!path)
            return
        if (filetreePanelOpen && filetreeMoving)
            return
        if (rowArtWarmQueue.indexOf(path) >= 0)
            return
        if (rowArtWarmQueue.length >= 24)
            rowArtWarmQueue.shift()
        rowArtWarmQueue.push(path)
        pumpRowArtWarm()
    }

    function pauseFiletreeArtWarm() {
        filetreeMoving = true
        rowArtWarmQueue = []
    }

    function resumeFiletreeArtWarm() {
        filetreeMoving = false
        if (filetreePanelOpen)
            scheduleVisibleArtWarm()
    }

    function warmTracksArt(entries) {
        if (!entries || !entries.length)
            return
        for (var i = 0; i < entries.length; i++) {
            var entry = entries[i]
            if (!entry)
                continue
            var path = String(entry.path || "")
            if (!path)
                continue
            var art = String(entry.art || "")
            var thumb = String(entry.thumb || "")
            var wf = String(entry.waveform || "")
            if (art.charAt(0) === "/" && thumb.charAt(0) === "/" && wf.charAt(0) === "/")
                continue
            queueRowArtWarm(path)
        }
    }

    function warmVisibleTracks(listView, model, filetreeMode) {
        if (!listView || !model || !model.length)
            return
        if (filetreeMode && filetreeMoving)
            return
        var buffer = filetreeMode ? 4 : 10
        var top = Math.max(0, listView.indexAt(0, listView.contentY + 1) - buffer)
        var bottom = listView.indexAt(0, listView.contentY + listView.height - 1) + buffer
        if (bottom < 0)
            bottom = model.length - 1
        bottom = Math.min(model.length - 1, bottom)
        var slice = []
        var i, entry, track
        var warmLimit = filetreeMode ? 8 : 32
        for (i = top; i <= bottom && slice.length < warmLimit; i++) {
            entry = model[i]
            if (!entry)
                continue
            if (filetreeMode) {
                if (entry.type === "track")
                    slice.push(entry.track || entry)
            } else {
                slice.push(entry)
            }
        }
        warmTracksArt(slice)
    }

    function noteFiletreeScrollActivity() {
        if (!filetreePanelOpen)
            return
        filetreeUserScrolling = true
        filetreeScrollIdleTimer.restart()
    }

    function scheduleVisibleArtWarm() {
        if (filetreePanelOpen && filetreeDeferVisibleArtWarm) {
            filetreeDeferVisibleArtWarm = false
            return
        }
        if (filetreePanelOpen && (filetreeMoving || filetreeUserScrolling))
            return
        if (playlistsPanelOpen && playlistUserScrolling)
            return
        visibleArtWarmTimer.restart()
    }

    function notePlaylistScrollActivity() {
        if (!playlistsPanelOpen || playlistPanelMode !== "tracks")
            return
        playlistUserScrolling = true
        playlistScrollIdleTimer.restart()
    }

    function pumpRowArtWarm() {
        if (rowArtWarmInflight >= rowArtWarmMaxInflight || !rowArtWarmQueue.length)
            return
        var path = rowArtWarmQueue.shift()
        rowArtWarmInflight++
        function finish() {
            rowArtWarmInflight = Math.max(0, rowArtWarmInflight - 1)
            if (rowArtWarmQueue.length)
                Qt.callLater(pumpRowArtWarm)
        }
        function applyWarm(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                var art = data && data.art ? String(data.art) : ""
                var thumb = data && data.thumb ? String(data.thumb) : ""
                if (root.filetreePanelOpen && !root.playlistsPanelOpen) {
                    if (art.charAt(0) === "/" || thumb.charAt(0) === "/")
                        root.queueFiletreeArtPatch(path, art, true, thumb)
                } else {
                    if (art.charAt(0) === "/")
                        patchTrackArtInLists(path, art, thumb)
                    else if (thumb.charAt(0) === "/")
                        patchTrackArtInLists(path, "", thumb)
                }
                if (data && data.waveform) {
                    var wf = String(data.waveform)
                    if (wf.charAt(0) === "/") {
                        if (root.filetreePanelOpen && !root.playlistsPanelOpen)
                            root.patchFiletreeWaveform(path, wf)
                        else
                            patchWaveformInLists(path, wf)
                    }
                    root.onWarmWaveformResult(path, wf)
                } else {
                    root.onWarmWaveformResult(path, "")
                }
            } catch (e) {
            }
            finish()
        }
        function runWarmCLI() {
            rowArtWarmProc.command = playerCmd(["warm", path, "--json"])
            rowArtWarmProc._onDone = applyWarm
            rowArtWarmProc.running = true
        }
        if (!runDaemonJSON("library.warm", { path: path }, applyWarm, runWarmCLI))
            runWarmCLI()
    }

    function queueWaveformWarm(path) {
        path = String(path || "")
        if (!path)
            return
        function applyWaveform(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                var wf = data && data.waveform ? String(data.waveform) : ""
                if (wf.charAt(0) === "/") {
                    patchWaveformInLists(path, wf)
                    rememberWaveformPath(path, wf)
                }
                root.onWarmWaveformResult(path, wf)
            } catch (e) {
                root.onWarmWaveformResult(path, "")
            }
        }
        function runCLI() {
            runMusic(["warm", path, "--json"], function(text) {
                applyWaveform(text)
            }, cmdProc)
        }
        if (!runDaemonJSON("library.warm.waveform", { path: path }, applyWaveform, runCLI))
            runCLI()
    }

    function invalidateResolvedArt(path) {
        var p = String(path || "")
        if (!p)
            return
        if (resolvedArtPath === p) {
            resolvedArt = ""
            resolvedArtPath = ""
            bumpArtRevision()
        }
    }

    function applyPlayerArt(path, art) {
        var p = String(path || "")
        if (!p)
            return
        var nextArt = String(art || "")
        if (!nextArt || nextArt.charAt(0) !== "/") {
            invalidateResolvedArt(p)
            if (String(player.path || "") === p)
                player = Object.assign({}, player, { art: "" })
            return
        }
        var prevArt = resolvedArtPath === p ? String(resolvedArt || "") : ""
        resolvedArtPath = p
        resolvedArt = nextArt
        if (nextArt && nextArt.charAt(0) === "/")
            patchTrackArtInLists(p, nextArt)
        if (String(player.path || "") === p) {
            if (String(player.art || "") !== nextArt)
                player = Object.assign({}, player, { art: nextArt })
            if (nextArt !== prevArt && prevArt !== "")
                bumpArtRevision()
            else
                syncSideArtImageSource()
        }
    }

    function pumpDisplayArtQueue() {
        if (displayArtCacheProc.running)
            return
        var job = displayArtCacheProc._pendingJob
        if (!job)
            return
        displayArtCacheProc._pendingJob = null
        var p = String(job.path || "")
        if (!p) {
            if (job.onDone)
                job.onDone()
            if (displayArtCacheProc._pendingJob)
                pumpDisplayArtQueue()
            return
        }
        displayArtCacheProc._requestedPath = p
        displayArtCacheProc.command = playerCmd(["art", "notify-cache", p])
        displayArtCacheProc._onDone = function(text) {
            var requested = String(displayArtCacheProc._requestedPath || "")
            if (requested === p) {
                var dest = String(text || "").trim()
                if (dest && dest.charAt(0) === "/")
                    applyPlayerArt(p, dest)
            }
            if (job.onDone)
                job.onDone()
            if (displayArtCacheProc._pendingJob)
                pumpDisplayArtQueue()
        }
        displayArtCacheProc.running = true
    }

    function applyDisplayArtForPath(path, onDone) {
        var p = String(path || "")
        if (!p) {
            if (onDone)
                onDone()
            return
        }
        if (resolvedArtPath === p && resolvedArt) {
            if (onDone)
                onDone()
            return
        }
        displayArtCacheProc._pendingJob = { path: p, onDone: onDone || null }
        pumpDisplayArtQueue()
    }

    function pumpWarmArtQueue() {
        if (warmArtProc.running)
            return
        var job = warmArtProc._pendingJob
        if (!job)
            return
        warmArtProc._pendingJob = null
        var p = String(job.path || "")
        if (!p) {
            if (job.onDone)
                job.onDone()
            if (warmArtProc._pendingJob)
                pumpWarmArtQueue()
            return
        }
        function finishWarm(text) {
            warmArtProc._onDone = null
            try {
                var data = JSON.parse(String(text || "{}"))
                if (data && data.waveform) {
                    var wf = String(data.waveform)
                    if (wf.charAt(0) === "/")
                        patchWaveformInLists(p, wf)
                    root.onWarmWaveformResult(p, wf)
                } else {
                    root.onWarmWaveformResult(p, "")
                }
            } catch (e) {
            }
            applyDisplayArtForPath(p, job.onDone)
            if (warmArtProc._pendingJob)
                pumpWarmArtQueue()
        }
        function runWarmCLI() {
            warmArtProc.command = playerCmd(["warm", p, "--json"])
            warmArtProc._onDone = function() {
                finishWarm("")
            }
            warmArtProc.running = true
        }
        if (runDaemonJSON("library.warm", { path: p }, finishWarm, runWarmCLI))
            return
        runWarmCLI()
    }

    function warmArtForPath(path, onDone) {
        var p = String(path || "")
        if (!p) {
            if (onDone)
                onDone()
            return
        }
        warmArtProc._pendingJob = { path: p, onDone: onDone || null }
        pumpWarmArtQueue()
    }

    function ensureNowPlayingFromCurrentPlaylist() {
        if (!currentPlaylistTracks.length)
            return
        var track = null
        for (var i = 0; i < currentPlaylistTracks.length; i++) {
            if (currentPlaylistTracks[i] && currentPlaylistTracks[i].path) {
                track = currentPlaylistTracks[i]
                break
            }
        }
        if (!track)
            return
        var p = String(track.path)
        var existingPath = String(player.path || "")
        if (existingPath && existingPath !== p)
            return
        if (existingPath && resolvedArt && resolvedArtPath === p) {
            prioritizeCurrentAssets()
            return
        }
        if (existingPath && existingPath === p) {
            applyDisplayArtForPath(p)
            prioritizeCurrentAssets()
            return
        }
        var t = trackForPath(p) || track
        var wf = (t && t.waveform) || waveformPathByTrack[p] || ""
        var next = Object.assign({}, player, playerFieldsFromTrack(t), {
            path: p,
            state: existingPath ? (player.state || "stopped") : "stopped",
            position: existingPath ? (Number(player.position) || 0) : 0,
            position_label: existingPath
                ? (player.position_label || formatPlaybackTime(Number(player.position) || 0))
                : formatPlaybackTime(0),
            art: (t && t.art) || "",
            waveform: wf
        })
        player = next
        if (wf)
            rememberWaveformPath(p, wf)
        applyCachedWaveform(p)
        prefetchNeighbors(p)
        applyDisplayArtForPath(p)
        prioritizeCurrentAssets()
    }

    function pumpCurrentPlaylistLoadQueue() {
        if (currentPlaylistLoadProc.running)
            return
        var job = currentPlaylistLoadProc._pendingJob
        if (!job)
            return
        currentPlaylistLoadProc._pendingJob = null
        function finishCurrentLoad(text) {
            try {
                var list = JSON.parse(String(text || "[]"))
                if (Array.isArray(list) && list.length > 0) {
                    currentPlaylistTracks = root.mergeCurrentTrackMeta(currentPlaylistTracks, list)
                    currentPlaylistActive = true
                    _lastSavedCurrentPathsKey = pathsFromTracks(currentPlaylistTracks).join("\n")
                    rebuildPlaylistTabs()
                    if (selectedPlaylist === currentPlaylistId && playlistsPanelOpen)
                        root.applyCurrentPlaylistTracksView()
                }
            } catch (e) {
            }
            if (job.onDone)
                job.onDone()
            if (currentPlaylistLoadProc._pendingJob)
                pumpCurrentPlaylistLoadQueue()
        }
        function runCurrentCLI() {
            currentPlaylistLoadProc.command = playerCmd(["current", "load", "--json"])
            currentPlaylistLoadProc._onDone = function(text) {
                finishCurrentLoad(text)
            }
            currentPlaylistLoadProc.running = true
        }
        if (runDaemonJSON("library.current.load", {}, finishCurrentLoad, runCurrentCLI))
            return
        runCurrentCLI()
    }

    function trackTitleLooksLikeFilename(title, path) {
        title = String(title || "").trim()
        if (!title)
            return true
        if (/\.(mp3|flac|ogg|m4a|wav|opus|aac|wma)$/i.test(title))
            return true
        var base = String(path || "").split("/").pop() || ""
        base = base.replace(/\.(mp3|flac|ogg|m4a|wav|opus|aac|wma)$/i, "")
        return title === base
    }

    function trackNeedsLibraryMeta(track) {
        if (!track || !track.path)
            return false
        if (!String(track.artist || "").trim() && !String(track.album || "").trim())
            return true
        return root.trackTitleLooksLikeFilename(track.title, track.path)
    }

    function currentPlaylistNeedsMeta() {
        var n = Math.min(currentPlaylistTracks.length, playlistPageSize)
        for (var i = 0; i < n; i++) {
            if (root.trackNeedsLibraryMeta(currentPlaylistTracks[i]))
                return true
        }
        return false
    }

    function mergeCurrentTrackMeta(existing, loaded) {
        loaded = loaded || []
        existing = existing || []
        if (!loaded.length)
            return existing
        if (!existing.length)
            return loaded.slice()
        var byPath = {}
        var i, path, meta
        for (i = 0; i < loaded.length; i++) {
            path = loaded[i] && String(loaded[i].path || "")
            if (path)
                byPath[path] = loaded[i]
        }
        var out = []
        for (i = 0; i < existing.length; i++) {
            path = existing[i] && String(existing[i].path || "")
            meta = path ? byPath[path] : null
            if (meta && root.trackNeedsLibraryMeta(existing[i]))
                out.push(Object.assign({}, existing[i], meta))
            else
                out.push(existing[i])
        }
        return out
    }

    function applyCurrentPlaylistTracksView() {
        if (selectedPlaylist !== currentPlaylistId)
            return
        tracksLoading = false
        playlistTrackTotal = currentPlaylistTracks.length
        var cachedY = playlistViewByKey[currentPlaylistId]
            && playlistViewByKey[currentPlaylistId].contentY >= 0
            ? Number(playlistViewByKey[currentPlaylistId].contentY)
            : -1
        var pageEnd = Math.min(playlistPageSize, currentPlaylistTracks.length)
        assignTracks(currentPlaylistTracks.slice(0, pageEnd), { scrollY: playlistFocusNowPlaying ? -1 : cachedY })
        playlistTrackOffset = tracks.length
        syncNowPlayingInPlaylist({ force: playlistFocusNowPlaying, ensureLoaded: playlistFocusNowPlaying })
        playlistFocusNowPlaying = false
        mergePlayerFromTrackList()
        Qt.callLater(function() { snapshotPlaylistTracksCache(currentPlaylistId) })
    }

    function loadCurrentPlaylist(onDone) {
        currentPlaylistLoadProc._pendingJob = { onDone: onDone || null }
        pumpCurrentPlaylistLoadQueue()
    }

    function prioritizeCurrentAssets(onDone) {
        if (!currentPlaylistActive || !currentPlaylistTracks.length) {
            if (onDone)
                onDone()
            return
        }
        var paths = []
        var limit = Math.min(currentPlaylistTracks.length, 32)
        for (var i = 0; i < limit; i++) {
            if (currentPlaylistTracks[i].path)
                paths.push(currentPlaylistTracks[i].path)
        }
        if (!paths.length) {
            if (onDone)
                onDone()
            return
        }
        var finish = function() {
            var currentPath = String(root.player.path || "")
            if (currentPath)
                root.ensureWaveformForPath(currentPath)
            if (onDone)
                onDone()
        }
        if (playerMonitor && typeof playerMonitor.ipcCallVoid === "function") {
            playerMonitor.ipcCallVoid("library.warm.batch", {
                paths: paths,
                workers: 8,
                art: true
            })
        }
        finish()
    }

    function stageCurrentPlaylistFromFiletree(entry) {
        if (!trackEntryPath(entry))
            return
        setCurrentPlaylistFromBrowse(entry)
        selectedPlaylist = currentPlaylistId
        commitCurrentPlaylist()
    }

    function clearCurrentPlaylist() {
        if (!currentPlaylistActive)
            return
        currentPlaylistActive = false
        currentPlaylistTracks = []
        currentPlaylistPath = ""
        rebuildPlaylistTabs()
        if (selectedPlaylist === currentPlaylistId)
            tracks = []
        runMusic(["current", "clear"], null, saveCurrentProc)
    }

    function filetreeTracksFrom(entry) {
        var path = trackEntryPath(entry)
        if (!path)
            return []
        var folderPath = String(entry.folderPath || "")
        var kids = filetreeChildren[folderPath] || []
        var out = []
        var fromHere = false
        for (var i = 0; i < kids.length; i++) {
            if (kids[i].type !== "track")
                continue
            var kidPath = trackEntryPath(kids[i])
            if (!kidPath)
                continue
            if (!fromHere) {
                if (kidPath !== path)
                    continue
                fromHere = true
            }
            out.push(kids[i].track || kids[i])
        }
        if (out.length)
            return out
        fromHere = false
        for (var r = 0; r < filetreeRows.length; r++) {
            var row = filetreeRows[r]
            if (row.type !== "track")
                continue
            if (String(row.folderPath || "") !== folderPath)
                continue
            var rowPath = trackEntryPath(row)
            if (!rowPath)
                continue
            if (!fromHere) {
                if (rowPath !== path)
                    continue
                fromHere = true
            }
            out.push(row.track || row)
        }
        return out
    }

    function setCurrentPlaylistFromBrowse(entry) {
        var path = trackEntryPath(entry)
        var idx = -1
        for (var i = 0; i < tracks.length; i++) {
            if (tracks[i].path === path) {
                idx = i
                break
            }
        }
        var queued = []
        if (idx >= 0) {
            for (var j = idx; j < tracks.length; j++)
                queued.push(tracks[j])
        } else {
            queued = filetreeTracksFrom(entry)
            if (!queued.length)
                queued = path ? [entry.track || entry] : tracks.slice()
        }
        currentPlaylistTracks = queued
        currentPlaylistPath = String(entry.folderPath || "")
        currentPlaylistActive = queued.length > 0
        if (currentPlaylistActive)
            rebuildPlaylistTabs()
    }

    function playlistTabLabel(name) {
        if (String(name || "") === currentPlaylistId)
            return "current"
        if (currentPlaylistActive && String(name || "") === currentPlaylistPath)
            return "current"
        if (String(name || "") === "all")
            return "all likes"
        return String(name || "")
    }

    function nowplayingPlaylistLabel() {
        if (currentPlaylistActive || selectedPlaylist === currentPlaylistId)
            return "current"
        return playlistTabLabel(selectedPlaylist)
    }

    function syncVizSubscription() {
        var snap = root.transportSnap
        // Keep viz subscribed while audio is playing, even in playlist/filetree panels,
        // so switching to Now Playing does not churn subscribe/unsubscribe IPC.
        var want = root.active && String(snap.path || "") !== ""
            && (root.nowplayingTabActive || String(snap.state || "") === "playing")
        // #region agent log
        agentDebug("H4", "DashboardModule.qml:syncVizSubscription", "viz sync", {
            want: want,
            subscribed: _vizSubscribed,
            nowplaying: root.nowplayingTabActive,
            state: String(snap.state || ""),
            path: String(snap.path || "").split("/").pop()
        })
        // #endregion
        if (want) {
            if (_vizSubscribed)
                return
            if (playerMonitor && typeof playerMonitor.ipcCallVoid === "function") {
                playerMonitor.ipcCallVoid("viz.subscribe")
                _vizSubscribed = true
            }
            return
        }
        if (_vizSubscribed) {
            _vizSubscribed = false
            if (playerMonitor && typeof playerMonitor.ipcCallVoid === "function")
                playerMonitor.ipcCallVoid("viz.unsubscribe")
        }
    }

    function onActivated() {
        _startupBootstrapOpenDone = false
        _startupBootstrapCurrentDone = false
        loadLibraryStats()
        loadPlaylists()
        loadCurrentPlaylist(function() {
            _startupBootstrapCurrentDone = true
            finishStartupBootstrap()
        })
        if (!runPlayerQuery(["open", "--json"], function(text) {
            try {
                var saved = JSON.parse(String(text || "{}"))
                resumePlaylist = root.normalizePlaylistName(saved.playlist || "")
                applyStatus(text)
            } catch (e) {
                resumePlaylist = ""
            }
            _startupBootstrapOpenDone = true
            finishStartupBootstrap()
        }))
            Qt.callLater(onActivated)
        if (!playerMonitor)
            statusTimer.start()
        if (playerMonitor)
            mergeMonitorStatus()
        else
            pollStatus(applyStatus)
        saveStateTimer.start()
        jobStatusTimer.start()
        syncExternalJobStatus()
        maybeRefreshQueueUpNext()
        loadPlayerSettings()
        syncVizSubscription()
    }

    function onDeactivated() {
        if (_vizSubscribed) {
            _vizSubscribed = false
            monitorTransport("viz.unsubscribe")
        }
        cancelTrashTrack()
        statusTimer.stop()
        saveStateTimer.stop()
        jobStatusTimer.stop()
        if (volumeApplyPending) {
            volumeApplyTimer.stop()
            volumeSettleTimer.stop()
            flushVolumeApply()
        }
        if (transportApplyPending) {
            transportApplyTimer.stop()
            transportSettleTimer.stop()
            flushTransportApply()
        }
        if (trackTransitionPending)
            forceRevealNowPlaying()
        if (playbackStatePending) {
            playbackToggleTimer.stop()
            playbackSettleTimer.stop()
            finishPlaybackSettle()
        }
        runPlayer(["save"], null, cmdProc)
    }

    function stopPlayback() {
        runPlayer(["stop"], function() {
            trackTransitionPending = false
            transportPreviewPath = ""
            transportApplyPending = false
            player = {}
            waveformSamples = []
            waveformLoading = false
        }, deactivateProc)
    }

    function normalizePlaylistName(name) {
        var n = String(name || "")
        if (n === "favorites")
            return "all"
        if (n.endsWith("-fav"))
            return n.slice(0, -4)
        return n
    }

    function loadPlaylists(force) {
        if (playlistsLoading && !force)
            return
        if (force)
            playlistViewByKey = {}
        playlistsLoading = true
        runPlaylistQuery(["playlist", "--json"], function(text) {
            playlistsLoading = false
            try {
                applyPlaylists(JSON.parse(String(text || "[]")))
            } catch (e) {
                applyPlaylists([])
            }
            var preferred = normalizePlaylistName(resumePlaylist)
            resumePlaylist = ""
            if (preferred === currentPlaylistId) {
                if (currentPlaylistActive) {
                    selectedPlaylist = currentPlaylistId
                    loadPlaylistTracks(currentPlaylistId)
                }
                syncPlaylistTabPosition()
            } else if (preferred === "all") {
                selectPlaylist("all", false)
            } else if (preferred) {
                for (var i = 0; i < playlists.length; i++) {
                    if (playlists[i].name === preferred) {
                        selectPlaylist(preferred, false)
                        return
                    }
                }
                syncPlaylistTabPosition()
            } else {
                syncPlaylistTabPosition()
            }
        })
    }

    function syncPlaylistTabPosition() {
        if (!playlistTabBar || playlistTabModel.count === 0 || !selectedPlaylist)
            return
        for (var i = 0; i < playlistTabModel.count; i++) {
            if (playlistTabModel.get(i).name === selectedPlaylist) {
                playlistTabBar.positionViewAtIndex(i, ListView.Center)
                return
            }
        }
    }

    function openBrowse() {
        toggleFiletreePanel()
    }

    function openPlaylistLibrary() {
        playerScreen = "nowplaying"
        filetreePanelOpen = false
        settingsPanelOpen = false
        statsPanelOpen = false
        playlistsPanelOpen = true
        playlistPanelMode = "library"
        if (!libraryPlaylists.length && !playlistsLoading)
            loadPlaylists()
        else
            Qt.callLater(root.ensureLibraryData)
    }

    function selectGenrePlaylist(name) {
        if (!name)
            return
        var next = normalizePlaylistName(name)
        selectedLibraryPlaylist = next
    }

    function openSelectedLibraryPlaylist() {
        if (!selectedLibraryPlaylist)
            return
        selectPlaylist(selectedLibraryPlaylist)
    }

    function togglePlaylistStar(name) {
        var playlistName = String(name || "")
        if (!playlistCanStar(playlistName))
            return
        var nextList = []
        for (var i = 0; i < libraryPlaylists.length; i++) {
            var item = libraryPlaylists[i]
            if (item.name === playlistName)
                nextList.push(Object.assign({}, item, { starred: !item.starred }))
            else
                nextList.push(item)
        }
        applyPlaylists(nextList)
        runPlaylistQuery(["playlist", "star", "toggle", playlistName, "--json"], function(text) {
            try {
                JSON.parse(String(text || "{}"))
            } catch (e) {
            }
            root.loadPlaylists(true)
        })
    }

    function runFiletreeQuery(args, onDone) {
        _filetreeQueryQueue.push({
            args: (args || []).slice(),
            onDone: onDone || null
        })
        pumpBrowseQuery()
    }

    function pumpBrowseQuery() {
        if (filetreeProc.running)
            return
        var job = _filetreeQueryQueue[0]
        if (!job)
            return
        filetreeProc._jobDone = false
        filetreeProc.command = playerCmd(job.args || [])
        filetreeProc._onDone = function(text) {
            if (filetreeProc._jobDone)
                return
            filetreeProc._jobDone = true
            _filetreeQueryQueue.shift()
            if (job.onDone)
                job.onDone(text)
            Qt.callLater(pumpBrowseQuery)
        }
        filetreeProc.running = true
    }

    function fetchFolderQueueTracks(folderPath, onDone) {
        folderPath = String(folderPath || "")
        function deliver(text) {
            var tracks = FiletreeLogic.parseBrowseQueueTracks(text)
            if (!tracks.length)
                tracks = FiletreeLogic.collectFolderTracksFromChildren(filetreeChildren, folderPath)
            if (onDone)
                onDone(tracks)
        }
        var rel = filetreeRelForBrowse(folderPath)
        function runCLI() {
            runFiletreeQuery(["browse", folderPath, "--json", "--queue", "--paths-only"], deliver)
        }
        if (!runDaemonJSON("library.browse", {
            path: rel,
            queue: true,
            queue_paths_only: true
        }, deliver, runCLI))
            runCLI()
    }

    function filetreeRelForBrowse(absPath) {
        absPath = String(absPath || "")
        if (!absPath)
            return ""
        var root = String(musicRoot || "")
        if (root && absPath.indexOf(root) === 0) {
            var rel = absPath.substring(root.length)
            if (rel.charAt(0) === "/")
                rel = rel.substring(1)
            return rel
        }
        return absPath
    }

    function applyFiletreeFolderTracks(folderPath, folderTracks, mode) {
        folderPath = String(folderPath || "")
        folderTracks = folderTracks || []
        if (!folderTracks.length) {
            notify("no tracks in folder", 2500)
            return
        }
        if (mode === "append") {
            runDaemonJSON("queue.append_folder", { path: folderPath }, function(text) {
                var result = {}
                try {
                    result = JSON.parse(String(text || "{}"))
                } catch (e) {
                }
                var added = Number(result.added || 0)
                if (added <= 0) {
                    notify("no new tracks to add", 2500)
                    return
                }
                var merge = FiletreeLogic.appendTracksUnique(currentPlaylistTracks, folderTracks)
                currentPlaylistTracks = merge.tracks
                currentPlaylistActive = true
                currentPlaylistPath = folderPath
                selectedPlaylist = currentPlaylistId
                rebuildPlaylistTabs()
                commitCurrentPlaylist({ appendCount: added, skipPersist: true })
                scheduleCurrentTracksEnrich()
                var appendLabel = folderPath.split("/").pop() || "folder"
                notify("+" + added + " to current from " + appendLabel, 2500)
            }, function() {
                var merge = FiletreeLogic.appendTracksUnique(currentPlaylistTracks, folderTracks)
                if (merge.added <= 0) {
                    notify("no new tracks to add", 2500)
                    return
                }
                currentPlaylistTracks = merge.tracks
                currentPlaylistActive = true
                currentPlaylistPath = folderPath
                selectedPlaylist = currentPlaylistId
                rebuildPlaylistTabs()
                commitCurrentPlaylist({ appendCount: merge.added })
                scheduleCurrentTracksEnrich()
                var appendLabel = folderPath.split("/").pop() || "folder"
                notify("+" + merge.added + " to current from " + appendLabel, 2500)
            })
            return
        }
        setCurrentPlaylistFromTracks(folderPath, folderTracks)
        selectedPlaylist = currentPlaylistId
        selectedTrackIndex = 0
        var paths = FiletreeLogic.pathsFromTracks(folderTracks)
        daemonVoid("queue.play_current", {
            paths: paths,
            start_path: paths[0]
        }, function(ok) {
            if (ok)
                mergeMonitorStatus()
        }, function() {
            syncCurrentQueueAt(paths[0])
        })
        commitCurrentPlaylist({ skipPersist: true })
        var playLabel = folderPath.split("/").pop() || "folder"
        notify("playing " + paths.length + " from " + playLabel, 2000)
    }

    function beginFiletreeFolderQueue(entry, onReady) {
        entry = FiletreeLogic.normalizeFolderEntry(entry)
        if (!entry) {
            notify("invalid folder", 2000)
            return false
        }
        if (filetreeQueueBusy) {
            notify("busy — folder queue", 2000)
            return false
        }
        filetreeQueueBusy = true
        filetreeQueueBusyTimer.restart()
        var folderPath = String(entry.path)
        var cached = FiletreeLogic.collectFolderTracksFromChildren(filetreeChildren, folderPath)
        if (cached.length && FiletreeLogic.folderTracksComplete(filetreeChildren, filetreeFolderMeta, folderPath)) {
            filetreeQueueBusy = false
            filetreeQueueBusyTimer.stop()
            if (onReady)
                onReady(folderPath, cached)
            return true
        }
        fetchFolderQueueTracks(folderPath, function(folderTracks) {
            filetreeQueueBusy = false
            filetreeQueueBusyTimer.stop()
            if (onReady)
                onReady(folderPath, folderTracks)
        })
        return true
    }

    function filetreeFolderEntry(path, name) {
        return { type: "dir", path: String(path || ""), name: String(name || "") }
    }

    function expansionStateKey(expanded) {
        return FiletreeLogic.expansionStateKey(
            expanded !== undefined ? expanded : filetreeExpanded)
    }

    function saveFiletreeScroll(key) {
        if (!filetreeListView || filetreeScrollLocked)
            return
        var map = Object.assign({}, filetreeScrollByKey)
        map[key !== undefined ? String(key) : expansionStateKey()] = filetreeListView.contentY
        filetreeScrollByKey = map
    }

    function restoreFiletreeScroll(key) {
        var k = key !== undefined ? String(key) : expansionStateKey()
        filetreeRestoreY = FiletreeLogic.scrollRestoreY(filetreeScrollByKey, k)
    }

    function scrollFiletreeViewportAfterLoad(key) {
        restoreFiletreeScroll(key)
        if (filetreeRestoreY >= 0) {
            filetreeKickScrollTimer.restart()
            return
        }
        if (filetreeListView)
            filetreeListView.contentY = 0
    }

    function pruneFiletreeExpansion(path) {
        var nextExp = Object.assign({}, filetreeExpanded)
        var prefix = String(path || "")
        for (var p in nextExp) {
            if (p === prefix || (prefix && p.indexOf(prefix + "/") === 0))
                delete nextExp[p]
        }
        return nextExp
    }

    function rebuildFiletreeRows(preserveScroll) {
        var rows = FiletreeLogic.rebuildFiletreeRows(filetreeChildren, filetreeExpanded)
        var list = filetreeListView
        var holdY = filetreeHoldY
        var anchorY = -1
        var anchorIndex = -1
        if (preserveScroll && holdY < 0 && list) {
            anchorY = list.contentY
            anchorIndex = list.count > 0
                ? Math.max(0, list.indexAt(0, list.contentY + 1))
                : -1
        }
        filetreeRows = rows
        var plan = FiletreeLogic.planFiletreeReflowScroll({
            preserveScroll: preserveScroll,
            holdY: holdY,
            restoreY: filetreeRestoreY,
            anchorY: anchorY,
            anchorIndex: anchorIndex
        })
        if (plan.action === "hold" && list) {
            filetreeHoldSafetyTimer.restart()
            Qt.callLater(function() {
                applyFiletreeHoldViewport()
            })
        } else if (plan.action === "restore" && list) {
            filetreeScrollLocked = true
            Qt.callLater(function() {
                restoreListViewport(list, plan.y, plan.anchorIndex, function() {
                    filetreeScrollLocked = false
                })
            })
        } else if (plan.action === "resetTop" && list) {
            list.contentY = 0
        }
    }

    function applyFiletreeHoldViewport() {
        var list = filetreeListView
        if (!list || filetreeHoldY < 0)
            return
        list.contentY = FiletreeLogic.clampContentY(
            filetreeHoldY, list.contentHeight, list.height)
    }

    function patchTrackLikedInFiletree(trackPath, liked) {
        trackPath = String(trackPath || "")
        if (!trackPath)
            return false
        var nextChildren = {}
        var changed = false
        for (var key in filetreeChildren) {
            var kids = filetreeChildren[key]
            var nextKids = []
            for (var i = 0; i < kids.length; i++) {
                var kid = kids[i]
                if (kid.type === "track" && String(kid.path) === trackPath) {
                    var patched = Object.assign({}, kid, { liked: liked })
                    if (kid.track)
                        patched.track = Object.assign({}, kid.track, { liked: liked })
                    nextKids.push(patched)
                    changed = true
                } else {
                    nextKids.push(kid)
                }
            }
            nextChildren[key] = nextKids
        }
        if (!changed)
            return false
        filetreeChildren = nextChildren
        rebuildFiletreeRows(true)
        return true
    }

    function queueFiletreeArtPatch(trackPath, art, albumScope, thumb) {
        trackPath = String(trackPath || "")
        if (!trackPath)
            return
        var q = Object.assign({}, _filetreeArtPatchQueue)
        q[trackPath] = {
            art: String(art || ""),
            thumb: thumb !== undefined ? String(thumb || "") : "",
            albumScope: !!albumScope
        }
        _filetreeArtPatchQueue = q
        if (!_filetreeArtPatchFlushPending) {
            _filetreeArtPatchFlushPending = true
            Qt.callLater(flushFiletreeArtPatches)
        }
    }

    function flushFiletreeArtPatches() {
        if (FiletreeLogic.shouldDeferFiletreeArtFlush({
            scrollLocked: filetreeScrollLocked,
            userScrolling: filetreeUserScrolling
        })) {
            if (!_filetreeArtPatchFlushPending)
                _filetreeArtPatchFlushPending = true
            Qt.callLater(flushFiletreeArtPatches)
            return false
        }
        _filetreeArtPatchFlushPending = false
        var queue = _filetreeArtPatchQueue
        _filetreeArtPatchQueue = ({})
        if (!Object.keys(queue).length)
            return false
        var result = FiletreeLogic.patchFiletreeArtBatch(
            filetreeChildren, filetreeRows, queue)
        if (!result.changed)
            return false
        filetreeChildren = result.children
        filetreeArtRevision++
        return true
    }

    function patchFiletreeArt(trackPath, art, albumScope, thumb) {
        queueFiletreeArtPatch(trackPath, art, albumScope, thumb)
        return true
    }

    function patchFiletreeWaveform(trackPath, waveform) {
        trackPath = String(trackPath || "")
        waveform = String(waveform || "")
        if (!trackPath)
            return false
        var nextChildren = {}
        var changed = false
        for (var key in filetreeChildren) {
            var kids = filetreeChildren[key]
            var nextKids = []
            for (var i = 0; i < kids.length; i++) {
                var kid = kids[i]
                if (kid.type === "track" && String(kid.path) === trackPath) {
                    var patched = Object.assign({}, kid, { waveform: waveform })
                    if (kid.track)
                        patched.track = Object.assign({}, kid.track, { waveform: waveform })
                    nextKids.push(patched)
                    changed = true
                } else {
                    nextKids.push(kid)
                }
            }
            nextChildren[key] = nextKids
        }
        if (!changed)
            return false
        filetreeChildren = nextChildren
        filetreeArtRevision++
        return true
    }

    function findTrackLikedInFiletree(trackPath) {
        trackPath = String(trackPath || "")
        for (var key in filetreeChildren) {
            var kids = filetreeChildren[key]
            for (var i = 0; i < kids.length; i++) {
                var kid = kids[i]
                if (kid.type === "track" && String(kid.path) === trackPath)
                    return !!kid.liked
            }
        }
        return undefined
    }

    function patchFiletreeFolderArtImmediate(folderEntries) {
        if (!folderEntries || !folderEntries.length)
            return false
        var shared = FiletreeLogic.folderArtFromEntries(folderEntries)
        if (!shared.art && !shared.thumb)
            return false
        var warmPath = ""
        var warmCandidates = FiletreeLogic.warmPathsForFolder(folderEntries, 1)
        if (warmCandidates.length)
            warmPath = warmCandidates[0]
        else {
            var wi
            for (wi = 0; wi < folderEntries.length; wi++) {
                if (folderEntries[wi] && folderEntries[wi].type === "track") {
                    warmPath = FiletreeLogic.trackPathForEntry(folderEntries[wi])
                    break
                }
            }
        }
        if (!warmPath)
            return false
        queueFiletreeArtPatch(warmPath, shared.art, true, shared.thumb)
        return flushFiletreeArtPatches()
    }

    function warmFiletreeFolder(entries) {
        if (!entries || !entries.length)
            return false
        var paths = FiletreeLogic.warmPathsForFolder(entries, playlistPageSize)
        if (!paths.length)
            return false
        if (!playerMonitor || typeof playerMonitor.ipcCallVoid !== "function")
            return false
        filetreeDeferVisibleArtWarm = true
        playerMonitor.ipcCallVoid("library.warm.batch", {
            paths: paths,
            workers: 8,
            art: true
        })
        return true
    }

    function applyFiletreeEntries(relPath, entries, meta, append) {
        relPath = String(relPath || "")
        entries = FiletreeLogic.hydrateFolderArtInEntries(entries || [])
        var next = Object.assign({}, filetreeChildren)
        var nextMeta = Object.assign({}, filetreeFolderMeta)
        var trackCount = 0
        var i
        for (i = 0; i < entries.length; i++) {
            if (entries[i].type === "track")
                trackCount++
        }
        if (append && next[relPath]) {
            var existing = next[relPath]
            var dirs = []
            var tracks = []
            var j
            for (j = 0; j < existing.length; j++) {
                if (existing[j].type === "dir")
                    dirs.push(existing[j])
                else if (existing[j].type === "track")
                    tracks.push(existing[j])
            }
            for (j = 0; j < entries.length; j++) {
                if (entries[j].type === "track")
                    tracks.push(entries[j])
            }
            next[relPath] = dirs.concat(tracks)
            trackCount = tracks.length
        } else {
            next[relPath] = entries
        }
        if (relPath !== "") {
            nextMeta[relPath] = {
                total: Number(meta.trackTotal) || trackCount,
                loaded: trackCount
            }
        }
        filetreeChildren = next
        filetreeFolderMeta = nextMeta
        if (relPath !== "")
            patchFiletreeFolderArtImmediate(next[relPath])
        var preservingScroll = !!append || filetreeHoldY >= 0
        rebuildFiletreeRows(preservingScroll)
        if (relPath !== "" && !warmFiletreeFolder(next[relPath]))
            scheduleVisibleArtWarm()
        else if (relPath === "")
            scheduleVisibleArtWarm()
    }

    function flushPendingFiletreeRefresh() {
        if (!_filetreeRefreshPending)
            return
        _filetreeRefreshPending = false
        Qt.callLater(function() {
            refreshFiletree()
        })
    }

    function fetchFiletreeEntries(relPath, offset, append, onDone) {
        relPath = String(relPath || "")
        function deliver(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                var entries = data.entries || []
                if (onDone) {
                    onDone(entries, {
                        trackTotal: relPath === ""
                            ? 0
                            : (Number(data.trackTotal) || 0),
                        trackOffset: Number(data.trackOffset) || 0,
                        trackLimit: Number(data.trackLimit) || playlistPageSize
                    })
                }
            } catch (e) {
                if (onDone)
                    onDone([], { trackTotal: 0, trackOffset: 0, trackLimit: playlistPageSize })
            }
        }
        var params = {
            path: relPath,
            offset: Number(offset) || 0,
            limit: playlistPageSize
        }
        function runCLI() {
            var args = ["browse", relPath, "--json"]
            if (relPath !== "") {
                args = args.concat([
                    "--offset", String(offset || 0),
                    "--limit", String(playlistPageSize)
                ])
            }
            runFiletreeQuery(args, deliver)
        }
        if (!runDaemonJSON("library.browse", params, deliver, runCLI))
            runCLI()
    }

    function loadMoreFiletreeTracks() {
        if (filetreeLoadingMore || filetreeTreeLoading)
            return
        var folderPath = ""
        var i
        for (i = filetreeRows.length - 1; i >= 0; i--) {
            if (filetreeRows[i].type === "track") {
                folderPath = String(filetreeRows[i].folderPath || "")
                break
            }
        }
        if (!folderPath)
            return
        var meta = filetreeFolderMeta[folderPath]
        if (!meta || meta.loaded >= meta.total)
            return
        var list = filetreeListView
        var anchorY = list ? list.contentY : -1
        var anchorIndex = list && list.count > 0
            ? Math.max(0, list.indexAt(0, list.contentY + 1))
            : -1
        filetreeLoadingMore = true
        fetchFiletreeEntries(folderPath, meta.loaded, true, function(entries, pagMeta) {
            filetreeLoadingMore = false
            applyFiletreeEntries(folderPath, entries, pagMeta, true)
            restoreListViewport(list, anchorY, anchorIndex)
        })
    }

    function loadFiletreeRoot() {
        filetreeTreeLoading = true
        filetreeLoadSafetyTimer.restart()
        fetchFiletreeEntries("", 0, false, function(entries) {
            filetreeTreeLoading = false
            applyFiletreeEntries("", entries, {}, false)
            scrollFiletreeViewportAfterLoad("")
            flushPendingFiletreeRefresh()
        })
    }

    function refreshFiletree() {
        if (filetreeTreeLoading) {
            _filetreeRefreshPending = true
            return
        }
        _filetreeRefreshPending = false
        saveFiletreeScroll()
        var anchorY = filetreeListView ? filetreeListView.contentY : -1
        var expanded = []
        for (var p in filetreeExpanded) {
            if (filetreeExpanded[p])
                expanded.push(String(p))
        }
        expanded.sort(function(a, b) {
            var da = a === "" ? 0 : a.split("/").length
            var db = b === "" ? 0 : b.split("/").length
            return da - db || a.localeCompare(b)
        })
        var paths = [""].concat(expanded)
        var pathIndex = 0
        var nextChildren = {}
        var nextMeta = {}
        filetreeTreeLoading = true
        filetreeLoadSafetyTimer.restart()

        function step() {
            if (pathIndex >= paths.length) {
                filetreeChildren = nextChildren
                filetreeFolderMeta = nextMeta
                filetreeTreeLoading = false
                beginFiletreeReflowHold(anchorY)
                rebuildFiletreeRows(false)
                restoreFiletreeViewport(anchorY, -1)
                flushPendingFiletreeRefresh()
                return
            }
            var path = paths[pathIndex++]
            fetchFiletreeEntries(path, 0, false, function(entries, meta) {
                var trackCount = 0
                for (var i = 0; i < entries.length; i++) {
                    if (entries[i].type === "track")
                        trackCount++
                }
                nextChildren[path] = entries
                if (path !== "") {
                    nextMeta[path] = {
                        total: Number(meta.trackTotal) || trackCount,
                        loaded: trackCount
                    }
                }
                step()
            })
        }
        step()
    }

    function reloadFiletreeView() {
        if (!filetreePanelOpen && filetreeRows.length === 0)
            return
        for (var p in filetreeExpanded) {
            if (filetreeExpanded[p]) {
                refreshFiletree()
                return
            }
        }
        loadFiletreeRoot()
    }

    function filetreeDirIndex(path) {
        path = String(path || "")
        for (var i = 0; i < filetreeRows.length; i++) {
            var row = filetreeRows[i]
            if (row && row.type === "dir" && String(row.path) === path)
                return i
        }
        return -1
    }

    function restoreFiletreeViewport(contentY, anchorIndex) {
        filetreeRestoreY = -1
        filetreeReflowHidden = false
        if (contentY >= 0)
            beginFiletreeReflowHold(contentY)
        restoreListViewport(filetreeListView, contentY, anchorIndex, function() {
            applyFiletreeHoldViewport()
        })
    }

    function beginFiletreeReflowHold(contentY) {
        if (contentY < 0)
            return
        filetreeHoldY = contentY
        filetreeHoldSafetyTimer.restart()
    }

    function toggleFiletreeNode(path) {
        path = String(path || "")
        saveFiletreeScroll()
        var list = filetreeListView
        var anchorY = list ? list.contentY : -1
        var anchorIndex = filetreeDirIndex(path)
        var nextExp = Object.assign({}, filetreeExpanded)
        if (nextExp[path]) {
            nextExp = pruneFiletreeExpansion(path)
            filetreeExpanded = nextExp
            beginFiletreeReflowHold(anchorY)
            rebuildFiletreeRows(false)
            restoreFiletreeViewport(anchorY, anchorIndex)
            return
        }
        nextExp[path] = true
        filetreeExpanded = nextExp
        if (filetreeChildren[path]) {
            beginFiletreeReflowHold(anchorY)
            rebuildFiletreeRows(false)
            warmFiletreeFolder(filetreeChildren[path])
            scheduleVisibleArtWarm()
            restoreFiletreeViewport(anchorY, anchorIndex)
            return
        }
        fetchFiletreeEntries(path, 0, false, function(entries, meta) {
            beginFiletreeReflowHold(anchorY)
            applyFiletreeEntries(path, entries, meta, false)
            restoreFiletreeViewport(anchorY, anchorIndex)
        })
    }

    function playFiletreeTrack(entry) {
        var track = (entry && entry.track && typeof entry.track === "object") ? entry.track : entry
        if (!track || !track.path)
            return
        if (!isTrackSelected(track.path))
            selectTrackEntry(entry)
        stageCurrentPlaylistFromFiletree(entry)
        selectedPlaylist = currentPlaylistId
        playPathFromList(track.path, true)
    }

    function filetreeDirGenre(path) {
        var p = String(path || "")
        if (!p)
            return ""
        var slash = p.indexOf("/")
        return slash < 0 ? p : p.substring(0, slash)
    }

    function selectFiletreeTrack(entry) {
        if (!entry || !entry.path)
            return
        selectedTrackPath = String(entry.path)
        for (var i = 0; i < tracks.length; i++) {
            if (tracks[i].path === entry.path) {
                selectedTrackIndex = i
                stageCurrentPlaylistFromFiletree(entry)
                return
            }
        }
        selectedTrackIndex = -1
    }

    function setCurrentPlaylistFromTracks(folderPath, folderTracks) {
        currentPlaylistTracks = folderTracks.slice()
        currentPlaylistPath = String(folderPath || "")
        currentPlaylistActive = true
        rebuildPlaylistTabs()
        if (playlistsPanelOpen && selectedPlaylist === currentPlaylistId)
            loadPlaylistTracks(currentPlaylistId)
    }

    function filetreeQueueFolder(entry) {
        beginFiletreeFolderQueue(entry, function(folderPath, folderTracks) {
            applyFiletreeFolderTracks(folderPath, folderTracks, "replace")
        })
    }

    function filetreeSortFolder(entry) {
        if (!entry || entry.type !== "dir" || !entry.path)
            return
        if (sortProc.running) {
            notify("busy — sort", 2000)
            return
        }
        if (libraryJobBusy) {
            notify("busy — " + libraryJobActiveLabel, 2000)
            return
        }
        var folderPath = String(entry.path)
        var label = "sort " + folderPath.split("/").pop()
        runJob(["sort", folderPath, "--json"], label, { stayOnScreen: false })
    }

    function onSortFinished(exitCode) {
        var label = sortProc._label || "sort"
        sortProc._label = ""
        if (sortStopRequested) {
            sortStopRequested = false
            return
        }
        if (exitCode === 0) {
            notify(label + " complete", 4000)
            jobLog = String(jobLog || "") + "\n" + label + " complete"
            if (filetreePanelOpen || filetreeRows.length > 0)
                reloadFiletreeView()
        } else if (exitCode === 2) {
            notify("busy — cannot sort now", 4000)
        } else {
            var err = sortErr.text ? String(sortErr.text).trim().split("\n").pop() : ""
            notify(label + " failed" + (err ? " — " + err : ""), 5000)
        }
    }

    function filetreeAppendFolder(entry) {
        beginFiletreeFolderQueue(entry, function(folderPath, folderTracks) {
            applyFiletreeFolderTracks(folderPath, folderTracks, "append")
        })
    }

    function openFilter(kind, value, label) {
        filetreePanelOpen = false
        playlistsPanelOpen = false
        settingsPanelOpen = false
        downloadsPanelOpen = false
        statsPanelOpen = false
        filterKind = String(kind || "")
        filterLabel = String(label || value || "")
        filterTracks = []
        filterLoading = true
        playerScreen = "filter"
        var mode = kind === "search" ? "search" : String(kind || "search")
        var query = String(value || "")
        var applyFilter = function(text) {
            filterLoading = false
            try {
                filterTracks = JSON.parse(String(text || "[]"))
            } catch (e) {
                filterTracks = []
            }
            warmTracksArt(filterTracks)
        }
        var args = ["find"]
        if (kind === "artist")
            args = args.concat(["--artist", query])
        else if (kind === "album")
            args = args.concat(["--album", query])
        else if (kind === "label")
            args = args.concat(["--label", query])
        else if (kind === "genre")
            args = args.concat(["--genre", query])
        else if (kind === "year")
            args = args.concat(["--year", query])
        else
            args.push(query)
        args.push("--json")
        runDaemonJSON("library.search", { mode: mode, query: query }, applyFilter, function() {
            runQuery(args, applyFilter)
        })
    }

    function runTabSearch() {
        var q = String(tabSearchText || "").trim()
        if (!q)
            return
        openFilter("search", q, q)
    }

    function filterKindTitle() {
        if (filterKind === "artist")
            return "Artist"
        if (filterKind === "album")
            return "Album"
        if (filterKind === "label")
            return "Label"
        if (filterKind === "genre")
            return "Genre"
        if (filterKind === "year")
            return "Year"
        if (filterKind === "search")
            return "Search"
        return "Filter"
    }

    function filterKindIcon() {
        if (filterKind === "artist")
            return "󰠃"
        if (filterKind === "album")
            return "󰀥"
        if (filterKind === "label")
            return "󰈿"
        if (filterKind === "genre")
            return "󰓤"
        if (filterKind === "year")
            return "󰃭"
        return "󰍉"
    }

    function filterHeaderTitle() {
        var kind = filterKindTitle()
        var label = String(filterLabel || "").trim()
        if (!label)
            return kind
        return kind + " · " + label
    }

    function playFilterTrackAt(index) {
        if (index < 0 || index >= filterTracks.length)
            return
        var folderTracks = filterTracks.slice()
        var label = filterHeaderTitle()
        setCurrentPlaylistFromTracks(label, folderTracks)
        selectedPlaylist = currentPlaylistId
        selectedTrackIndex = index
        selectedTrackPath = String(filterTracks[index].path || "")
        var paths = pathsFromTracks(folderTracks)
        syncCurrentQueueAt(paths[index])
        commitCurrentPlaylist()
    }

    function selectFilterTrack(index) {
        if (index < 0 || index >= filterTracks.length)
            return
        selectedTrackPath = String(filterTracks[index].path || "")
    }

    function playFiletreeTrackEntry(entry) {
        if (!entry || !entry.path)
            return
        if (!root.isTrackSelected(entry.path)) {
            selectFiletreeTrack(entry)
            return
        }
        stageCurrentPlaylistFromFiletree(entry)
        selectedPlaylist = currentPlaylistId
        playPathFromList(entry.path, true)
    }

    function cacheSelectedTrack(entry) {
        var track = (entry && entry.track) ? entry.track : entry
        if (!track)
            return
        selectedTrackCache = {
            path: String(track.path || (entry && entry.path) || ""),
            art: String(track.art || (entry && entry.art) || ""),
            album: String(track.album || ""),
            title: String(track.title || (entry && entry.name) || ""),
            artist: String(track.artist || (entry && entry.artist) || "")
        }
    }

    function selectFiletreeFolder(path) {
        path = String(path || "")
        if (!path)
            return
        selectedFiletreeFolderPath = path
        if (selectedTrackPath && selectedTrackPath !== String(player.path || ""))
            selectedTrackPath = ""
    }

    function isTrackPlaying(path) {
        return String(player.path || "") === String(path || "")
            && String(player.state || "") === "playing"
    }

    function isTrackSelected(path) {
        path = String(path || "")
        if (!path)
            return false
        if (isTrackPlaying(path))
            return true
        return String(selectedTrackPath) === path
    }

    function trackEntryPath(entry) {
        if (!entry)
            return ""
        var track = (entry.track && typeof entry.track === "object") ? entry.track : entry
        return String(track.path || entry.path || "")
    }

    function selectTrackEntry(entry) {
        var path = trackEntryPath(entry)
        if (!path)
            return
        if (path !== String(selectedTrackPath || ""))
            closeArtPicker()
        selectedTrackPath = path
        cacheSelectedTrack(entry)
        selectedFiletreeFolderPath = ""
        primePlayerForPath(path, false)
        var list = filetreeListView
        var anchorY = filetreePanelOpen && list ? list.contentY : -1
        var anchorIndex = list && list.count > 0
            ? Math.max(0, list.indexAt(0, list.contentY + 1))
            : -1
        applyDisplayArtForPath(path, function() {
            if (resolvedArtPath === path && resolvedArt)
                cacheSelectedTrack(Object.assign({}, entry, { art: resolvedArt }))
            refreshNowPlayingArtDisplay()
            if (filetreePanelOpen && list && anchorY >= 0) {
                Qt.callLater(function() {
                    restoreFiletreeViewport(anchorY, anchorIndex)
                })
            }
        })
    }

    function openTrackFolder(entry) {
        if (!entry)
            return
        var folderPath = String(entry.folderPath || "").trim()
        if (folderPath) {
            openBrowseFolder({ path: folderPath })
            return
        }
        var path = String(entry.path || "").trim()
        if (!path)
            return
        var slash = path.lastIndexOf("/")
        if (slash > 0)
            openBrowseFolder({ path: path.substring(0, slash) })
        else
            openTrackInThunar(path)
    }

    function requestTrashTrack(trackOrPath) {
        var path = ""
        var title = ""
        if (typeof trackOrPath === "string") {
            path = String(trackOrPath || "").trim()
        } else if (trackOrPath) {
            path = String(trackOrPath.path || "").trim()
            title = String(trackOrPath.title || "").trim()
        }
        if (!path || trashTrackProc.running)
            return
        if (!title) {
            var slash = path.lastIndexOf("/")
            title = slash >= 0 ? path.substring(slash + 1) : path
        }
        trashConfirmPath = path
        trashConfirmTitle = title
    }

    function cancelTrashTrack() {
        trashConfirmPath = ""
        trashConfirmTitle = ""
    }

    function confirmTrashTrack() {
        var path = String(trashConfirmPath || "").trim()
        cancelTrashTrack()
        trashTrack(path)
    }

    function trashTrack(trackPath) {
        var path = String(trackPath || "").trim()
        if (!path || trashTrackProc.running)
            return
        trashTrackProc._path = path
        trashTrackProc.command = ["gio", "trash", path]
        trashTrackProc.running = true
    }

    function removeTrackFromViews(path) {
        path = String(path || "")
        var i
        var nextTracks = []
        for (i = 0; i < tracks.length; i++) {
            if (tracks[i].path !== path)
                nextTracks.push(tracks[i])
        }
        if (nextTracks.length !== tracks.length)
            assignTracks(nextTracks)
        var nextFilter = []
        for (i = 0; i < filterTracks.length; i++) {
            if (filterTracks[i].path !== path)
                nextFilter.push(filterTracks[i])
        }
        if (nextFilter.length !== filterTracks.length)
            filterTracks = nextFilter
        if (currentPlaylistActive) {
            var nextCurrent = []
            for (i = 0; i < currentPlaylistTracks.length; i++) {
                if (currentPlaylistTracks[i].path !== path)
                    nextCurrent.push(currentPlaylistTracks[i])
            }
            if (nextCurrent.length !== currentPlaylistTracks.length)
                currentPlaylistTracks = nextCurrent
        }
    }

    function onTrackTrashed(exitCode) {
        var path = trashTrackProc._path || ""
        trashTrackProc._path = ""
        if (exitCode !== 0) {
            notify("could not trash file", 3000)
            return
        }
        if (String(selectedTrackPath) === path)
            selectedTrackPath = ""
        if (selectedTrackIndex >= 0 && selectedTrackIndex < tracks.length
                && tracks[selectedTrackIndex].path === path)
            selectedTrackIndex = -1
        if (isTrackPlaying(path))
            runPlayer(["stop"], root.refreshStatus, cmdProc)
        removeTrackFromViews(path)
        savePlaylistView(selectedPlaylist)
        reloadFiletreeView()
        notify("moved to trash", 2500)
    }

    function selectPlaylist(name, switchScreen, focusNowPlaying) {
        var next = normalizePlaylistName(name)
        if (playlistPanelMode === "tracks" && selectedPlaylist && selectedPlaylist !== next)
            savePlaylistView(selectedPlaylist)
        selectedPlaylist = next
        syncPlaylistTabPosition()
        if (switchScreen !== false) {
            playerScreen = "nowplaying"
            filetreePanelOpen = false
            settingsPanelOpen = false
            statsPanelOpen = false
            playlistsPanelOpen = true
            playlistPanelMode = "tracks"
        }
        playlistFocusNowPlaying = focusNowPlaying !== false
        loadPlaylistTracks(selectedPlaylist)
    }

    function savePlaylistView(name) {
        snapshotPlaylistTracksCache(name)
    }

    function snapshotPlaylistTracksCache(name) {
        var key = String(name || selectedPlaylist || "")
        if (!key || playlistScrollLocked)
            return
        if (!tracks.length)
            return
        var list = playlistTrackList
        var y = -1
        if (playlistRestoreY >= 0)
            y = playlistRestoreY
        else if (list && playlistPanelMode === "tracks" && list.visible && list.height > 0)
            y = list.contentY
        else if (playlistViewByKey[key] && playlistViewByKey[key].contentY >= 0)
            y = Number(playlistViewByKey[key].contentY)
        if (y < 0)
            y = 0
        var map = Object.assign({}, playlistViewByKey)
        map[key] = {
            contentY: y,
            tracks: tracks.slice(),
            offset: playlistTrackOffset,
            total: playlistTrackTotal
        }
        playlistViewByKey = map
    }

    function savePlaylistScrollPosition(name) {
        var key = String(name || selectedPlaylist || "")
        if (!key || playlistScrollLocked || !tracks.length)
            return
        var existing = playlistViewByKey[key]
        if (!existing || !existing.tracks || !existing.tracks.length)
            return
        var list = playlistTrackList
        var y = -1
        if (list && playlistPanelMode === "tracks" && list.visible && list.height > 0)
            y = list.contentY
        else
            return
        if (y < 0 || Math.abs(Number(existing.contentY || 0) - y) < 1)
            return
        var map = Object.assign({}, playlistViewByKey)
        map[key] = Object.assign({}, existing, { contentY: y })
        playlistViewByKey = map
    }

    function restorePlaylistScroll(name) {
        var key = String(name || selectedPlaylist || "")
        var saved = playlistViewByKey[key]
        var y = saved && saved.contentY !== undefined && saved.contentY !== null
            ? Number(saved.contentY)
            : -1
        playlistRestoreY = y >= 0 ? y : -1
    }

    function applyCachedPlaylistView(name) {
        var key = String(name || "")
        var saved = playlistViewByKey[key]
        if (!key || !saved || !saved.tracks || !saved.tracks.length)
            return false
        tracksLoading = false
        playlistTracksLoadingMore = false
        if (!playlistFocusNowPlaying)
            restorePlaylistScroll(key)
        var restoreY = playlistFocusNowPlaying ? -1 : playlistRestoreY
        playlistRestoreY = -1
        assignTracks(saved.tracks.slice(), { scrollY: restoreY })
        playlistTrackOffset = Number(saved.offset) || tracks.length
        playlistTrackTotal = Number(saved.total) || tracks.length
        mergePlayerFromTrackList()
        return true
    }

    function loadPlaylistTracks(name, force) {
        if (!name) {
            tracks = []
            tracksLoading = false
            playlistTrackTotal = 0
            playlistTrackOffset = 0
            return
        }
        var requested = String(name)
        if (requested === currentPlaylistId) {
            root.applyCurrentPlaylistTracksView()
            if (root.currentPlaylistNeedsMeta())
                root.loadCurrentPlaylist()
            return
        }
        if (!force && applyCachedPlaylistView(requested)) {
            syncNowPlayingInPlaylist({ force: playlistFocusNowPlaying, ensureLoaded: true })
            playlistFocusNowPlaying = false
            return
        }
        tracksLoading = true
        playlistTracksLoadingMore = false
        playlistTrackOffset = 0
        playlistTrackTotal = 0
        if (playlistFocusNowPlaying)
            playlistRestoreY = -1
        else
            restorePlaylistScroll(requested)
        runPlaylistQuery([
            "playlist", requested, "--json",
            "--offset", "0",
            "--limit", String(playlistPageSize)
        ], function(text) {
            if (selectedPlaylist !== requested)
                return
            tracksLoading = false
            var restoreY = playlistRestoreY
            playlistRestoreY = -1
            applyPlaylistTracksPage(text, false, restoreY)
            syncNowPlayingInPlaylist({ force: playlistFocusNowPlaying, ensureLoaded: true })
            playlistFocusNowPlaying = false
            mergePlayerFromTrackList()
        })
    }

    function applyPlaylistTracksPage(text, append, restoreY) {
        try {
            var parsed = JSON.parse(String(text || "{}"))
            if (parsed && Array.isArray(parsed.items)) {
                var items = parsed.items
                if (append) {
                    tracks = tracks.concat(items)
                } else {
                    assignTracks(items, { scrollY: restoreY })
                }
                playlistTrackTotal = Number(parsed.total) || tracks.length
                playlistTrackOffset = tracks.length
                warmPlaylistPage(items)
                scheduleVisibleArtWarm()
                return
            }
            if (Array.isArray(parsed)) {
                if (append) {
                    tracks = tracks.concat(parsed)
                } else {
                    assignTracks(parsed, { scrollY: restoreY })
                }
                playlistTrackTotal = tracks.length
                playlistTrackOffset = tracks.length
                warmPlaylistPage(parsed)
                scheduleVisibleArtWarm()
                return
            }
        } catch (e) {
        }
        if (!append)
            assignTracks([], { scrollY: restoreY })
        playlistTrackTotal = tracks.length
        playlistTrackOffset = tracks.length
    }

    function loadMoreCurrentPlaylistTracks(onDone) {
        if (playlistTrackOffset >= playlistTrackTotal) {
            if (onDone)
                onDone()
            return
        }
        var list = playlistTrackList
        var anchorY = list ? list.contentY : -1
        var anchorIndex = list && list.count > 0
            ? Math.max(0, list.indexAt(0, list.contentY + 1))
            : -1
        var end = Math.min(playlistTrackOffset + playlistPageSize, playlistTrackTotal)
        var items = currentPlaylistTracks.slice(playlistTrackOffset, end)
        tracks = tracks.concat(items)
        playlistTrackOffset = tracks.length
        tracksRevision++
        warmPlaylistPage(items)
        scheduleVisibleArtWarm()
        restoreListViewport(list, anchorY, anchorIndex, onDone)
    }

    function loadMorePlaylistTracks(onDone) {
        if (playlistTracksLoadingMore || tracksLoading)
            return
        if (!selectedPlaylist || playlistTrackOffset >= playlistTrackTotal) {
            if (onDone)
                onDone()
            return
        }
        if (selectedPlaylist === currentPlaylistId) {
            playlistTracksLoadingMore = true
            loadMoreCurrentPlaylistTracks(function() {
                playlistTracksLoadingMore = false
                if (onDone)
                    onDone()
            })
            return
        }
        var requested = String(selectedPlaylist)
        var list = playlistTrackList
        var anchorY = list ? list.contentY : -1
        var anchorIndex = list && list.count > 0
            ? Math.max(0, list.indexAt(0, list.contentY + 1))
            : -1
        playlistTracksLoadingMore = true
        runPlaylistQuery([
            "playlist", requested, "--json",
            "--offset", String(playlistTrackOffset),
            "--limit", String(playlistPageSize)
        ], function(text) {
            if (selectedPlaylist !== requested) {
                playlistTracksLoadingMore = false
                if (onDone)
                    onDone()
                return
            }
            applyPlaylistTracksPage(text, true)
            playlistTracksLoadingMore = false
            restoreListViewport(list, anchorY, anchorIndex, onDone)
        })
    }

    function indexOfTrackInPlaylist(path) {
        path = String(path || "")
        if (!path)
            return -1
        for (var i = 0; i < tracks.length; i++) {
            if (String(tracks[i].path || "") === path)
                return i
        }
        return -1
    }

    function focusPlaylistTrackIndex(index, retries) {
        if (index < 0 || index >= tracks.length)
            return
        var entry = tracks[index]
        if (!entry || !entry.path)
            return
        selectedTrackIndex = index
        selectedTrackPath = String(entry.path)
        cacheSelectedTrack(entry)
        if (!playlistsPanelOpen || playlistPanelMode !== "tracks")
            return
        var list = playlistTrackList
        if (!list) {
            if ((retries || 0) < 8)
                Qt.callLater(function() { focusPlaylistTrackIndex(index, (retries || 0) + 1) })
            return
        }
        var focusIdx = index
        playlistScrollLocked = true
        Qt.callLater(function() {
            restoreListViewport(list, -1, focusIdx, function() {
                if (list && list.count > focusIdx)
                    list.positionViewAtIndex(focusIdx, ListView.Center)
                playlistScrollLocked = false
            })
        })
    }

    function ensureCurrentPlaylistWindow(idx) {
        if (idx < 0)
            return
        playlistTrackTotal = currentPlaylistTracks.length
        var end = Math.min(
            Math.ceil((idx + 1) / playlistPageSize) * playlistPageSize,
            currentPlaylistTracks.length
        )
        if (end <= tracks.length)
            return
        var items = currentPlaylistTracks.slice(tracks.length, end)
        tracks = tracks.concat(items)
        playlistTrackOffset = tracks.length
        tracksRevision++
        warmPlaylistPage(items)
        scheduleVisibleArtWarm()
    }

    function loadPlaylistUntilTrack(path) {
        path = String(path || "")
        if (!path || !selectedPlaylist)
            return
        if (selectedPlaylist === currentPlaylistId) {
            var idx = indexInCurrentPlaylist(path)
            if (idx < 0)
                return
            ensureCurrentPlaylistWindow(idx)
            focusPlaylistTrackIndex(idx)
            return
        }
        var found = indexOfTrackInPlaylist(path)
        if (found >= 0) {
            focusPlaylistTrackIndex(found)
            return
        }
        if (tracksLoading || playlistTracksLoadingMore) {
            Qt.callLater(function() { loadPlaylistUntilTrack(path) })
            return
        }
        if (playlistTrackOffset >= playlistTrackTotal)
            return
        loadMorePlaylistTracks(function() { loadPlaylistUntilTrack(path) })
    }

    function syncNowPlayingInPlaylist(opts) {
        opts = opts || {}
        if (!player.path || !tracks.length)
            return
        if (transportApplyPending && !opts.force)
            return
        var path = String(player.path)
        if (!opts.force && selectedTrackPath && selectedTrackPath !== path)
            return
        var idx = indexOfTrackInPlaylist(path)
        if (idx >= 0) {
            focusPlaylistTrackIndex(idx)
            return
        }
        if (opts.ensureLoaded)
            loadPlaylistUntilTrack(path)
    }

    function syncSelectedTrackIndex() {
        syncNowPlayingInPlaylist({ force: false, ensureLoaded: false })
    }

    function mergeMonitorStatus() {
        var snap = playerMonitor && playerMonitor.player
        if (!snap || typeof snap !== "object")
            return
        if (snap.queue_revision !== undefined)
            queueRevision = Number(snap.queue_revision) || 0
        var snapPath = String(snap.path || "")
        var prevPath = String(player.path || "")
        if (!snapPath && prevPath && !artPreviewActive)
            return
        if (transportTargetPath && snapPath === transportTargetPath)
            transportTargetPath = ""
        if (snapPath && snapPath !== prevPath) {
            transportPreviewPath = ""
            transportApplyPending = false
            trackTransitionPending = false
            waveformLoading = false
            var idx = indexInCurrentPlaylist(snapPath)
            if (idx >= 0) {
                selectedTrackIndex = idx
                selectedTrackPath = snapPath
            }
        }
        var samePath = snapPath !== "" && snapPath === prevPath
        if (snapPath && snapPath !== prevPath && !trackTransitionPending) {
            scannerReverse = trackStepReverse(prevPath, snapPath)
        }
        if (samePath) {
            var next = Object.assign({}, player)
            if (snap.state !== undefined) {
                if (playbackStatePending && playbackStateTarget) {
                    var reportedState = String(snap.state || "")
                    if (reportedState === playbackStateTarget) {
                        playbackStatePending = false
                        playbackStateTarget = ""
                        playbackSettleTimer.stop()
                        next.state = reportedState
                    } else {
                        next.state = playbackStateTarget
                    }
                } else {
                    next.state = snap.state
                }
            }
            if (seekApplyPending) {
                var reportedPos = Number(snap.position !== undefined ? snap.position : seekApplyTarget)
                if (Math.abs(reportedPos - seekApplyTarget) <= 2) {
                    seekApplyPending = false
                    seekSettleTimer.stop()
                    next.position = reportedPos
                    next.position_label = snap.position_label || formatPlaybackTime(reportedPos)
                } else {
                    next.position = seekApplyTarget
                    next.position_label = formatPlaybackTime(seekApplyTarget)
                }
            } else if (snap.position !== undefined) {
                next.position = Number(snap.position) || 0
                next.position_label = snap.position_label || formatPlaybackTime(next.position)
            }
            if (snap.duration !== undefined) {
                var snapDur = Number(snap.duration) || 0
                if (snapDur > 0 || !(Number(player.duration) > 0)) {
                    next.duration = snapDur
                    next.duration_label = snap.duration_label || formatPlaybackTime(snapDur)
                }
            }
            if (volumeApplyPending) {
                var reportedVol = Number(snap.volume !== undefined ? snap.volume : volumeApplyTarget)
                if (Math.abs(reportedVol - volumeApplyTarget) <= 1) {
                    volumeApplyPending = false
                    volumeSettleTimer.stop()
                    next.volume = reportedVol
                } else {
                    next.volume = volumeApplyTarget
                }
            } else if (snap.volume !== undefined) {
                var vol = Number(snap.volume)
                if (!isNaN(vol))
                    next.volume = vol
            }
            if (snap.shuffle !== undefined)
                next.shuffle = !!snap.shuffle
            if (snap.liked !== undefined)
                next.liked = !!snap.liked
            var snapArt = String(snap.art || "")
            if (snapArt.charAt(0) === "/")
                next.art = snapArt
            player = next
            if (!resolvedArt && prevPath) {
                if (snapArt.charAt(0) === "/")
                    applyPlayerArt(prevPath, snapArt)
                else
                    applyDisplayArtForPath(prevPath)
            }
            syncSideArtImageSource()
            maybeRefreshQueueUpNext()
            return
        }
        if (transportTargetPath && snapPath && snapPath !== transportTargetPath)
            return
        applyStatus(JSON.stringify(snap))
    }

    function refreshStatus() {
        if (!active || playerMonitor)
            return
        pollStatus(applyStatus)
    }

    function pollStatus(onDone) {
        if (statusQueryProc.running)
            return false
        statusQueryProc.command = playerCmd(["status", "--json"])
        statusQueryProc._onDone = onDone || applyStatus
        statusQueryProc.running = true
        return true
    }

    function trackForPath(path) {
        var p = String(path || "")
        for (var i = 0; i < tracks.length; i++) {
            if (tracks[i].path === p)
                return tracks[i]
        }
        for (var c = 0; c < currentPlaylistTracks.length; c++) {
            if (currentPlaylistTracks[c].path === p)
                return currentPlaylistTracks[c]
        }
        for (var key in filetreeChildren) {
            var kids = filetreeChildren[key]
            for (var k = 0; k < kids.length; k++) {
                if (kids[k].type === "track" && String(kids[k].path) === p)
                    return kids[k]
            }
        }
        for (var r = 0; r < filetreeRows.length; r++) {
            var row = filetreeRows[r]
            if (row.type === "track" && String(row.path) === p)
                return row.track || row
        }
        if (String(selectedTrackCache.path || "") === p)
            return selectedTrackCache
        return null
    }

    function playerFieldsFromTrack(t) {
        if (!t)
            return ({})
        var next = {}
        if (t.title)
            next.title = t.title
        if (t.artist)
            next.artist = t.artist
        if (t.genre)
            next.genre = t.genre
        if (t.album)
            next.album = t.album
        if (t.year)
            next.year = t.year
        if (t.label)
            next.label = t.label
        if (t.art)
            next.art = t.art
        if (t.liked !== undefined)
            next.liked = !!t.liked
        if (t.waveform)
            next.waveform = t.waveform
        var dur = Number(t.duration)
        if (isFinite(dur) && dur > 0) {
            next.duration = dur
            next.duration_label = formatPlaybackTime(dur)
        }
        return next
    }

    function rememberWaveformPath(path, wf) {
        var p = String(path || "")
        var file = String(wf || "")
        if (!p || !file)
            return
        if (waveformPathByTrack[p] === file)
            return
        var next = Object.assign({}, waveformPathByTrack)
        next[p] = file
        waveformPathByTrack = next
    }

    function rememberWaveformSamples(path, samples) {
        var p = String(path || "")
        if (!p)
            return
        var next = Object.assign({}, waveformCache)
        next[p] = samples || []
        var keys = Object.keys(next)
        while (keys.length > 24) {
            delete next[keys[0]]
            keys = Object.keys(next)
        }
        waveformCache = next
    }

    function parseWaveformText(text) {
        try {
            var wf = JSON.parse(String(text || "{}"))
            var raw = wf.data || []
            var ch = wf.channels || 1
            var out = []
            if (ch >= 2) {
                for (var i = 0; i < raw.length; i += 2)
                    out.push(Math.max(Number(raw[i]) || 0, Number(raw[i + 1]) || 0))
            } else {
                for (var j = 0; j < raw.length; j++)
                    out.push(Number(raw[j]) || 0)
            }
            return out
        } catch (e) {
            return []
        }
    }

    function cacheWaveformText(path, text) {
        rememberWaveformSamples(path, parseWaveformText(text))
    }

    function beginWaveformLoading(path) {
        var p = String(path || "")
        if (!p || String(player.path || "") !== p)
            return
        var cached = waveformCache[p]
        waveformLoading = !(cached && cached.length)
    }

    function finishWaveformLoading(path, hasSamples) {
        var p = String(path || "")
        if (!p || String(player.path || "") !== p)
            return
        if (hasSamples || waveformSamples.length > 0) {
            if (!trackTransitionPending)
                waveformLoading = false
            tryRevealNowPlaying()
            return
        }
        if (!String(player.waveform || "") && !trackTransitionPending)
            waveformLoading = false
        tryRevealNowPlaying()
    }

    function onWarmWaveformResult(path, waveform) {
        var p = String(path || "")
        if (String(player.path || "") !== p)
            return
        var wf = String(waveform || "")
        if (wf.charAt(0) === "/")
            return
        finishWaveformLoading(p, false)
        scheduleWaveformRecheck(p)
    }

    function scheduleWaveformRecheck(path) {
        path = String(path || "")
        if (!path)
            return
        waveformRecheckPath = path
        waveformRecheckTimer.restart()
    }

    function recheckWaveformForPath() {
        var p = String(waveformRecheckPath || "")
        if (!p || String(player.path || "") !== p)
            return
        function applyMeta(text) {
            try {
                var data = JSON.parse(String(text || "{}"))
                var wf = data && data.waveform ? String(data.waveform) : ""
                if (wf.charAt(0) !== "/")
                    return
                patchWaveformInLists(p, wf)
                rememberWaveformPath(p, wf)
                if (String(player.path || "") === p) {
                    player = Object.assign({}, player, { waveform: wf })
                    applyCachedWaveform(p)
                }
            } catch (e) {
            }
        }
        function runCLI() {
            runMusic(["meta", p, "--json"], applyMeta, cmdProc)
        }
        if (!runDaemonJSON("library.meta", { path: p }, applyMeta, runCLI))
            runCLI()
    }

    function applyCachedWaveform(path) {
        var p = String(path || "")
        if (p && waveformCache[p]) {
            waveformSamples = waveformCache[p]
            if (String(player.path || "") === p) {
                if (!trackTransitionPending) {
                    waveformLoading = false
                    tryRevealNowPlaying()
                }
            }
            return true
        }
        waveformSamples = []
        return false
    }

    function indexInCurrentPlaylist(path) {
        var p = String(path || "")
        if (!p)
            return -1
        for (var i = 0; i < currentPlaylistTracks.length; i++) {
            if (currentPlaylistTracks[i] && String(currentPlaylistTracks[i].path || "") === p)
                return i
        }
        return -1
    }

    function trackStepReverse(fromPath, toPath) {
        var prev = String(fromPath || "")
        var next = String(toPath || "")
        if (!prev || !next || prev === next)
            return false
        var prevIdx = indexInCurrentPlaylist(prev)
        var nextIdx = indexInCurrentPlaylist(next)
        var n = currentPlaylistTracks.length
        if (prevIdx < 0 || nextIdx < 0 || n < 2)
            return false
        var forward = (nextIdx - prevIdx + n) % n
        var backward = (prevIdx - nextIdx + n) % n
        return backward < forward
    }

    function prefetchNeighbors(path) {
        var idx = indexInCurrentPlaylist(path)
        var n = currentPlaylistTracks.length
        if (idx < 0 || n < 2) {
            prefetchArtSources = []
            neighborWaveformJobs = []
            return
        }
        var neighbors = [
            currentPlaylistTracks[(idx - 1 + n) % n],
            currentPlaylistTracks[(idx + 1) % n]
        ]
        var arts = []
        var jobs = []
        for (var i = 0; i < neighbors.length; i++) {
            var t = neighbors[i]
            if (!t || !t.path)
                continue
            if (t.art)
                arts.push(artUrl(t.art))
            var wf = t.waveform || waveformPathByTrack[t.path] || ""
            if (wf && !waveformCache[t.path])
                jobs.push({ path: t.path, file: wf })
        }
        prefetchArtSources = arts
        neighborWaveformJobs = jobs
        neighborWaveformJobIndex = 0
        startNeighborWaveformJob()
    }

    function startNeighborWaveformJob() {
        if (neighborWaveformJobIndex >= neighborWaveformJobs.length) {
            neighborWaveformFile.trackPath = ""
            neighborWaveformFile.path = ""
            return
        }
        var job = neighborWaveformJobs[neighborWaveformJobIndex]
        if (!job || !job.file) {
            neighborWaveformJobIndex++
            startNeighborWaveformJob()
            return
        }
        neighborWaveformFile.trackPath = job.path
        neighborWaveformFile.path = job.file
    }

    function advanceNeighborWaveform() {
        if (neighborWaveformJobIndex >= neighborWaveformJobs.length)
            return
        neighborWaveformJobIndex++
        startNeighborWaveformJob()
    }

    function waveformCachePath(trackPath) {
        return ""
    }

    function ensureWaveformForPath(path) {
        var p = String(path || "")
        if (!p)
            return
        if (applyCachedWaveform(p))
            return
        var t = trackForPath(p)
        var wf = String((t && t.waveform) || waveformPathByTrack[p] || "")
        if (wf.charAt(0) === "/") {
            patchWaveformInLists(p, wf)
            return
        }
        queueWaveformWarm(p)
    }

    function primePlayerForPath(path, playing) {
        var p = String(path || "")
        if (!p)
            return
        var preview = playing !== false
        var t = trackForPath(p)
        var wf = (t && t.waveform) || waveformPathByTrack[p] || ""
        var next = Object.assign({}, player, playerFieldsFromTrack(t), {
            path: p,
            state: preview ? "playing" : "stopped",
            position: 0,
            position_label: formatPlaybackTime(0),
            art: (t && t.art) || "",
            waveform: wf
        })
        player = next
        if (wf)
            rememberWaveformPath(p, wf)
        if (!applyCachedWaveform(p))
            beginWaveformLoading(p)
        if (!waveformCache[p] || !waveformCache[p].length)
            ensureWaveformForPath(p)
        prefetchNeighbors(p)
    }

    function beginTrackTransition(path, reverse, options) {
        var p = String(path || "")
        if (!p)
            return
        var opts = options || {}
        scannerReverse = !!reverse
        trackTransitionPending = true
        transportPreviewPath = p
        scannerHoldPending = true
        scannerHoldTimer.restart()
        if (opts.awaitTransport === true) {
            transportApplyPending = true
            transportSettleTimer.restart()
        }
        trackRevealTimer.restart()
        waveformSamples = []
        waveformLoading = true
    }

    function tryRevealNowPlaying() {
        if (!trackTransitionPending)
            return
        if (scannerHoldPending)
            return
        var path = String(player.path || "")
        var target = String(transportPreviewPath || "")
        if (!path || !target || path !== target)
            return
        if (transportApplyPending)
            return
        trackTransitionPending = false
        transportPreviewPath = ""
        trackRevealTimer.stop()
        waveformLoading = false
    }

    function forceRevealNowPlaying() {
        if (!trackTransitionPending)
            return
        scannerHoldPending = false
        scannerHoldTimer.stop()
        trackTransitionPending = false
        transportPreviewPath = ""
        trackRevealTimer.stop()
        waveformLoading = false
    }

    function togglePlayback() {
        // #region agent log
        agentDebug("H1", "DashboardModule.qml:togglePlayback", "toggle requested", {
            guard: panelTransitionGuard,
            playing: playerPlaying,
            path: String(transportSnap.path || "").split("/").pop()
        })
        // #endregion
        traceIPC("togglePlayback guard=" + panelTransitionGuard + " playing=" + playerPlaying)
        if (panelTransitionGuard)
            return
        if (playerMonitor && !playerMonitor.ipcReady) {
            if (typeof playerMonitor.ensurePlayer === "function")
                playerMonitor.ensurePlayer()
        }
        var target = playerPlaying ? "paused" : "playing"
        if (!playerMonitor || playerMonitor.ipcSynced)
            previewPlaybackState(target)
        queuePlaybackState(target)
        sendPlaybackToggle()
    }

    function previewPlaybackState(state) {
        var next = Object.assign({}, player)
        next.state = state
        player = next
    }

    function queuePlaybackState(target) {
        playbackStateTarget = target
        playbackStatePending = true
        playbackSettleTimer.restart()
    }

    function sendPlaybackToggle() {
        if (!playbackStatePending)
            return
        playbackToggleTimer.restart()
    }

    function flushPlaybackToggle() {
        if (!playbackStatePending)
            return
        if (panelTransitionGuard) {
            // #region agent log
            agentDebug("H5", "DashboardModule.qml:flushPlaybackToggle", "deferred by guard", {})
            // #endregion
            playbackToggleTimer.restart()
            return
        }
        // #region agent log
        agentDebug("H1", "DashboardModule.qml:flushPlaybackToggle", "sending toggle", {})
        // #endregion
        runPlayer(["toggle"], null, cmdProc)
    }

    function finishPlaybackSettle() {
        playbackStatePending = false
        playbackStateTarget = ""
    }

    function mergePlayerFromTrackList() {
        if (!player.path)
            return
        var t = trackForPath(player.path)
        if (!t)
            return
        var fields = playerFieldsFromTrack(t)
        delete fields.liked
        var playerArt = String(player.art || "")
        if (playerArt && playerArt.charAt(0) === "/")
            delete fields.art
        var next = Object.assign({}, player, fields)
        if (favoriteApplyPending && favoriteApplyPath === String(player.path || "")) {
            next.liked = favoriteApplyLiked
        } else {
            var likedPath = String(player.path || "")
            var overlay = likedByPath[likedPath]
            if (overlay === true || overlay === false)
                next.liked = overlay
            else if (t.liked === true)
                next.liked = true
        }
        player = next
    }

    function applyStatus(text, forceSync) {
        var prevPath = String(player.path || "")
        var parsed
        try {
            parsed = JSON.parse(String(text || "{}"))
        } catch (e) {
            parsed = {}
        }
        if (!String(parsed.path || "") && prevPath)
            return
        var incomingPath = String(parsed.path || "")
        if (!forceSync && transportTargetPath && incomingPath && incomingPath !== transportTargetPath)
            return
        if (transportTargetPath && incomingPath === transportTargetPath)
            transportTargetPath = ""
        if (incomingPath && incomingPath !== prevPath) {
            transportPreviewPath = ""
            transportApplyPending = false
            trackTransitionPending = false
            waveformLoading = false
            var idx = indexInCurrentPlaylist(incomingPath)
            if (idx >= 0) {
                selectedTrackIndex = idx
                selectedTrackPath = incomingPath
            }
            scannerReverse = trackStepReverse(prevPath, incomingPath)
        }
        if (volumeApplyPending) {
            var reported = Number(parsed.volume !== undefined ? parsed.volume : volumeApplyTarget)
            if (Math.abs(reported - volumeApplyTarget) <= 1) {
                volumeApplyPending = false
                volumeSettleTimer.stop()
            } else {
                parsed = Object.assign({}, parsed, { volume: volumeApplyTarget })
            }
        }
        if (seekApplyPending) {
            var reportedPos = Number(parsed.position !== undefined ? parsed.position : seekApplyTarget)
            if (Math.abs(reportedPos - seekApplyTarget) <= 2) {
                seekApplyPending = false
                seekSettleTimer.stop()
            } else {
                parsed = Object.assign({}, parsed, {
                    position: seekApplyTarget,
                    position_label: root.formatPlaybackTime(seekApplyTarget)
                })
            }
        }
        if (playbackStatePending && playbackStateTarget) {
            var reportedState = String(parsed.state || "")
            if (reportedState === playbackStateTarget) {
                playbackStatePending = false
                playbackStateTarget = ""
                playbackSettleTimer.stop()
            } else {
                parsed = Object.assign({}, parsed, { state: playbackStateTarget })
            }
        }
        if (favoriteApplyPending && favoriteApplyPath) {
            var likedPath = String(parsed.path || player.path || "")
            if (likedPath === favoriteApplyPath) {
                if (parsed.liked !== undefined && !!parsed.liked === !!favoriteApplyLiked) {
                    favoriteApplyPending = false
                    favoriteApplyPath = ""
                    favoriteSettleTimer.stop()
                    tracksRevision++
                } else {
                    parsed = Object.assign({}, parsed, { liked: favoriteApplyLiked })
                }
            }
        }
        if (parsed.duration !== undefined) {
            var parsedDur = Number(parsed.duration) || 0
            if (parsedDur <= 0 && Number(player.duration) > 0)
                parsed = Object.assign({}, parsed, {
                    duration: player.duration,
                    duration_label: player.duration_label
                })
        }
        if (parsed.waveform)
            rememberWaveformPath(parsed.path || player.path, parsed.waveform)
        var newPathBefore = String(parsed.path || player.path || "")
        if (newPathBefore !== prevPath && prevPath && resolvedArtPath === prevPath) {
            resolvedArtPath = ""
            resolvedArt = ""
        }
        player = Object.assign({}, player, parsed)
        var newPath = String(player.path || "")
        if (newPath && (newPath !== prevPath || resolvedArtPath !== newPath || !resolvedArt)) {
            var knownArt = String(player.art || "")
            if (knownArt.charAt(0) === "/")
                applyPlayerArt(newPath, knownArt)
            else
                applyDisplayArtForPath(newPath)
        }
        if (newPath !== prevPath) {
            if (!applyCachedWaveform(newPath))
                beginWaveformLoading(newPath)
            ensureWaveformForPath(newPath)
            prefetchNeighbors(newPath)
            if (trackTransitionPending && newPath === String(transportPreviewPath || "")) {
                if (transportApplyPending) {
                    transportApplyPending = false
                    transportApplyTarget = null
                    transportSettleTimer.stop()
                }
                tryRevealNowPlaying()
            }
        }
        checkAutoExtendQueue()
        mergePlayerFromTrackList()
        syncSideArtImageSource()
        kickSideArtUntilLoaded()
        maybeRefreshQueueUpNext()
    }

    function applyWaveform(text) {
        var samples = parseWaveformText(text)
        if (samples.length === 0 && player.path && waveformCache[player.path] && waveformCache[player.path].length)
            samples = waveformCache[player.path]
        Qt.callLater(function() {
            var p = String(player.path || "")
            waveformSamples = samples
            if (p && samples.length)
                rememberWaveformSamples(p, samples)
            if (p)
                finishWaveformLoading(p, samples.length > 0)
        })
    }

    function currentWaveformFilePath() {
        return (player.path && player.waveform && active) ? String(player.waveform) : ""
    }

    function formatPlaybackTime(sec) {
        var s = Math.max(0, Math.floor(Number(sec) || 0))
        var h = Math.floor(s / 3600)
        var m = Math.floor((s % 3600) / 60)
        var r = s % 60
        var mm = h > 0 ? (m < 10 ? "0" : "") + m : String(m)
        var rr = (r < 10 ? "0" : "") + r
        return h > 0 ? (h + ":" + mm + ":" + rr) : (mm + ":" + rr)
    }

    function scrubSecondsFromX(x, width) {
        if (!player.duration || width <= 0)
            return -1
        var ratio = Math.max(0, Math.min(1, x / width))
        return ratio * player.duration
    }

    function scrubTo(seconds) {
        var dur = Number(player.duration) || 0
        if (dur <= 0)
            return
        var sec = Math.max(0, Math.min(dur, Number(seconds) || 0))
        seekApplyTarget = sec
        seekApplyPending = true
        seekSettleTimer.stop()
        var next = Object.assign({}, player)
        next.position = sec
        next.position_label = formatPlaybackTime(sec)
        player = next
        if (waveformViz)
            waveformViz.scheduleVizUpdate()
    }

    function previewSeekFromX(x, width) {
        var sec = scrubSecondsFromX(x, width)
        if (sec >= 0)
            scrubTo(sec)
    }

    function queueSeek(seconds) {
        var dur = Number(player.duration) || 0
        if (dur <= 0)
            return
        seekApplyTarget = Math.max(0, Math.min(dur, Number(seconds) || 0))
        seekApplyPending = true
        seekSettleTimer.stop()
        seekApplyTimer.restart()
    }

    function flushSeekApply() {
        if (!seekApplyPending)
            return
        if (seekProc.running) {
            seekApplyTimer.restart()
            return
        }
        runPlayer(["seek", String(seekApplyTarget)], null, seekProc)
        seekSettleTimer.restart()
    }

    function finishSeekSettle() {
        if (!seekApplyPending)
            return
        seekApplyPending = false
    }

    function commitSeekFromX(x, width) {
        var sec = scrubSecondsFromX(x, width)
        if (sec < 0)
            return
        scrubTo(sec)
        queueSeek(sec)
    }

    function previewVolume(percent) {
        var v = Math.max(0, Math.min(100, Math.round(percent)))
        var next = Object.assign({}, player)
        next.volume = v
        player = next
    }

    function queueVolumeApply(target) {
        volumeApplyTarget = Math.max(0, Math.min(100, Math.round(target)))
        volumeApplyPending = true
        volumeSettleTimer.stop()
        volumeApplyTimer.restart()
    }

    function flushVolumeApply() {
        if (!volumeApplyPending)
            return
        if (volumeProc.running) {
            volumeApplyTimer.restart()
            return
        }
        runPlayer(["volume", "set", String(volumeApplyTarget)], null, volumeProc)
        volumeSettleTimer.restart()
    }

    function finishVolumeSettle() {
        if (!volumeApplyPending)
            return
        volumeApplyPending = false
    }

    function nudgeSystemVolume(delta) {
        if (!delta)
            return
        Quickshell.execDetached([
            "omarchy-audio-output-volume",
            delta > 0 ? "raise" : "lower"
        ])
    }

    function adjustVolume(delta) {
        if (!delta) return
        var cur = Math.round(Number(player.volume !== undefined ? player.volume : 100))
        var step = Number(delta)
        if (step > 0 && cur >= 100) {
            nudgeSystemVolume(step)
            return
        }
        var next = Math.max(0, Math.min(100, cur + step))
        previewVolume(next)
        queueVolumeApply(next)
    }

    function setVolume(percent) {
        var v = Math.max(0, Math.min(100, Math.round(percent)))
        previewVolume(v)
        queueVolumeApply(v)
    }

    function toggleVolumeMute() {
        var cur = Number(player.volume !== undefined ? player.volume : 100)
        setVolume(cur <= 0 ? 100 : 0)
    }

    function volumeIcon(level) {
        var v = Number(level || 0)
        if (v <= 0) return "󰝟"
        if (v < 34) return "󰕿"
        if (v < 67) return "󰖀"
        return "󰕾"
    }

    function startLoad(path, useFolder) {
        if (!path)
            return
        var args = ["load", path]
        if (useFolder)
            args.push("--folder")
        loadProc.command = playerCmd(args)
        loadProc.running = true
    }

    function playPath(path, useFolder) {
        if (!path)
            return
        selectedTrackPath = String(path)
        primePlayerForPath(path)
        var trackPath = String(path)
        var folderQueue = !!useFolder
        if (loadProc.running) {
            loadProc.pendingPath = trackPath
            loadProc.pendingFolder = folderQueue
            return
        }
        startLoad(trackPath, folderQueue)
    }

    function pathsFromTracks(trackList) {
        var paths = []
        for (var i = 0; i < trackList.length; i++) {
            if (trackList[i] && trackList[i].path)
                paths.push(trackList[i].path)
        }
        return paths
    }

    function appendTrackToCurrent(track) {
        var path = ""
        var entry = track
        if (typeof track === "string") {
            path = String(track || "").trim()
            entry = { path: path }
        } else if (track) {
            if (track.track && typeof track.track === "object" && track.track.path)
                entry = track.track
            else if (track.path)
                entry = track
            else
                entry = track
            path = String(entry.path || "").trim()
        }
        if (!path)
            return
        var before = currentPlaylistTracks.length
        appendTracksToCurrent([entry])
        if (currentPlaylistTracks.length === before) {
            notify("already in current", 2000)
            return
        }
        selectedPlaylist = currentPlaylistId
        commitCurrentPlaylist()
        notify("added to current", 2000)
    }

    function appendTracksToCurrent(slice) {
        var seen = {}
        var i
        for (i = 0; i < currentPlaylistTracks.length; i++) {
            if (currentPlaylistTracks[i].path)
                seen[currentPlaylistTracks[i].path] = true
        }
        var merged = currentPlaylistTracks.slice()
        for (i = 0; i < slice.length; i++) {
            if (!slice[i] || !slice[i].path || seen[slice[i].path])
                continue
            seen[slice[i].path] = true
            merged.push(slice[i])
        }
        currentPlaylistTracks = merged
        currentPlaylistActive = true
    }

    function appendPathsToQueue(paths) {
        if (!paths || !paths.length)
            return
        function runCLI() {
            if (!runMusic(["queue", "append"].concat(paths), function() {
                mergeMonitorStatus()
            }, queuePlayProc))
                runMusic(["queue", "append"].concat(paths), function() {
                    mergeMonitorStatus()
                }, cmdProc)
        }
        daemonVoid("queue.append", { paths: paths }, function(ok) {
            if (ok)
                mergeMonitorStatus()
            else
                runCLI()
        }, runCLI)
    }

    function playQueueAt(startPath, pathList) {
        if (!startPath || !pathList || !pathList.length)
            return
        traceIPC("playQueueAt start=" + String(startPath).split("/").pop()
            + " count=" + pathList.length + " guard=" + panelTransitionGuard)
        // #region agent log
        agentDebug("H2", "DashboardModule.qml:playQueueAt", "queue replace requested", {
            start: String(startPath).split("/").pop(),
            count: pathList.length,
            guard: panelTransitionGuard
        })
        // #endregion
        if (panelTransitionGuard)
            return
        startPath = String(startPath)
        selectedTrackPath = startPath
        if (queueAlreadyPlaying(startPath, pathList)) {
            traceIPC("playQueueAt skip already playing")
            transportTargetPath = ""
            syncPlayerFromMonitor()
            return
        }
        if (!playerMonitor || !playerMonitor.ipcReady) {
            notify("player not connected", 2000)
            return
        }
        transportTargetPath = startPath
        if (resolvedArtPath && resolvedArtPath !== startPath) {
            resolvedArtPath = ""
            resolvedArt = ""
        }
        daemonVoid("queue.replace", {
            paths: pathList,
            start_path: startPath
        }, function(ok) {
            if (!ok) {
                transportTargetPath = ""
                notify("could not start playback", 2500)
                return
            }
            Qt.callLater(syncPlayerFromMonitor)
        })
    }

    function syncCurrentQueueAt(startPath, opts) {
        opts = opts || {}
        startPath = String(startPath || "")
        var paths = pathsFromTracks(currentPlaylistTracks)
        if (!paths.length)
            return false
        if (!startPath || paths.indexOf(startPath) < 0)
            startPath = paths[0]
        selectedTrackPath = startPath
        if (!opts.skipSave)
            persistCurrentPlaylist()
        playQueueAt(startPath, paths)
        return true
    }

    function tryPlayPathInQueue(path, onDone) {
        path = String(path || "")
        if (!path) {
            if (onDone)
                onDone(false)
            return false
        }
        daemonVoid("queue.play_path", { path: path }, function(ok) {
            if (ok)
                mergeMonitorStatus()
            if (onDone)
                onDone(ok)
        }, function() {
            if (!runMusic(["queue", "play", path, path], function() {
                mergeMonitorStatus()
                if (onDone)
                    onDone(true)
            }, queuePlayProc)) {
                if (onDone)
                    onDone(false)
            }
        })
        return true
    }

    function jumpCurrentAtNow(index) {
        if (index < 0 || index >= currentPlaylistTracks.length)
            return
        var startPath = currentPlaylistTracks[index] ? String(currentPlaylistTracks[index].path || "") : ""
        if (!startPath)
            return
        selectedTrackIndex = index
        selectedTrackPath = startPath
        playQueueAt(startPath, pathsFromTracks(currentPlaylistTracks))
    }

    function playFromPlaylistAtNow(index) {
        if (index < 0 || index >= tracks.length)
            return
        var startPath = tracks[index].path
        if (!startPath)
            return
        var slice = tracks.slice(index)
        appendTracksToCurrent(slice)
        var paths = pathsFromTracks(currentPlaylistTracks)
        selectedTrackIndex = index
        selectedTrackPath = String(startPath)
        syncCurrentQueueAt(startPath)
        commitCurrentPlaylist()
    }

    function resolveCurrentTrackIndex() {
        var path = String(player.path || "")
        var i
        if (path) {
            for (i = 0; i < tracks.length; i++) {
                if (tracks[i].path === path)
                    return i
            }
            for (i = 0; i < currentPlaylistTracks.length; i++) {
                if (currentPlaylistTracks[i].path === path)
                    return i
            }
        }
        if (selectedTrackIndex >= 0)
            return selectedTrackIndex
        return 0
    }

    function previewTrackIndex(index) {
        var path = ""
        if (index >= 0 && index < tracks.length && tracks[index] && tracks[index].path)
            path = tracks[index].path
        else if (index >= 0 && index < currentPlaylistTracks.length && currentPlaylistTracks[index])
            path = currentPlaylistTracks[index].path
        if (!path)
            return false
        selectedTrackIndex = index
        selectedTrackPath = String(path)
        beginTrackTransition(path)
        return true
    }

    function queueCurrentJump(index) {
        transportApplyTarget = { kind: "current", index: index }
        transportApplyPending = true
        transportSettleTimer.stop()
        transportApplyTimer.restart()
    }

    function queueTransportAction(target) {
        transportApplyTarget = target
        transportApplyPending = true
        transportSettleTimer.stop()
        transportApplyTimer.restart()
    }

    function flushTransportApply() {
        if (!transportApplyPending || !transportApplyTarget)
            return
        if (panelTransitionGuard) {
            // #region agent log
            agentDebug("H5", "DashboardModule.qml:flushTransportApply", "deferred by guard", {
                kind: String(transportApplyTarget.kind || "")
            })
            // #endregion
            transportApplyTimer.restart()
            return
        }
        if (transportProc.running) {
            transportApplyTimer.restart()
            return
        }
        var t = transportApplyTarget
        // #region agent log
        agentDebug("H2", "DashboardModule.qml:flushTransportApply", "flush transport", {
            kind: String(t.kind || ""),
            index: t.index
        })
        // #endregion
        if (t.kind === "jump")
            jumpCurrentAtNow(t.index)
        else if (t.kind === "current")
            jumpCurrentAtNow(t.index)
        else if (t.kind === "playlist")
            playFromPlaylistAtNow(t.index)
        else if (t.kind === "daemon")
            runPlayer([t.forward ? "next" : "prev"], function() { root.refreshStatus() }, transportProc)
        transportSettleTimer.restart()
    }

    function finishTransportSettle() {
        if (!transportApplyPending)
            return
        transportApplyPending = false
        transportApplyTarget = null
        tryRevealNowPlaying()
    }

    function jumpCurrentAt(index) {
        var path = ""
        if (index >= 0 && index < tracks.length && tracks[index])
            path = String(tracks[index].path || "")
        else if (index >= 0 && index < currentPlaylistTracks.length && currentPlaylistTracks[index])
            path = String(currentPlaylistTracks[index].path || "")
        if (!path)
            return
        jumpCurrentAtPath(path)
    }

    function previewTrackPath(path) {
        path = String(path || "")
        if (!path)
            return false
        var queueIndex = indexInCurrentPlaylist(path)
        if (queueIndex >= 0)
            selectedTrackIndex = queueIndex
        selectedTrackPath = path
        beginTrackTransition(path)
        return true
    }

    function jumpCurrentAtPath(path) {
        path = String(path || "")
        if (!path)
            return
        var queueIndex = indexInCurrentPlaylist(path)
        if (queueIndex >= 0) {
            selectedTrackIndex = queueIndex
            selectedTrackPath = path
            playQueueAt(path, pathsFromTracks(currentPlaylistTracks))
            return
        }
        var paths = pathsFromTracks(tracks)
        if (paths.indexOf(path) < 0)
            return
        selectedTrackIndex = paths.indexOf(path)
        selectedTrackPath = path
        playQueueAt(path, paths)
    }

    function playFromPlaylistAt(index) {
        previewTrackIndex(index)
        queueTransportAction({ kind: "playlist", index: index })
    }

    function playPathFromList(path, useFolder) {
        if (!path)
            return
        playPath(path, useFolder)
    }

    function playTrackAt(index) {
        if (index < 0 || index >= tracks.length)
            return
        var path = String(tracks[index].path || "")
        if (!path)
            return
        selectedTrackPath = path
        selectedTrackIndex = index
        if (selectedPlaylist === currentPlaylistId) {
            syncCurrentQueueAt(path)
            return
        }
        previewTrackPath(path)
        playFromPlaylistAtNow(index)
    }

    function selectPlaylistTrack(index) {
        if (index < 0 || index >= tracks.length)
            return
        var path = String(tracks[index].path || "")
        if (path !== String(selectedTrackPath || ""))
            closeArtPicker()
        selectedTrackIndex = index
        selectedTrackPath = path
        cacheSelectedTrack(tracks[index])
        applyDisplayArtForPath(path, function() {
            if (resolvedArtPath === path && resolvedArt)
                cacheSelectedTrack(Object.assign({}, tracks[index], { art: resolvedArt }))
            refreshNowPlayingArtDisplay()
        })
    }

    function rememberLiked(path, liked) {
        path = String(path || "")
        if (!path)
            return
        var next = Object.assign({}, likedByPath)
        next[path] = !!liked
        likedByPath = next
        tracksRevision++
    }

    function trackIsLiked(path, fallback) {
        path = String(path || "")
        var overlay = likedByPath[path]
        if (overlay === true || overlay === false)
            return overlay
        if (fallback === true || fallback === false)
            return !!fallback
        if (path && path === String(player.path || ""))
            return !!player.liked
        return false
    }

    function toggleFavorite() {
        if (!player.path)
            return
        toggleTrackFavorite(player.path)
    }

    function runFavoriteQuery(args, onDone) {
        args = args || []
        var path = ""
        for (var i = 0; i < args.length; i++) {
            var a = String(args[i])
            if (a === "favorite" || a === "toggle" || a === "--json")
                continue
            if (!a.startsWith("-") && !path)
                path = a
        }
        if (path && runDaemonJSON("library.favorite.toggle", { path: path }, onDone, function() {
            if (favoriteProc.running) {
                favoriteProc._queuedArgs = args || []
                favoriteProc._queuedOnDone = onDone || null
                return
            }
            favoriteProc.command = playerCmd(args || [])
            favoriteProc._onDone = onDone || null
            favoriteProc.running = true
        }))
            return
        if (favoriteProc.running) {
            favoriteProc._queuedArgs = args || []
            favoriteProc._queuedOnDone = onDone || null
            return
        }
        favoriteProc.command = playerCmd(args || [])
        favoriteProc._onDone = onDone || null
        favoriteProc.running = true
    }

    function restoreListViewport(listView, contentY, anchorIndex, onSettled) {
        if (!listView) {
            if (onSettled)
                onSettled()
            return
        }
        var attempts = 0
        function step() {
            if (!listView) {
                if (onSettled)
                    onSettled()
                return
            }
            attempts++
            if (contentY >= 0) {
                listView.contentY = FiletreeLogic.clampContentY(
                    contentY, listView.contentHeight, listView.height)
            } else if (anchorIndex >= 0 && anchorIndex < listView.count) {
                listView.positionViewAtIndex(anchorIndex, ListView.Beginning)
            }
            if (listView.contentHeight > 0 || attempts >= 20) {
                if (onSettled)
                    onSettled()
                return
            }
            Qt.callLater(step)
        }
        step()
    }

    function assignTracks(nextTracks, opts) {
        opts = opts || {}
        var list = playlistTrackList
        var y = opts.scrollY
        if (y === undefined || y === null) {
            if (opts.preserveScroll === false)
                y = -1
            else if (list && playlistsPanelOpen && playlistPanelMode === "tracks" && list.height > 0)
                y = list.contentY
            else
                y = -1
        }
        playlistScrollLocked = true
        tracks = nextTracks
        tracksRevision++
        if (y >= 0 && list) {
            restoreListViewport(list, y, -1, function() {
                playlistScrollLocked = false
            })
        } else {
            playlistScrollLocked = false
        }
    }

    function toggleTrackFavorite(path, trackObj) {
        var trackPath = String(path || (trackObj && trackObj.path) || "")
        if (!trackPath)
            return
        var applyFavorite = function(text) {
            try {
                var result = JSON.parse(String(text || "{}"))
                if (result.liked === undefined)
                    return
                var liked = !!result.liked
                rememberLiked(trackPath, liked)
                if (String(player.path || "") === trackPath) {
                    var p = Object.assign({}, player)
                    p.liked = liked
                    player = p
                }
                var i, entry, nextTracks = [], nextCurrent = [], nextFilter = []
                for (i = 0; i < tracks.length; i++) {
                    entry = tracks[i]
                    if (entry && String(entry.path) === trackPath) {
                        entry.liked = liked
                        nextTracks.push(Object.assign({}, entry, { liked: liked }))
                    } else {
                        nextTracks.push(entry)
                    }
                }
                for (i = 0; i < currentPlaylistTracks.length; i++) {
                    entry = currentPlaylistTracks[i]
                    if (entry && String(entry.path) === trackPath) {
                        entry.liked = liked
                        nextCurrent.push(Object.assign({}, entry, { liked: liked }))
                    } else {
                        nextCurrent.push(entry)
                    }
                }
                for (i = 0; i < filterTracks.length; i++) {
                    entry = filterTracks[i]
                    if (entry && String(entry.path) === trackPath) {
                        entry.liked = liked
                        nextFilter.push(Object.assign({}, entry, { liked: liked }))
                    } else {
                        nextFilter.push(entry)
                    }
                }
                var playlistY = playlistTrackList ? playlistTrackList.contentY : -1
                assignTracks(nextTracks, { scrollY: playlistY })
                currentPlaylistTracks = nextCurrent
                filterTracks = nextFilter
                patchTrackLikedInFiletree(trackPath, liked)
                tracksRevision++
            } catch (e) {
            }
        }
        var optimisticLiked = true
        var j, t, found = false
        for (j = 0; j < tracks.length; j++) {
            if (tracks[j].path === trackPath) {
                optimisticLiked = !tracks[j].liked
                found = true
                break
            }
        }
        if (!found) {
            for (j = 0; j < currentPlaylistTracks.length; j++) {
                if (currentPlaylistTracks[j].path === trackPath) {
                    optimisticLiked = !currentPlaylistTracks[j].liked
                    found = true
                    break
                }
            }
        }
        if (!found) {
            for (j = 0; j < filterTracks.length; j++) {
                if (filterTracks[j].path === trackPath) {
                    optimisticLiked = !filterTracks[j].liked
                    found = true
                    break
                }
            }
        }
        if (!found) {
            var treeLiked = findTrackLikedInFiletree(trackPath)
            if (treeLiked !== undefined) {
                optimisticLiked = !treeLiked
                found = true
            }
        }
        if (!found) {
            if (trackObj && trackObj.liked !== undefined)
                optimisticLiked = !trackObj.liked
            else if (String(player.path || "") === trackPath)
                optimisticLiked = !player.liked
            else
                optimisticLiked = true
        }
        favoriteApplyPending = true
        favoriteApplyPath = trackPath
        favoriteApplyLiked = optimisticLiked
        favoriteSettleTimer.restart()
        applyFavorite(JSON.stringify({ liked: optimisticLiked }))
        if (trackObj) {
            try {
                trackObj.liked = optimisticLiked
            } catch (e) {
            }
        }
        runFavoriteQuery(["favorite", "toggle", trackPath, "--json"], function(text) {
            var confirmed = null
            try {
                confirmed = JSON.parse(String(text || "{}"))
            } catch (e) {
                confirmed = null
            }
            if (!confirmed || confirmed.liked === undefined)
                return
            if (favoriteApplyPath === trackPath && !!confirmed.liked !== !!favoriteApplyLiked)
                return
            applyFavorite(text)
        })
    }

    function previewCurrentIndex(index, reverse) {
        if (index < 0 || index >= currentPlaylistTracks.length)
            return false
        var path = currentPlaylistTracks[index] ? currentPlaylistTracks[index].path : ""
        if (!path)
            return false
        selectedTrackIndex = index
        selectedTrackPath = String(path)
        beginTrackTransition(path, reverse)
        return true
    }

    function skipTrack(forward) {
        var reverse = !forward
        scannerReverse = reverse
        if (!currentPlaylistTracks.length) {
            daemonVoid(forward ? "playback.next" : "playback.prev", {}, function(ok) {
                if (ok)
                    mergeMonitorStatus()
            }, function() {
                runPlayer([forward ? "next" : "prev"], function() { mergeMonitorStatus() }, transportProc)
            })
            return
        }
        var snap = playerMonitor && playerMonitor.player
        var currentPath = String((snap && snap.path) || player.path || "")
        var idx = indexInCurrentPlaylist(currentPath)
        if (idx < 0)
            idx = Math.max(0, Number((snap && snap.playlist_pos) || selectedTrackIndex || 0))
        var n = currentPlaylistTracks.length
        var nextIdx = forward ? (idx + 1) % n : (idx - 1 + n) % n
        var nextPath = currentPlaylistTracks[nextIdx] ? String(currentPlaylistTracks[nextIdx].path || "") : ""
        if (!nextPath)
            return
        selectedTrackIndex = nextIdx
        selectedTrackPath = nextPath
        playQueueAt(nextPath, pathsFromTracks(currentPlaylistTracks))
    }

    function toggleFiletreeFavorite(path) {
        root.toggleTrackFavorite(path)
    }

    function filetreeAbsPath(relPath) {
        var rel = String(relPath || "").trim()
        var rootPath = String(musicRoot || "").replace(/\/+$/, "")
        if (!rel)
            return rootPath
        if (rel.charAt(0) === "/")
            return rel
        if (rootPath && (rel === rootPath || rel.indexOf(rootPath + "/") === 0))
            return rel
        return rootPath ? rootPath + "/" + rel.replace(/^\/+/, "") : rel
    }

    function openInThunar(targetPath) {
        var target = String(targetPath || "").trim()
        if (!target)
            return
        if (openDirProc.running)
            return
        openDirProc.command = ["bash", "-lc",
            "GDK_WAYLAND_APP_ID=floating-window exec -a floating-window thunar "
                + JSON.stringify(target)]
        openDirProc.running = true
    }

    function openBrowseFolder(entry) {
        if (!entry)
            return
        openInThunar(filetreeAbsPath(entry.path))
    }

    function openTrackInThunar(trackPath) {
        var path = String(trackPath || "").trim()
        if (!path)
            return
        openInThunar(path)
    }

    function openPlaylistFolder(playlistName) {
        var name = String(playlistName || "").trim()
        if (!name)
            return
        if (name === currentPlaylistId) {
            var rel = String(currentPlaylistPath || "").trim()
            openInThunar(rel ? filetreeAbsPath(rel) : filetreeAbsPath(""))
            return
        }
        if (name === "all" || name === "mixes") {
            openInThunar(filetreeAbsPath(""))
            return
        }
        openInThunar(filetreeAbsPath(name))
    }

    Process {
        id: jobProc
        stdout: StdioCollector {
            id: jobOut
            waitForEnd: false
        }
        stderr: StdioCollector {
            id: jobErr
            waitForEnd: false
        }
        onExited: function(exitCode) {
            root.onJobFinished(exitCode)
        }
    }

    Process {
        id: sortProc
        property string _label: ""
        stdout: StdioCollector {}
        stderr: StdioCollector {
            id: sortErr
        }
        onExited: function(exitCode) {
            root.onSortFinished(exitCode)
        }
    }

    Timer {
        id: visibleArtWarmTimer
        interval: 120
        repeat: false
        onTriggered: {
            if (root.playlistsPanelOpen && root.playlistTrackList)
                root.warmVisibleTracks(root.playlistTrackList, root.tracks, false)
            else if (root.filetreePanelOpen && root.filetreeListView)
                root.warmVisibleTracks(root.filetreeListView, root.filetreeRows, true)
        }
    }

    Timer {
        id: libraryCpuPollTimer
        interval: 2000
        repeat: true
        running: root.libraryActivityBusy || root.externalJobBusy || root.scanJobRunning
        onTriggered: {
            if (!libraryCpuPollProc.running)
                libraryCpuPollProc.running = true
        }
        onRunningChanged: {
            if (!running) {
                root.ffmpegCpuPercent = 0
                root.evoplayerCpuPercent = 0
            }
        }
    }

    Process {
        id: libraryCpuPollProc
        command: ["bash", "-c",
            "ffmpeg=$(ps -C ffmpeg -o %cpu= 2>/dev/null | awk '{s+=$1} END {printf \"%.0f\", s+0}'); " +
            "evo=$(ps -C evoplayer -o %cpu= 2>/dev/null | awk '{s+=$1} END {printf \"%.0f\", s+0}'); " +
            "echo $ffmpeg $evo"]
        stdout: StdioCollector {
            onStreamFinished: {
                var parts = String(text || "").trim().split(/\s+/)
                root.ffmpegCpuPercent = parseInt(parts[0], 10) || 0
                root.evoplayerCpuPercent = parseInt(parts[1], 10) || 0
            }
        }
    }

    Timer {
        id: daemonJobPollTimer
        interval: 1000
        repeat: true
        onTriggered: root.pollDaemonJobStatus()
    }

    Timer {
        id: jobStatusTimer
        interval: 2000
        repeat: true
        running: root.jobBusy || root.externalJobBusy
        onTriggered: root.syncExternalJobStatus()
    }

    Timer {
        id: jobLogTimer
        interval: 100
        repeat: true
        running: root.jobBusy || root.externalJobBusy
        onTriggered: root.syncJobLog()
    }

    Process {
        id: cmdProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (cmdProc._onDone)
                    cmdProc._onDone(text)
            }
        }
    }

    Process {
        id: seekProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (seekProc._onDone)
                    seekProc._onDone(text)
            }
        }
    }

    Process {
        id: volumeProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (volumeProc._onDone)
                    volumeProc._onDone(text)
            }
        }
    }

    Process {
        id: deactivateProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (deactivateProc._onDone)
                    deactivateProc._onDone(text)
            }
        }
    }

    Process {
        id: openDirProc
    }

    Process {
        id: trashTrackProc
        property string _path: ""
        onExited: function(exitCode) {
            root.onTrackTrashed(exitCode)
        }
    }

    Process {
        id: loadProc
        property string pendingPath: ""
        property bool pendingFolder: false
        stdout: StdioCollector {}
        onExited: function() {
            root.refreshStatus()
            if (pendingPath) {
                var next = pendingPath
                var folder = pendingFolder
                pendingPath = ""
                pendingFolder = false
                root.startLoad(next, folder)
            }
        }
    }

    Process {
        id: filetreeProc
        property var _onDone: null
        property bool _jobDone: false
        stdout: StdioCollector {
            onStreamFinished: {
                if (filetreeProc._onDone) {
                    var done = filetreeProc._onDone
                    filetreeProc._onDone = null
                    done(text)
                }
            }
        }
        onExited: {
            if (filetreeProc._onDone) {
                var fallback = filetreeProc._onDone
                filetreeProc._onDone = null
                fallback("")
            }
            Qt.callLater(pumpBrowseQuery)
        }
    }

    Process {
        id: queryProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (queryProc._onDone) {
                    var done = queryProc._onDone
                    queryProc._onDone = null
                    done(text)
                }
            }
        }
        onExited: {
            if (queryProc._onDone) {
                var fallback = queryProc._onDone
                queryProc._onDone = null
                fallback("")
            }
        }
    }

    Process {
        id: settingsLoadProc
        command: root.playerCmd(["config", "get", "--json"])
        stdout: StdioCollector {
            onStreamFinished: root.parsePlayerSettings(text)
        }
    }

    Process {
        id: settingsSetProc
        property string key: ""
        property string value: ""
        command: root.playerCmd(["config", "set", settingsSetProc.key, settingsSetProc.value, "--json"])
        stdout: StdioCollector {
            onStreamFinished: {
                root.parsePlayerSettings(text)
                if (String(settingsSetProc.key || "").indexOf("viz.") === 0)
                    applyVizProc.running = true
            }
        }
    }

    Process {
        id: applyVizProc
        command: root.playerCmd(["viz", "apply"])
    }

    Process {
        id: settingsPickProc
        command: root.playerCmd(["config", "pick"])
        stdout: StdioCollector {
            onStreamFinished: {
                if (String(text || "").trim())
                    root.parsePlayerSettings(text)
            }
        }
    }

    Process {
        id: favoriteProc
        property var _onDone: null
        property var _queuedArgs: null
        property var _queuedOnDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                var cb = favoriteProc._onDone
                favoriteProc._onDone = null
                if (cb)
                    cb(text)
                if (favoriteProc._queuedArgs) {
                    var args = favoriteProc._queuedArgs
                    var done = favoriteProc._queuedOnDone
                    favoriteProc._queuedArgs = null
                    favoriteProc._queuedOnDone = null
                    root.runFavoriteQuery(args, done)
                }
            }
        }
    }

    Process {
        id: queuePlayProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (queuePlayProc._onDone)
                    queuePlayProc._onDone(text)
            }
        }
    }

    Process {
        id: appendFiletreeProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (appendFiletreeProc._onDone)
                    appendFiletreeProc._onDone(text)
                appendFiletreeProc._onDone = null
            }
        }
        onExited: appendFiletreeProc._onDone = null
    }

    Process {
        id: transportProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (transportProc._onDone)
                    transportProc._onDone(text)
            }
        }
    }

    Process {
        id: saveCurrentProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (saveCurrentProc._onDone)
                    saveCurrentProc._onDone(text)
            }
        }
    }

    Process {
        id: currentPlaylistLoadProc
        property var _onDone: null
        property var _pendingJob: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (currentPlaylistLoadProc._onDone) {
                    var done = currentPlaylistLoadProc._onDone
                    currentPlaylistLoadProc._onDone = null
                    done(text)
                }
            }
        }
        onExited: currentPlaylistLoadProc._onDone = null
    }

    Process {
        id: rowArtWarmProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (rowArtWarmProc._onDone) {
                    var done = rowArtWarmProc._onDone
                    rowArtWarmProc._onDone = null
                    done(text)
                }
            }
        }
    }

    Process {
        id: warmArtProc
        property var _onDone: null
        property var _pendingJob: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (warmArtProc._onDone) {
                    var done = warmArtProc._onDone
                    warmArtProc._onDone = null
                    done(text)
                }
            }
        }
    }

    Process {
        id: displayArtCacheProc
        property var _onDone: null
        property string _requestedPath: ""
        property var _pendingJob: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (displayArtCacheProc._onDone) {
                    var done = displayArtCacheProc._onDone
                    displayArtCacheProc._onDone = null
                    done(text)
                }
            }
        }
    }

    Process {
        id: playlistQueryProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (playlistQueryProc._onDone) {
                    var done = playlistQueryProc._onDone
                    playlistQueryProc._onDone = null
                    done(text)
                }
            }
        }
        onExited: playlistQueryProc._onDone = null
    }

    ListModel {
        id: playlistTabModel
    }

    Process {
        id: playerQueryProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (playerQueryProc._onDone)
                    playerQueryProc._onDone(text)
                playerQueryProc._onDone = null
            }
        }
        onExited: playerQueryProc._onDone = null
    }

    Process {
        id: agentDebugLogProc
        property var _queue: []
        property string _line: ""
        function enqueue(line) {
            _queue.push(String(line || ""))
            pumpAgentDebugLog()
        }
        onExited: Qt.callLater(pumpAgentDebugLog)
    }

    Process {
        id: statusQueryProc
        property var _onDone: null
        stdout: StdioCollector {
            onStreamFinished: {
                if (statusQueryProc._onDone)
                    statusQueryProc._onDone(text)
                statusQueryProc._onDone = null
            }
        }
        onExited: statusQueryProc._onDone = null
    }

    FileView {
        id: waveformFile
        path: root.currentWaveformFilePath()
        watchChanges: true
        onLoaded: {
            if (String(path) !== root.currentWaveformFilePath())
                return
            root.applyWaveform(text())
        }
        onLoadFailed: {
            var p = String(root.player.path || "")
            if (!p) {
                root.applyWaveform("")
                root.waveformLoading = false
                return
            }
            if (root.player.waveform) {
                var cleared = Object.assign({}, root.player, { waveform: "" })
                root.player = cleared
            }
            root.ensureWaveformForPath(p)
            if (!String(root.player.waveform || ""))
                root.finishWaveformLoading(p, false)
        }
    }

    FileView {
        id: neighborWaveformFile
        property string trackPath: ""
        path: ""
        printErrors: false
        onLoaded: {
            if (!trackPath || !path)
                return
            root.cacheWaveformText(trackPath, text())
            root.advanceNeighborWaveform()
        }
        onLoadFailed: {
            if (path)
                root.advanceNeighborWaveform()
        }
    }

    Repeater {
        model: root.prefetchArtSources
        Image {
            required property string modelData
            source: modelData
            asynchronous: true
            cache: true
            visible: false
            width: 1
            height: 1
        }
    }

    Timer {
        id: sideArtKickTimer
        interval: 100
        repeat: true
        property int tries: 0
        onTriggered: {
            root.syncSideArtImageSource()
            tries++
            if (tries >= 24 || root.sideArtLoaded)
                stop()
        }
        onRunningChanged: if (!running)
            tries = 0
    }

    Timer {
        id: tabSearchDebounce
        interval: 650
        repeat: false
        onTriggered: {
            var q = String(root.tabSearchText || "").trim()
            if (!q)
                return
            root.runTabSearch()
        }
    }

    Timer {
        id: artSearchDebounce
        interval: 500
        repeat: false
        onTriggered: {
            var q = String(root.artPickerSearchText || "").trim()
            if (!q || !root.artPickerOpen)
                return
            root.searchArtPicker(q)
        }
    }

    Timer {
        id: statusNoteTimer
        interval: 3000
        repeat: false
        onTriggered: root.statusNote = ""
    }

    Timer {
        id: statusTimer
        interval: 500
        repeat: true
        running: false
        onTriggered: root.refreshStatus()
    }

    Connections {
        target: root.playerMonitor
        enabled: root.playerMonitor !== null
        function onIpcReadyChanged() {
            if (!root.playerMonitor || !root.playerMonitor.ipcReady) {
                root._vizSubscribed = false
                return
            }
            // Daemon restart drops server-side viz subs; resubscribe.
            root._vizSubscribed = false
            root.syncVizSubscription()
        }
    }

    Connections {
        target: root
        function onActiveChanged() { root.syncVizSubscription() }
        function onNowplayingTabActiveChanged() {
            // #region agent log
            root.agentDebug("H5", "DashboardModule.qml:onNowplayingTabActiveChanged", "tab active changed", {
                active: root.nowplayingTabActive,
                playing: root.playerPlaying
            })
            // #endregion
            root.syncVizSubscription()
        }
        function onPlayerPlayingChanged() { root.syncVizSubscription() }
        function onPlayerPathForVizChanged() { root.syncVizSubscription() }
    }

    Connections {
        target: root.playerMonitor
        enabled: root.playerMonitor !== null
        function onDaemonJobUpdated(data) {
            root.onDaemonJobEvent(data)
        }
    }

    Connections {
        target: root.playerMonitor
        enabled: root.playerMonitor !== null
        function onPlayerChanged() {
            if (root.active)
                monitorMergeTimer.restart()
        }
    }

    Timer {
        id: monitorMergeTimer
        interval: 120
        repeat: false
        onTriggered: root.mergeMonitorStatus()
    }

    Connections {
        target: root
        function onTracksRevisionChanged() {
            var path = String(root.player.path || "")
            if (!path)
                return
            var listArt = root.artForTrackPath(path)
            if (!listArt || listArt.charAt(0) !== "/")
                return
            if (String(root.player.art || "") === listArt)
                return
            root.player = Object.assign({}, root.player, { art: listArt })
            if (root.resolvedArtPath !== path || !root.resolvedArt)
                root.applyPlayerArt(path, listArt)
        }
    }

    Timer {
        id: sidePanelKickTimer
        interval: 32
        repeat: true
        property int tries: 0
        onTriggered: {
            var browseDone = refreshFiletreeListView()
            var playlistDone = refreshPlaylistListView()
            tries++
            if ((browseDone && playlistDone) || tries >= 60)
                stop()
        }
        onRunningChanged: if (!running)
            tries = 0
    }

    Timer {
        id: filetreeScrollIdleTimer
        interval: 150
        onTriggered: {
            root.filetreeUserScrolling = false
            if (Object.keys(root._filetreeArtPatchQueue).length)
                Qt.callLater(root.flushFiletreeArtPatches)
            root.scheduleVisibleArtWarm()
        }
    }

    Timer {
        id: playlistScrollIdleTimer
        interval: 150
        onTriggered: {
            root.playlistUserScrolling = false
            root.savePlaylistScrollPosition(root.selectedPlaylist)
            root.scheduleVisibleArtWarm()
        }
    }

    Timer {
        id: filetreeHoldSafetyTimer
        interval: 500
        onTriggered: {
            if (root.filetreeHoldY >= 0)
                root.filetreeHoldY = -1
        }
    }

    Timer {
        id: filetreeQueueBusyTimer
        interval: 15000
        repeat: false
        onTriggered: {
            if (root.filetreeQueueBusy)
                root.filetreeQueueBusy = false
        }
    }

    Timer {
        id: filetreeLoadSafetyTimer
        interval: 20000
        repeat: false
        onTriggered: {
            if (!root.filetreeTreeLoading)
                return
            root.filetreeTreeLoading = false
            if (root.filetreePanelOpen || root.filetreeRows.length > 0)
                root.loadFiletreeRoot()
        }
    }

    Timer {
        id: filetreeKickScrollTimer
        interval: 0
        repeat: true
        property int attempts: 0
        onTriggered: {
            if (root.filetreeRestoreY < 0 || !root.filetreeListView) {
                stop()
                attempts = 0
                return
            }
            root.filetreeListView.contentY = root.filetreeRestoreY
            attempts++
            if (root.filetreeListView.contentHeight > 0 || attempts >= 8) {
                root.filetreeRestoreY = -1
                stop()
                attempts = 0
            }
        }
        onRunningChanged: if (!running)
            attempts = 0
    }

    Connections {
        target: root
        function onFiletreeRowsChanged() {
            if (root.filetreePanelOpen && root.filetreeRows.length > 0 && !root.filetreeArtOnlyPatch)
                root.kickSidePanels()
            if (root.filetreeRestoreY < 0)
                return
            filetreeKickScrollTimer.attempts = 0
            filetreeKickScrollTimer.restart()
        }
        function onFiletreePanelOpenChanged() {
            if (root.filetreePanelOpen)
                root.kickSidePanels()
        }
        function onPlaylistsPanelOpenChanged() {
            if (root.playlistsPanelOpen)
                root.kickSidePanels()
        }
        function onLibraryPlaylistsChanged() {
            if (root.playlistsPanelOpen && root.playlistPanelMode === "library")
                root.kickSidePanels()
        }
        function onSplitSidePanelModeChanged() {
            if (root.splitSidePanelMode)
                root.kickSidePanels()
        }
    }

    Timer {
        id: panelTransitionTimer
        interval: 350
        repeat: false
        onTriggered: root.panelTransitionGuard = false
    }

    Timer {
        id: favoriteSettleTimer
        interval: 4000
        repeat: false
        onTriggered: {
            root.favoriteApplyPending = false
            root.favoriteApplyPath = ""
        }
    }

    Timer {
        id: saveStateTimer
        interval: 10000
        repeat: true
        onTriggered: root.runPlayer(["save"], null, cmdProc)
    }

    Timer {
        id: seekApplyTimer
        interval: 120
        repeat: false
        onTriggered: root.flushSeekApply()
    }

    Timer {
        id: seekSettleTimer
        interval: 1500
        repeat: false
        onTriggered: root.finishSeekSettle()
    }

    Timer {
        id: volumeApplyTimer
        interval: 120
        repeat: false
        onTriggered: root.flushVolumeApply()
    }

    Timer {
        id: volumeSettleTimer
        interval: 1500
        repeat: false
        onTriggered: root.finishVolumeSettle()
    }

    Timer {
        id: transportApplyTimer
        interval: 120
        repeat: false
        onTriggered: root.flushTransportApply()
    }

    Timer {
        id: transportSettleTimer
        interval: 1500
        repeat: false
        onTriggered: root.finishTransportSettle()
    }

    Timer {
        id: scannerHoldTimer
        interval: 400
        repeat: false
        onTriggered: {
            root.scannerHoldPending = false
            root.tryRevealNowPlaying()
        }
    }

    Timer {
        id: trackRevealTimer
        interval: 2500
        repeat: false
        onTriggered: root.forceRevealNowPlaying()
    }

    Timer {
        id: playbackToggleTimer
        interval: 80
        repeat: false
        onTriggered: root.flushPlaybackToggle()
    }

    Timer {
        id: persistCurrentTimer
        interval: 500
        repeat: false
        onTriggered: root.flushPersistCurrentPlaylist()
    }

    Timer {
        id: currentEnrichTimer
        interval: 800
        repeat: false
        onTriggered: {
            root.enrichCurrentTracksBatch()
            root.prioritizeCurrentAssets()
        }
    }

    Timer {
        id: waveformRecheckTimer
        interval: 1500
        repeat: false
        onTriggered: root.recheckWaveformForPath()
    }

    Timer {
        id: playbackSettleTimer
        interval: 1500
        repeat: false
        onTriggered: root.finishPlaybackSettle()
    }

    Flickable {
        id: playerScroller
        anchors.fill: parent
        anchors.topMargin: columnTopPad
        anchors.leftMargin: columnPad
        anchors.rightMargin: columnPad
        anchors.bottomMargin: columnPad
        clip: !root.settingsPanelOpen
        contentWidth: width
        contentHeight: Math.max(height, rootLayout.implicitHeight)
        boundsBehavior: Flickable.StopAtBounds
        interactive: contentHeight > height && !root.settingsPanelOpen

        ColumnLayout {
            id: rootLayout
            width: playerScroller.width
            height: Math.max(playerScroller.height, implicitHeight)
            spacing: root.playerSectionGap

        // Tabs
        SectionPanel {
            label: ""
            notchLegend: true
            legendText: "Menu"
            legendIcon: "󰍜"
            legendBackground: Theme.background
            visible: !root.menubarHidden
            Layout.fillWidth: true
            contentPad: root.contentPad
            fillHeight: false

            RowLayout {
                id: tabBarHost
                Layout.fillWidth: true
                Layout.preferredHeight: root.genreTabHeight
                Layout.maximumHeight: root.genreTabHeight
                Layout.alignment: Qt.AlignVCenter
                spacing: Theme.spacingM

                IconTab {
                    dashboard: root
                    icon: "󰎆"
                    active: root.nowplayingTabActive
                    onActivated: root.showNowplaying()
                }
                IconTab {
                    dashboard: root
                    icon: "󰉋"
                    active: root.filetreePanelOpen
                    onActivated: root.toggleFiletreePanel()
                }
                IconTab {
                    dashboard: root
                    icon: "󰲸"
                    active: root.playlistsPanelOpen
                    onActivated: root.togglePlaylistsPanel()
                }
                IconTab {
                    dashboard: root
                    icon: "󰄨"
                    active: root.statsPanelOpen
                    onActivated: root.toggleStatsPanel()
                }
                IconTab {
                    dashboard: root
                    icon: "󰇚"
                    active: root.downloadsPanelOpen
                    onActivated: root.toggleDownloadsPanel()
                }
                IconTab {
                    dashboard: root
                    icon: "󰒓"
                    active: root.settingsPanelOpen
                    onActivated: root.toggleSettingsPanel()
                }

                Rectangle {
                    Layout.preferredWidth: 1
                    Layout.preferredHeight: Math.max(12, root.genreTabHeight - 16)
                    Layout.alignment: Qt.AlignVCenter
                    color: Theme.foregroundDivider
                }

                Item {
                    id: playlistTabBarHost
                    visible: !root.scanJobRunning
                    Layout.fillWidth: true
                    Layout.preferredHeight: root.genreTabHeight
                    Layout.maximumHeight: root.genreTabHeight
                    Layout.alignment: Qt.AlignVCenter
                    Layout.preferredWidth: playlistTabBar.contentWidth
                    Layout.maximumWidth: playlistTabBar.contentWidth
                    Layout.minimumWidth: 0

                    ListView {
                        id: playlistTabBar
                        anchors.fill: parent
                        orientation: ListView.Horizontal
                        spacing: Theme.spacingS
                        clip: true
                        boundsBehavior: Flickable.StopAtBounds
                        model: playlistTabModel

                        onCountChanged: Qt.callLater(root.syncPlaylistTabPosition)
                        Component.onCompleted: root.syncPlaylistTabPosition()

                        delegate: Item {
                            required property int index
                            required property string name
                            required property int count
                            width: playlistTabContent.implicitWidth + 20
                            height: root.genreTabHeight

                            Rectangle {
                                anchors.fill: parent
                                radius: 6
                                color: root.playlistsPanelOpen && root.selectedPlaylist === name
                                    ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.2)
                                    : (playlistTabMouse.containsMouse
                                        ? Theme.foregroundWash
                                        : "transparent")
                            }

                            Row {
                                id: playlistTabContent
                                anchors.centerIn: parent
                                spacing: Theme.spacingS

                                Text {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: root.playlistTabLabel(name)
                                    color: root.playlistsPanelOpen && root.selectedPlaylist === name ? Theme.accent : Theme.foreground
                                    font.family: Theme.fontFamily
                                    font.pixelSize: root.listFont
                                    font.bold: root.playlistsPanelOpen && root.selectedPlaylist === name && Theme.fontBold
                                    opacity: root.playlistsPanelOpen && root.selectedPlaylist === name ? 1 : 0.78
                                }

                                HoverPanelLabelPill {
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: String(count || 0)
                                    fieldsetLegend: false
                                    fontSize: Theme.fontSizeXs
                                    textOpacity: 0.62
                                }
                            }

                            MouseArea {
                                id: playlistTabMouse
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: root.selectPlaylist(name)
                            }
                        }
                    }
                }

                Item {
                    visible: !root.scanJobRunning
                    Layout.fillWidth: true
                    Layout.minimumWidth: 120
                    Layout.preferredHeight: root.genreTabHeight
                    Layout.alignment: Qt.AlignVCenter

                    Rectangle {
                        anchors.fill: parent
                        radius: 6
                        color: Theme.foregroundWash
                        border.color: tabSearchInput.activeFocus
                            ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.35)
                            : Theme.foregroundDivider
                        border.width: 1

                        TextInput {
                            id: tabSearchInput
                            anchors.fill: parent
                            anchors.leftMargin: 8
                            anchors.rightMargin: tabSearchClear.visible ? 26 : 8
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: root.libraryFont
                            selectionColor: Theme.accent
                            selectedTextColor: Theme.mantle
                            verticalAlignment: TextInput.AlignVCenter
                            clip: true
                            onTextChanged: {
                                root.tabSearchText = text
                                tabSearchDebounce.restart()
                            }
                            onAccepted: {
                                tabSearchDebounce.stop()
                                root.runTabSearch()
                            }
                            Keys.onEscapePressed: {
                                tabSearchDebounce.stop()
                                root.tabSearchText = ""
                                text = ""
                                focus = false
                            }
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            anchors.left: parent.left
                            anchors.leftMargin: 8
                            visible: !tabSearchInput.text && !tabSearchInput.activeFocus
                            text: "search…"
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: root.libraryFont
                            opacity: 0.4
                        }

                        Item {
                            id: tabSearchClear
                            visible: tabSearchInput.text !== ""
                            anchors.right: parent.right
                            anchors.rightMargin: 4
                            anchors.verticalCenter: parent.verticalCenter
                            width: 22
                            height: 22
                            z: 1

                            Text {
                                anchors.centerIn: parent
                                text: "󰅖"
                                color: Theme.foreground
                                font.family: Theme.fontFamily
                                font.pixelSize: root.libraryFont
                                opacity: tabSearchClearMouse.containsMouse ? 0.95 : 0.45
                            }

                            MouseArea {
                                id: tabSearchClearMouse
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: {
                                    tabSearchDebounce.stop()
                                    root.tabSearchText = ""
                                    tabSearchInput.text = ""
                                    tabSearchInput.focus = false
                                }
                            }
                        }
                    }
                }

                LibraryScanStatusBar {
                    dashboard: root
                    visible: root.scanJobRunning
                    Layout.alignment: Qt.AlignVCenter
                }
            }
        }

        // Now playing / library panels
        Item {
            id: nowplayingSection
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumHeight: root.nowplayingMinBodyHeight

            RowLayout {
                    anchors.fill: parent
                    spacing: columnPad

                    ColumnLayout {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        Layout.minimumWidth: root.nowplayingMinBioWidth
                        Layout.maximumWidth: root.nowplayingCompact
                            ? -1
                            : Math.max(1, nowplayingSection.width - root.albumartWidth - columnPad)
                        spacing: root.playerSectionGap

                        Item {
                            Layout.fillWidth: true
                            Layout.fillHeight: true

                            StackLayout {
                                id: sideContentStack
                                anchors.fill: parent
                                currentIndex: root.sideContentStackIndex

                                onCurrentIndexChanged: Qt.callLater(root.kickSidePanels)

                        SectionPanel {
                            label: ""
                            notchLegend: true
                            legendText: "Now playing"
                            legendIcon: "󰎆"
                            legendBackground: Theme.background
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            Layout.minimumHeight: root.nowplayingFieldsetMinHeight
                            fillHeight: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.fillHeight: true
                                spacing: columnPad
                                clip: true

                                RowLayout {
                                    id: titleRow
                                    Layout.fillWidth: true
                                    spacing: Theme.spacingL

                                    AlbumArtThumbnail {
                    dashboard: root
                                        visible: root.nowplayingCompact
                                        side: root.nowplayingInlineArtSize
                                        showPickerOverlay: false
                                        useRevision: true
                                        art: root.nowplayingArt
                                        Layout.alignment: Qt.AlignTop | Qt.AlignLeft
                                        Layout.rightMargin: contentPad
                                    }

                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: Theme.spacingM
                                        opacity: root.nowplayingContentVisible ? 1 : 0
                                        Behavior on opacity {
                                            NumberAnimation {
                                                duration: 180
                                                easing.type: Easing.OutCubic
                                            }
                                        }

                                        Item {
                                            Layout.fillWidth: true
                                            Layout.preferredHeight: titleLabel.implicitHeight
                                            clip: true

                                            Text {
                                                id: titleLabel
                                                width: parent.width
                                                text: root.nowplayingTitle
                                                color: Theme.foreground
                                                font.family: Theme.fontFamily
                                                font.pixelSize: root.nowplayingTitleFont
                                                font.bold: Theme.fontBold
                                                wrapMode: Text.Wrap
                                                maximumLineCount: 2
                                                elide: Text.ElideRight
                                                opacity: titleMouse.containsMouse && root.nowplayingTitle !== "No track"
                                                    ? 0.82
                                                    : 1
                                            }

                                            MouseArea {
                                                id: titleMouse
                                                anchors.fill: titleLabel
                                                enabled: root.nowplayingTitle !== "No track"
                                                hoverEnabled: true
                                                cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
                                                onClicked: root.copyTitleToClipboard()
                                            }
                                        }

                                        Flow {
                                            id: bylineFlow
                                            Layout.fillWidth: true
                                            spacing: 0
                                            visible: root.nowplayingBylineVisible
                                            clip: true

                                            Text {
                                                visible: root.nowplayingArtist !== ""
                                                width: bylineFlow.width > 0
                                                    ? Math.min(implicitWidth, bylineFlow.width)
                                                    : implicitWidth
                                                text: root.nowplayingArtist
                                                color: Theme.foreground
                                                font.family: Theme.fontFamily
                                                font.pixelSize: Theme.fontSizeXl
                                                wrapMode: Text.Wrap
                                                opacity: artistMouse.containsMouse ? 1 : 0.62

                                                MouseArea {
                                                    id: artistMouse
                                                    anchors.fill: parent
                                                    hoverEnabled: true
                                                    cursorShape: Qt.PointingHandCursor
                                                    onClicked: function(mouse) {
                                                        var artist = String(root.nowplayingArtist || "").trim()
                                                        if (!artist)
                                                            return
                                                        mouse.accepted = true
                                                        root.openFilter("artist", artist, artist)
                                                    }
                                                }
                                            }

                                            Text {
                                                visible: root.nowplayingArtist !== "" && root.nowplayingAlbum !== ""
                                                text: " - "
                                                color: Theme.foreground
                                                font.family: Theme.fontFamily
                                                font.pixelSize: Theme.fontSizeXl
                                                opacity: 0.62
                                            }

                                            Text {
                                                visible: root.nowplayingAlbum !== ""
                                                width: bylineFlow.width > 0
                                                    ? Math.min(implicitWidth, bylineFlow.width)
                                                    : implicitWidth
                                                text: root.nowplayingAlbum
                                                color: Theme.foreground
                                                font.family: Theme.fontFamily
                                                font.pixelSize: Theme.fontSizeXl
                                                wrapMode: Text.Wrap
                                                opacity: albumMouse.containsMouse ? 1 : 0.62

                                                MouseArea {
                                                    id: albumMouse
                                                    anchors.fill: parent
                                                    hoverEnabled: true
                                                    cursorShape: Qt.PointingHandCursor
                                                    onClicked: function(mouse) {
                                                        var album = String(root.nowplayingAlbum || "").trim()
                                                        if (!album)
                                                            return
                                                        mouse.accepted = true
                                                        root.openFilter("album", album, album)
                                                    }
                                                }
                                            }
                                        }

                                        Flow {
                                            Layout.fillWidth: true
                                            spacing: Theme.spacingS
                                            visible: root.nowplayingMetaChips.length > 0

                                            Repeater {
                                                model: root.nowplayingMetaChips

                                                delegate: MetaChip {
                                                    dashboard: root
                                                    required property var modelData
                                                    fontSize: root.libraryFont
                                                    label: modelData.label
                                                    accent: false
                                                    tintColor: modelData.tint || Theme.foreground
                                                    tinted: !!modelData.tint
                                                    clickable: modelData.kind === "label"
                                                        || modelData.kind === "genre"
                                                        || modelData.kind === "year"
                                                    onActivated: {
                                                        if (modelData.kind === "label")
                                                            root.openFilter("label", modelData.value, modelData.value)
                                                        else if (modelData.kind === "genre")
                                                            root.openFilter("genre", modelData.value, modelData.value)
                                                        else if (modelData.kind === "year")
                                                            root.openFilter("year", modelData.value, modelData.value)
                                                    }
                                                }
                                            }
                                        }

                                    }
                                }

                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.fillHeight: true
                                spacing: Theme.spacing2

                                WaveformViz {
                                    id: waveformViz
                                    dashboard: root
                                    Layout.fillWidth: true
                                    Layout.fillHeight: true
                                    Layout.minimumHeight: root.nowplayingWaveformMinHeight
                                }

                                UpNextStrip {
                                    dashboard: root
                                    Layout.fillWidth: true
                                    tracks: root.upNextTracks
                                    onRevealFirst: root.focusUpNextInPlaylist()
                                }

                            Controls {
                    dashboard: root
                                visible: root.showInlineTransport
                                Layout.fillWidth: true
                                Layout.preferredHeight: root.controlsHeight
                                Layout.maximumHeight: root.controlsHeight
                                showTimestamps: false
                            }
                            }
                        }
                        }

                        FiletreePanel {
                    dashboard: root
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                        }

                        PlaylistsPanel {
                    dashboard: root
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                        }

                        Item {
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            z: 20

                            Flickable {
                                id: settingsScroll
                                anchors.fill: parent
                                clip: true
                                contentWidth: width
                                contentHeight: Math.max(height, settingsScrollCol.implicitHeight)
                                boundsBehavior: Flickable.StopAtBounds
                                interactive: contentHeight > height

                                ColumnLayout {
                                    id: settingsScrollCol
                                    width: settingsScroll.width
                                    spacing: columnPad

                                    PlayerSettingsContent {
                                        dashboard: root
                                        Layout.fillWidth: true
                                    }
                                }
                            }
                        }

                        StatsPanel {
                    dashboard: root
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                        }

                        DownloadsPanel {
                    dashboard: root
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                        }

                        FilterPanel {
                    dashboard: root
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                        }
                            }
                        }

                        SectionPanel {
                            label: ""
                            notchLegend: true
                            legendText: "Transport"
                            legendIcon: "󰐊"
                            legendBackground: Theme.background
                            visible: root.showBottomTransport
                            contentPad: root.contentPad
                            Layout.fillWidth: true

                            Controls {
                    dashboard: root
                                Layout.fillWidth: true
                                Layout.preferredHeight: root.controlsHeight
                                Layout.maximumHeight: root.controlsHeight
                                showTimestamps: false
                            }
                        }
                    }

                    SectionPanel {
                        label: ""
                        notchLegend: true
                        legendText: "Album art"
                        legendIcon: "󰎈"
                        legendBackground: Theme.background
                        visible: !root.nowplayingCompact
                        opacity: root.nowplayingContentVisible ? 1 : 0
                        Behavior on opacity {
                            NumberAnimation {
                                duration: 180
                                easing.type: Easing.OutCubic
                            }
                        }
                        Layout.fillHeight: true
                        Layout.preferredWidth: root.albumartWidth
                        Layout.maximumWidth: root.albumartWidth
                        Layout.minimumWidth: root.albumartWidth
                        fillHeight: true

                        onVisibleChanged: {
                            if (visible)
                                Qt.callLater(root.refreshNowPlayingArtDisplay)
                        }

                        Item {
                            id: albumartHost
                            Layout.fillWidth: true
                            Layout.fillHeight: true

                            readonly property bool layoutReady: width > 8 && height > 8
                            readonly property bool heldReady: sideArtHeldImage.status === Image.Ready
                                && root.sideArtHeldSource !== ""
                            readonly property bool incomingReady: sideArtIncomingImage.status === Image.Ready
                                && root.sideArtIncomingSource !== ""
                            readonly property bool artShowing: heldReady || incomingReady
                            property bool retryQueued: false

                            onLayoutReadyChanged: {
                                root.sideArtLayoutReady = layoutReady
                                if (layoutReady)
                                    root.syncSideArtImageSource()
                            }

                            Rectangle {
                                anchors.fill: parent
                                radius: Theme.fieldsetCornerRadius
                                clip: true
                                color: Theme.foregroundFaint
                                visible: !albumartHost.artShowing

                                Text {
                                    anchors.centerIn: parent
                                    text: "󰎈"
                                    color: Theme.accent
                                    font.family: Theme.fontFamily
                                    font.pixelSize: Math.round(Math.min(albumartHost.width, albumartHost.height) * 0.22)
                                    opacity: 0.5
                                }
                            }

                            Rectangle {
                                anchors.fill: parent
                                radius: Theme.fieldsetCornerRadius
                                clip: true
                                color: "transparent"

                                Image {
                                    id: sideArtHeldImage
                                    anchors.fill: parent
                                    z: 0
                                    visible: albumartHost.heldReady
                                    source: root.sideArtHeldSource
                                    fillMode: Image.PreserveAspectCrop
                                    smooth: true
                                    asynchronous: true
                                    cache: false
                                    onStatusChanged: {
                                        if (status === Image.Ready) {
                                            if (root.sideArtIncomingSource
                                                    && root.sideArtHeldSource === root.sideArtIncomingSource)
                                                root.finishSideArtIncoming()
                                            else if (!root.sideArtIncomingSource && root.sideArtHeldSource) {
                                                root.sideArtLoaded = true
                                                sideArtKickTimer.stop()
                                            }
                                        }
                                        if (status === Image.Error
                                                && root.sideArtHeldSource
                                                && !albumartHost.retryQueued) {
                                            albumartHost.retryQueued = true
                                            root.applyDisplayArtForPath(String(player.path || ""), function() {
                                                albumartHost.retryQueued = false
                                            })
                                        }
                                    }
                                }

                                Image {
                                    id: sideArtIncomingImage
                                    anchors.fill: parent
                                    z: 1
                                    visible: albumartHost.incomingReady
                                    source: root.sideArtIncomingSource
                                    fillMode: Image.PreserveAspectCrop
                                    smooth: true
                                    asynchronous: true
                                    cache: false
                                    onStatusChanged: {
                                        if (status === Image.Ready && root.sideArtIncomingSource) {
                                            if (root.sideArtHeldSource === root.sideArtIncomingSource
                                                    && sideArtHeldImage.status === Image.Ready)
                                                root.finishSideArtIncoming()
                                            else
                                                root.beginSideArtHeldPromote(root.sideArtIncomingSource)
                                        }
                                        if (status === Image.Error
                                                && root.sideArtIncomingSource
                                                && !albumartHost.retryQueued) {
                                            albumartHost.retryQueued = true
                                            root.applyDisplayArtForPath(String(player.path || ""), function() {
                                                albumartHost.retryQueued = false
                                            })
                                        }
                                    }
                                }
                            }

                            DropArea {
                                id: albumartDrop
                                anchors.fill: parent
                                z: 2
                                keys: ["text/uri-list"]
                                onEntered: function(drag) {
                                    var ok = false
                                    if (root.artTargetPath && drag.hasUrls) {
                                        for (var i = 0; i < drag.urls.length; i++) {
                                            if (root.isImagePath(root.localPathFromUrl(drag.urls[i]))) {
                                                ok = true
                                                break
                                            }
                                        }
                                    }
                                    drag.accepted = ok
                                }
                                onDropped: function(drop) {
                                    if (!drop.hasUrls || !root.artTargetPath)
                                        return
                                    for (var j = 0; j < drop.urls.length; j++) {
                                        var p = root.localPathFromUrl(drop.urls[j])
                                        if (root.isImagePath(p)) {
                                            root.openArtPickerForDrop(p)
                                            break
                                        }
                                    }
                                }
                            }

                            MouseArea {
                                z: 3
                                anchors.fill: parent
                                enabled: !root.artPickerOpen
                                hoverEnabled: true
                                cursorShape: (root.artTargetPath || "") !== ""
                                    ? Qt.PointingHandCursor : Qt.ArrowCursor
                                onClicked: function(mouse) {
                                    if (!(root.artTargetPath || ""))
                                        return
                                    mouse.accepted = true
                                    root.openArtPicker()
                                }
                                onWheel: function(wheel) {
                                    if (!wheel.angleDelta.y)
                                        return
                                    if (root.volumeTransportBtn)
                                        root.volumeTransportBtn.nudgeVolume(wheel.angleDelta.y > 0 ? 5 : -5)
                                    else
                                        root.adjustVolume(wheel.angleDelta.y > 0 ? 5 : -5)
                                    wheel.accepted = true
                                }
                            }

                            ArtPickerOverlay {
                    dashboard: root
                                z: 4
                                anchors.fill: parent
                                visible: root.artPickerOpen
                            }

                            Rectangle {
                                visible: !root.artPickerOpen && root.playerStatusText !== ""
                                z: 5
                                anchors.right: albumartHost.right
                                anchors.bottom: albumartHost.bottom
                                anchors.rightMargin: 8
                                anchors.bottomMargin: 8
                                radius: Theme.radiusM
                                color: Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.82)
                                border.color: Theme.foregroundSubtle
                                border.width: 1
                                implicitWidth: artStatusText.width + 12
                                implicitHeight: artStatusText.height + 8

                                Text {
                                    id: artStatusText
                                    anchors.centerIn: parent
                                    width: Math.min(implicitWidth, Math.max(48, albumartHost.width - 28))
                                    text: root.playerStatusText
                                    color: Theme.foreground
                                    font.family: Theme.fontFamily
                                    font.pixelSize: root.libraryFont
                                    wrapMode: Text.Wrap
                                    maximumLineCount: 2
                                    elide: Text.ElideRight
                                    opacity: 0.72
                                }
                            }
                        }
                    }

                Item {
                    Layout.fillWidth: true
                    Layout.fillHeight: root.nowplayingCompact && root.artPickerOpen
                    Layout.preferredHeight: root.nowplayingCompact && root.artPickerOpen ? -1 : 0
                    visible: root.nowplayingCompact && root.artPickerOpen
                    z: 100

                    ArtPickerOverlay {
                        dashboard: root
                        anchors.fill: parent
                        anchors.margins: columnPad
                    }
                }
            }
        }
        }
    }

    Rectangle {
        id: trashConfirmScrim
        anchors.fill: parent
        visible: root.trashConfirmOpen
        z: 400
        color: Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.72)

        MouseArea {
            anchors.fill: parent
            onClicked: root.cancelTrashTrack()
        }

        MouseArea {
            anchors.centerIn: parent
            width: trashConfirmCard.width
            height: trashConfirmCard.height
            onClicked: function(mouse) { mouse.accepted = true }

            Rectangle {
                id: trashConfirmCard
                width: Math.min(420, trashConfirmScrim.width - root.columnPad * 4)
                height: trashConfirmCol.implicitHeight + contentPad * 2
                radius: Theme.radiusL
                color: Theme.overlaySurface
                border.color: Theme.accent
                border.width: 1

                ColumnLayout {
                    id: trashConfirmCol
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.margins: contentPad
                    spacing: Theme.spacingL

                    Text {
                        Layout.fillWidth: true
                        text: "trash this track?"
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: root.sectionLabelFont
                        font.bold: Theme.fontBold
                    }

                    Text {
                        Layout.fillWidth: true
                        visible: root.trashConfirmTitle !== ""
                        text: root.trashConfirmTitle
                        color: Theme.accent
                        font.family: Theme.fontFamily
                        font.pixelSize: root.listFont
                        wrapMode: Text.Wrap
                        maximumLineCount: 3
                        elide: Text.ElideRight
                    }

                    RowLayout {
                        Layout.fillWidth: true
                        Layout.topMargin: Theme.spacingS
                        spacing: Theme.spacingM

                        Item { Layout.fillWidth: true }

                        MetaChip {
                    dashboard: root
                            label: "cancel"
                            clickable: true
                            onActivated: root.cancelTrashTrack()
                        }

                        MetaChip {
                    dashboard: root
                            label: "trash"
                            accent: true
                            clickable: true
                            onActivated: root.confirmTrashTrack()
                        }
                    }
                }
            }
        }
    }
}
