import QtQuick
import "../compat"

Text {
    id: link

    signal activated()

    property color baseColor: Theme.muted
    property color hoverColor: Theme.foreground
    readonly property bool linkEnabled: text !== ""
    readonly property bool hovered: area.containsMouse

    textFormat: Text.PlainText
    elide: Text.ElideRight
    color: hovered ? hoverColor : baseColor
    font.family: Theme.fontFamily
    font.underline: hovered

    MouseArea {
        id: area
        width: Math.min(link.width, link.implicitWidth)
        height: link.height
        enabled: link.linkEnabled
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: link.activated()
    }
}
