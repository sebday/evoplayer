import QtQuick
import "../compat"
import "../widgets"

// Library menu along the top. The folder tree drops down under filesystem.
Item {
    id: pane

    required property var view

    readonly property int barHeight: 36

    Row {
        id: bar
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: pane.barHeight
        spacing: 10

        Item {
            id: transport
            width: transportRow.implicitWidth
            height: parent.height

            Row {
                id: transportRow
                anchors.verticalCenter: parent.verticalCenter
                spacing: 0

                Item {
                    width: 26
                    height: 28

                    Canvas {
                        anchors.fill: parent
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var s = Math.min(width, height)
                            var cy = height / 2
                            var th = s * 0.42
                            var tw = s * 0.34
                            var barW = Math.max(1.5, s * 0.1)
                            var gap = s * 0.08
                            var right = (width - (barW + gap + tw)) / 2
                            ctx.fillStyle = Theme.withOpacity(Theme.foreground, prevArea.containsMouse ? 1 : 0.78)
                            ctx.fillRect(right, cy - th / 2, barW, th)
                            ctx.beginPath()
                            ctx.moveTo(right + barW + gap + tw, cy - th / 2)
                            ctx.lineTo(right + barW + gap + tw, cy + th / 2)
                            ctx.lineTo(right + barW + gap, cy)
                            ctx.closePath()
                            ctx.fill()
                        }
                    }

                    MouseArea {
                        id: prevArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.transport("prev")
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }

                PlayCircle {
                    size: 28
                    pause: view.playing
                    onActivated: view.togglePlay()
                }

                Item {
                    width: 26
                    height: 28

                    Canvas {
                        anchors.fill: parent
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var s = Math.min(width, height)
                            var cy = height / 2
                            var th = s * 0.42
                            var tw = s * 0.34
                            var barW = Math.max(1.5, s * 0.1)
                            var gap = s * 0.08
                            var left = (width - (tw + gap + barW)) / 2
                            ctx.fillStyle = Theme.withOpacity(Theme.foreground, nextArea.containsMouse ? 1 : 0.78)
                            ctx.beginPath()
                            ctx.moveTo(left, cy - th / 2)
                            ctx.lineTo(left, cy + th / 2)
                            ctx.lineTo(left + tw, cy)
                            ctx.closePath()
                            ctx.fill()
                            ctx.fillRect(left + tw + gap, cy - th / 2, barW, th)
                        }
                    }

                    MouseArea {
                        id: nextArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.transport("next")
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }

                Item {
                    width: 26
                    height: 28

                    Canvas {
                        anchors.fill: parent
                        property bool liked: view.liked
                        onLikedChanged: requestPaint()
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var s = Math.min(width, height) * 0.58
                            var x = (width - s) / 2
                            var y = (height - s) / 2 + s * 0.04
                            ctx.beginPath()
                            ctx.moveTo(x + s / 2, y + s * 0.9)
                            ctx.bezierCurveTo(x - s * 0.02, y + s * 0.55, x + s * 0.08, y + s * 0.12, x + s * 0.28, y + s * 0.12)
                            ctx.bezierCurveTo(x + s * 0.44, y + s * 0.12, x + s * 0.5, y + s * 0.28, x + s / 2, y + s * 0.4)
                            ctx.bezierCurveTo(x + s * 0.5, y + s * 0.28, x + s * 0.56, y + s * 0.12, x + s * 0.72, y + s * 0.12)
                            ctx.bezierCurveTo(x + s * 0.92, y + s * 0.12, x + s * 1.02, y + s * 0.55, x + s / 2, y + s * 0.9)
                            ctx.closePath()
                            ctx.fillStyle = liked
                                ? Theme.liked
                                : Theme.withOpacity(Theme.foreground, likeArea.containsMouse ? 0.95 : 0.72)
                            ctx.fill()
                        }
                    }

                    MouseArea {
                        id: likeArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.likePlaying()
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }
            }
        }

        Item {
            id: tabs
            width: parent.width - searchSlot.width - transport.width - viewSwitch.width - bar.spacing * 3
            height: parent.height
            clip: true

            Row {
                anchors.verticalCenter: parent.verticalCenter
                height: 34
                spacing: 2

                Repeater {
                    model: view.menuTabs()

                    delegate: Item {
                        id: tab
                        required property var modelData

                        readonly property var row: modelData.row || {}
                        readonly property int sourceIndex: modelData.index
                        readonly property bool currentPage: row.kind === "page" && view.page && view.page.kind === row.id
                        readonly property bool shownList: row.kind === "playlist" && view.libraryView === "" && !view.page
                            && view.mode !== "download" && view.mode !== "discover" && !view.searchActive()
                            && row.id === (view.shownPlaylist !== "" ? view.shownPlaylist : "current")
                        readonly property bool shownLists: row.id === "playlists" && view.libraryView === "lists" && !view.searchActive()
                        readonly property bool on: currentPage || shownList || shownLists

                        width: tabBody.implicitWidth + 22
                        height: 34

                        Rectangle {
                            anchors.fill: parent
                            radius: height / 2
                            color: tab.on
                                ? Theme.withOpacity(Theme.border, 0.22)
                                : (tabArea.containsMouse ? Theme.foregroundFaint : "transparent")
                        }

                        Row {
                            id: tabBody
                            anchors.centerIn: parent
                            spacing: 6

                            Text {
                                visible: row.id === "filesystem"
                                width: visible ? 12 : 0
                                height: 34
                                verticalAlignment: Text.AlignVCenter
                                textFormat: Text.PlainText
                                text: view.treeRootOpen ? "▾" : "▸"
                                color: Theme.muted
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeS
                            }

                            Text {
                                height: 34
                                verticalAlignment: Text.AlignVCenter
                                textFormat: Text.PlainText
                                text: String(row.label || "")
                                color: tab.on ? Theme.border : Theme.foreground
                                font.family: Theme.fontFamily
                                font.bold: tab.on
                                font.pixelSize: Theme.fontSizeM
                            }

                            Text {
                                visible: row.count > 0
                                height: 34
                                verticalAlignment: Text.AlignVCenter
                                textFormat: Text.PlainText
                                text: row.count > 0 ? String(row.count) : ""
                                color: Theme.muted
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeS
                            }
                        }

                        MouseArea {
                            id: tabArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            acceptedButtons: Qt.LeftButton | Qt.RightButton
                            onClicked: function(mouse) {
                                if (mouse.button === Qt.RightButton) {
                                    view.clickBrowse(tab.sourceIndex)
                                    view.playBrowseAt(tab.sourceIndex)
                                    return
                                }
                                view.clickBrowse(tab.sourceIndex)
                            }
                            onDoubleClicked: {
                                if (row.id === "filesystem")
                                    return
                                view.clickBrowse(tab.sourceIndex)
                                view.playBrowseAt(tab.sourceIndex)
                            }
                        }
                    }
                }
            }
        }

        Item {
            id: viewSwitch
            width: viewSwitchRow.implicitWidth
            height: parent.height

            Row {
                id: viewSwitchRow
                anchors.verticalCenter: parent.verticalCenter
                spacing: 2

                Item {
                    width: 28
                    height: 28

                    Rectangle {
                        anchors.fill: parent
                        radius: 6
                        color: view.asGrid
                            ? Theme.withOpacity(Theme.border, 0.22)
                            : (gridArea.containsMouse ? Theme.foregroundFaint : "transparent")
                    }

                    Grid {
                        anchors.centerIn: parent
                        columns: 2
                        spacing: 2

                        Repeater {
                            model: 4
                            Rectangle {
                                width: 5
                                height: 5
                                radius: 1
                                color: view.asGrid ? Theme.border : Theme.foreground
                            }
                        }
                    }

                    MouseArea {
                        id: gridArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.setLayout(true)
                    }
                }

                Item {
                    width: 28
                    height: 28

                    Rectangle {
                        anchors.fill: parent
                        radius: 6
                        color: !view.asGrid
                            ? Theme.withOpacity(Theme.border, 0.22)
                            : (listArea.containsMouse ? Theme.foregroundFaint : "transparent")
                    }

                    Column {
                        anchors.centerIn: parent
                        spacing: 3

                        Repeater {
                            model: 3
                            Rectangle {
                                width: 12
                                height: 2
                                radius: 1
                                color: !view.asGrid ? Theme.border : Theme.foreground
                            }
                        }
                    }

                    MouseArea {
                        id: listArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.setLayout(false)
                    }
                }
            }
        }

        Item {
            id: searchSlot
            width: Math.min(240, pane.width * 0.28)
            height: parent.height

        Rectangle {
            id: searchBox
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width
            height: 32
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
        }
    }

    Rectangle {
        id: treePanel
        parent: pane.parent
        z: 4
        visible: view.treeRows().length > 0
        x: pane.x
        y: pane.y + pane.height + 8
        width: Math.min(320, Math.max(220, pane.width * 0.34))
        height: Math.min(420, Math.max(34, view.treeRows().length * 36 + 16))
        radius: 8
        color: Theme.mixColors(Theme.background, Qt.color("#000000"), 0.55)
        clip: true

        ListView {
            id: treeList
            anchors.fill: parent
            anchors.margins: 8
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            spacing: 2
            model: view.treeRows()

            delegate: Item {
                id: branch
                required property int index
                required property var modelData

                readonly property var row: modelData.row || {}
                readonly property int sourceIndex: modelData.index

                width: treeList.width
                height: 34

                Rectangle {
                    anchors.fill: parent
                    radius: height / 2
                    color: branchArea.containsMouse ? Theme.foregroundFaint : "transparent"
                }

                Row {
                    anchors.left: parent.left
                    anchors.right: actions.left
                    anchors.leftMargin: 8 + (Number(row.depth) || 0) * 14
                    anchors.rightMargin: 8
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 6

                    Text {
                        width: 12
                        height: 34
                        verticalAlignment: Text.AlignVCenter
                        textFormat: Text.PlainText
                        text: row.open ? "▾" : "▸"
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: parent.width - 18
                        height: 34
                        verticalAlignment: Text.AlignVCenter
                        elide: Text.ElideRight
                        text: String(row.label || "")
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeM
                    }
                }

                Row {
                    id: actions
                    z: 2
                    anchors.right: parent.right
                    anchors.rightMargin: 10
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: 2

                    Item {
                        id: playMark
                        width: 22
                        height: 34

                        Text {
                            anchors.centerIn: parent
                            textFormat: Text.PlainText
                            text: "▶"
                            color: playArea.containsMouse ? Theme.foreground : Theme.muted
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeXs
                        }

                        MouseArea {
                            id: playArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: view.playFolder(String(row.path || ""))
                        }
                    }

                    Item {
                        id: addMark
                        width: 22
                        height: 34

                        Text {
                            anchors.centerIn: parent
                            textFormat: Text.PlainText
                            text: "+"
                            color: addArea.containsMouse ? Theme.foreground : Theme.muted
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeM
                        }

                        MouseArea {
                            id: addArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: view.addFolder(String(row.path || ""))
                        }
                    }

                    Text {
                        id: countText
                        width: 40
                        horizontalAlignment: Text.AlignRight
                        textFormat: Text.PlainText
                        height: 34
                        verticalAlignment: Text.AlignVCenter
                        text: row.count > 0 ? String(row.count) : ""
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }
                }

                MouseArea {
                    id: branchArea
                    anchors.fill: parent
                    hoverEnabled: true
                    preventStealing: true
                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                    onClicked: function(mouse) {
                        if (mouse.button === Qt.RightButton) {
                            view.playFolder(String(row.path || ""))
                            return
                        }
                        view.clickBrowse(branch.sourceIndex)
                    }
                }
            }
        }
    }

    Connections {
        target: view
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
}
