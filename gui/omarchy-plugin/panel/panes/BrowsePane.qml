import QtQuick
import "../compat"

Fieldset {
    id: pane

    required property var view

    number: 2
    legend: "browse"
    legendRight: view.searchQuery ? "" : (view.browsePath || (view.browseFiles ? "filesystem" : ""))
    active: view.pane === "browse" || view.pane === "search"
    hints: [
        { key: "⏎", label: "play" },
        { key: "d", label: "add dir" }
    ]

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
                text: "search"
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
                anchors.rightMargin: 8
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                clip: true
                selectByMouse: true
                selectionColor: Theme.good
                selectedTextColor: Theme.background
                onTextChanged: {
                    if (view.searchQuery !== text)
                        view.setSearch(text)
                }
                onActiveFocusChanged: {
                    view.textCapture = activeFocus
                    if (activeFocus && view.pane !== "search")
                        view.openSearch()
                }
                Keys.priority: Keys.BeforeItem
                Keys.onPressed: function(event) {
                    if (view.dispatch(event))
                        event.accepted = true
                }
            }
        }

        Text {
            textFormat: Text.PlainText
            visible: view.pane !== "search" && (view.browsePath !== "" || view.browseFiles) && !view.searchQuery
            width: parent.width
            elide: Text.ElideLeft
            text: view.browsePath || "filesystem"
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        ListView {
            id: list
            width: parent.width
            height: parent.height - searchBox.height - 4 - (view.pane !== "search" && (view.browsePath !== "" || view.browseFiles) && !view.searchQuery ? 18 : 0)
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.sidebar
            onHeightChanged: view.browsePage = Math.max(1, Math.floor(height / 22))

            Text {
                textFormat: Text.PlainText
                anchors.left: parent.left
                anchors.top: parent.top
                visible: list.count === 0
                text: view.scanRunning ? "Please wait for library to finish scanning" : (view.searchQuery ? "no matches" : "loading…")
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                width: parent.width
                wrapMode: Text.WordWrap
            }

            Connections {
                target: view
                function onBrowseIdxChanged() {
                    if (view.browseIdx >= 0 && view.browseIdx < list.count) {
                        list.currentIndex = view.browseIdx
                        list.positionViewAtIndex(view.browseIdx, ListView.Contain)
                    }
                }
                function onPaneChanged() {
                    if (view.pane === "search") {
                        if (searchField.text !== view.searchQuery)
                            searchField.text = view.searchQuery
                        searchField.forceActiveFocus()
                    }
                }
                function onSearchQueryChanged() {
                    if (!searchField.activeFocus && searchField.text !== view.searchQuery)
                        searchField.text = view.searchQuery
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: list.width
                height: modelData.kind === "rule" ? 10 : 22

                readonly property bool selected: (view.pane === "browse" || view.pane === "search") && index === view.browseIdx && modelData.kind !== "rule"

                Rectangle {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    height: 1
                    visible: modelData.kind === "rule"
                    color: Theme.border
                }

                Row {
                    id: line
                    anchors.fill: parent
                    visible: modelData.kind !== "rule"
                    spacing: 4

                    Text {
                        id: mark
                        textFormat: Text.PlainText
                        width: 12
                        height: 22
                        verticalAlignment: Text.AlignVCenter
                        text: selected ? ">" : " "
                        color: selected ? Theme.border : Theme.muted
                        font.family: Theme.fontFamily
                        font.bold: true
                        font.pixelSize: Theme.fontSizeM
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.max(20, line.width - mark.width - line.spacing * 2 - (countText.visible ? countText.width : 0))
                        height: 22
                        verticalAlignment: Text.AlignVCenter
                        text: String(modelData.label || "")
                        elide: Text.ElideRight
                        color: selected ? Theme.border : (modelData.type === "dir" || modelData.kind === "tool" ? Theme.muted : Theme.foreground)
                        font.family: Theme.fontFamily
                        font.bold: selected
                        font.pixelSize: Theme.fontSizeM
                    }

                    Text {
                        id: countText
                        textFormat: Text.PlainText
                        visible: modelData.count > 0
                        width: visible ? implicitWidth : 0
                        height: 22
                        verticalAlignment: Text.AlignVCenter
                        horizontalAlignment: Text.AlignRight
                        text: modelData.count > 0 ? String(modelData.count) : ""
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    enabled: modelData.kind !== "rule"
                    preventStealing: true
                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                    onClicked: function(mouse) {
                        if (mouse.button === Qt.RightButton) {
                            view.clickBrowse(index)
                            view.playBrowseAt(index)
                            return
                        }
                        view.clickBrowse(index)
                    }
                    onDoubleClicked: {
                        view.clickBrowse(index)
                        if (modelData.id === "filesystem" || (modelData.kind === "entry" && modelData.type === "dir"))
                            view.enterFolder()
                        else
                            view.playBrowseAt(index)
                    }
                }
            }
        }
    }
}
