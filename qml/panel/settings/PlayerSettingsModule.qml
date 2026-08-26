import Quickshell
import Quickshell.Io
import QtQuick
import QtQuick.Layouts
import "../compat"
import "../panels"
import "."

Item {
    id: root

    readonly property string home: Quickshell.env("HOME") || ""
    readonly property int libraryFont: Theme.fontSizeS
    readonly property int listFont: Theme.fontSizeS

    property var libraryStats: ({ tracks: 0, genres: 0 })
    property string settingsMusicLibrary: ""
    property string settingsScUser: ""
    property string settingsScOAuthToken: ""
    property bool settingsReady: false
    property bool settingsFieldFocused: false
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

    readonly property var libraryActions: [
        {
            key: "download",
            icon: "󰕧",
            label: "download soundcloud",
            button: "Download SoundCloud",
            hint: "download soundcloud",
            args: ["download"]
        },
        {
            key: "import",
            icon: "󰉍",
            label: "import incoming",
            button: "Import incoming",
            hint: "import incoming",
            args: ["import"]
        }
    ]

    property bool jobBusy: false
    property string jobLabel: ""
    property string jobLog: ""
    property string activeLibraryJobKey: ""
    readonly property bool libraryJobBusy: jobBusy
    readonly property string libraryJobActiveLabel: jobLabel
    readonly property bool libraryActivityBusy: libraryJobBusy

    implicitWidth: contentCol.implicitWidth
    implicitHeight: contentCol.implicitHeight

    function playerCmd(args) {
        return Util.evoplayerCommand(home, args || [])
    }

    function parsePlayerSettings(raw) {
        var trimmed = String(raw || "").trim()
        if (!trimmed)
            return
        try {
            var data = JSON.parse(trimmed)
            var sc = data.soundcloud || {}
            var paths = data.paths || {}
            var viz = data.viz || {}
            settingsScUser = String(sc.user || "")
            settingsScOAuthToken = String(sc.oauth_token || "")
            settingsMusicLibrary = String(paths.root || "")
            settingsVizFrameRate = parseInt(viz.frame_rate, 10) || 45
            settingsVizSensitivity = parseInt(viz.sensitivity, 10) || 145
            settingsVizAutosens = parseInt(viz.autosens, 10) || 2
            settingsVizNoiseReduction = parseInt(viz.noise_reduction, 10) || 34
            settingsVizMonstercat = parseFloat(viz.monstercat) || 1
            settingsVizLowCutoff = parseInt(viz.low_cutoff, 10) || 50
            settingsVizHighCutoff = parseInt(viz.high_cutoff, 10) || 10000
            settingsReady = true
        } catch (e) {
            settingsReady = false
        }
    }

    function loadPlayerSettings() {
        if (settingsLoadProc.running)
            return
        settingsLoadProc.running = true
    }

    function loadLibraryStats() {
        if (statsProc.running)
            return
        statsProc.running = true
    }

    function setPlayerSetting(key, value) {
        if (!settingsReady || settingsSetProc.running || settingsPickProc.running)
            return
        settingsSetProc.key = String(key || "")
        settingsSetProc.value = String(value || "")
        settingsSetProc.running = true
    }

    function setVizSetting(field, value) {
        setPlayerSetting("viz." + String(field || ""), String(value))
    }

    function setMusicLibrary(path) {
        setPlayerSetting("paths.root", path)
    }

    function pickMusicLibrary() {
        if (settingsPickProc.running || settingsSetProc.running)
            return
        settingsPickProc.running = true
    }

    function runLibraryAction(action) {
        if (!action || libraryJobBusy)
            return
        jobBusy = true
        jobLabel = String(action.label || "library task")
        activeLibraryJobKey = String(action.key || "")
        jobLog = jobLabel + "…\n"
        jobProc.command = playerCmd(action.args || [])
        jobProc.running = true
    }

    function jobLogInline() {
        if (libraryJobBusy)
            return libraryJobActiveLabel + "…"
        return ""
    }

    function stopLibraryJob() {
        if (!libraryJobBusy)
            return
        jobProc.running = false
        jobBusy = false
        jobLabel = ""
        activeLibraryJobKey = ""
        jobLog = String(jobLog || "") + "\nstopped"
    }

    Component.onCompleted: {
        loadPlayerSettings()
        loadLibraryStats()
    }

    onVisibleChanged: {
        if (visible) {
            loadPlayerSettings()
            loadLibraryStats()
        }
    }

    ColumnLayout {
        id: contentCol
        width: parent.width
        spacing: Theme.hoverPanelSectionSpacing

        PlayerSettingsContent {
            dashboard: root
            Layout.fillWidth: true
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
        id: applyVizProc
        command: root.playerCmd(["viz", "apply"])
    }

    Process {
        id: statsProc
        command: root.playerCmd(["stats", "--json"])
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    root.libraryStats = JSON.parse(String(text || "{}"))
                } catch (e) {
                    root.libraryStats = { tracks: 0, genres: 0 }
                }
            }
        }
    }

    Process {
        id: jobProc
        stdout: StdioCollector {
            onStreamFinished: {
                var out = String(text || "").trim()
                if (out)
                    root.jobLog = String(root.jobLog || "") + "\n" + out
                root.jobBusy = false
                root.jobLabel = ""
                root.activeLibraryJobKey = ""
            }
        }
        stderr: StdioCollector {
            onStreamFinished: {
                var err = String(text || "").trim()
                if (err)
                    root.jobLog = String(root.jobLog || "") + "\n" + err
            }
        }
        onExited: function(code) {
            if (code !== 0 && root.jobBusy) {
                root.jobBusy = false
                root.jobLabel = ""
                root.activeLibraryJobKey = ""
            }
        }
    }
}
