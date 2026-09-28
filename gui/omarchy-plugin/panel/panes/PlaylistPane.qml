import QtQuick
import "../compat"
import "../overlays"
import "../widgets"

Fieldset {
    id: pane

    required property var view

    // Ghostty's configured size is 9pt, which is 12px here — the size this
    // list already used. The terminal that is actually open is 12pt.
    readonly property int termPt: 12
    readonly property int metaPt: 9

    FontMetrics {
        id: term
        font.family: Theme.fontFamily
        font.pointSize: pane.termPt
    }

    FontMetrics {
        id: metaFont
        font.family: Theme.fontFamily
        font.pointSize: pane.metaPt
    }

    readonly property int rowH: Math.ceil(term.height) + 4

    number: 3
    legend: view.playlistLegend()
    active: view.pane === "playlist"
    hints: view.playlistHints()

    Item {
        anchors.fill: parent

        Text {
            id: errLine
            textFormat: Text.PlainText
            visible: view.err !== ""
            width: parent.width
            height: visible ? 18 : 0
            text: view.err
            color: Theme.liked
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        Text {
            id: noteLine
            textFormat: Text.PlainText
            anchors.top: errLine.bottom
            visible: view.mode === "discover" && view.discoverNote !== ""
            width: parent.width
            height: visible ? 18 : 0
            text: view.discoverNote
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        ListView {
            id: list
            anchors.top: errLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            visible: view.mode === "queue"
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.shownPlaylist ? view.shownTracks : view.queue
            onHeightChanged: view.playlistPage = Math.max(1, Math.floor(height / pane.rowH))

            // A drag reorders the live queue. Wheel scrolling stays.
            readonly property bool canReorder: !view.shuffle && !view.shownPlaylist && view.mode === "queue"
            property int dragFrom: -1
            property int dragTo: -1
            property real dragOffset: 0

            Connections {
                target: view
                function onQueueChanged() {
                    if (view.reorderScroll < 0)
                        return
                    var y = view.reorderScroll
                    Qt.callLater(function() {
                        list.contentY = y
                        if (view.reorderPending === 0)
                            view.reorderScroll = -1
                    })
                }
            }

            Text {
                textFormat: Text.PlainText
                visible: list.count === 0
                text: "no tracks"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pointSize: pane.termPt
            }

            Connections {
                target: view
                function onPlaylistIdxChanged() {
                    if (view.mode !== "queue" || view.playlistIdx < 0)
                        return
                    list.currentIndex = view.playlistIdx
                    if (view.playlistIdx < list.count)
                        list.positionViewAtIndex(view.playlistIdx, ListView.Contain)
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: list.width
                height: pane.rowH
                z: list.dragFrom === index ? 2 : 0

                readonly property bool playing: String(modelData.path || "") !== "" && String(modelData.path) === view.trackPath
                readonly property bool selected: view.pane === "playlist" && view.mode === "queue" && index === view.playlistIdx

                Item {
                    id: body
                    width: parent.width
                    height: parent.height
                    y: {
                        var from = list.dragFrom
                        var to = list.dragTo
                        if (from < 0 || pane.rowH < 1)
                            return 0
                        if (index === from)
                            return list.dragOffset
                        if (from < to && index > from && index <= to)
                            return -height
                        if (to < from && index >= to && index < from)
                            return height
                        return 0
                    }

                Rectangle {
                    anchors.fill: parent
                    radius: 2
                    color: selected ? Theme.good : "transparent"
                }

                Row {
                    anchors.fill: parent
                    anchors.leftMargin: 2
                    anchors.rightMargin: 2
                    spacing: 8

                    Row {
                        id: titleRow
                        spacing: 4
                        width: parent.width - meta.implicitWidth - 8
                        anchors.verticalCenter: parent.verticalCenter

                        Text {
                            id: mark
                            textFormat: Text.PlainText
                            width: Math.ceil(term.advanceWidth(">")) + 2
                            height: pane.rowH
                            verticalAlignment: Text.AlignVCenter
                            text: playing ? ">" : " "
                            color: selected ? Theme.background : Theme.good
                            font.family: Theme.fontFamily
                            font.bold: true
                            font.pointSize: pane.termPt
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: Math.max(40, titleRow.width - mark.width - titleRow.spacing)
                            height: pane.rowH
                            verticalAlignment: Text.AlignVCenter
                            text: view.trackLabel(modelData)
                            elide: Text.ElideRight
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.foreground)
                            font.family: Theme.fontFamily
                            font.pointSize: pane.termPt
                        }
                    }

                    Row {
                        id: meta
                        spacing: 6
                        anchors.verticalCenter: parent.verticalCenter

                        Text {
                            textFormat: Text.PlainText
                            width: Math.ceil(term.advanceWidth("♥"))
                            height: pane.rowH
                            verticalAlignment: Text.AlignVCenter
                            horizontalAlignment: Text.AlignHCenter
                            text: modelData.liked ? "♥" : ""
                            color: selected ? Theme.background : Theme.good
                            font.family: Theme.fontFamily
                            font.pointSize: pane.termPt
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: Math.ceil(metaFont.advanceWidth("000:00"))
                            height: pane.rowH
                            verticalAlignment: Text.AlignVCenter
                            horizontalAlignment: Text.AlignRight
                            text: view.trackClock(modelData)
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.muted)
                            font.family: Theme.fontFamily
                            font.pointSize: pane.metaPt
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: Math.ceil(metaFont.advanceWidth("0000"))
                            height: pane.rowH
                            verticalAlignment: Text.AlignVCenter
                            text: String(modelData.year || "")
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.muted)
                            font.family: Theme.fontFamily
                            font.pointSize: pane.metaPt
                        }
                    }
                }
                }

                MouseArea {
                    anchors.fill: parent
                    preventStealing: list.canReorder
                    onWheel: function(wheel) {
                        var dy = wheel.pixelDelta.y
                        if (!dy)
                            dy = wheel.angleDelta.y / 8
                        if (!dy)
                            return
                        var maxY = Math.max(0, list.contentHeight - list.height)
                        list.contentY = Math.max(0, Math.min(maxY, list.contentY - dy))
                        wheel.accepted = true
                    }
                    property bool dragged: false
                    property real grabY: 0
                    onPressed: function(mouse) {
                        dragged = false
                        grabY = mouse.y
                        view.clickPlaylist(index)
                    }
                    onPositionChanged: function(mouse) {
                        if (!list.canReorder || pane.rowH < 1)
                            return
                        if (!dragged) {
                            if (Math.abs(mouse.y - grabY) < 8)
                                return
                            dragged = true
                            list.dragFrom = index
                            list.dragTo = index
                            list.dragOffset = 0
                        }
                        var pointerY = mapToItem(list, 0, mouse.y).y
                        var rowTop = index * pane.rowH - list.contentY
                        list.dragOffset = pointerY - grabY - rowTop
                        var dest = Math.floor((list.contentY + pointerY - grabY + pane.rowH / 2) / pane.rowH)
                        if (dest < 0)
                            dest = 0
                        if (dest > list.count - 1)
                            dest = list.count - 1
                        list.dragTo = dest
                    }
                    onReleased: {
                        if (!dragged)
                            return
                        var from = list.dragFrom
                        var delta = list.dragTo - from
                        var scrollY = list.contentY
                        list.dragFrom = -1
                        list.dragTo = -1
                        list.dragOffset = 0
                        if (delta !== 0)
                            view.reorderFrom(from, delta, scrollY)
                    }
                    onCanceled: {
                        list.dragFrom = -1
                        list.dragTo = -1
                        list.dragOffset = 0
                    }
                    onClicked: {
                        if (dragged)
                            return
                        view.clickPlaylist(index)
                    }
                    onDoubleClicked: {
                        if (dragged)
                            return
                        view.playSelected()
                    }
                }
            }
        }

        ListView {
            id: discoverList
            anchors.top: noteLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            visible: view.mode === "discover"
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.discoverTracks

            Text {
                textFormat: Text.PlainText
                visible: discoverList.count === 0 && view.discoverNote === ""
                text: "nothing new"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pointSize: pane.termPt
            }

            Connections {
                target: view
                function onDiscoverIdxChanged() {
                    if (view.mode !== "discover" || view.discoverIdx < 0)
                        return
                    discoverList.currentIndex = view.discoverIdx
                    if (view.discoverIdx < discoverList.count)
                        discoverList.positionViewAtIndex(view.discoverIdx, ListView.Contain)
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: discoverList.width
                height: pane.rowH

                readonly property bool selected: view.pane === "playlist" && view.mode === "discover" && index === view.discoverIdx
                readonly property string mark: view.discoverMark(modelData)

                Rectangle {
                    anchors.fill: parent
                    radius: 2
                    color: selected ? Theme.good : "transparent"
                }

                Row {
                    anchors.fill: parent
                    anchors.leftMargin: 2
                    anchors.rightMargin: 2
                    spacing: 8

                    Text {
                        textFormat: Text.PlainText
                        width: Math.max(40, parent.width - Math.ceil(term.advanceWidth("♥")) - Math.ceil(term.advanceWidth("000:00")) - 16)
                        text: view.trackLabel(modelData)
                        elide: Text.ElideRight
                        color: selected ? Theme.background : (mark !== "" ? Theme.good : Theme.foreground)
                        font.family: Theme.fontFamily
                        font.pointSize: pane.termPt
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.ceil(term.advanceWidth("♥"))
                        horizontalAlignment: Text.AlignHCenter
                        text: mark
                        color: selected ? Theme.background : Theme.good
                        font.family: Theme.fontFamily
                        font.pointSize: pane.termPt
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.ceil(term.advanceWidth("000:00"))
                        horizontalAlignment: Text.AlignRight
                        text: view.trackClock(modelData)
                        color: selected ? Theme.background : Theme.muted
                        font.family: Theme.fontFamily
                        font.pointSize: pane.termPt
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    onClicked: view.clickDiscover(index)
                    onDoubleClicked: view.previewDiscover()
                }
            }
        }

        HelpOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "help"
            view: pane.view
        }

        TagEditor {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "tags"
            view: pane.view
        }

        MovePicker {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "move"
            view: pane.view
        }

        ArtPickerOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "art"
            view: pane.view
        }

        SettingsOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "settings"
            view: pane.view
        }

        DownloadOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "download"
            view: pane.view
        }
    }
}
