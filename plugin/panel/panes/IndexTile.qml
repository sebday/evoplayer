import QtQuick
import "../compat"
import "../widgets"

Item {
    id: tile

    required property var view
    required property var row
    property bool round: false
    property bool starred: false
    property bool selected: false
    property string playKind: ""

    signal activated()
    signal contextActivated()

    readonly property bool hot: tileHover.hovered || selected

    Rectangle {
        anchors.fill: parent
        anchors.margins: 4
        radius: 8
        color: tile.selected ? Theme.withOpacity(Theme.border, 0.16) : (tileHover.hovered ? Theme.foregroundWash : "transparent")
        border.width: tile.selected ? 1 : 0
        border.color: Theme.border
    }

    Cover {
        id: face
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.top: parent.top
        anchors.topMargin: 12
        width: parent.width - 36
        height: width
        round: tile.round
        corner: 6
        name: String(tile.row.name || "")
        source: tile.view.rowThumbUrl(tile.row)
        decodeSize: Math.ceil(Math.max(96, width) * Screen.devicePixelRatio)
        scale: tile.hot ? 1.04 : 1

        Behavior on scale {
            NumberAnimation { duration: Theme.motionFast }
        }
    }

    Text {
        visible: tile.starred
        anchors.right: face.right
        anchors.top: face.top
        anchors.rightMargin: 6
        anchors.topMargin: 4
        textFormat: Text.PlainText
        text: "♥"
        color: Theme.liked
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        id: nameText
        anchors.top: face.bottom
        anchors.topMargin: 8
        anchors.horizontalCenter: parent.horizontalCenter
        width: parent.width - 16
        horizontalAlignment: Text.AlignHCenter
        textFormat: Text.PlainText
        text: String(tile.row.name || "")
        elide: Text.ElideRight
        color: tile.hot ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.85)
        font.family: Theme.fontFamily
        font.bold: true
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        anchors.top: nameText.bottom
        anchors.topMargin: 2
        anchors.horizontalCenter: parent.horizontalCenter
        textFormat: Text.PlainText
        text: tile.view.counted(tile.row.count, "track")
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeXs
    }

    HoverHandler {
        id: tileHover
    }

    MouseArea {
        id: cellArea
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        cursorShape: Qt.PointingHandCursor
        onClicked: function(mouse) {
            if (mouse.button === Qt.RightButton)
                tile.contextActivated()
            else
                tile.activated()
        }
    }

    Row {
        id: gridActions
        visible: tile.playKind !== "" && tileHover.hovered
        z: 3
        anchors.horizontalCenter: face.horizontalCenter
        anchors.bottom: face.bottom
        anchors.bottomMargin: 8
        spacing: 6

        Chip {
            text: "▶"
            color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
            onActivated: tile.view.playGridRow(tile.playKind, tile.row)
        }

        Chip {
            text: "+"
            color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
            onActivated: tile.view.addGridRow(tile.playKind, tile.row)
        }
    }
}
