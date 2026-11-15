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

    readonly property int artSide: 46
    readonly property int rowH: artSide + 14
    readonly property int discoverArt: 32
    readonly property int discoverRowH: discoverArt + 8

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
            visible: view.mode === "queue" && !view.indexPage()
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            highlightFollowsCurrentItem: false
            ScrollBar.vertical: ThinScrollBar {}

            function scrollBy(dy) {
                var minY = list.originY
                var maxY = Math.max(minY, list.originY + list.contentHeight - list.height)
                list.contentY = Math.max(minY, Math.min(maxY, list.contentY - dy))
            }

            WheelHandler {
                blocking: true
                onWheel: function(event) {
                    var dy = event.pixelDelta.y
                    if (dy)
                        dy *= 2
                    else
                        dy = event.angleDelta.y / 4
                    list.scrollBy(dy)
                }
            }
            model: view.playlistRows()
            onHeightChanged: view.playlistPage = Math.max(1, Math.floor(height / pane.rowH))
            onContentYChanged: {
                if (view.reorderScroll < 0)
                    view.playlistScroll = contentY
            }

            header: PageHeader {
                view: pane.view
                width: list.width
            }

            footer: Item {
                width: list.width
                height: list.count === 0 ? 48 : 12

                Text {
                    anchors.centerIn: parent
                    textFormat: Text.PlainText
                    visible: list.count === 0
                    text: view.pageBusy ? "loading…" : (view.listSource() === "search" ? "no matches" : "no tracks")
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }
            }

            // A drag reorders the live queue. Wheel scrolling stays.
            readonly property bool canReorder: !view.shuffle && view.mode === "queue" && view.listSource() === "queue"
            property int dragFrom: -1
            property int dragTo: -1
            property real dragOffset: 0

            Connections {
                target: view
                function onQueueChanged() { list.restoreScroll() }
                function onShownTracksChanged() { list.restoreScroll() }
                function onSearchHitsChanged() { list.restoreScroll() }
                function onPageTracksChanged() {
                    if (view.reorderScroll >= 0)
                        list.restoreScroll()
                    else
                        Qt.callLater(list.positionViewAtBeginning)
                }
                function onPageChanged() {
                    if (view.reorderScroll >= 0)
                        list.restoreScroll()
                    else
                        Qt.callLater(list.positionViewAtBeginning)
                }
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

            Connections {
                target: view
                function onPlaylistIdxChanged() {
                    if (view.mode !== "queue" || view.playlistIdx < 0)
                        return
                    list.currentIndex = view.playlistIdx
                    if (view.reorderScroll >= 0)
                        return
                    if (view.playlistIdx === 0)
                        list.positionViewAtBeginning()
                    else if (view.playlistIdx < list.count)
                        list.positionViewAtIndex(view.playlistIdx, ListView.Contain)
                }
                function onPlayingRevealChanged() {
                    Qt.callLater(function() {
                        if (view.mode !== "queue" || view.listSource() !== "queue" || view.playlistIdx < 0)
                            return
                        if (view.playlistIdx >= list.count)
                            return
                        list.currentIndex = view.playlistIdx
                        list.positionViewAtIndex(view.playlistIdx, ListView.Center)
                    })
                }
            }

            delegate: Item {
                id: row
                required property int index
                required property var modelData

                width: list.width
                height: pane.rowH
                z: list.dragFrom === index ? 2 : 0

                readonly property bool playing: String(modelData.path || "") !== "" && String(modelData.path) === view.trackPath
                readonly property bool inPlaylist: view.mode === "queue" && (view.pane === "playlist" || (view.searchActive() && view.pane === "search"))
                readonly property bool selected: inPlaylist && (view.trackPicked(String(modelData.path || "")) || (!(view.playlistPicks && view.playlistPicks.length) && index === view.playlistIdx))
                readonly property bool filled: selected || playing
                readonly property bool hovered: rowHover.hovered
                readonly property color ink: filled ? Theme.background : Theme.foreground
                readonly property color dim: filled ? Theme.withOpacity(Theme.background, 0.72) : Theme.muted
                readonly property bool roomy: list.width > 640
                readonly property string artist: String(modelData.artist || "").replace(/^\s+|\s+$/g, "")
                readonly property string album: String(modelData.album || "").replace(/^\s+|\s+$/g, "")
                readonly property string title: {
                    var t = String(modelData.title || "").replace(/^\s+|\s+$/g, "")
                    if (t)
                        return t
                    var p = String(modelData.path || "")
                    return p.slice(p.lastIndexOf("/") + 1)
                }

                HoverHandler {
                    id: rowHover
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
                        list.scrollBy(dy)
                        wheel.accepted = true
                    }
                    property bool dragged: false
                    property bool rangePress: false
                    property bool clearOnClick: false
                    property bool suppressClick: false
                    property real grabY: 0
                    onPressed: function(mouse) {
                        dragged = false
                        var shift = (mouse.modifiers & Qt.ShiftModifier) !== 0
                        var ctrl = (mouse.modifiers & Qt.ControlModifier) !== 0
                        rangePress = shift || ctrl
                        grabY = mouse.y
                        clearOnClick = !shift && !ctrl && view.playlistRowSelected(index)
                        if (!clearOnClick)
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
                        if (dragged || suppressClick) {
                            suppressClick = false
                            return
                        }
                        if (clearOnClick) {
                            view.unselectPlaylist(index)
                            return
                        }
                        var shift = (mouse.modifiers & Qt.ShiftModifier) !== 0
                        var ctrl = (mouse.modifiers & Qt.ControlModifier) !== 0
                        view.clickPlaylist(index, shift, ctrl && !shift)
                    }
                    onDoubleClicked: {
                        if (dragged)
                            return
                        suppressClick = true
                        view.playSelected()
                    }
                }

                Item {
                    id: body
                    z: 1
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
                        anchors.topMargin: 1
                        anchors.bottomMargin: 1
                        radius: 6
                        color: row.selected ? Theme.good
                            : (row.playing ? Theme.mixColors(Theme.background, Theme.good, 0.65)
                            : (row.hovered ? Theme.foregroundWash : "transparent"))
                    }

                    Rectangle {
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.leftMargin: pane.artSide + 18
                        anchors.bottom: parent.bottom
                        height: 1
                        color: Theme.foregroundGhost
                        visible: !row.filled && !row.hovered
                    }

                    Cover {
                        id: cover
                        x: 6
                        anchors.verticalCenter: parent.verticalCenter
                        width: pane.artSide
                        height: width
                        corner: 4
                        name: row.artist || row.title
                        placeholder: row.filled ? Theme.withOpacity(Theme.background, 0.25) : Theme.mantle
                        decodeSize: 96
                        source: {
                            if (row.playing) {
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

                        Rectangle {
                            anchors.fill: parent
                            radius: 4
                            color: Qt.rgba(0, 0, 0, 0.45)
                            visible: row.hovered || (row.playing && view.playing)

                            Text {
                                anchors.centerIn: parent
                                textFormat: Text.PlainText
                                text: row.playing && view.playing ? "❚❚" : "▶"
                                color: "white"
                                font.pixelSize: Math.round(cover.height * 0.36)
                            }

                            MouseArea {
                                anchors.fill: parent
                                cursorShape: Qt.PointingHandCursor
                                onClicked: {
                                    if (row.playing) {
                                        view.togglePlay()
                                        return
                                    }
                                    view.clickPlaylist(index, false, false)
                                    view.playSelected()
                                }
                            }
                        }
                    }

                    Column {
                        id: words
                        anchors.left: cover.right
                        anchors.leftMargin: 12
                        anchors.right: meta.left
                        anchors.rightMargin: 12
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 3

                        Row {
                            width: parent.width
                            height: artistLink.implicitHeight
                            spacing: 6
                            clip: true

                            LinkText {
                                id: artistLink
                                visible: row.artist !== ""
                                width: Math.min(implicitWidth, parent.width * (row.album ? 0.6 : 1))
                                text: row.artist
                                baseColor: row.dim
                                hoverColor: row.filled ? Theme.background : Theme.foreground
                                font.pixelSize: Theme.fontSizeS
                                onActivated: view.openArtist(row.artist)
                            }

                            Text {
                                visible: row.artist !== "" && row.album !== ""
                                textFormat: Text.PlainText
                                text: "·"
                                color: row.dim
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeS
                            }

                            LinkText {
                                visible: row.album !== ""
                                width: Math.min(implicitWidth, parent.width - (artistLink.visible ? artistLink.width + 20 : 0))
                                text: row.album
                                baseColor: row.filled ? Theme.withOpacity(Theme.background, 0.6) : Theme.withOpacity(Theme.muted, 0.8)
                                hoverColor: row.filled ? Theme.background : Theme.foreground
                                font.pixelSize: Theme.fontSizeS
                                onActivated: view.openAlbum(row.album, modelData.path)
                            }
                        }

                        Text {
                            width: parent.width
                            textFormat: Text.PlainText
                            text: row.title
                            elide: Text.ElideRight
                            color: row.ink
                            font.family: Theme.fontFamily
                            font.pointSize: pane.termPt
                            font.bold: row.playing
                        }
                    }

                    Row {
                        id: meta
                        anchors.right: parent.right
                        anchors.rightMargin: 10
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 8

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: row.roomy && String(modelData.label || "") !== ""
                            text: String(modelData.label || "")
                            maxWidth: 150
                            inverted: row.filled
                            onActivated: view.openLabel(text)
                        }

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: row.roomy && list.width > 820 && String(modelData.genre || "") !== ""
                            text: String(modelData.genre || "")
                            prefix: "#"
                            maxWidth: 140
                            inverted: row.filled
                            onActivated: view.openGenre(text)
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            textFormat: Text.PlainText
                            width: Math.ceil(term.advanceWidth("♥"))
                            horizontalAlignment: Text.AlignHCenter
                            text: modelData.liked ? "♥" : (row.hovered ? "♡" : "")
                            color: row.filled ? Theme.background : (modelData.liked ? Theme.liked : Theme.muted)
                            font.family: Theme.fontFamily
                            font.pointSize: pane.termPt

                            MouseArea {
                                anchors.fill: parent
                                anchors.margins: -4
                                enabled: row.hovered || !!modelData.liked
                                cursorShape: Qt.PointingHandCursor
                                onClicked: view.toggleLike(String(modelData.path || ""))
                            }
                        }

                        Column {
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 2

                            Text {
                                width: Math.ceil(metaFont.advanceWidth("000:00"))
                                horizontalAlignment: Text.AlignRight
                                textFormat: Text.PlainText
                                text: view.trackClock(modelData)
                                color: row.filled ? Theme.background : Theme.withOpacity(Theme.foreground, 0.75)
                                font.family: Theme.fontFamily
                                font.pointSize: pane.metaPt
                            }

                            Text {
                                width: Math.ceil(metaFont.advanceWidth("000:00"))
                                horizontalAlignment: Text.AlignRight
                                textFormat: Text.PlainText
                                text: String(modelData.year || "")
                                color: row.dim
                                font.family: Theme.fontFamily
                                font.pointSize: pane.metaPt
                            }
                        }
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
                height: pane.discoverRowH

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
                        width: pane.discoverArt
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

        IndexGrid {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "queue" && view.indexPage()
            view: pane.view
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
