import QtQuick
import "../compat"

Column {
    id: settings

    required property var view

    spacing: 8
    width: parent ? parent.width : implicitWidth

    Text {
        textFormat: Text.PlainText
        text: "library"
        color: Theme.border
        font.family: Theme.fontFamily
        font.bold: true
        font.pixelSize: Theme.fontSizeS
    }

    TextInput {
        id: pathField
        width: settings.width
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
        clip: true
        selectByMouse: true
        selectionColor: Theme.good
        selectedTextColor: Theme.background
        onTextChanged: {
            if (view.settingsRoot !== text)
                view.settingsRoot = text
        }
        Component.onCompleted: text = view.settingsRoot
        onActiveFocusChanged: view.textCapture = activeFocus
        Keys.priority: Keys.BeforeItem
        Keys.onPressed: function(event) {
            if (view.dispatch(event))
                event.accepted = true
        }
    }

    Text {
        textFormat: Text.PlainText
        text: "soundcloud"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
    }

    Text {
        textFormat: Text.PlainText
        width: settings.width
        elide: Text.ElideRight
        text: view.scUser || "unset"
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        textFormat: Text.PlainText
        text: "oauth  " + (view.scOauth || "missing")
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Text {
        textFormat: Text.PlainText
        text: "visualizer"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
    }

    Text {
        textFormat: Text.PlainText
        text: view.vizOn ? "bars" : "off"
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    Connections {
        target: view
        function onModeChanged() {
            if (view.mode === "settings") {
                pathField.text = view.settingsRoot
                Qt.callLater(function() { pathField.forceActiveFocus() })
            }
        }
        function onSettingsRootChanged() {
            if (!pathField.activeFocus && pathField.text !== view.settingsRoot)
                pathField.text = view.settingsRoot
        }
    }
}
