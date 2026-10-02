import QtQuick
import "../compat"

Item {
    id: pane

    required property var view

    readonly property int artSide: 32
    readonly property int rowH: artSide + 8
    readonly property bool onResults: view.downloadHit >= 0
    readonly property bool focused: view.pane === "playlist"

    FontMetrics {
        id: metaFont
        font.family: Theme.fontFamily
        font.pointSize: 9
    }

    component FieldInput: Item {
        id: box

        property int index: 0
        property string placeholder: ""
        signal edited(string text)
        signal accepted()

        width: parent ? parent.width : 0
        height: 24

        Text {
            anchors.fill: parent
            visible: field.text === ""
            textFormat: Text.PlainText
            text: box.placeholder
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }

        TextInput {
            id: field
            anchors.fill: parent
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
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

    component NoteLine: Text {
        width: parent ? parent.width : 0
        textFormat: Text.PlainText
        text: pane.view.downloadNote
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
        wrapMode: Text.Wrap
    }

    Fieldset {
        id: downloadSet
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        height: downloadCol.implicitHeight + 16 + (hints.length > 0 ? 16 : 10)
        number: 3
        legend: "download"
        active: pane.focused && !pane.onResults
        hints: pane.onResults ? [] : view.playlistHints()

        Column {
            id: downloadCol
            width: parent.width
            spacing: 4

            FieldInput {
                id: queryBox
                index: 0
                placeholder: "Search youtube, soundcloud or download a link"
                onEdited: function(text) {
                    if (view.downloadQuery !== text)
                        view.downloadQuery = text
                }
                onAccepted: view.runDownloadBox()
            }
        }
    }

    Fieldset {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: downloadSet.bottom
        anchors.topMargin: view.paneGap
        anchors.bottom: parent.bottom
        legend: "results"
        active: pane.focused && pane.onResults
        hints: pane.onResults ? view.playlistHints() : []

        NoteLine {
            id: searchNote
            anchors.top: parent.top
            visible: view.downloadNote !== ""
            height: visible ? implicitHeight : 0
        }

        Flickable {
            id: logView
            anchors.top: searchNote.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.topMargin: searchNote.height > 0 ? 4 : 0
            readonly property real cap: hits.count > 0 ? parent.height * 0.45 : Math.max(0, parent.height - searchNote.height - anchors.topMargin)
            height: logCol.implicitHeight > 0 ? Math.min(logCol.implicitHeight, cap) : 0
            visible: height > 0
            contentWidth: width
            contentHeight: logCol.implicitHeight
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            onContentHeightChanged: {
                if (contentHeight > height)
                    contentY = contentHeight - height
            }

            Column {
                id: logCol
                width: logView.width
                spacing: 2

                Text {
                    textFormat: Text.PlainText
                    width: logCol.width
                    visible: view.downloadLog !== ""
                    text: view.downloadLog
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    wrapMode: Text.Wrap
                }

                Repeater {
                    model: view.downloadFiles || []

                    Text {
                        required property var modelData
                        textFormat: Text.PlainText
                        width: logCol.width
                        elide: Text.ElideRight
                        text: view.trackLabel(modelData)
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeM
                    }
                }
            }
        }

        ListView {
            id: hits
            anchors.top: logView.bottom
            anchors.topMargin: logView.height > 0 ? 4 : 0
            anchors.bottom: parent.bottom
            width: parent.width
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.downloadHits || []

            delegate: Item {
                required property int index
                required property var modelData
                width: hits.width
                height: pane.rowH

                readonly property bool selected: view.downloadHit === index

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
                            source: view.soundcloudArtURL(modelData.artwork)
                        }
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.max(40, parent.width - cover.width - Math.ceil(metaFont.advanceWidth("000:00")) - 16)
                        height: pane.rowH
                        verticalAlignment: Text.AlignVCenter
                        elide: Text.ElideRight
                        text: view.trackLabel(modelData)
                        color: selected ? Theme.background : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pointSize: 12
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: Math.ceil(metaFont.advanceWidth("000:00"))
                        height: pane.rowH
                        verticalAlignment: Text.AlignVCenter
                        horizontalAlignment: Text.AlignRight
                        text: view.trackClock(modelData)
                        color: selected ? Theme.background : Theme.muted
                        font.family: Theme.fontFamily
                        font.pointSize: 9
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    acceptedButtons: Qt.LeftButton | Qt.RightButton
                    onClicked: function(mouse) {
                        view.downloadHit = index
                        if (mouse.button === Qt.RightButton)
                            view.downloadSoundCloudHit()
                    }
                    onDoubleClicked: function(mouse) {
                        if (mouse.button !== Qt.LeftButton)
                            return
                        view.downloadHit = index
                        view.playSoundCloudHit()
                    }
                }
            }
        }
    }

    Connections {
        target: view
        function onModeChanged() {
            if (view.mode === "download")
                pane.focusCurrent()
        }
        function onDownloadIdxChanged() { pane.focusCurrent() }
        function onDownloadHitChanged() {
            if (view.downloadHit >= 0 && view.downloadHit < hits.count)
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
