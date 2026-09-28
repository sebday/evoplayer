import QtQuick
import "../compat"

ListView {
    id: list

    required property var view

    clip: true
    boundsBehavior: Flickable.StopAtBounds
    spacing: 0
    highlightMoveDuration: 0
    model: view.artHits || []

    Text {
        textFormat: Text.PlainText
        anchors.centerIn: parent
        visible: view.artBusy && list.count === 0
        text: "searching discogs…"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        textFormat: Text.PlainText
        anchors.centerIn: parent
        visible: !view.artBusy && list.count === 0
        text: "no covers"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Connections {
        target: view
        function onArtIdxChanged() {
            list.currentIndex = view.artIdx
            if (view.artIdx >= 0)
                list.positionViewAtIndex(view.artIdx, ListView.Contain)
        }
    }

    delegate: Item {
        required property int index
        required property var modelData

        width: list.width
        height: 22

        readonly property bool selected: view.pane === "playlist" && index === view.artIdx

        Rectangle {
            anchors.fill: parent
            color: selected ? Theme.good : "transparent"
        }

        Row {
            anchors.fill: parent
            anchors.leftMargin: 2
            anchors.rightMargin: 4
            spacing: 6

            Text {
                textFormat: Text.PlainText
                width: 14
                text: selected ? ">" : " "
                color: selected ? Theme.background : Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                textFormat: Text.PlainText
                width: parent.width - 70
                text: view.artHitLabel(modelData)
                color: selected ? Theme.background : Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                elide: Text.ElideRight
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                textFormat: Text.PlainText
                width: 44
                horizontalAlignment: Text.AlignRight
                text: String(modelData.year || "")
                color: selected ? Theme.background : Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        MouseArea {
            anchors.fill: parent
            onClicked: view.clickArt(index)
            onDoubleClicked: view.applyArt("album")
        }
    }
}
