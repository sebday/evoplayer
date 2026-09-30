import QtQuick
import qs.Commons
import "."

Item {
    id: root
    clip: false

    default property alias content: contentHost.data

    property string legend: ""
    property int number: 0
    property string legendRight: ""
    property var hints: []
    property bool hintsRight: false
    property bool active: false
    property bool framed: true
    property int pad: 10

    readonly property color borderColor: active ? Color.popups.border : Color.foreground
    readonly property var superscripts: ["", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"]

    Item {
        id: contentHost
        z: 1
        anchors.fill: parent
        anchors.leftMargin: root.pad
        anchors.rightMargin: root.pad
        anchors.topMargin: 16
        anchors.bottomMargin: root.hints && root.hints.length > 0 ? 16 : 10
    }

    // These two are declared as children, so the default property parks them
    // inside contentHost. onCompleted moves them back onto the frame.
    Rectangle {
        id: borderRect
        visible: root.framed
        anchors.fill: parent
        radius: 6
        color: "transparent"
        border.width: 1
        border.color: root.borderColor
    }

    Item {
        id: legendHost
        visible: root.framed
        anchors.fill: parent

        Rectangle {
            visible: legendRow.visible
            x: legendRow.x - 4
            y: legendRow.y - 2
            width: legendRow.width + 8
            height: legendRow.height + 4
            color: Theme.background
        }

        Row {
            id: legendRow
            x: 12
            y: -height / 2
            spacing: 4
            visible: root.legend !== "" || (root.number >= 1 && root.number <= 9)

            Text {
                textFormat: Text.PlainText
                visible: root.number >= 1 && root.number <= 9
                text: root.superscripts[root.number]
                color: root.borderColor
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
            }

            Text {
                textFormat: Text.PlainText
                text: root.legend
                color: root.borderColor
                font.family: Theme.fontFamily
                font.bold: true
                font.pixelSize: Theme.fontSizeS
            }
        }

        Rectangle {
            visible: rightLegend.visible
            x: rightLegend.x - 4
            y: rightLegend.y - 2
            width: rightLegend.width + 8
            height: rightLegend.height + 4
            color: Theme.background
        }

        Row {
            id: rightLegend
            visible: root.legendRight !== ""
            y: -height / 2
            x: Math.max(12, parent.width - width - 12)

            Text {
                textFormat: Text.PlainText
                width: Math.min(implicitWidth, Math.max(0, root.width * 0.45))
                elide: Text.ElideLeft
                text: root.legendRight
                color: root.borderColor
                font.family: Theme.fontFamily
                font.bold: true
                font.pixelSize: Theme.fontSizeS
            }
        }

        Rectangle {
            visible: hintRow.visible
            x: hintRow.x - 4
            y: hintRow.y - 2
            width: hintRow.width + 8
            height: hintRow.height + 4
            color: Theme.background
        }

        Row {
            id: hintRow
            x: root.hintsRight ? Math.max(12, parent.width - width - 12) : 12
            y: parent.height - height / 2
            spacing: 10
            visible: root.hints && root.hints.length > 0
            width: Math.min(implicitWidth, Math.max(0, parent.width - 24))
            clip: true

            Repeater {
                model: root.hints || []

                Row {
                    spacing: 4

                    Text {
                        textFormat: Text.PlainText
                        text: String(modelData.key || "")
                        color: root.borderColor
                        font.family: Theme.fontFamily
                        font.bold: true
                        font.pixelSize: Theme.fontSizeS
                    }

                    Text {
                        textFormat: Text.PlainText
                        text: String(modelData.label || "")
                        color: Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }
                }
            }
        }
    }

    Component.onCompleted: {
        borderRect.parent = root
        legendHost.parent = root
        borderRect.z = 0
        contentHost.z = 1
        legendHost.z = 2
    }
}
