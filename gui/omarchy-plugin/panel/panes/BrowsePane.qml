import QtQuick
import "../compat"

Fieldset {
    id: pane

    required property var view

    number: 2
    legend: "browse"
    legendRight: view.searchQuery ? "" : view.browsePath
    active: view.pane === "browse" || view.pane === "search"
    hints: [
        { key: "⏎", label: "play" },
        { key: "d", label: "add dir" }
    ]

    Column {
        anchors.fill: parent
        spacing: 4

        TextInput {
            id: searchField
            width: parent.width
            height: 20
            visible: view.pane === "search"
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
            onActiveFocusChanged: view.textCapture = activeFocus
            Keys.priority: Keys.BeforeItem
            Keys.onPressed: function(event) {
                if (view.dispatch(event))
                    event.accepted = true
            }
        }

        Text {
            textFormat: Text.PlainText
            visible: view.pane !== "search" && view.browsePath !== "" && !view.searchQuery
            width: parent.width
            elide: Text.ElideLeft
            text: view.browsePath
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        ListView {
            id: list
            width: parent.width
            height: parent.height - (searchField.visible ? searchField.height + 4 : 0) - (view.pane !== "search" && view.browsePath !== "" && !view.searchQuery ? 18 : 0)
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
                    anchors.fill: parent
                    visible: modelData.kind !== "rule"
                    spacing: 4

                    Text {
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
                        width: Math.max(20, list.width - 56)
                        height: 22
                        verticalAlignment: Text.AlignVCenter
                        text: String(modelData.label || "")
                        elide: Text.ElideRight
                        color: selected ? Theme.border : (modelData.type === "dir" ? Theme.muted : Theme.foreground)
                        font.family: Theme.fontFamily
                        font.bold: selected
                        font.pixelSize: Theme.fontSizeM
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: 36
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
                        if (modelData.kind === "entry" && modelData.type === "dir")
                            view.loadBrowse(String(modelData.path || ""))
                        else
                            view.playBrowseAt(index)
                    }
                }
            }
        }
    }
}
