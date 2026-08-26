//@ pragma Env QT_QPA_PLATFORM=offscreen

import QtQuick
import Quickshell

ShellRoot {
    id: root

    readonly property bool interact: String(Quickshell.env("EVOPLAYER_SMOKE_INTERACT") || "") === "1"
    readonly property string pluginId: "seb.evoplayer"
    readonly property string playerServicePath: "vendor/evoplayer/qml/panel/Service.qml"
    readonly property string playerDashboardPath: "vendor/evoplayer/qml/panel/Player.qml"

    property var _services: ({})
    property var panelLoaders: ({})
    readonly property var barConfig: ({ output: "" })

    function fail(msg) {
        console.error("FAIL:", msg)
        Qt.quit()
    }

    function pass(msg) {
        console.log("PASS:", msg)
        Qt.quit()
    }

    function serviceFor(id) {
        return _services[String(id)] || null
    }

    function registerService(id, inst) {
        var next = ({})
        for (var k in _services)
            next[k] = _services[k]
        next[String(id)] = inst
        _services = next
    }

    function hide(pluginId) {
    }

    function isPluginOpen(pluginId) {
        return false
    }

    function ensurePlayerService() {
        var comp = Qt.createComponent(playerServicePath, Component.PreferSynchronous)
        if (comp.status !== Component.Ready) {
            fail("service component: " + comp.errorString())
            return null
        }
        var inst = comp.createObject(serviceHost, { shell: root })
        if (!inst) {
            fail("service createObject failed")
            return null
        }
        registerService(pluginId, inst)
        return inst
    }

    function loadPlayerDashboard() {
        playerLoader.source = playerDashboardPath
        playerLoader.active = true
    }

    function onPlayerReady() {
        if (!playerLoader.item) {
            fail("player loader item missing")
            return
        }
        playerLoader.item.shell = root
        var loaders = ({})
        loaders[pluginId] = playerLoader
        panelLoaders = loaders
        if (!interact) {
            pass("player loaded")
            return
        }
        interactTimer.restart()
    }

    function runInteraction() {
        interactTimer.attempts = (interactTimer.attempts || 0) + 1
        var monitor = serviceFor(pluginId)
        if (!monitor) {
            fail("player service missing")
            return
        }
        if (typeof monitor.ensurePlayer === "function")
            monitor.ensurePlayer()
        if (!monitor.ipcReady) {
            if (interactTimer.attempts > 100)
                fail("ipc not ready (socket " + String(Quickshell.env("XDG_RUNTIME_DIR") || "") + "/evoplayer.sock)")
            return
        }
        interactTimer.stop()
        var dash = playerLoader.item.dashboard
        if (!dash || typeof dash.togglePlayback !== "function") {
            fail("dashboard togglePlayback unavailable")
            return
        }
        dash.togglePlayback()
        dispatchTimer.restart()
    }

    Item {
        id: serviceHost
        visible: false
    }

    Loader {
        id: playerLoader
        active: false
        onLoaded: Qt.callLater(root.onPlayerReady)
        onStatusChanged: {
            if (status === Loader.Error)
                root.fail("player loader: " + String(errorString))
        }
    }

    Timer {
        id: interactTimer
        property int attempts: 0
        interval: 100
        repeat: true
        onTriggered: root.runInteraction()
    }

    Timer {
        id: dispatchTimer
        interval: 500
        onTriggered: root.pass("toggle dispatched")
    }

    Timer {
        id: hardTimeout
        interval: 15000
        running: true
        onTriggered: root.fail("timeout")
    }

    Component.onCompleted: {
        if (!ensurePlayerService())
            return
        loadPlayerDashboard()
    }
}
