import QtQuick
import "../compat"

Column {
    id: pane

    required property var view

    spacing: 8
    width: parent ? parent.width : implicitWidth

    Text {
        textFormat: Text.PlainText
        text: "youtube or soundcloud url"
        color: Theme.border
        font.family: Theme.fontFamily
        font.bold: true
        font.pixelSize: Theme.fontSizeS
    }

    TextInput {
        id: urlField
        width: pane.width
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
        clip: true
        selectByMouse: true
        selectionColor: Theme.good
        selectedTextColor: Theme.background
        onTextChanged: {
            if (view.downloadUrl !== text)
                view.downloadUrl = text
        }
        Component.onCompleted: text = view.downloadUrl
        onActiveFocusChanged: view.textCapture = activeFocus
        Keys.priority: Keys.BeforeItem
        Keys.onPressed: function(event) {
            if (view.dispatch(event))
                event.accepted = true
        }
    }

    Text {
        textFormat: Text.PlainText
        width: pane.width
        text: "enter downloads and imports"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
    }

    Rectangle {
        width: pane.width
        height: 22
        radius: 2
        color: view.downloadIdx === 1 ? Theme.good : "transparent"

        Text {
            anchors.fill: parent
            anchors.leftMargin: 2
            textFormat: Text.PlainText
            text: "sync likes"
            verticalAlignment: Text.AlignVCenter
            color: view.downloadIdx === 1 ? Theme.background : Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
        }

        MouseArea {
            anchors.fill: parent
            onClicked: {
                view.downloadIdx = 1
                view.syncLikes()
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
                pane.focusUrl()
        }
        function onDownloadIdxChanged() {
            if (view.mode !== "download")
                return
            if (view.downloadIdx === 0)
                pane.focusUrl()
            else
                urlField.focus = false
        }
        function onDownloadUrlChanged() {
            if (!urlField.activeFocus && urlField.text !== view.downloadUrl)
                urlField.text = view.downloadUrl
        }
    }

    function focusUrl() {
        Qt.callLater(function() { urlField.forceActiveFocus() })
    }
}
