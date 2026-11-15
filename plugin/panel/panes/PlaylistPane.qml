import QtQuick
import QtQuick.Controls
import "../compat"
import "../overlays"
import "../widgets"

Card {
    id: pane

    required property var view

    pad: 0

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

    active: view.pane === "playlist"
    property int sideMargin: 0
    readonly property int scrollGutter: 12
    readonly property bool likesOpen: view.shownPlaylist === "all" && view.listSource() === "shown"
    readonly property bool likesGrid: likesOpen && view.asGrid
    readonly property bool showLibrary: view.mode === "queue" || view.mode === "tags"
    readonly property bool coverList: view.mode === "library"
    readonly property bool trackGrid: view.asGrid && pane.showLibrary && !view.indexPage()
        && (view.listSource() === "shown" || view.listSource() === "page" || view.listSource() === "search")
    readonly property bool plainHead: view.listSource() === "files" || view.listSource() === "lists"
    readonly property bool trackHeading: view.listSource() === "page"
    readonly property bool releaseColumn: trackHeading && view.page && String(view.page.kind) !== "album"
        && view.pageData && (view.pageData.releases || []).length > 0
    readonly property bool titledHead: view.listSource() === "files"
    property int menuTop: 0

    function activeFlick() {
        if (view.indexPage())
            return indexGrid
        if (trackGrid)
            return trackView
        if (view.listSource() === "lists")
            return playlistGrid
        return list
    }

    function captureScroll() {
        var key = view.scrollKey()
        if (!key)
            return
        var flick = activeFlick()
        if (!flick)
            return
        view.rememberScroll(key, flick.readScroll ? flick.readScroll() : flick.contentY)
    }

    function restoreActiveScroll() {
        var key = view.scrollKey()
        if (!key)
            return
        var y = view.savedScroll(key)
        if (y < 0)
            return
        placeScroll(activeFlick(), y, 6)
    }

    function placeScroll(flick, y, tries) {
        Qt.callLater(function() {
            if (!flick)
                return
            var maxY = flick.scrollMax ? flick.scrollMax() : Math.max(0, flick.contentHeight - flick.height)
            if (y > 8 && maxY < y && (tries > 0 || maxY < 8)) {
                if (tries > 0)
                    placeScroll(flick, y, tries - 1)
                return
            }
            var next = Math.max(0, Math.min(maxY, y))
            if (flick.writeScroll)
                flick.writeScroll(next)
            else
                flick.contentY = next
            view.scrollPending = false
        })
    }

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

        Item {
            id: head
            anchors.top: errLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.rightMargin: pane.plainHead ? pane.sideMargin : 0
            visible: pane.showLibrary && !view.indexPage()
            height: visible ? (pane.plainHead ? (pane.titledHead ? plainHead.implicitHeight : 0) : banner.implicitHeight) : 0
            onHeightChanged: pane.menuTop = pane.plainHead ? 0 : height
            onVisibleChanged: pane.menuTop = visible && !pane.plainHead ? height : 0

            PageHeader {
                id: banner
                view: pane.view
                width: head.width
                visible: !pane.plainHead
            }

            Column {
                id: plainHead
                width: parent.width
                visible: pane.titledHead
                spacing: 4
                bottomPadding: 10

                Row {
                    width: parent.width
                    height: 40
                    spacing: 12

                    Text {
                        visible: view.listSource() === "files" && view.browsePath !== ""
                        anchors.verticalCenter: parent.verticalCenter
                        textFormat: Text.PlainText
                        text: "‹"
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSize2xl

                        MouseArea {
                            anchors.fill: parent
                            anchors.margins: -8
                            cursorShape: Qt.PointingHandCursor
                            onClicked: view.leaveFolder()
                        }
                    }

                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        width: Math.min(implicitWidth, parent.width - 90)
                        textFormat: Text.PlainText
                        text: pane.likesGrid ? "Likes" : (view.listSource() === "files" ? (view.browsePath || "Filesystem") : "Playlists")
                        elide: Text.ElideMiddle
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.bold: true
                        font.pixelSize: Theme.fontSize2xl
                    }

                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        textFormat: Text.PlainText
                        text: pane.likesGrid ? String((view.shownTracks || []).length)
                            : (view.listSource() === "files"
                            ? String((view.browseEntries || []).length)
                            : String(view.playlistIndexRows().length))
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }
                }
            }
        }

        GridView {
            id: playlistGrid
            visible: view.listSource() === "lists"
            anchors.top: head.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.rightMargin: pane.sideMargin
            anchors.bottom: parent.bottom
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: view.playlistIndexRows()
            currentIndex: view.playlistIdx
            highlightFollowsCurrentItem: false

            readonly property int columns: Math.max(2, Math.floor(width / 160))
            cellWidth: Math.floor(width / columns)
            cellHeight: cellWidth + 36
            onColumnsChanged: view.playlistColumns = columns
            Component.onCompleted: view.playlistColumns = columns

            Text {
                anchors.centerIn: parent
                visible: playlistGrid.count === 0
                textFormat: Text.PlainText
                text: "no playlists"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: playlistGrid.cellWidth
                height: playlistGrid.cellHeight

                readonly property bool selected: view.pane === "playlist" && index === view.playlistIdx
                readonly property bool hot: cellArea.containsMouse || selected

                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 4
                    radius: 8
                    color: selected ? Theme.withOpacity(Theme.border, 0.16) : (cellArea.containsMouse ? Theme.foregroundWash : "transparent")
                    border.width: selected ? 1 : 0
                    border.color: Theme.border
                }

                Cover {
                    id: face
                    anchors.horizontalCenter: parent.horizontalCenter
                    anchors.top: parent.top
                    anchors.topMargin: 12
                    width: parent.width - 36
                    height: width
                    corner: 6
                    name: String(modelData.title || "")
                    source: view.rowCoverUrl(modelData)
                    decodeSize: Math.ceil((width > 0 ? width : 160) * Screen.devicePixelRatio)
                    scale: hot ? 1.04 : 1

                    Behavior on scale {
                        NumberAnimation { duration: Theme.motionFast }
                    }
                }

                Text {
                    id: nameText
                    anchors.top: face.bottom
                    anchors.topMargin: 8
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: parent.width - 16
                    horizontalAlignment: Text.AlignHCenter
                    textFormat: Text.PlainText
                    text: String(modelData.title || "")
                    elide: Text.ElideRight
                    color: hot ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.85)
                    font.family: Theme.fontFamily
                    font.bold: true
                    font.pixelSize: Theme.fontSizeM
                }

                Text {
                    anchors.top: nameText.bottom
                    anchors.topMargin: 2
                    anchors.horizontalCenter: parent.horizontalCenter
                    textFormat: Text.PlainText
                    text: view.counted(modelData.count, "track")
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeXs
                }

                MouseArea {
                    id: cellArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        view.playlistIdx = index
                        view.openListedPlaylist(String(modelData.id || ""))
                    }
                }
            }
        }

        GridView {
            id: trackView
            visible: pane.trackGrid
            anchors.top: head.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: view.listSource() === "page" ? (view.pageTracks || [])
                : (view.listSource() === "search" ? (view.searchHits || []) : (view.shownTracks || []))
            currentIndex: view.playlistIdx
            highlightFollowsCurrentItem: false

            readonly property int columns: Math.max(2, Math.floor(width / 160))
            cellWidth: Math.floor(width / columns)
            cellHeight: cellWidth + 48
            onColumnsChanged: view.playlistColumns = columns
            Component.onCompleted: view.playlistColumns = columns

            Text {
                anchors.centerIn: parent
                visible: trackView.count === 0
                textFormat: Text.PlainText
                text: view.listSource() === "search" ? "no matches"
                    : (view.shownPlaylist === "all" ? "no likes" : "no tracks")
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: trackView.cellWidth
                height: trackView.cellHeight

                readonly property bool selected: view.pane === "playlist" && index === view.playlistIdx
                readonly property bool hot: likeArea.containsMouse || selected

                Rectangle {
                    anchors.fill: parent
                    anchors.margins: 4
                    radius: 8
                    color: selected ? Theme.withOpacity(Theme.border, 0.16) : (likeArea.containsMouse ? Theme.foregroundWash : "transparent")
                    border.width: selected ? 1 : 0
                    border.color: Theme.border
                }

                Cover {
                    id: likeFace
                    anchors.horizontalCenter: parent.horizontalCenter
                    anchors.top: parent.top
                    anchors.topMargin: 12
                    width: parent.width - 36
                    height: width
                    corner: 6
                    name: String(modelData.title || "")
                    source: view.rowCoverUrl(modelData)
                    decodeSize: Math.ceil((width > 0 ? width : 160) * Screen.devicePixelRatio)
                    scale: hot ? 1.04 : 1

                    Behavior on scale {
                        NumberAnimation { duration: Theme.motionFast }
                    }
                }

                Text {
                    id: likeTitle
                    anchors.top: likeFace.bottom
                    anchors.topMargin: 8
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: parent.width - 16
                    horizontalAlignment: Text.AlignHCenter
                    textFormat: Text.PlainText
                    text: String(modelData.title || "")
                    elide: Text.ElideRight
                    color: hot ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.85)
                    font.family: Theme.fontFamily
                    font.bold: true
                    font.pixelSize: Theme.fontSizeM
                }

                Text {
                    anchors.top: likeTitle.bottom
                    anchors.topMargin: 2
                    anchors.horizontalCenter: parent.horizontalCenter
                    width: parent.width - 16
                    horizontalAlignment: Text.AlignHCenter
                    textFormat: Text.PlainText
                    text: String(modelData.artist || "")
                    elide: Text.ElideRight
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeXs
                }

                MouseArea {
                    id: likeArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        view.playlistIdx = index
                        view.focusPane("playlist")
                        if (modelData.path)
                            view.playTrackList(view.playlistRows(), String(modelData.path))
                    }
                }
            }
        }

        Item {
            id: tracksLabel
            anchors.top: head.bottom
            anchors.left: parent.left
            anchors.right: pane.releaseColumn ? releaseCol.left : parent.right
            anchors.rightMargin: pane.releaseColumn ? 16 : 0
            height: pane.trackHeading ? 28 : 0
            visible: pane.trackHeading

            Text {
                anchors.left: parent.left
                anchors.leftMargin: 2
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 6
                textFormat: Text.PlainText
                text: "tracks"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                font.bold: true
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1
            }
        }

        Item {
            id: releaseCol
            visible: pane.releaseColumn
            width: visible ? Math.min(320, Math.max(240, parent.width * 0.32)) : 0
            anchors.top: head.bottom
            anchors.right: parent.right
            anchors.bottom: parent.bottom

            Text {
                id: releaseLabel
                anchors.top: parent.top
                anchors.left: parent.left
                width: parent.width
                height: 28
                verticalAlignment: Text.AlignBottom
                bottomPadding: 6
                textFormat: Text.PlainText
                text: "releases"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                font.bold: true
                font.capitalization: Font.AllUppercase
                font.letterSpacing: 1
            }

            ListView {
                id: releaseList
                anchors.top: releaseLabel.bottom
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.rightMargin: pane.scrollGutter
                anchors.bottom: parent.bottom
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                model: pane.releaseColumn && view.pageData ? (view.pageData.releases || []) : []
                ScrollBar.vertical: ThinScrollBar {
                    parent: releaseList.parent
                    visible: releaseList.visible
                    anchors.top: releaseList.top
                    anchors.bottom: releaseList.bottom
                    anchors.left: releaseList.right
                    anchors.leftMargin: Math.max(0, pane.scrollGutter - implicitWidth)
                }

                delegate: Item {
                    required property var modelData

                    width: releaseList.width
                    height: 60

                    readonly property string meta: [String(modelData.year || ""), view.page && view.page.kind === "artist" ? "" : String(modelData.artist || "")]
                        .filter(function(s) { return s !== "" }).join(" · ")

                    Rectangle {
                        anchors.fill: parent
                        anchors.rightMargin: 8
                        radius: 6
                        color: releaseArea.containsMouse ? Theme.foregroundWash : "transparent"
                    }

                    Cover {
                        anchors.left: parent.left
                        anchors.leftMargin: 4
                        anchors.verticalCenter: parent.verticalCenter
                        width: 44
                        height: 44
                        corner: 4
                        source: view.rowCoverUrl(modelData)
                        name: String(modelData.name || "")
                        decodeSize: Math.ceil(width * Screen.devicePixelRatio)
                    }

                    Column {
                        anchors.left: parent.left
                        anchors.leftMargin: 58
                        anchors.right: parent.right
                        anchors.rightMargin: 12
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 2

                        Text {
                            width: parent.width
                            textFormat: Text.PlainText
                            text: String(modelData.name || "")
                            elide: Text.ElideRight
                            color: releaseArea.containsMouse ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.9)
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                            font.bold: true
                        }

                        Text {
                            width: parent.width
                            visible: text !== ""
                            textFormat: Text.PlainText
                            text: parent.parent.meta
                            elide: Text.ElideRight
                            color: Theme.muted
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeXs
                        }
                    }

                    MouseArea {
                        id: releaseArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: view.openPage("album", modelData.name, modelData.dir)
                    }
                }
            }
        }

        ListView {
            id: list
            anchors.top: pane.trackHeading ? tracksLabel.bottom : head.bottom
            anchors.left: parent.left
            anchors.right: pane.releaseColumn ? releaseCol.left : parent.right
            anchors.rightMargin: (pane.releaseColumn ? 16 : pane.sideMargin) + pane.scrollGutter
            anchors.bottom: parent.bottom
            visible: pane.showLibrary && !pane.coverList && !view.indexPage() && view.listSource() !== "lists" && !pane.trackGrid
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            highlightFollowsCurrentItem: false
            ScrollBar.vertical: ThinScrollBar {
                parent: list.parent
                visible: list.visible
                anchors.top: list.top
                anchors.bottom: list.bottom
                anchors.left: list.right
                anchors.leftMargin: Math.max(0, pane.scrollGutter - implicitWidth)
            }

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
                view.noteRelatedScroll(contentY, height, contentHeight)
            }

            footer: Column {
                width: list.width
                height: implicitHeight
                spacing: 0

                Item {
                    width: parent.width
                    height: list.count === 0 ? 48 : 12

                    Text {
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        visible: list.count === 0
                        text: view.pageBusy ? "loading…"
                            : (view.listSource() === "search" ? "no matches"
                            : (view.listSource() === "files" ? "empty folder" : "no tracks"))
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeM
                    }
                }

                    Column {
                        id: relatedBlock
                        width: parent.width
                        visible: view.listSource() === "queue" && ((view.relatedTracks && view.relatedTracks.length) || view.relatedNote === "finding similar…")
                        spacing: 0

                    Text {
                        width: parent.width
                        topPadding: 16
                        bottomPadding: 6
                        leftPadding: 8
                        textFormat: Text.PlainText
                        text: "Related"
                        color: Theme.withOpacity(Theme.foreground, 0.38)
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        font.bold: true
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1
                    }

                    Text {
                        width: parent.width
                        leftPadding: 8
                        bottomPadding: 8
                        visible: !(view.relatedTracks && view.relatedTracks.length)
                        textFormat: Text.PlainText
                        text: "finding similar…"
                        color: Theme.withOpacity(Theme.foreground, 0.32)
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }

                    Repeater {
                        model: view.relatedPage()

                        Item {
                            id: relatedWrap
                            required property int index
                            required property var modelData
                            width: relatedBlock.width
                            height: pane.rowH
                            opacity: relatedHover.hovered || view.isPreviewing(modelData.id) ? 0.82 : 0.36

                            HoverHandler {
                                id: relatedHover
                            }

                            SoundRow {
                                id: relatedRow
                                width: parent.width
                                artSide: pane.artSide
                                titlePt: pane.termPt
                                art: view.soundcloudArtURL(modelData.artwork)
                                upper: String(modelData.artist || "")
                                title: String(modelData.title || "")
                                genre: String(modelData.genre || "")
                                plays: Number(modelData.plays) || 0
                                likes: Number(modelData.likes) || 0
                                year: modelData.year ? String(modelData.year) : ""
                                clock: view.trackClock(modelData)
                                badge: modelData.kept ? "kept" : ""
                                current: view.isPreviewing(modelData.id)
                                audible: view.playing
                                onClicked: view.previewRelated(index)
                                onActivated: view.previewRelated(index)
                                onPlayRequested: view.previewRelated(index)

                                Chip {
                                    visible: !relatedWrap.modelData.kept
                                    text: "keep"
                                    prefix: "+"
                                    inverted: relatedRow.filled
                                    onActivated: view.keepRelated(relatedWrap.index)
                                }

                                Chip {
                                    text: "similar"
                                    inverted: relatedRow.filled
                                    onActivated: view.moreRelated(relatedWrap.index)
                                }

                                Chip {
                                    text: "✕"
                                    inverted: relatedRow.filled
                                    onActivated: view.dismissRelated(relatedWrap.index)
                                }
                            }
                        }
                    }

                    Text {
                        width: parent.width
                        leftPadding: 8
                        topPadding: 8
                        bottomPadding: 10
                        visible: view.relatedShown < (view.relatedTracks ? view.relatedTracks.length : 0)
                        textFormat: Text.PlainText
                        text: "more"
                        color: moreRelatedArea.containsMouse ? Theme.withOpacity(Theme.foreground, 0.7) : Theme.withOpacity(Theme.foreground, 0.32)
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        font.bold: true
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1

                        MouseArea {
                            id: moreRelatedArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: view.revealRelated()
                        }
                    }
                }
            }

            // A drag reorders the live queue. Wheel scrolling stays.
            readonly property bool canReorder: !view.shuffle && view.mode === "queue" && view.listSource() === "queue"
            property int dragFrom: -1
            property int dragTo: -1
            property real dragOffset: 0

            Connections {
                target: view
                function onQueueChanged() {
                    list.restoreScroll()
                    if (view.scrollPending)
                        pane.restoreActiveScroll()
                }
                function onShownTracksChanged() {
                    list.restoreScroll()
                    if (view.scrollPending)
                        pane.restoreActiveScroll()
                }
                function onSearchHitsChanged() { list.restoreScroll() }
                function onPageTracksChanged() {
                    if (view.reorderScroll >= 0)
                        list.restoreScroll()
                    else if (view.scrollPending)
                        pane.restoreActiveScroll()
                }
                function onPageChanged() {
                    if (view.reorderScroll >= 0)
                        list.restoreScroll()
                    else if (view.scrollPending)
                        pane.restoreActiveScroll()
                }
                function onScrollEpochChanged() { pane.restoreActiveScroll() }
                function onIndexRowsChanged() {
                    if (view.scrollPending)
                        pane.restoreActiveScroll()
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
                    if (view.mode !== "queue" || view.playlistIdx < 0 || view.holdScroll)
                        return
                    if (pane.trackGrid && view.playlistIdx < trackView.count)
                        trackView.positionViewAtIndex(view.playlistIdx, GridView.Contain)
                    if (view.listSource() === "lists" && view.playlistIdx < playlistGrid.count)
                        playlistGrid.positionViewAtIndex(view.playlistIdx, GridView.Contain)
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

                readonly property bool folder: String(modelData.type || "") === "dir"
                readonly property bool listPick: String(modelData.type || "") === "playlist"
                readonly property bool playing: !folder && !listPick && String(modelData.path || "") !== "" && String(modelData.path) === view.trackPath
                readonly property bool filled: playing
                readonly property bool hovered: rowHover.hovered
                readonly property color ink: filled ? Theme.background : Theme.foreground
                readonly property color dim: filled ? Theme.withOpacity(Theme.background, 0.72) : Theme.muted
                readonly property bool roomy: list.width > 640
                readonly property string year: {
                    var match = String(modelData.year || "").match(/\d{4}/)
                    return match ? match[0] : ""
                }
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
                    acceptedButtons: Qt.LeftButton | Qt.RightButton
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
                        if (mouse.button === Qt.RightButton) {
                            dragged = false
                            return
                        }
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
                        if (mouse.button === Qt.RightButton) {
                            if (!row.folder && !row.listPick)
                                view.editTrack(String(modelData.path || ""))
                            return
                        }
                        if (dragged || suppressClick) {
                            suppressClick = false
                            return
                        }
                        if (row.folder) {
                            view.openBrowsePath(String(modelData.path || ""))
                            return
                        }
                        if (row.listPick) {
                            view.openListedPlaylist(String(modelData.id || ""))
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
                        color: row.playing ? Theme.good
                            : (row.hovered ? Theme.foregroundWash : "transparent")
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
                            visible: !row.folder && !row.listPick && (row.hovered || (row.playing && view.playing))

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
                                onContextActivated: view.openFieldTags("artist", row.artist)
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
                        anchors.rightMargin: 22
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: 8

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: row.folder
                            text: "play"
                            inverted: row.filled
                            onActivated: view.playFolder(String(modelData.path || ""))
                        }

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: row.folder
                            text: "add"
                            inverted: row.filled
                            onActivated: view.addFolder(String(modelData.path || ""))
                        }

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: row.roomy && list.width > 820 && String(modelData.genre || "") !== ""
                            text: String(modelData.genre || "")
                            prefix: "#"
                            maxWidth: 140
                            inverted: row.filled
                            onActivated: view.openGenre(text)
                            onContextActivated: view.openFieldTags("genre", text)
                        }

                        Chip {
                            anchors.verticalCenter: parent.verticalCenter
                            visible: !row.folder && !row.listPick && row.year !== ""
                            text: row.year
                            inverted: row.filled
                            onActivated: view.openYear(text)
                            onContextActivated: view.openFieldTags("year", text)
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            width: Math.ceil(metaFont.advanceWidth("000:00"))
                            horizontalAlignment: Text.AlignRight
                            textFormat: Text.PlainText
                            text: row.folder || row.listPick
                                ? (Number(modelData.count) > 0 ? String(modelData.count) : "")
                                : view.trackClock(modelData)
                            color: row.filled ? Theme.background : Theme.withOpacity(Theme.foreground, 0.75)
                            font.family: Theme.fontFamily
                            font.pointSize: pane.metaPt
                        }

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            textFormat: Text.PlainText
                            visible: !row.folder && !row.listPick
                            width: row.folder || row.listPick ? 0 : Math.ceil(term.advanceWidth("♥"))
                            horizontalAlignment: Text.AlignHCenter
                            text: modelData.liked || row.hovered ? "♥" : ""
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
                    }
                }
            }
        }

        ListView {
            id: discoverList
            anchors.top: errLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.rightMargin: pane.sideMargin + pane.scrollGutter
            anchors.bottom: parent.bottom
            visible: view.mode === "discover"
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            highlightFollowsCurrentItem: false
            ScrollBar.vertical: ThinScrollBar {
                parent: discoverList.parent
                visible: discoverList.visible
                anchors.top: discoverList.top
                anchors.bottom: discoverList.bottom
                anchors.left: discoverList.right
                anchors.leftMargin: Math.max(0, pane.scrollGutter - implicitWidth)
            }
            model: view.discoverTracks

            header: Item {
                width: discoverList.width
                height: seedBanner.height + 14
                onHeightChanged: {
                    if (view.discoverIdx <= 0)
                        Qt.callLater(discoverList.positionViewAtBeginning)
                }

                Banner {
                    id: seedBanner
                    width: parent.width
                    caption: view.discoverSeed.title ? "similar to" : "discover"
                    title: view.discoverSeed.title || "Discover"
                    subtitle: view.discoverSeed.artist || ""
                    stats: view.discoverStats()
                    art: view.discoverSeed.art || ""
                    pause: view.playing && !!view.currentDiscover() && view.isPreviewing(view.currentDiscover().id)
                    onPlay: view.discoverPlay()

                    Row {
                        spacing: 8

                        Chip {
                            visible: view.trackPath !== ""
                            text: "seed from now playing"
                            color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
                            onActivated: view.loadDiscover()
                        }
                    }
                }
            }

            footer: Text {
                width: discoverList.width
                topPadding: 8
                visible: discoverList.count === 0
                textFormat: Text.PlainText
                horizontalAlignment: Text.AlignHCenter
                text: view.discoverNote === "finding similar…" ? "finding similar…" : "nothing new"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            Connections {
                target: view
                function onDiscoverIdxChanged() {
                    if (view.mode !== "discover" || view.discoverIdx < 0)
                        return
                    discoverList.currentIndex = view.discoverIdx
                    if (view.discoverIdx === 0)
                        discoverList.positionViewAtBeginning()
                    else if (view.discoverIdx < discoverList.count)
                        discoverList.positionViewAtIndex(view.discoverIdx, ListView.Contain)
                }
            }

            delegate: SoundRow {
                id: discoverRow
                required property int index
                required property var modelData

                width: discoverList.width
                artSide: pane.artSide
                titlePt: pane.termPt
                art: view.soundcloudArtURL(modelData.artwork)
                upper: String(modelData.artist || "")
                title: String(modelData.title || "")
                genre: String(modelData.genre || "")
                plays: Number(modelData.plays) || 0
                likes: Number(modelData.likes) || 0
                year: modelData.year ? String(modelData.year) : ""
                clock: view.trackClock(modelData)
                badge: modelData.kept ? "kept" : ""
                selected: view.pane === "playlist" && view.mode === "discover" && index === view.discoverIdx
                current: view.isPreviewing(modelData.id)
                audible: view.playing
                onClicked: view.clickDiscover(index)
                onActivated: view.discoverAction(index, "play")
                onPlayRequested: view.discoverAction(index, "play")

                Chip {
                    visible: !discoverRow.modelData.kept
                    text: "keep"
                    prefix: "+"
                    inverted: discoverRow.filled
                    onActivated: view.discoverAction(discoverRow.index, "keep")
                }

                Chip {
                    text: "similar"
                    inverted: discoverRow.filled
                    onActivated: view.discoverAction(discoverRow.index, "more")
                }

                Chip {
                    text: "✕"
                    inverted: discoverRow.filled
                    onActivated: view.discoverAction(discoverRow.index, "dismiss")
                }
            }
        }

        IndexGrid {
            id: indexGrid
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: pane.showLibrary && view.indexPage()
            view: pane.view
        }

        HelpOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: view.mode === "help"
            view: pane.view
        }

        TagEditor {
            z: 8
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: view.mode === "tags"
            view: pane.view
        }

        MovePicker {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: view.mode === "move"
            view: pane.view
        }

        ArtPickerOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: view.mode === "art"
            view: pane.view
        }

        LibraryOverlay {
            z: 6
            anchors.top: errLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            visible: view.mode === "library"
            view: pane.view
        }

        DownloadOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            anchors.rightMargin: pane.sideMargin
            visible: view.mode === "download"
            view: pane.view
        }
    }
}
