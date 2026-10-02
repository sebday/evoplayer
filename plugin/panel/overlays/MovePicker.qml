import QtQuick
import "../compat"

ListView {
    id: list

    required property var view

    clip: true
    boundsBehavior: Flickable.StopAtBounds
    highlightMoveDuration: 0
    model: view.moveFolders || []

    Text {
        textFormat: Text.PlainText
        anchors.centerIn: parent
        visible: !view.moveBusy && list.count === 0
        text: "no folders"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        textFormat: Text.PlainText
        anchors.centerIn: parent
        visible: view.moveBusy
        text: "moving…"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Connections {
        target: view
        function onMoveIdxChanged() {
            list.currentIndex = view.moveIdx
            if (view.moveIdx >= 0)
                list.positionViewAtIndex(view.moveIdx, ListView.Contain)
        }
    }

    delegate: Item {
        required property int index
        required property string modelData

        width: list.width
        height: 22

        readonly property bool selected: view.pane === "playlist" && index === view.moveIdx

        Rectangle {
            anchors.fill: parent
            color: selected ? Theme.good : "transparent"
        }

        Text {
            textFormat: Text.PlainText
            anchors.fill: parent
            anchors.leftMargin: 4
            anchors.rightMargin: 4
            verticalAlignment: Text.AlignVCenter
            text: (selected ? "> " : "  ") + modelData
            color: selected ? Theme.background : Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            elide: Text.ElideRight
        }

        MouseArea {
            anchors.fill: parent
            onClicked: view.clickMove(index)
            onDoubleClicked: view.applyMove()
        }
    }
}
