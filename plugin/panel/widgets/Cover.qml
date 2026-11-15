import QtQuick
import QtQuick.Effects
import "../compat"

Item {
    id: cover

    property string source: ""
    property bool round: false
    property int corner: 4
    property string name: ""
    property int decodeSize: 144
    property bool cached: true
    property color placeholder: Theme.mantle

    readonly property bool ready: img.status === Image.Ready && source !== ""
    readonly property real cornerPx: round ? width / 2 : corner

    Rectangle {
        anchors.fill: parent
        radius: cover.cornerPx
        color: cover.placeholder
        visible: !cover.ready

        Text {
            anchors.centerIn: parent
            visible: cover.source === ""
            textFormat: Text.PlainText
            text: String(cover.name || "").replace(/^[^A-Za-z0-9]+/, "").charAt(0).toUpperCase()
            color: Theme.muted
            font.family: Theme.fontFamily
            font.bold: true
            font.pixelSize: Math.max(10, Math.round(cover.height * 0.38))
        }
    }

    Image {
        id: img
        anchors.fill: parent
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        cache: cover.cached
        sourceSize.width: cover.decodeSize
        sourceSize.height: cover.decodeSize
        source: cover.source
        visible: false
    }

    MultiEffect {
        anchors.fill: parent
        source: img
        visible: cover.ready
        maskEnabled: true
        maskSource: mask
        maskThresholdMin: 0.5
        maskSpreadAtMin: 1.0
    }

    Item {
        id: mask
        anchors.fill: parent
        layer.enabled: true
        visible: false

        Rectangle {
            anchors.fill: parent
            radius: cover.cornerPx
        }
    }
}
