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
            id: menuBtn
            width: 34
            height: parent.height

            Rectangle {
                anchors.centerIn: parent
                width: 34
                height: 34
                radius: 8
                color: view.libraryView === "files"
                    ? Theme.withOpacity(Theme.border, 0.22)
                    : (menuArea.containsMouse ? Theme.foregroundFaint : "transparent")
            }

            Column {
                anchors.centerIn: parent
                spacing: 4

                Repeater {
                    model: 3
                    Rectangle {
                        width: 14
                        height: 2
                        radius: 1
                        color: view.libraryView === "files"
                            ? Theme.border
                            : Theme.withOpacity(Theme.foreground, menuArea.containsMouse ? 1 : 0.78)
                    }
                }
            }

            MouseArea {
                id: menuArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: view.toggleFilesMenu()
            }
        }

        Item {
            id: transport
            width: transportRow.implicitWidth
            height: parent.height

            Row {
                id: transportRow
                anchors.verticalCenter: parent.verticalCenter
                spacing: 0

                Item {
                    width: 30
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var s = height
                            var cy = height / 2
                            var th = s * 0.44
                            var tw = s * 0.30
                            var barW = Math.max(1.6, s * 0.08)
                            var gap = s * 0.06
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

                Item {
                    width: 34
                    height: 34

                    PlayCircle {
                        anchors.centerIn: parent
                        size: 28
                        pause: view.playing
                        onActivated: view.togglePlay()
                    }
                }

                Item {
                    width: 30
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var s = height
                            var cy = height / 2
                            var th = s * 0.44
                            var tw = s * 0.30
                            var barW = Math.max(1.6, s * 0.08)
                            var gap = s * 0.06
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
                    width: 34
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        property bool shuffleOn: view.shuffle
                        property bool repeating: view.repeatOn
                        onShuffleOnChanged: requestPaint()
                        onRepeatingChanged: requestPaint()
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var active = Theme.good
                            var idle = Theme.withOpacity(Theme.foreground, modeArea.containsMouse ? 0.95 : 0.55)
                            function paintShuffle(cx, cy, scale, color) {
                                ctx.strokeStyle = color
                                ctx.fillStyle = color
                                ctx.lineWidth = Math.max(1.4, 1.7 * scale)
                                ctx.lineCap = "round"
                                ctx.lineJoin = "round"
                                var w = 12 * scale
                                var h = 8 * scale
                                function head(x, y, ang) {
                                    var ah = 3.3 * scale
                                    ctx.beginPath()
                                    ctx.moveTo(x, y)
                                    ctx.lineTo(x - ah * Math.cos(ang - 0.5), y - ah * Math.sin(ang - 0.5))
                                    ctx.lineTo(x - ah * Math.cos(ang + 0.5), y - ah * Math.sin(ang + 0.5))
                                    ctx.closePath()
                                    ctx.fill()
                                }
                                var x0 = cx - w / 2
                                var x1 = cx - w * 0.08
                                var x2 = cx + w / 2
                                ctx.beginPath()
                                ctx.moveTo(x0, cy + h * 0.22)
                                ctx.lineTo(x1, cy + h * 0.22)
                                ctx.lineTo(x2, cy - h * 0.55)
                                ctx.stroke()
                                head(x2, cy - h * 0.55, Math.atan2(-h * 0.77, x2 - x1))
                                ctx.beginPath()
                                ctx.moveTo(x0, cy - h * 0.22)
                                ctx.lineTo(x1, cy - h * 0.22)
                                ctx.lineTo(x2, cy + h * 0.55)
                                ctx.stroke()
                                head(x2, cy + h * 0.55, Math.atan2(h * 0.77, x2 - x1))
                            }
                            function paintRepeat(cx, cy, scale, color) {
                                ctx.strokeStyle = color
                                ctx.fillStyle = color
                                ctx.lineWidth = Math.max(1.4, 1.7 * scale)
                                ctx.lineCap = "round"
                                var r = 5.4 * scale
                                var a0 = 0.85
                                var a1 = Math.PI * 1.72
                                ctx.beginPath()
                                ctx.arc(cx, cy, r, a0, a1)
                                ctx.stroke()
                                var x = cx + r * Math.cos(a1)
                                var y = cy + r * Math.sin(a1)
                                var dir = a1 + Math.PI / 2
                                var ah = 3.6 * scale
                                ctx.beginPath()
                                ctx.moveTo(x + ah * 0.15 * Math.cos(dir), y + ah * 0.15 * Math.sin(dir))
                                ctx.lineTo(x - ah * Math.cos(dir - 0.7), y - ah * Math.sin(dir - 0.7))
                                ctx.lineTo(x - ah * Math.cos(dir + 0.9), y - ah * Math.sin(dir + 0.9))
                                ctx.closePath()
                                ctx.fill()
                            }
                            if (shuffleOn && repeating) {
                                paintShuffle(width / 2 - 7, height / 2, 0.72, active)
                                paintRepeat(width / 2 + 7, height / 2, 0.72, active)
                            } else if (repeating) {
                                paintRepeat(width / 2, height / 2, 1, active)
                            } else {
                                paintShuffle(width / 2, height / 2, 1, shuffleOn ? active : idle)
                            }
                        }
                    }

                    MouseArea {
                        id: modeArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.cycleShuffleRepeat()
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }

                Item {
                    width: 34
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        property bool liked: view.liked
                        onLikedChanged: requestPaint()
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var scale = 16 / 20
                            var cx = width / 2
                            var cy = height / 2 + 1.3
                            var ox = cx - 10 * scale
                            var oy = cy - 9.175 * scale
                            function X(x) { return ox + (x - 2) * scale }
                            function Y(y) { return oy + (y - 3) * scale }
                            ctx.beginPath()
                            ctx.moveTo(X(12), Y(21.35))
                            ctx.bezierCurveTo(X(5.4), Y(15.36), X(2), Y(12.28), X(2), Y(8.5))
                            ctx.bezierCurveTo(X(2), Y(5.42), X(4.42), Y(3), X(7.5), Y(3))
                            ctx.bezierCurveTo(X(9.24), Y(3), X(10.91), Y(3.81), X(12), Y(5.09))
                            ctx.bezierCurveTo(X(13.09), Y(3.81), X(14.76), Y(3), X(16.5), Y(3))
                            ctx.bezierCurveTo(X(19.58), Y(3), X(22), Y(5.42), X(22), Y(8.5))
                            ctx.bezierCurveTo(X(22), Y(12.28), X(18.6), Y(15.36), X(12), Y(21.35))
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

                Item {
                    width: 34
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        property bool hot: view.mode === "eq" || view.eqHot()
                        onHotChanged: requestPaint()
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var color = hot
                                ? Theme.good
                                : Theme.withOpacity(Theme.foreground, eqArea.containsMouse ? 0.95 : 0.55)
                            ctx.strokeStyle = color
                            ctx.fillStyle = color
                            ctx.lineWidth = 1.6
                            ctx.lineCap = "round"
                            var xs = [width / 2 - 5, width / 2, width / 2 + 5]
                            var knobs = [0.72, 0.28, 0.52]
                            var top = height / 2 - 6
                            var span = 12
                            for (var i = 0; i < xs.length; i++) {
                                ctx.beginPath()
                                ctx.moveTo(xs[i], top)
                                ctx.lineTo(xs[i], top + span)
                                ctx.stroke()
                                ctx.beginPath()
                                ctx.arc(xs[i], top + span * knobs[i], 2.1, 0, Math.PI * 2)
                                ctx.fill()
                            }
                        }
                    }

                    MouseArea {
                        id: eqArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.toggleEq()
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }

                Item {
                    width: 34
                    height: 34

                    Canvas {
                        anchors.fill: parent
                        property bool open: view.mode === "library"
                        onOpenChanged: requestPaint()
                        onWidthChanged: requestPaint()
                        onHeightChanged: requestPaint()
                        onPaint: {
                            var ctx = getContext("2d")
                            ctx.clearRect(0, 0, width, height)
                            var color = open
                                ? Theme.good
                                : Theme.withOpacity(Theme.foreground, libArea.containsMouse ? 0.95 : 0.55)
                            ctx.strokeStyle = color
                            ctx.lineWidth = 1.6
                            ctx.lineCap = "round"
                            var left = width / 2 - 7
                            var right = width / 2 + 7
                            var ys = [height / 2 - 5, height / 2, height / 2 + 5]
                            for (var i = 0; i < ys.length; i++) {
                                ctx.beginPath()
                                ctx.moveTo(left, ys[i])
                                ctx.lineTo(right, ys[i])
                                ctx.stroke()
                            }
                        }
                    }

                    MouseArea {
                        id: libArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.toggleLibrary()
                        onContainsMouseChanged: parent.children[0].requestPaint()
                    }
                }
            }
        }

        Item {
            id: tabs
            width: parent.width - searchSlot.width - transport.width - menuBtn.width - bar.spacing * 3
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
