import QtQuick
import "../compat"

Fieldset {
    id: pane

    required property var view

    number: 4
    legend: view.artLegend()
    active: false
    hints: [{ key: "a", label: "art" }]

    Image {
        anchors.centerIn: parent
        width: Math.min(parent.width, parent.height)
        height: width
        fillMode: Image.PreserveAspectFit
        asynchronous: true
        cache: false
        sourceSize.width: 512
        sourceSize.height: 512
        source: view.artSource()
        visible: source !== ""
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
}
