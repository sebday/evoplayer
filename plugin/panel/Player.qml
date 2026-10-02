import Quickshell
import QtQuick
import "compat"
import "compat/PluginIds.js" as PluginIds

Item {
    id: root

    property var shell: null
    property bool opened: false
    property bool closingFromHost: false

    readonly property var dashScreen: Util.screenForOutput(
        shell && shell.barConfig ? shell.barConfig.output : "",
        ""
    )

    function close() {
        closingFromHost = true
        opened = false
        closingFromHost = false
    }

    function open(payloadJson) {
        closingFromHost = false
        if (dashScreen && dashWindow)
            dashWindow.screen = dashScreen
        opened = true
    }

    function toggle() {
        if (opened)
            close()
        else
            open("{}")
    }

    function requestClose() {
        if (shell && typeof shell.hide === "function")
            shell.hide(PluginIds.pluginId)
        else
            close()
    }

    function applyDisplayArt(trackPath, artPath) {
        if (playerView)
            playerView.applyDisplayArt(trackPath, artPath)
    }

    function forceKeyFocus() {
        keySurface.forceActiveFocus()
    }

    onOpenedChanged: {
        if (opened)
            playerView.activate()
        else
            playerView.deactivate()
        if (opened)
            Qt.callLater(function() { keySurface.forceActiveFocus() })
    }

    FloatingWindow {
        id: dashWindow
        visible: root.opened && root.dashScreen !== null
        title: PluginIds.pluginId
        screen: root.dashScreen
        color: Theme.background
        implicitWidth: 1180
        implicitHeight: 720
        minimumSize: Qt.size(720, 420)

        onVisibleChanged: {
            if (visible)
                Qt.callLater(function() { keySurface.forceActiveFocus() })
            else if (root.opened && !root.closingFromHost)
                root.requestClose()
        }

        Item {
            id: keySurface
            anchors.fill: parent
            focus: true

            MouseArea {
                anchors.fill: parent
                propagateComposedEvents: true
                onPressed: function(mouse) {
                    keySurface.forceActiveFocus()
                    mouse.accepted = false
                }
            }

            Keys.onPressed: function(event) {
                playerView.handleKey(event)
            }

            PlayerView {
                id: playerView
                anchors.fill: parent
                shell: root.shell
                host: root
            }
        }
    }
}
