import QtQuick
import QtQuick.Layouts
import "../compat"
import "../widgets"

Item {
    id: editor

    required property var view
    property bool confirmDelete: false

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
        if (editor.view.tagField === "artist" || editor.view.tagField === "album")
            return [fieldOnly, tagsField]
        if (editor.view.tagField)
            return [fieldOnly]
        if (editor.view.tagMany)
            return [albumField, yearField]
        return [titleField, artistField, albumField, yearField, genreField, labelField, tagsField]
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
                width: parent.width - 28
                visible: editor.view.tagField !== "" || editor.view.tagMany
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
                id: labelField
                visible: editor.view.tagField === "" && !editor.view.tagMany
                label: "label"
                index: 5
                view: editor.view
                value: editor.view.tagLabel
                onEdited: function(text) { if (editor.view.tagLabel !== text) editor.view.tagLabel = text }
            }
            TagField {
                id: tagsField
                visible: editor.view.tagField === "artist" || editor.view.tagField === "album" || (editor.view.tagField === "" && !editor.view.tagMany)
                label: "tags"
                suggest: true
                index: (editor.view.tagField === "artist" || editor.view.tagField === "album") ? 1 : 6
                view: editor.view
                value: editor.view.tagTags
                onEdited: function(text) { if (editor.view.tagTags !== text) editor.view.tagTags = text }
            }
            Row {
                width: parent.width
                spacing: 6

                Rectangle {
                    id: openBtn
                    visible: editor.showArt
                    width: visible ? folderText.implicitWidth + 20 : 0
                    height: 32
                    radius: 8
                    color: folderArea.containsMouse ? Theme.foregroundSubtle : Theme.foregroundFaint

                    Text {
                        id: folderText
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        text: "Open"
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }

                    MouseArea {
                        id: folderArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: editor.view.openTrackFolder()
                    }
                }

                Rectangle {
                    id: deleteBtn
                    visible: editor.showArt
                    width: visible ? deleteText.implicitWidth + 20 : 0
                    height: 32
                    radius: 8
                    color: editor.confirmDelete
                        ? (deleteArea.containsMouse ? Theme.mixColors(Theme.liked, Theme.foreground, 0.18) : Theme.liked)
                        : (deleteArea.containsMouse ? Theme.foregroundSubtle : Theme.fillUrgentSubtle)

                    Text {
                        id: deleteText
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        text: editor.confirmDelete ? "Delete?" : "Delete"
                        color: editor.confirmDelete ? Theme.background : Theme.liked
                        font.family: Theme.fontFamily
                        font.bold: editor.confirmDelete
                        font.pixelSize: Theme.fontSizeS
                    }

                    MouseArea {
                        id: deleteArea
                        anchors.fill: parent
                        enabled: !editor.view.tagBusy
                        hoverEnabled: true
                        cursorShape: enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
                        onClicked: {
                            if (!editor.confirmDelete) {
                                editor.confirmDelete = true
                                return
                            }
                            editor.confirmDelete = false
                            editor.view.deleteTrack()
                        }
                    }
                }

                Item {
                    height: 32
                    width: {
                        var gaps = openBtn.visible ? 3 : 1
                        return Math.max(0, parent.width - openBtn.width - deleteBtn.width - saveBtn.width - parent.spacing * gaps)
                    }
                }

                Rectangle {
                    id: saveBtn
                    width: saveText.implicitWidth + 20
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
            }
            }

            ArtPickerOverlay {
                visible: editor.showArt
                width: editor.showArt ? body.width - form.width - body.spacing : 0
                height: body.height
                view: editor.view
            }
        }

        Text {
            anchors.top: parent.top
            anchors.right: parent.right
            anchors.margins: 8
            z: 2
            width: 28
            height: 28
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            textFormat: Text.PlainText
            text: "×"
            color: closeArea.containsMouse ? Theme.foreground : Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeL

            MouseArea {
                id: closeArea
                anchors.fill: parent
                anchors.margins: -6
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: editor.view.closeMode()
            }
        }
    }

    component TagField: Column {
        id: field
        property string label: ""
        property int index: 0
        property string value: ""
        property bool suggest: false
        property bool syncing: false
        property bool picksOpen: false
        property var view: null
        signal edited(string text)

        function chosenKeys() {
            var parts = String(input.text || "").split(/[;,]/)
            var keys = {}
            for (var i = 0; i < parts.length; i++) {
                var name = parts[i].replace(/^\s+|\s+$/g, "")
                if (name)
                    keys[name.toLowerCase()] = true
            }
            return keys
        }

        function suggestChoices() {
            if (!suggest || !view || !view.tagChoices)
                return []
            var have = chosenKeys()
            var all = view.tagChoices()
            var out = []
            for (var i = 0; i < all.length; i++) {
                if (!have[String(all[i] || "").toLowerCase()])
                    out.push(all[i])
            }
            return out
        }

        function addTag(name) {
            name = String(name || "").replace(/^\s+|\s+$/g, "")
            if (!name || chosenKeys()[name.toLowerCase()])
                return
            var parts = String(input.text || "").split(/[;,]/)
            var kept = []
            for (var i = 0; i < parts.length; i++) {
                var bit = parts[i].replace(/^\s+|\s+$/g, "")
                if (bit)
                    kept.push(bit)
            }
            kept.push(name)
            var next = kept.join("; ")
            syncing = true
            input.text = next
            syncing = false
            edited(next)
            picksOpen = true
            input.forceActiveFocus()
        }

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
                        parent.parent.picksOpen = true
                        parent.parent.view.artPicking = false
                        parent.parent.view.artQueryFocus = false
                        parent.parent.view.tagFocus = parent.parent.index
                    } else {
                        pickClose.restart()
                    }
                }
                Keys.priority: Keys.BeforeItem
                Keys.onPressed: function(event) {
                    if (parent.parent.view && parent.parent.view.dispatch(event))
                        event.accepted = true
                }
            }
        }

        Flickable {
            id: tagPick
            visible: field.suggest && field.picksOpen && tagPick.picks.length > 0
            width: parent.width
            height: visible ? Math.min(tagPick.contentHeight, 132) : 0
            contentHeight: pickFlow.implicitHeight
            clip: true
            boundsBehavior: Flickable.StopAtBounds

            readonly property var picks: {
                var listed = field.view ? field.view.playlists : null
                var typed = input.text
                return listed !== undefined && typed !== undefined ? field.suggestChoices() : []
            }

            MouseArea {
                id: pickHover
                anchors.fill: parent
                hoverEnabled: true
                z: 0
                onPressed: function(mouse) { mouse.accepted = false }
            }

            Flow {
                id: pickFlow
                width: tagPick.width
                spacing: 6
                z: 1

                Repeater {
                    model: tagPick.picks

                    delegate: Rectangle {
                        required property var modelData
                        width: pickText.implicitWidth + 16
                        height: 24
                        radius: height / 2
                        color: pickArea.containsMouse ? Theme.foregroundSubtle : Theme.foregroundFaint

                        Text {
                            id: pickText
                            anchors.centerIn: parent
                            textFormat: Text.PlainText
                            text: String(modelData)
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                        }

                        MouseArea {
                            id: pickArea
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: field.addTag(modelData)
                        }
                    }
                }
            }
        }

        Timer {
            id: pickClose
            interval: 180
            onTriggered: {
                if (!pickHover.containsMouse)
                    field.picksOpen = false
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
        function onModeChanged() {
            editor.confirmDelete = false
            if (view.mode === "tags")
                Qt.callLater(function() { editor.focusIndex(view.tagFocus) })
        }
        function onTagFocusChanged() {
            editor.confirmDelete = false
            if (view.mode === "tags")
                editor.focusIndex(view.tagFocus)
        }
    }
}
