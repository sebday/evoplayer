import QtQuick
import "../compat"

Item {
    id: root

    required property var view

    Column {
        anchors.fill: parent
        spacing: 4

        Rectangle {
            id: searchBox
            width: parent.width
            height: 26
            radius: Theme.fieldsetCornerRadius
            color: Theme.mantle
            border.width: 1
            border.color: searchField.activeFocus ? Theme.good : Theme.inactiveBorder

            Text {
                anchors.fill: searchField
                visible: searchField.text === ""
                text: "search discogs"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                verticalAlignment: Text.AlignVCenter
                elide: Text.ElideRight
            }

            TextInput {
                id: searchField
                anchors.fill: parent
                anchors.leftMargin: 8
                anchors.rightMargin: clearMark.visible ? clearMark.width + 2 : 8
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                clip: true
                selectByMouse: true
                selectionColor: Theme.good
                selectedTextColor: Theme.background
                onTextChanged: {
                    if (view.artQuery !== text)
                        view.artQuery = text
                }
                onActiveFocusChanged: {
                    view.textCapture = activeFocus
                    view.artQueryFocus = activeFocus
                }
                Keys.priority: Keys.BeforeItem
                Keys.onPressed: function(event) {
                    if (view.dispatch(event))
                        event.accepted = true
                }
            }

            Item {
                id: clearMark
                visible: searchField.text !== ""
                width: 22
                height: parent.height
                anchors.right: parent.right
                z: 1

                Text {
                    anchors.centerIn: parent
                    text: "×"
                    color: clearArea.containsMouse ? Theme.foreground : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }

                MouseArea {
                    id: clearArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        searchField.text = ""
                        searchField.forceActiveFocus()
                    }
                }
            }
        }

        ListView {
            id: list
            width: parent.width
            height: parent.height - searchBox.height - 4
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
    }

    Connections {
        target: view
        function onArtIdxChanged() {
            list.currentIndex = view.artIdx
            if (view.artIdx >= 0 && view.artIdx < list.count)
                list.positionViewAtIndex(view.artIdx, ListView.Contain)
        }
        function onArtQueryChanged() {
            if (!searchField.activeFocus && searchField.text !== view.artQuery)
                searchField.text = view.artQuery
        }
        function onArtQueryFocusChanged() {
            if (view.artQueryFocus)
                searchField.forceActiveFocus()
            else if (searchField.activeFocus)
                searchField.focus = false
        }
        function onModeChanged() {
            if (view.mode === "art" && searchField.text !== view.artQuery)
                searchField.text = view.artQuery
        }
    }
}
