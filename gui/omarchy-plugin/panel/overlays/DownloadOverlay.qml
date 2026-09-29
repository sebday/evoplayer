import QtQuick
import "../compat"

Column {
    id: pane

    required property var view

    spacing: 8
    width: parent ? parent.width : implicitWidth

    component FieldBox: Rectangle {
        id: box

        property int index: 0
        property string label: ""
        property string placeholder: ""
        property bool selected: false
        property string value: ""
        signal edited(string text)
        signal accepted()

        width: pane.width
        implicitHeight: 48
        radius: Theme.fieldsetCornerRadius
        color: Theme.mantle
        border.width: 1
        border.color: selected ? Theme.good : Theme.inactiveBorder

        Text {
            id: caption
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.leftMargin: 8
            anchors.rightMargin: 8
            anchors.topMargin: 4
            text: box.label
            color: box.selected ? Theme.border : Theme.muted
            font.family: Theme.fontFamily
            font.bold: true
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        Text {
            anchors.left: field.left
            anchors.right: field.right
            anchors.verticalCenter: field.verticalCenter
            visible: field.text === ""
            text: box.placeholder
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }

        TextInput {
            id: field
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: caption.bottom
            anchors.leftMargin: 8
            anchors.rightMargin: 8
            height: 22
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

    FieldBox {
        id: urlBox
        index: 0
        label: "url"
        placeholder: "youtube or soundcloud link"
        selected: view.downloadIdx === 0 && view.downloadHit < 0
        onEdited: function(text) {
            if (view.downloadUrl !== text)
                view.downloadUrl = text
        }
        onAccepted: view.submitDownload()
    }

    FieldBox {
        id: searchBox
        index: 1
        label: "soundcloud"
        placeholder: "search, enter plays a result"
        selected: view.downloadIdx === 1 && view.downloadHit < 0
        onEdited: function(text) {
            if (view.downloadQuery !== text)
                view.downloadQuery = text
        }
        onAccepted: view.searchSoundCloud()
    }

    ListView {
        id: hits
        width: pane.width
        height: Math.min(6, count) * 22
        visible: count > 0
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        interactive: false
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
                anchors.leftMargin: 8
                anchors.rightMargin: 8
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

    Text {
        textFormat: Text.PlainText
        width: pane.width
        visible: view.downloadNote !== ""
        text: view.downloadNote
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
        wrapMode: Text.Wrap
    }

    Flickable {
        width: pane.width
        height: Math.max(0, (pane.parent ? pane.parent.height : 0) - y - 8)
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

    Connections {
        target: view
        function onModeChanged() {
            if (view.mode === "download")
                pane.focusCurrent()
        }
        function onDownloadIdxChanged() { pane.focusCurrent() }
        function onDownloadHitChanged() { pane.focusCurrent() }
        function onPaneChanged() {
            if (view.pane === "playlist")
                pane.focusCurrent()
        }
        function onDownloadUrlChanged() { urlBox.show(view.downloadUrl) }
        function onDownloadQueryChanged() { searchBox.show(view.downloadQuery) }
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
            urlBox.drop()
            searchBox.drop()
            return
        }
        Qt.callLater(function() {
            if (view.downloadIdx === 1)
                searchBox.grab()
            else
                urlBox.grab()
        })
    }

    Component.onCompleted: {
        urlBox.show(view.downloadUrl)
        searchBox.show(view.downloadQuery)
        if (view.mode === "download")
            focusCurrent()
    }
}
