import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: rowIconBtn
    property string icon: ""
    property string tooltip: ""
    property color iconColor: Theme.foreground
    property real opacityIdle: 0.42
    property real opacityHover: 0.9
    property bool enabled: true
    property bool flashing: false
    property int iconSize: dashboard && dashboard.listFont !== undefined
        ? dashboard.listFont
        : Theme.fontSizeL
    property int hitPadding: 6
    signal activated()

    function restingOpacity() {
        if (!rowIconBtn.enabled)
            return 0.2
        return rowIconMouse.containsMouse ? rowIconBtn.opacityHover : rowIconBtn.opacityIdle
    }

    implicitWidth: 22
    implicitHeight: 22
    Layout.preferredWidth: 22
    Layout.preferredHeight: 22
    Layout.alignment: Qt.AlignVCenter
    z: 2

    Text {
        id: rowIconGlyph
        anchors.centerIn: parent
        text: rowIconBtn.icon
        color: rowIconBtn.iconColor
        opacity: rowIconBtn.flashing ? 1 : rowIconBtn.restingOpacity()
        font.family: Theme.fontFamily
        font.pixelSize: rowIconBtn.iconSize

        SequentialAnimation on opacity {
            running: rowIconBtn.flashing
            loops: Animation.Infinite
            NumberAnimation {
                from: 0.35
                to: 1.0
                duration: 600
                easing.type: Easing.InOutSine
            }
            NumberAnimation {
                from: 1.0
                to: 0.35
                duration: 600
                easing.type: Easing.InOutSine
            }
        }
    }

    Connections {
        target: rowIconBtn
        function onFlashingChanged() {
            if (!rowIconBtn.flashing)
                rowIconGlyph.opacity = rowIconBtn.restingOpacity()
        }
        function onEnabledChanged() {
            if (!rowIconBtn.flashing)
                rowIconGlyph.opacity = rowIconBtn.restingOpacity()
        }
    }

    MouseArea {
        id: rowIconMouse
        anchors.fill: parent
        anchors.margins: -rowIconBtn.hitPadding
        hoverEnabled: true
        enabled: rowIconBtn.enabled
        cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
        onClicked: function(mouse) {
            mouse.accepted = true
            rowIconBtn.activated()
        }
    }

    BriefTooltip {
        dashboard: rowIconBtn.dashboard
        show: rowIconMouse.containsMouse
        text: rowIconBtn.tooltip
    }
}
