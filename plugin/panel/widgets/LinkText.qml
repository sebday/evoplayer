import QtQuick
import "../compat"

Text {
    id: link

    signal activated()
    signal contextActivated()

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
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        cursorShape: Qt.PointingHandCursor
        onClicked: function(mouse) {
            if (mouse.button === Qt.RightButton)
                link.contextActivated()
            else
                link.activated()
        }
    }
}
