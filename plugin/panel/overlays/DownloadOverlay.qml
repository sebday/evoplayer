import QtQuick
import QtQuick.Controls
import "../compat"
import "../widgets"

Item {
    id: pane

    required property var view

    readonly property int artSide: 46
    readonly property bool onResults: view.downloadHit >= 0
    readonly property bool focused: view.pane === "playlist"
    readonly property bool linkTyped: view.isDownloadLink(view.downloadQuery)
    readonly property var incoming: view.downloadFiles || []

    component SectionLabel: Text {
        textFormat: Text.PlainText
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
        font.bold: true
        font.capitalization: Font.AllUppercase
        font.letterSpacing: 1
    }

    component FieldInput: Item {
        id: box

        property int index: 0
        property string placeholder: ""
        readonly property bool editing: field.activeFocus
        signal edited(string text)
        signal accepted()

        height: 24

        Text {
            anchors.fill: parent
            visible: field.text === ""
            textFormat: Text.PlainText
            text: box.placeholder
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeL
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }

        TextInput {
            id: field
            anchors.fill: parent
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeL
            clip: true
            selectByMouse: true
            selectionColor: Theme.good
            selectedTextColor: Theme.background
            verticalAlignment: TextInput.AlignVCenter
            onTextChanged: box.edited(text)
            onAccepted: box.accepted()
            onActiveFocusChanged: {
                pane.view.textCapture = activeFocus
                if (!activeFocus)
                    return
                pane.view.downloadHit = -1
                pane.view.downloadIdx = box.index
            }
            Keys.priority: Keys.BeforeItem
            Keys.onPressed: function(event) {
                if (pane.view.dispatch(event))
                    event.accepted = true
            }
        }

        function grab() {
            field.forceActiveFocus()
        }

        function drop() {
            field.focus = false
        }

        function show(text) {
            if (field.activeFocus || field.text === text)
                return
            field.text = text
        }
    }

    Banner {
        id: banner
        width: parent.width
        caption: "download"
        title: "Get music"
        stats: {
            if (view.downloadNote)
                return view.downloadNote
            var n = (view.downloadHits || []).length
            if (n)
                return view.counted(n, "result") + " on soundcloud"
            return "search soundcloud, or paste a soundcloud / youtube link"
        }
        art: view.downloadPreviewId && view.isPreviewing(view.downloadPreviewId) ? view.downloadPreviewArt : ""
        showPlay: false

        Rectangle {
            id: searchPill
            width: parent.width
            height: 40
            radius: height / 2
            color: Theme.withOpacity(Theme.background, 0.7)
            border.width: 1
            border.color: queryBox.editing ? Theme.border : Theme.foregroundDivider

            Text {
                id: glyph
                anchors.left: parent.left
                anchors.leftMargin: 16
                anchors.verticalCenter: parent.verticalCenter
                textFormat: Text.PlainText
                text: pane.linkTyped ? "↓" : "›"
                color: queryBox.editing ? Theme.border : Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXl
            }

            FieldInput {
                id: queryBox
                anchors.left: glyph.right
                anchors.leftMargin: 10
                anchors.right: goChip.visible ? goChip.left : parent.right
                anchors.rightMargin: goChip.visible ? 10 : 18
                anchors.verticalCenter: parent.verticalCenter
                index: 0
                placeholder: "artist, track, or a link"
                onEdited: function(text) {
                    if (view.downloadQuery !== text)
                        view.downloadQuery = text
                }
                onAccepted: view.runDownloadBox()
            }

            MouseArea {
                anchors.fill: queryBox
                cursorShape: Qt.IBeamCursor
                visible: !queryBox.editing
                onClicked: {
                    view.focusPane("playlist")
                    view.downloadHit = -1
                    queryBox.grab()
                }
            }

            Chip {
                id: goChip
                anchors.right: parent.right
                anchors.rightMargin: 9
                anchors.verticalCenter: parent.verticalCenter
                visible: String(view.downloadQuery || "").replace(/^\s+|\s+$/g, "") !== ""
                text: pane.linkTyped ? "download" : "search"
                inverted: true
                color: hovered ? Theme.mixColors(Theme.border, Theme.foreground, 0.2) : Theme.border
                onActivated: view.runDownloadBox()
            }
        }

        Row {
            spacing: 8

            Chip {
                visible: view.downloadBusy
                text: "stop"
                inverted: true
                color: hovered ? Theme.mixColors(Theme.liked, Theme.foreground, 0.2) : Theme.liked
                onActivated: view.stopDownload()
            }

            Chip {
                text: "sync soundcloud likes"
                color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
                onActivated: view.syncLikes()
            }
        }
    }

    ListView {
        id: hits
        anchors.top: banner.bottom
        anchors.topMargin: 14
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        highlightMoveDuration: 0
        highlightFollowsCurrentItem: false
        model: view.downloadHits || []
        ScrollBar.vertical: ScrollBar {
            policy: ScrollBar.AsNeeded
            implicitWidth: 7
            leftPadding: 3
            rightPadding: 1
            contentItem: Rectangle {
                implicitWidth: 3
                radius: width / 2
                color: Theme.muted
                opacity: 0.5
            }
            background: Item {}
        }

        header: Column {
            width: hits.width
            spacing: 8
            bottomPadding: hits.count > 0 ? 6 : 0
            onHeightChanged: {
                if (view.downloadHit <= 0)
                    Qt.callLater(hits.positionViewAtBeginning)
            }

            Rectangle {
                width: parent.width
                height: logView.height + 20
                visible: view.downloadLog !== ""
                radius: 8
                color: Theme.mantle

                Flickable {
                    id: logView
                    x: 12
                    y: 10
                    width: parent.width - 24
                    height: Math.min(logText.implicitHeight, 6 * 16)
                    contentWidth: width
                    contentHeight: logText.implicitHeight
                    clip: true
                    boundsBehavior: Flickable.StopAtBounds
                    onContentHeightChanged: {
                        if (contentHeight > height)
                            contentY = contentHeight - height
                    }

                    Text {
                        id: logText
                        width: logView.width
                        textFormat: Text.PlainText
                        text: view.downloadLog
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        wrapMode: Text.Wrap
                    }
                }
            }

            SectionLabel {
                visible: hits.count > 0
                topPadding: view.downloadLog !== "" ? 6 : 0
                text: "soundcloud"
            }
        }

        delegate: SoundRow {
            id: hitRow
            required property int index
            required property var modelData

            width: hits.width
            artSide: pane.artSide
            art: view.soundcloudArtURL(modelData.artwork)
            upper: String(modelData.artist || "")
            title: String(modelData.title || "")
            genre: String(modelData.genre || "")
            plays: Number(modelData.plays) || 0
            likes: Number(modelData.likes) || 0
            year: modelData.year ? String(modelData.year) : ""
            clock: view.trackClock(modelData)
            selected: pane.focused && view.downloadHit === index
            current: view.isPreviewing(modelData.id)
            audible: view.playing
            onClicked: {
                view.downloadHit = index
                view.focusPane("playlist")
            }
            onActivated: view.downloadAction(index, "play")
            onPlayRequested: view.downloadAction(index, "play")

            Chip {
                text: "download"
                prefix: "↓"
                inverted: hitRow.filled
                onActivated: view.downloadAction(hitRow.index, "download")
            }
        }

        footer: Column {
            width: hits.width
            topPadding: hits.count > 0 ? 18 : 0
            spacing: 8
            visible: pane.incoming.length > 0

            SectionLabel {
                text: "recent downloads  " + pane.incoming.length
            }

            Column {
                width: parent.width

                Repeater {
                    model: pane.incoming

                    SoundRow {
                        required property var modelData

                        width: parent.width
                        artSide: 38
                        titlePt: 11
                        art: modelData.art ? Util.fileUrl(String(modelData.art)) : ""
                        upper: String(modelData.artist || "")
                        upperNote: String(modelData.album || "")
                        title: String(modelData.title || "")
                        genre: String(modelData.genre || "")
                        badge: modelData.needs_folder ? "unsorted" : ""
                        year: String(modelData.year || "")
                        clock: view.trackClock(modelData)
                        current: String(modelData.path || "") === view.trackPath
                        audible: view.playing
                        onActivated: pane.playIncoming(String(modelData.path || ""))
                        onPlayRequested: {
                            if (current)
                                view.togglePlay()
                            else
                                pane.playIncoming(String(modelData.path || ""))
                        }
                    }
                }
            }
        }
    }

    function playIncoming(path) {
        var paths = []
        for (var i = 0; i < incoming.length; i++)
            paths.push(String(incoming[i].path || ""))
        view.playPaths(paths, path)
    }

    Connections {
        target: view
        function onModeChanged() {
            if (view.mode === "download")
                pane.focusCurrent()
        }
        function onDownloadIdxChanged() { pane.focusCurrent() }
        function onDownloadHitChanged() {
            if (view.downloadHit === 0)
                hits.positionViewAtBeginning()
            else if (view.downloadHit > 0 && view.downloadHit < hits.count)
                hits.positionViewAtIndex(view.downloadHit, ListView.Contain)
            pane.focusCurrent()
        }
        function onPaneChanged() {
            if (view.pane === "playlist")
                pane.focusCurrent()
        }
        function onDownloadQueryChanged() { queryBox.show(view.downloadQuery) }
    }

    function focusCurrent() {
        if (view.mode !== "download")
            return
        if (view.downloadHit >= 0) {
            queryBox.drop()
            view.textCapture = false
            if (view.host && view.host.forceKeyFocus)
                view.host.forceKeyFocus()
            return
        }
        Qt.callLater(function() { queryBox.grab() })
    }

    Component.onCompleted: {
        queryBox.show(view.downloadQuery)
        if (view.mode === "download")
            focusCurrent()
    }
}
