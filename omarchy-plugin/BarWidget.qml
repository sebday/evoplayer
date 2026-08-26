import QtQuick
import Quickshell
import qs.Ui
import qs.Commons
import "panel/compat/PluginIds.js" as PluginIds

BarWidget {
    id: root
    moduleName: PluginIds.pluginId

    readonly property var playerService: bar?.shell?.serviceFor(PluginIds.pluginId)
    readonly property var player: playerService && playerService.player ? playerService.player : ({})
    readonly property bool hasTrack: String(player.path || "") !== ""
    readonly property bool isPlaying: String(player.state || "") === "playing"
    readonly property string playIcon: isPlaying ? "󰏤" : "󰐊"
    readonly property string title: String(player.title || "")
    readonly property string artist: String(player.artist || "")

    property real maxLabelWidth: 180

    visible: hasTrack
    implicitWidth: hasTrack ? row.implicitWidth + Style.space(14) : 0
    implicitHeight: barSize

    Row {
        id: row
        anchors.centerIn: parent
        spacing: Style.space(6)

        Text {
            id: glyph
            anchors.verticalCenter: parent.verticalCenter
            text: root.playIcon
            color: root.bar.barForeground
            opacity: root.isPlaying ? pulseOpacity : 0.45
            font.family: root.bar.fontFamily
            font.pixelSize: Style.font.body
            property real pulseOpacity: 1

            SequentialAnimation on pulseOpacity {
                running: root.isPlaying
                loops: Animation.Infinite
                NumberAnimation { from: 1.0; to: 0.42; duration: 880; easing.type: Easing.InOutSine }
                NumberAnimation { from: 0.42; to: 1.0; duration: 880; easing.type: Easing.InOutSine }
            }
        }

        Item {
            id: scrollClip
            width: Math.min(root.maxLabelWidth, labelText.implicitWidth)
            height: glyph.height
            clip: true
            anchors.verticalCenter: parent.verticalCenter
            visible: !root.bar.vertical && root.title !== ""

            Text {
                id: labelText
                text: root.title + (root.artist ? "  ·  " + root.artist : "")
                color: root.bar.barForeground
                font.family: root.bar.fontFamily
                font.pixelSize: Style.font.body
                anchors.verticalCenter: parent.verticalCenter

                property bool needsScroll: implicitWidth > scrollClip.width

                NumberAnimation on x {
                    running: labelText.needsScroll && !root.bar.vertical
                    loops: Animation.Infinite
                    duration: Math.max(6000, labelText.implicitWidth * 25)
                    from: scrollClip.width
                    to: -labelText.implicitWidth
                    easing.type: Easing.Linear
                }
            }
        }
    }

    MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: root.hasTrack ? Qt.PointingHandCursor : Qt.ArrowCursor
        acceptedButtons: Qt.LeftButton | Qt.MiddleButton | Qt.RightButton

        onClicked: function(mouse) {
            if (!root.playerService)
                return
            if (mouse.button === Qt.MiddleButton) {
                root.playerService.runTransport("next")
                return
            }
            if (mouse.button === Qt.RightButton) {
                if (root.bar && root.bar.shell)
                    root.bar.shell.summon(PluginIds.pluginId, "{}")
                return
            }
            root.playerService.runTransport("toggle")
        }

        onWheel: function(wheel) {
            if (!root.playerService)
                return
            root.playerService.runTransport(wheel.angleDelta.y > 0 ? "previous" : "next")
        }

        onDoubleClicked: function(mouse) {
            if (mouse.button !== Qt.LeftButton)
                return
            if (root.bar && root.bar.shell)
                root.bar.shell.summon(PluginIds.pluginId, "{}")
        }
    }
}
