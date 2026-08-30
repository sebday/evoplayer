import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: btn
    property string icon: ""
    property bool accent: false
    property bool dimmed: false
    property bool liked: false
    property bool smallGlyph: false
    property int btnSize: dashboard.transportBtnSize
    property real iconScale: 1
    property int iconOffsetX: 0
    property int iconOffsetY: 0
    signal activated()

    implicitWidth: btnSize
    implicitHeight: btnSize
    Layout.preferredWidth: btnSize
    Layout.preferredHeight: btnSize
    Layout.alignment: Qt.AlignVCenter

    Rectangle {
        anchors.fill: parent
        radius: btnSize / 2
        color: Theme.foregroundWash
        visible: transportMouse.containsMouse
    }

    Text {
        anchors.centerIn: parent
        anchors.horizontalCenterOffset: btn.iconOffsetX
        anchors.verticalCenterOffset: btn.iconOffsetY
        text: btn.icon
        color: liked ? Theme.urgent : (accent ? Theme.accent : Theme.foreground)
        opacity: dimmed
            ? (transportMouse.containsMouse ? 0.55 : 0.35)
            : (liked ? 1 : (transportMouse.containsMouse ? 1 : 0.9))
        font.family: Theme.fontFamily
        font.pixelSize: Math.max(10, Math.round((smallGlyph
            ? dashboard.transportSecondaryIconFont
            : dashboard.transportIconFont) * btn.iconScale))
    }

    MouseArea {
        id: transportMouse
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: btn.activated()
    }
}
