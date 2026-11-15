import QtQuick
import "../compat"

// The old sidebar, laid out as one scrolling row of tabs.
Item {
    id: pane

    required property var view

    Row {
        anchors.fill: parent
        spacing: 10

        Rectangle {
            id: searchBox
            width: 168
            height: parent.height
            radius: height / 2
            color: Theme.foregroundFaint
            border.width: 1
            border.color: searchField.activeFocus ? Theme.good : "transparent"

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
                anchors.leftMargin: 12
                anchors.rightMargin: clearMark.visible ? clearMark.width + 2 : 12
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
            width: parent.width - searchBox.width - parent.spacing
            height: parent.height
            orientation: ListView.Horizontal
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            highlightFollowsCurrentItem: false
            spacing: 4
            model: view.sidebar
            onWidthChanged: view.browsePage = Math.max(1, Math.floor(width / 110))

            WheelHandler {
                onWheel: function(event) {
                    var d = event.angleDelta.y !== 0 ? event.angleDelta.y : event.angleDelta.x
                    var maxX = Math.max(0, list.contentWidth - list.width)
                    list.contentX = Math.max(0, Math.min(maxX, list.contentX - d))
                }
            }

            Text {
                anchors.verticalCenter: parent.verticalCenter
                visible: list.count === 0
                textFormat: Text.PlainText
                text: view.scanRunning ? "Please wait for library to finish scanning" : "loading…"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
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
                id: tab
                required property int index
                required property var modelData

                readonly property bool rule: modelData.kind === "rule"
                readonly property bool selected: (view.pane === "browse" || view.pane === "search") && index === view.browseIdx && !rule
                readonly property bool currentPage: modelData.kind === "page" && view.page && view.page.kind === modelData.id
                readonly property bool shownList: modelData.kind === "playlist" && view.libraryView === "" && !view.page
                    && view.mode !== "download" && view.mode !== "discover" && !view.searchActive()
                    && modelData.id === (view.shownPlaylist !== "" ? view.shownPlaylist : "current")
                readonly property bool shownFiles: modelData.id === "filesystem" && view.libraryView === "files" && !view.searchActive()
                readonly property bool shownLists: modelData.id === "playlists" && view.libraryView === "lists" && !view.searchActive()
                readonly property bool shownTool: modelData.kind === "tool" && view.mode === modelData.id
                readonly property bool on: selected || currentPage || shownList || shownFiles || shownLists || shownTool

                width: rule ? 14 : line.implicitWidth + 20
                height: list.height

                Rectangle {
                    anchors.centerIn: parent
                    visible: tab.rule
                    width: 1
                    height: 14
                    color: Theme.foregroundDivider
                }

                Rectangle {
                    anchors.fill: parent
                    visible: !tab.rule
                    radius: height / 2
                    color: tab.on
                        ? Theme.withOpacity(Theme.border, 0.22)
                        : (tabArea.containsMouse ? Theme.foregroundFaint : "transparent")
                }

                Row {
                    id: line
                    anchors.centerIn: parent
                    visible: !tab.rule
                    spacing: 6

                    Text {
                        textFormat: Text.PlainText
                        height: tab.height
                        verticalAlignment: Text.AlignVCenter
                        text: String(modelData.label || "")
                        color: tab.on ? Theme.border
                            : (modelData.type === "dir" || modelData.kind === "tool" || modelData.kind === "page" ? Theme.muted : Theme.foreground)
                        font.family: Theme.fontFamily
                        font.bold: tab.on
                        font.pixelSize: Theme.fontSizeM
                    }

                    Text {
                        visible: modelData.count > 0
                        textFormat: Text.PlainText
                        height: tab.height
                        verticalAlignment: Text.AlignVCenter
                        text: modelData.count > 0 ? String(modelData.count) : ""
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }
                }

                MouseArea {
                    id: tabArea
                    anchors.fill: parent
                    enabled: !tab.rule
                    hoverEnabled: true
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
