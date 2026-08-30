import QtQuick
import Quickshell
import Quickshell.Io
import "CliampBackend.js" as CliampBackend

Item {
    id: bridge

    required property var service
    required property bool active

    readonly property bool ipcReady: active && socketReady
    property bool ipcSynced: false
    property bool socketReady: false
    property bool vizWanted: false

    readonly property string cliampBin: CliampBackend.cliampBin(Quickshell.env("EVOPLAYER_CLIAMP_BIN"))
    readonly property string socketPath: CliampBackend.cliampSocketPath(service ? service.home : "", Quickshell.env("EVOPLAYER_CLIAMP_SOCKET"))

    function ensurePlayer() {
        if (!active)
            return
        if (!socketReady && !startDaemonProc.running)
            startDaemonProc.running = true
        else if (socketReady && !libraryDaemonProc.running)
            libraryDaemonProc.running = true
        refreshSocketReady()
    }

    function refreshSocketReady() {
        socketProbe.running = true
    }

    function runTransport(action) {
        if (!active)
            return
        var cmd = cliampBin
        if (action === "toggle")
            transportProc.command = [cmd, "toggle"]
        else if (action === "next")
            transportProc.command = [cmd, "next"]
        else if (action === "prev" || action === "previous")
            transportProc.command = [cmd, "prev"]
        else if (action === "play")
            transportProc.command = [cmd, "play"]
        else if (action === "pause")
            transportProc.command = [cmd, "pause"]
        else
            return
        transportProc.running = true
        pollStatusSoon()
    }

    function ipcCall(method, params, onDone) {
        if (!active) {
            if (onDone)
                onDone(false, null)
            return
        }
        var m = String(method || "")
        if (CliampBackend.isLibraryMethod(m)) {
            service.callEvoplayerIPC(method, params, onDone)
            return
        }
        if (m === "viz.subscribe") {
            vizWanted = true
            startVisStream()
            if (onDone)
                onDone(true, { ok: true })
            return
        }
        if (m === "viz.unsubscribe") {
            vizWanted = false
            stopVisStream()
            if (onDone)
                onDone(true, { ok: true })
            return
        }
        if (m.indexOf("playback.") === 0) {
            if (m === "playback.toggle")
                runTransport("toggle")
            else if (m === "playback.next")
                runTransport("next")
            else if (m === "playback.prev")
                runTransport("prev")
            if (onDone)
                onDone(true, { ok: true })
            return
        }
        if (m === "state.get" || m === "subscribe") {
            pollStatusSoon()
            if (onDone)
                onDone(true, { ok: true, data: service.player })
            return
        }
        service.callEvoplayerIPC(method, params, onDone)
    }

    function ipcCallVoid(method, params) {
        ipcCall(method, params, null)
    }

    function pollStatusSoon() {
        statusPollTimer.restart()
    }

    function parseStatusText(text) {
        var parsed
        try {
            parsed = JSON.parse(String(text || "{}"))
        } catch (e) {
            return
        }
        if (!parsed || parsed.ok === false)
            return
        ipcSynced = true
        var track = parsed.track || {}
        var state = String(parsed.state || "stopped").toLowerCase()
        if (state === "paused")
            state = "paused"
        else if (state === "playing")
            state = "playing"
        else
            state = "stopped"
        var patch = {
            state: state,
            path: String(track.path || ""),
            title: String(track.title || ""),
            artist: String(track.artist || ""),
            album: String(track.album || ""),
            position: Number(parsed.position) || 0,
            duration: Number(parsed.duration) || 0,
            volume: Number(parsed.volume) || 100,
            shuffle: !!parsed.shuffle
        }
        if (service && typeof service.applyStatePayload === "function")
            service.applyStatePayload(patch)
    }

    function parseVisLine(line) {
        if (!vizWanted)
            return
        var parsed
        try {
            parsed = JSON.parse(String(line || "{}"))
        } catch (e) {
            return
        }
        if (!parsed || !Array.isArray(parsed.bands))
            return
        if (service && typeof service.applyVizPayload === "function") {
            service.applyVizPayload({
                levels: parsed.bands,
                sequence: (service.vizSequence || 0) + 1,
                generation: service.vizGeneration || 0
            })
        }
    }

    function startVisStream() {
        if (visProc.running)
            return
        visProc.command = [cliampBin, "visstream", "--fps", "15"]
        visProc.running = true
    }

    function stopVisStream() {
        if (visProc.running)
            visProc.running = false
        if (service) {
            service.vizLevels = []
            service.vizRevision++
        }
    }

    Process {
        id: startDaemonProc
        command: [bridge.cliampBin, "--daemon"]
        onExited: Qt.callLater(bridge.refreshSocketReady)
    }

    Process {
        id: libraryDaemonProc
        command: service ? service.playerCmd(["start"]) : []
        onExited: Qt.callLater(function() {
            if (service && typeof service.ensureEvoplayerConnect === "function")
                service.ensureEvoplayerConnect()
        })
    }

    Process {
        id: statusProc
        command: [bridge.cliampBin, "status", "--json"]
        stdout: SplitParser {
            onRead: line => bridge.parseStatusText(line)
        }
    }

    Process {
        id: transportProc
    }

    Process {
        id: visProc
        stdout: SplitParser {
            onRead: line => bridge.parseVisLine(line)
        }
        onExited: {
            if (bridge.vizWanted)
                Qt.callLater(bridge.startVisStream)
        }
    }

    Process {
        id: socketProbe
        command: ["test", "-S", bridge.socketPath]
        onExited: {
            bridge.socketReady = exitCode === 0
            if (bridge.socketReady && bridge.active) {
                bridge.ipcSynced = false
                bridge.pollStatusSoon()
                if (bridge.vizWanted)
                    bridge.startVisStream()
            }
        }
    }

    Timer {
        id: statusPollTimer
        interval: 60
        repeat: false
        onTriggered: {
            if (bridge.active && bridge.socketReady && !statusProc.running)
                statusProc.running = true
        }
    }

    Timer {
        id: statusPollLoop
        interval: 500
        repeat: true
        running: bridge.active && bridge.socketReady
        onTriggered: {
            if (!statusProc.running)
                statusProc.running = true
        }
    }

    onActiveChanged: {
        if (!active) {
            stopVisStream()
            socketReady = false
            ipcSynced = false
            return
        }
        ensurePlayer()
    }

    Component.onCompleted: {
        if (active)
            ensurePlayer()
    }
}
