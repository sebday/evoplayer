import QtQuick
import "../compat"

Column {
    id: editor

    required property var view

    spacing: 2

    function focusIndex(i) {
        var fields = [titleField, artistField, yearField, genreField, labelField]
        var n = i
        if (n < 0)
            n = 0
        if (n >= fields.length)
            n = fields.length - 1
        fields[n].grab()
    }

    component TagField: Column {
        property string label: ""
        property int index: 0
        property string value: ""
        property bool syncing: false
        property var view: null
        signal edited(string text)

        width: parent.width
        spacing: 0

        Text {
            textFormat: Text.PlainText
            text: parent.label
            color: view && view.tagFocus === parent.index ? Theme.border : Theme.muted
            font.family: Theme.fontFamily
            font.bold: view && view.tagFocus === parent.index
            font.pixelSize: Theme.fontSizeS
        }

        TextInput {
            id: input
            width: parent.width
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            clip: true
            selectByMouse: true
            selectionColor: Theme.good
            selectedTextColor: Theme.background
            onTextChanged: {
                if (!parent.syncing)
                    parent.edited(text)
            }
            Component.onCompleted: {
                parent.syncing = true
                text = parent.value
                parent.syncing = false
            }
            onActiveFocusChanged: {
                if (!parent.view)
                    return
                parent.view.textCapture = activeFocus
                if (activeFocus)
                    parent.view.tagFocus = parent.index
            }
            Keys.priority: Keys.BeforeItem
            Keys.onPressed: function(event) {
                if (parent.view && parent.view.dispatch(event))
                    event.accepted = true
            }
        }

        function grab() {
            input.forceActiveFocus()
        }

        onValueChanged: {
            if (input.text === value)
                return
            // The title is focused before the tags arrive. Keep what the user
            // typed; otherwise show the loaded value.
            if (input.activeFocus && input.text !== "")
                return
            syncing = true
            input.text = value
            syncing = false
        }
    }

    TagField {
        id: titleField
        label: "title"
        index: 0
        view: editor.view
        value: editor.view.tagTitle
        onEdited: function(text) { if (editor.view.tagTitle !== text) editor.view.tagTitle = text }
    }
    TagField {
        id: artistField
        label: "artist"
        index: 1
        view: editor.view
        value: editor.view.tagArtist
        onEdited: function(text) { if (editor.view.tagArtist !== text) editor.view.tagArtist = text }
    }
    TagField {
        id: yearField
        label: "year"
        index: 2
        view: editor.view
        value: editor.view.tagYear
        onEdited: function(text) { if (editor.view.tagYear !== text) editor.view.tagYear = text }
    }
    TagField {
        id: genreField
        label: "genre"
        index: 3
        view: editor.view
        value: editor.view.tagGenre
        onEdited: function(text) { if (editor.view.tagGenre !== text) editor.view.tagGenre = text }
    }
    TagField {
        id: labelField
        label: "label"
        index: 4
        view: editor.view
        value: editor.view.tagLabel
        onEdited: function(text) { if (editor.view.tagLabel !== text) editor.view.tagLabel = text }
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
