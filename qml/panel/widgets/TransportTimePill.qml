import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Rectangle {
    required property var dashboard

    property string label: ""
    property bool highlight: false
    property bool dark: false
    property bool compact: false

    radius: compact ? Theme.radiusM : Theme.radiusL
    color: dark
        ? Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.98)
        : Theme.foregroundWash
    border.color: dark ? Theme.foregroundSubtle : Theme.foregroundDivider
    border.width: 1
    implicitWidth: pillText.implicitWidth + (compact ? 12 : 20)
    implicitHeight: pillText.implicitHeight + (compact ? 4 : 10)
    Layout.alignment: Qt.AlignVCenter

    Text {
        id: pillText
        anchors.centerIn: parent
        text: parent.label
        color: parent.highlight ? Theme.accent : Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: parent.compact ? dashboard.libraryFont : dashboard.hintFont
        font.bold: parent.highlight && Theme.fontBold
        opacity: parent.highlight ? 1 : 0.65
    }
}
