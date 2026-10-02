import QtQuick
import QtQuick.Effects
import "../compat"

Fieldset {
    id: pane

    required property var view

    number: 4
    legend: view.artLegend()
    active: false
    hints: [{ key: "a", label: "art" }]

    Item {
        id: coverHost
        anchors.centerIn: parent
        width: Math.min(parent.width, parent.height)
        height: width
        visible: view.artSource() !== ""

        Image {
            id: cover
            anchors.fill: parent
            fillMode: Image.PreserveAspectFit
            asynchronous: true
            cache: false
            source: view.artSource()
            visible: false
        }

        MultiEffect {
            anchors.fill: parent
            source: cover
            maskEnabled: true
            maskSource: coverMask
            maskThresholdMin: 0.5
            maskSpreadAtMin: 1.0
        }

        Item {
            id: coverMask
            anchors.fill: parent
            layer.enabled: true
            visible: false

            Rectangle {
                anchors.fill: parent
                radius: 6
            }
        }
    }

    DropArea {
        id: drop
        anchors.fill: parent
        onDropped: function(dropEvent) {
            var raw = ""
            if (dropEvent.hasUrls && dropEvent.urls && dropEvent.urls.length)
                raw = String(dropEvent.urls[0])
            else if (dropEvent.text)
                raw = String(dropEvent.text)
            if (view.dropArt(raw))
                dropEvent.accept(Qt.CopyAction)
        }

        Rectangle {
            anchors.fill: parent
            visible: drop.containsDrag
            color: "transparent"
            border.width: 1
            border.color: Theme.good
            radius: 4
        }
    }

    Connections {
        target: view
        function onArtEpochChanged() {
            cover.source = ""
            cover.source = Qt.binding(function() { return view.artSource() })
        }
    }

    Text {
        textFormat: Text.PlainText
        anchors.centerIn: parent
        visible: view.artSource() === ""
        text: "no art"
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeM
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        onClicked: {
            if (view.mode === "art")
                return
            view.openArt(view.trackPath)
        }
    }
}
