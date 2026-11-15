import QtQuick
import "../compat"

Rectangle {
    id: chip

    signal activated()

    property string text: ""
    property string prefix: ""
    property string suffix: ""
    property bool inverted: false
    property int maxWidth: 220

    readonly property bool hovered: area.containsMouse

    height: 22
    width: Math.min(maxWidth, label.implicitWidth + 18)
    radius: height / 2
    color: inverted
        ? Theme.withOpacity(Theme.background, hovered ? 0.32 : 0.18)
        : (hovered ? Theme.foregroundSubtle : Theme.foregroundFaint)
    border.width: 1
    border.color: inverted ? "transparent" : (hovered ? Theme.foregroundBorder : "transparent")

    Text {
        id: label
        anchors.centerIn: parent
        width: Math.min(implicitWidth, chip.maxWidth - 18)
        textFormat: Text.PlainText
        elide: Text.ElideRight
        text: (chip.prefix ? chip.prefix + " " : "") + chip.text + (chip.suffix ? "  " + chip.suffix : "")
        color: chip.inverted ? Theme.background : (chip.hovered ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.78))
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
    }

    MouseArea {
        id: area
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: chip.activated()
    }
}
