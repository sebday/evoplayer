import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: transportProgress
    property int minBarWidth: 72
    property int preferredBarWidth: 140
    Layout.preferredWidth: visible ? preferredBarWidth : 0
    Layout.minimumWidth: visible ? minBarWidth : 0
    Layout.maximumWidth: visible ? (dashboard.compactLayout ? -1 : 240) : 0
    Layout.fillWidth: visible
    Layout.alignment: Qt.AlignVCenter
    implicitHeight: 20

    readonly property real value: dashboard.progress
    readonly property bool seekable: Number(dashboard.player.duration) > 0

    Rectangle {
        id: transportProgressTrack
        anchors.verticalCenter: parent.verticalCenter
        width: parent.width
        height: 4
        radius: 2
        color: Theme.foregroundFaint

        Rectangle {
            width: parent.width * transportProgress.value
            height: parent.height
            radius: parent.radius
            color: Theme.accent
            opacity: transportProgress.seekable ? 1 : 0.35
        }
    }

    MouseArea {
        anchors.fill: parent
        enabled: transportProgress.seekable
        cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
        onPressed: function(mouse) {
            dashboard.previewSeekFromX(mouse.x, width)
        }
        onPositionChanged: function(mouse) {
            if (pressed)
                dashboard.previewSeekFromX(mouse.x, width)
        }
        onReleased: function(mouse) {
            dashboard.commitSeekFromX(mouse.x, width)
        }
    }
}
