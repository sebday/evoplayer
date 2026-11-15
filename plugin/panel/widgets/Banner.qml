import QtQuick
import "../compat"

Item {
    id: banner

    signal play()

    property string caption: ""
    property string title: ""
    property string subtitle: ""
    property string stats: ""
    property string art: ""
    property string blur: art
    property bool round: false
    property bool showPlay: true
    property bool pause: false
    readonly property bool showArt: art !== "" && width > 520
    default property alias content: extra.data

    implicitHeight: Math.max(textCol.implicitHeight + 40, showArt ? 176 : 0)

    BlurBackdrop {
        anchors.fill: parent
        source: banner.blur
    }

    PlayCircle {
        id: playCircle
        visible: banner.showPlay
        x: 22
        y: 22
        pause: banner.pause
        onActivated: banner.play()
    }

    Column {
        id: textCol
        anchors.left: banner.showPlay ? playCircle.right : parent.left
        anchors.leftMargin: banner.showPlay ? 18 : 22
        anchors.right: artSlot.left
        anchors.rightMargin: banner.showArt ? 18 : 0
        anchors.top: parent.top
        anchors.topMargin: 18
        spacing: 6

        Rectangle {
            visible: banner.caption !== ""
            height: 18
            width: captionText.implicitWidth + 14
            radius: 3
            color: Theme.withOpacity(Theme.foreground, 0.12)

            Text {
                id: captionText
                anchors.centerIn: parent
                textFormat: Text.PlainText
                text: banner.caption
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                font.bold: true
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1
            }
        }

        Text {
            width: parent.width
            textFormat: Text.PlainText
            text: banner.title
            color: Theme.foreground
            elide: Text.ElideRight
            wrapMode: Text.Wrap
            maximumLineCount: 2
            font.family: Theme.fontFamily
            font.bold: true
            font.pixelSize: Theme.fontSize3xl
        }

        Text {
            width: parent.width
            visible: text !== ""
            textFormat: Text.PlainText
            text: banner.subtitle
            color: Theme.foreground
            elide: Text.ElideRight
            font.family: Theme.fontFamily
            font.bold: true
            font.pixelSize: Theme.fontSizeM
        }

        Text {
            width: parent.width
            visible: text !== ""
            textFormat: Text.PlainText
            text: banner.stats
            color: Theme.withOpacity(Theme.foreground, 0.7)
            elide: Text.ElideRight
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        Column {
            id: extra
            width: parent.width
            topPadding: 4
            spacing: 10
            visible: children.length > 0
        }
    }

    Cover {
        id: artSlot
        anchors.right: parent.right
        anchors.rightMargin: banner.showArt ? 22 : 0
        anchors.verticalCenter: parent.verticalCenter
        height: parent.height - 40
        width: banner.showArt ? height : 0
        visible: banner.showArt
        source: banner.art
        round: banner.round
        name: banner.title
        decodeSize: 400
        cached: false
    }
}
