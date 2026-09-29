import QtQuick
import QtQuick.Controls
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

    readonly property int artSide: 32
    readonly property int rowH: artSide + 8

    component ThinScrollBar: ScrollBar {
        id: bar
        policy: ScrollBar.AsNeeded
        implicitWidth: 7
        leftPadding: 3
        rightPadding: 1
        topPadding: 0
        bottomPadding: 0
        minimumSize: 0.06
        contentItem: Rectangle {
            implicitWidth: 3
            radius: width / 2
            color: Theme.muted
            opacity: bar.active || bar.hovered ? 0.9 : 0.35
            Behavior on opacity {
                NumberAnimation { duration: Theme.motionFast }
            }
        }
        background: Item {}
    }

    number: 3
    legend: view.playlistLegend()
    active: view.pane === "playlist"
    framed: view.mode !== "download"
    hints: view.mode === "download" ? [] : view.playlistHints()

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
            ScrollBar.vertical: ThinScrollBar {}
            WheelHandler {
                blocking: true
                onWheel: function(event) {
                    var dy = event.pixelDelta.y
                    if (dy)
                        dy *= 2
                    else
                        dy = event.angleDelta.y / 4
                    var maxY = Math.max(0, list.contentHeight - list.height)
                    var y = list.contentY - dy
                    if (y < 0)
                        y = 0
                    if (y > maxY)
                        y = maxY
                    list.contentY = y
                }
            }
            model: String(view.searchQuery || "").replace(/^\s+|\s+$/g, "") !== "" ? view.searchHits : (view.shownPlaylist ? view.shownTracks : view.queue)
            onHeightChanged: view.playlistPage = Math.max(1, Math.floor(height / pane.rowH))
            onContentYChanged: {
                if (view.reorderScroll < 0)
                    view.playlistScroll = contentY
            }

            // A drag reorders the live queue. Wheel scrolling stays.
            readonly property bool canReorder: !view.shuffle && !view.shownPlaylist && view.mode === "queue" && String(view.searchQuery || "").replace(/^\s+|\s+$/g, "") === ""
            property int dragFrom: -1
            property int dragTo: -1
            property real dragOffset: 0

            Connections {
                target: view
                function onQueueChanged() { list.restoreScroll() }
                function onShownTracksChanged() { list.restoreScroll() }
                function onSearchHitsChanged() { list.restoreScroll() }
            }

            function restoreScroll() {
                if (view.reorderScroll < 0)
                    return
                var y = view.reorderScroll
                Qt.callLater(function() {
                    list.contentY = y
                    if (view.reorderPending === 0)
                        view.reorderScroll = -1
                })
            }

            Text {
                textFormat: Text.PlainText
                visible: list.count === 0
                text: String(view.searchQuery || "").replace(/^\s+|\s+$/g, "") !== "" ? "no matches" : "no tracks"
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
                function onPlayingRevealChanged() {
                    Qt.callLater(function() {
                        if (view.mode !== "queue" || view.shownPlaylist || view.playlistIdx < 0)
                            return
                        if (view.playlistIdx >= list.count)
                            return
                        list.currentIndex = view.playlistIdx
                        list.positionViewAtIndex(view.playlistIdx, ListView.Center)
                    })
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: list.width
                height: pane.rowH
                z: list.dragFrom === index ? 2 : 0

                readonly property bool playing: String(modelData.path || "") !== "" && String(modelData.path) === view.trackPath
                readonly property bool inPlaylist: view.mode === "queue" && (view.pane === "playlist" || (String(view.searchQuery || "").replace(/^\s+|\s+$/g, "") !== "" && view.pane === "search"))
                readonly property bool selected: inPlaylist && (view.trackPicked(String(modelData.path || "")) || (!(view.playlistPicks && view.playlistPicks.length) && index === view.playlistIdx))

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
                    anchors.rightMargin: 8
                    spacing: 8

                    Rectangle {
                        id: cover
                        width: pane.artSide
                        height: width
                        radius: 2
                        anchors.verticalCenter: parent.verticalCenter
                        color: selected ? Theme.background : Theme.mantle
                        clip: true

                        Image {
                            anchors.fill: parent
                            fillMode: Image.PreserveAspectCrop
                            asynchronous: true
                            cache: true
                            sourceSize.width: 72
                            sourceSize.height: 72
                            source: {
                                if (playing) {
                                    var live = view.artSource()
                                    if (live)
                                        return live
                                }
                                var path = String(modelData.thumb || modelData.art || "")
                                if (!path)
                                    return ""
                                var url = Util.fileUrl(path)
                                if (view.artEpoch)
                                    url += "#" + view.artEpoch
                                return url
                            }
                        }
                    }

                    Row {
                        id: titleRow
                        spacing: 4
                        width: parent.width - meta.implicitWidth - cover.width - 16
                        anchors.verticalCenter: parent.verticalCenter

                        Text {
                            textFormat: Text.PlainText
                            width: titleRow.width
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
                    property bool rangePress: false
                    property real grabY: 0
                    onPressed: function(mouse) {
                        dragged = false
                        var shift = (mouse.modifiers & Qt.ShiftModifier) !== 0
                        var ctrl = (mouse.modifiers & Qt.ControlModifier) !== 0
                        rangePress = shift || ctrl
                        grabY = mouse.y
                        view.clickPlaylist(index, shift, ctrl && !shift)
                    }
                    onPositionChanged: function(mouse) {
                        if (rangePress || !list.canReorder || pane.rowH < 1)
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
                    onClicked: function(mouse) {
                        if (dragged)
                            return
                        var shift = (mouse.modifiers & Qt.ShiftModifier) !== 0
                        var ctrl = (mouse.modifiers & Qt.ControlModifier) !== 0
                        view.clickPlaylist(index, shift, ctrl && !shift)
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
            ScrollBar.vertical: ThinScrollBar {}
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
                    anchors.rightMargin: 8
                    spacing: 8

                    Rectangle {
                        id: discoverCover
                        width: pane.artSide
                        height: width
                        radius: 2
                        anchors.verticalCenter: parent.verticalCenter
                        color: selected ? Theme.background : Theme.mantle
                        clip: true

                        Image {
                            anchors.fill: parent
                            fillMode: Image.PreserveAspectCrop
                            asynchronous: true
                            cache: true
                            sourceSize.width: 72
                            sourceSize.height: 72
                            source: view.soundcloudArtURL(modelData.artwork)
                        }
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.max(40, parent.width - discoverCover.width - Math.ceil(term.advanceWidth("♥")) - Math.ceil(term.advanceWidth("000:00")) - 24)
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

        EqOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "eq"
            view: pane.view
        }

        DownloadOverlay {
            parent: pane
            anchors.fill: parent
            anchors.topMargin: errLine.visible ? errLine.y + errLine.height + 24 : 0
            z: 3
            visible: view.mode === "download"
            view: pane.view
        }
    }
}
