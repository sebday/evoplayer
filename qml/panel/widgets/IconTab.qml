import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: iconTab
    property string icon: ""
    property bool active: false
    property bool spinning: false
    signal activated()

    function restingOpacity() {
        return iconTab.active ? 1 : 0.78
    }

    implicitWidth: dashboard.genreTabHeight
    implicitHeight: dashboard.genreTabHeight
    clip: true

    Connections {
        target: iconTab
        function onSpinningChanged() {
            if (!iconTab.spinning)
                iconTabGlyph.opacity = iconTab.restingOpacity()
        }
        function onActiveChanged() {
            if (!iconTab.spinning)
                iconTabGlyph.opacity = iconTab.restingOpacity()
        }
    }

    Rectangle {
        anchors.fill: parent
        radius: 6
        color: iconTab.active
            ? "transparent"
            : (iconTabMouse.containsMouse
                ? Theme.foregroundWash
                : "transparent")
    }

    Text {
        id: iconTabGlyph
        anchors.centerIn: parent
        text: iconTab.icon
        color: iconTab.active || iconTab.spinning ? Theme.accent : Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSize7xl
        opacity: iconTab.spinning ? 1 : iconTab.restingOpacity()

        SequentialAnimation on opacity {
            running: iconTab.spinning
            loops: Animation.Infinite
            NumberAnimation {
                from: iconTab.active ? 0.55 : 0.35
                to: 1.0
                duration: 600
                easing.type: Easing.InOutSine
            }
            NumberAnimation {
                from: 1.0
                to: iconTab.active ? 0.55 : 0.35
                duration: 600
                easing.type: Easing.InOutSine
            }
        }
    }

    MouseArea {
        id: iconTabMouse
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: function(mouse) {
            mouse.accepted = true
            iconTab.activated()
        }
    }
}
