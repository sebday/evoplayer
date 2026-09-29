import QtQuick
import "../compat"

Item {
    id: pane

    required property var view

    readonly property bool onResults: view.downloadHit >= 0
    readonly property bool focused: view.pane === "playlist"

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

            NoteLine {
                visible: view.downloadNote !== "" && !pane.onResults
            }

            Flickable {
                width: parent.width
                height: Math.min(logCol.height, pane.height * 0.4)
                visible: logCol.height > 0
                contentWidth: width
                contentHeight: logCol.height
                clip: true
                boundsBehavior: Flickable.StopAtBounds

                Column {
                    id: logCol
                    width: parent.width
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
            visible: view.downloadNote !== "" && pane.onResults
            height: visible ? implicitHeight : 0
        }

        ListView {
            id: hits
            anchors.top: searchNote.bottom
            anchors.topMargin: searchNote.visible ? 4 : 0
            anchors.bottom: parent.bottom
            width: parent.width
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.downloadHits || []

            delegate: Rectangle {
                required property int index
                required property var modelData
                width: hits.width
                height: 22
                radius: 2
                color: view.downloadHit === index ? Theme.good : "transparent"

                Text {
                    anchors.fill: parent
                    anchors.leftMargin: 4
                    anchors.rightMargin: 4
                    textFormat: Text.PlainText
                    verticalAlignment: Text.AlignVCenter
                    elide: Text.ElideRight
                    text: pane.hitLabel(modelData)
                    color: view.downloadHit === index ? Theme.background : Theme.foreground
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
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

    function hitLabel(row) {
        var artist = String(row && row.artist || "").replace(/^\s+|\s+$/g, "")
        var title = String(row && row.title || "").replace(/^\s+|\s+$/g, "")
        var name = artist && title ? artist + " — " + title : (title || artist)
        var dur = view.clock(row ? row.duration : 0)
        return dur ? name + "  " + dur : name
    }

    function focusCurrent() {
        if (view.mode !== "download")
            return
        if (view.downloadHit >= 0) {
            queryBox.drop()
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
