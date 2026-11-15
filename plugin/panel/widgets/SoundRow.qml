import QtQuick
import "../compat"

Item {
    id: row

    signal clicked()
    signal activated()
    signal playRequested()

    property string art: ""
    property string upper: ""
    property string upperNote: ""
    property string title: ""
    property string genre: ""
    property real plays: 0
    property real likes: 0
    property string badge: ""
    property string clock: ""
    property string year: ""
    property bool selected: false
    property bool current: false
    property bool audible: false
    property int artSide: 46
    property int titlePt: 12
    default property alias actions: actionRow.data

    readonly property bool hovered: rowHover.hovered
    readonly property bool filled: selected || current
    readonly property color ink: filled ? Theme.background : Theme.foreground
    readonly property color dim: filled ? Theme.withOpacity(Theme.background, 0.72) : Theme.muted

    height: artSide + 14

    function compact(n) {
        n = Number(n) || 0
        if (n >= 1e6)
            return (n / 1e6).toFixed(n >= 1e7 ? 0 : 1).replace(/\.0$/, "") + "M"
        if (n >= 1e3)
            return (n / 1e3).toFixed(n >= 1e4 ? 0 : 1).replace(/\.0$/, "") + "k"
        return String(Math.round(n))
    }

    component Tag: Rectangle {
        property alias text: tagText.text
        property bool strong: false

        anchors.verticalCenter: parent ? parent.verticalCenter : undefined
        height: 22
        width: Math.min(150, tagText.implicitWidth + 18)
        radius: height / 2
        color: row.filled ? Theme.withOpacity(Theme.background, 0.18)
            : (strong ? Theme.withOpacity(Theme.good, 0.18) : Theme.foregroundFaint)

        Text {
            id: tagText
            anchors.centerIn: parent
            width: Math.min(implicitWidth, parent.width - 18)
            textFormat: Text.PlainText
            elide: Text.ElideRight
            color: row.filled ? Theme.background : (parent.strong ? Theme.good : Theme.withOpacity(Theme.foreground, 0.78))
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }
    }

    HoverHandler {
        id: rowHover
    }

    MouseArea {
        anchors.fill: parent
        onClicked: row.clicked()
        onDoubleClicked: row.activated()
    }

    Rectangle {
        anchors.fill: parent
        anchors.topMargin: 1
        anchors.bottomMargin: 1
        radius: 6
        color: row.selected ? Theme.good
            : (row.current ? Theme.mixColors(Theme.background, Theme.good, 0.65)
            : (row.hovered ? Theme.foregroundWash : "transparent"))
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: row.artSide + 18
        anchors.bottom: parent.bottom
        height: 1
        color: Theme.foregroundGhost
        visible: !row.filled && !row.hovered
    }

    Cover {
        id: cover
        x: 6
        anchors.verticalCenter: parent.verticalCenter
        width: row.artSide
        height: width
        corner: 4
        name: row.upper || row.title
        placeholder: row.filled ? Theme.withOpacity(Theme.background, 0.25) : Theme.mantle
        decodeSize: 96
        source: row.art

        Rectangle {
            anchors.fill: parent
            radius: 4
            color: Qt.rgba(0, 0, 0, 0.45)
            visible: row.hovered || (row.current && row.audible)

            Text {
                anchors.centerIn: parent
                textFormat: Text.PlainText
                text: row.current && row.audible ? "❚❚" : "▶"
                color: "white"
                font.pixelSize: Math.round(cover.height * 0.36)
            }

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: row.playRequested()
            }
        }
    }

    Column {
        anchors.left: cover.right
        anchors.leftMargin: 12
        anchors.right: meta.left
        anchors.rightMargin: 12
        anchors.verticalCenter: parent.verticalCenter
        spacing: 3

        Text {
            width: parent.width
            visible: text !== ""
            textFormat: Text.PlainText
            text: row.upper + (row.upper && row.upperNote ? "  ·  " : "") + row.upperNote
            elide: Text.ElideRight
            color: row.dim
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        Text {
            width: parent.width
            textFormat: Text.PlainText
            text: row.title
            elide: Text.ElideRight
            color: row.ink
            font.family: Theme.fontFamily
            font.pointSize: row.titlePt
            font.bold: row.current
        }
    }

    Row {
        id: meta
        anchors.right: parent.right
        anchors.rightMargin: 10
        anchors.verticalCenter: parent.verticalCenter
        spacing: 8

        Row {
            id: actionRow
            anchors.verticalCenter: parent.verticalCenter
            spacing: 6
            visible: row.hovered && children.length > 0
        }

        Tag {
            visible: row.badge !== ""
            text: row.badge
            strong: true
        }

        Tag {
            visible: !row.hovered && row.width > 700 && row.genre !== ""
            text: "# " + row.genre
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            visible: !row.hovered && row.width > 560 && (row.plays > 0 || row.likes > 0)
            textFormat: Text.PlainText
            text: (row.plays > 0 ? "▶ " + row.compact(row.plays) : "") + (row.plays > 0 && row.likes > 0 ? "   " : "") + (row.likes > 0 ? "♥ " + row.compact(row.likes) : "")
            color: row.dim
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        Column {
            anchors.verticalCenter: parent.verticalCenter
            spacing: 2

            Text {
                width: Math.max(implicitWidth, 44)
                horizontalAlignment: Text.AlignRight
                textFormat: Text.PlainText
                text: row.clock
                color: row.filled ? Theme.background : Theme.withOpacity(Theme.foreground, 0.75)
                font.family: Theme.fontFamily
                font.pointSize: 9
            }

            Text {
                width: Math.max(implicitWidth, 44)
                visible: text !== ""
                horizontalAlignment: Text.AlignRight
                textFormat: Text.PlainText
                text: row.year
                color: row.dim
                font.family: Theme.fontFamily
                font.pointSize: 9
            }
        }
    }
}
