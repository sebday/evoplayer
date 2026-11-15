import QtQuick
import "../compat"
import "../widgets"

Item {
    id: editor

    required property var view

    function fileName(path) {
        var s = String(path || "")
        var i = Math.max(s.lastIndexOf("/"), s.lastIndexOf("\\"))
        return i >= 0 ? s.slice(i + 1) : s
    }

    function heading() {
        if (editor.view.tagField)
            return editor.view.tagScope
        if (editor.view.tagMany)
            return editor.view.tagPaths.length + " tracks"
        return editor.view.tagTitle || fileName(editor.view.tagPath)
    }

    function fields() {
        if (editor.view.tagField === "artist")
            return [fieldOnly, tagsField]
        if (editor.view.tagField)
            return [fieldOnly]
        if (editor.view.tagMany)
            return [albumField, yearField]
        return [titleField, artistField, albumField, yearField, genreField, tagsField]
    }

    function focusIndex(i) {
        var list = fields()
        var n = i
        if (n < 0)
            n = 0
        if (n >= list.length)
            n = list.length - 1
        list[n].grab()
    }

    readonly property bool showArt: view.tagField === "" && !view.tagMany

    Rectangle {
        anchors.fill: parent
        color: Qt.rgba(0, 0, 0, 0.46)

        MouseArea {
            anchors.fill: parent
            onClicked: editor.view.closeMode()
        }
    }

    Rectangle {
        id: card
        width: Math.min(editor.showArt ? 780 : 440, parent.width - 32)
        height: editor.showArt
            ? Math.min(parent.height - 40, Math.max(form.implicitHeight + 36, 520))
            : form.implicitHeight + 36
        anchors.centerIn: parent
        radius: 14
        color: Theme.overlaySurface
        border.width: 1
        border.color: Theme.foregroundFaint

        MouseArea {
            anchors.fill: parent
        }

        Row {
            id: body
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            anchors.margins: 18
            spacing: 18

            Column {
                id: form
                width: editor.showArt ? Math.min(340, body.width * 0.46) : body.width
                spacing: 14

            Text {
                width: parent.width
                textFormat: Text.PlainText
                text: editor.heading()
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.bold: true
                font.pixelSize: Theme.fontSizeL
                elide: Text.ElideRight
                wrapMode: Text.Wrap
                maximumLineCount: 2
            }

            TagField {
                id: fieldOnly
                visible: editor.view.tagField !== ""
                label: editor.view.tagField
                index: 0
                view: editor.view
                value: editor.view.tagFieldValue
                onEdited: function(text) { if (editor.view.tagFieldValue !== text) editor.view.tagFieldValue = text }
            }
            TagField {
                id: titleField
                visible: editor.view.tagField === "" && !editor.view.tagMany
                label: "title"
                index: 0
                view: editor.view
                value: editor.view.tagTitle
                onEdited: function(text) { if (editor.view.tagTitle !== text) editor.view.tagTitle = text }
            }
            TagField {
                id: artistField
                visible: editor.view.tagField === "" && !editor.view.tagMany
                label: "artist"
                index: 1
                view: editor.view
                value: editor.view.tagArtist
                onEdited: function(text) { if (editor.view.tagArtist !== text) editor.view.tagArtist = text }
            }
            TagField {
                id: albumField
                visible: editor.view.tagField === ""
                label: "album"
                index: editor.view.tagMany ? 0 : 2
                view: editor.view
                value: editor.view.tagAlbum
                onEdited: function(text) { if (editor.view.tagAlbum !== text) editor.view.tagAlbum = text }
            }
            TagField {
                id: yearField
                visible: editor.view.tagField === ""
                label: "year"
                index: editor.view.tagMany ? 1 : 3
                view: editor.view
                value: editor.view.tagYear
                onEdited: function(text) { if (editor.view.tagYear !== text) editor.view.tagYear = text }
            }
            TagField {
                id: genreField
                visible: editor.view.tagField === "" && !editor.view.tagMany
                label: "genre"
                index: 4
                view: editor.view
                value: editor.view.tagGenre
                onEdited: function(text) { if (editor.view.tagGenre !== text) editor.view.tagGenre = text }
            }
            TagField {
                id: tagsField
                visible: editor.view.tagField === "artist" || (editor.view.tagField === "" && !editor.view.tagMany)
                label: "tags"
                index: editor.view.tagField === "artist" ? 1 : 5
                view: editor.view
                value: editor.view.tagTags
                onEdited: function(text) { if (editor.view.tagTags !== text) editor.view.tagTags = text }
            }
            Row {
                width: parent.width
                spacing: 8
                layoutDirection: Qt.RightToLeft

                Rectangle {
                    width: saveText.implicitWidth + 28
                    height: 32
                    radius: 8
                    color: editor.view.tagBusy ? Theme.foregroundFaint : Theme.good

                    Text {
                        id: saveText
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        text: editor.view.tagBusy ? "saving" : "Save"
                        color: Theme.background
                        font.family: Theme.fontFamily
                        font.bold: true
                        font.pixelSize: Theme.fontSizeS
                    }

                    MouseArea {
                        anchors.fill: parent
                        enabled: !editor.view.tagBusy
                        cursorShape: Qt.PointingHandCursor
                        onClicked: editor.view.saveTags()
                    }
                }

                Rectangle {
                    width: cancelText.implicitWidth + 28
                    height: 32
                    radius: 8
                    color: cancelArea.containsMouse ? Theme.foregroundSubtle : Theme.foregroundFaint

                    Text {
                        id: cancelText
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        text: "Cancel"
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }

                    MouseArea {
                        id: cancelArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: editor.view.closeMode()
                    }
                }
            }
            }

            ArtPickerOverlay {
                visible: editor.showArt
                width: editor.showArt ? body.width - form.width - body.spacing : 0
                height: body.height
                view: editor.view
            }
        }
    }

    component TagField: Column {
        property string label: ""
        property int index: 0
        property string value: ""
        property bool syncing: false
        property var view: null
        signal edited(string text)

        width: parent.width
        spacing: 6

        Text {
            textFormat: Text.PlainText
            text: parent.label
            color: view && view.tagFocus === parent.index ? Theme.foreground : Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        Rectangle {
            width: parent.width
            height: 36
            radius: 8
            color: Theme.foregroundFaint
            border.width: view && view.tagFocus === parent.index ? 1 : 0
            border.color: Theme.border

            TextInput {
                id: input
                anchors.fill: parent
                anchors.leftMargin: 10
                anchors.rightMargin: 10
                verticalAlignment: TextInput.AlignVCenter
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                clip: true
                selectByMouse: true
                selectionColor: Theme.good
                selectedTextColor: Theme.background
                onTextChanged: {
                    if (!parent.parent.syncing)
                        parent.parent.edited(text)
                }
                onActiveFocusChanged: {
                    if (!parent.parent.view)
                        return
                    parent.parent.view.textCapture = activeFocus
                    if (activeFocus) {
                        parent.parent.view.artPicking = false
                        parent.parent.view.artQueryFocus = false
                        parent.parent.view.tagFocus = parent.parent.index
                    }
                }
                Keys.priority: Keys.BeforeItem
                Keys.onPressed: function(event) {
                    if (parent.parent.view && parent.parent.view.dispatch(event))
                        event.accepted = true
                }
            }
        }

        function grab() {
            input.forceActiveFocus()
        }

        onValueChanged: {
            if (input.text === value)
                return
            if (input.activeFocus && input.text !== "")
                return
            syncing = true
            input.text = value
            syncing = false
        }

        Component.onCompleted: {
            syncing = true
            input.text = value
            syncing = false
        }
    }

    Connections {
        target: view
        function onTagFocusChanged() {
            if (view.mode === "tags")
                editor.focusIndex(view.tagFocus)
        }
        function onModeChanged() {
            if (view.mode === "tags")
                Qt.callLater(function() { editor.focusIndex(view.tagFocus) })
        }
    }
}
