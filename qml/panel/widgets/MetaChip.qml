import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Rectangle {
    required property var dashboard

    property string label: ""
    property bool accent: false
    property bool tinted: false
    property color tintColor: Theme.accent
    property bool clickable: false
    property int maxLabelWidth: 0
    property int fontSize: dashboard ? dashboard.listFont : Theme.fontSizeS
    signal activated()

    readonly property color chipTint: tinted ? tintColor : (accent ? Theme.accent : Theme.foreground)
    readonly property bool chipTinted: tinted || accent

    radius: 10
    color: chipTinted
        ? Theme.withOpacity(chipTint, (clickable && chipMouse.containsMouse) ? 0.22 : 0.14)
        : (clickable && chipMouse.containsMouse) ? Theme.foregroundHoverWash : Theme.foregroundWash
    border.color: chipTinted
        ? Theme.withOpacity(chipTint, 0.38)
        : Theme.foregroundDivider
    border.width: 1
    implicitWidth: (maxLabelWidth > 0 ? Math.min(chipText.implicitWidth, maxLabelWidth) : chipText.implicitWidth) + 16
    implicitHeight: chipText.implicitHeight + 6

    Text {
        id: chipText
        anchors.centerIn: parent
        width: parent.maxLabelWidth > 0 ? parent.maxLabelWidth : implicitWidth
        text: parent.label
        color: parent.chipTinted ? parent.chipTint : Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: parent.fontSize
        font.bold: parent.chipTinted && Theme.fontBold
        opacity: parent.chipTinted ? 0.95 : 0.68
        elide: parent.maxLabelWidth > 0 ? Text.ElideRight : Text.ElideNone
        horizontalAlignment: Text.AlignHCenter
    }

    MouseArea {
        id: chipMouse
        anchors.fill: parent
        z: 1
        enabled: parent.clickable
        hoverEnabled: true
        preventStealing: true
        cursorShape: parent.clickable ? Qt.PointingHandCursor : Qt.ArrowCursor
        onClicked: function(mouse) {
            mouse.accepted = true
            parent.activated()
        }
    }
}
