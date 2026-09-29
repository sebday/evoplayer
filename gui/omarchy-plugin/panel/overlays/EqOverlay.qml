import QtQuick
import "../compat"

Item {
    id: pane

    required property var view

    readonly property var marks: ["pre", "31", "62", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"]

    function dbFromY(y, h) {
        if (h < 1)
            return 0
        var t = y / h
        if (t < 0)
            t = 0
        if (t > 1)
            t = 1
        return Math.round(12 - t * 24)
    }

    function dbText(v) {
        var n = Math.round(Number(v) || 0)
        if (n > 0)
            return "+" + n
        return String(n)
    }

    function gainAt(index) {
        if (index <= 0)
            return view.eqPreamp
        return Number((view.eqBands || [])[index - 1]) || 0
    }

    Column {
        anchors.fill: parent
        spacing: 6

        Row {
            width: parent.width
            height: 18
            spacing: 16

            Text {
                textFormat: Text.PlainText
                text: view.eqEnabled ? "on" : "off"
                color: view.eqEnabled ? Theme.good : Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                anchors.verticalCenter: parent.verticalCenter

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -4
                    onClicked: view.toggleEqEnabled()
                }
            }

            Text {
                textFormat: Text.PlainText
                text: "flat"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                anchors.verticalCenter: parent.verticalCenter

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -4
                    onClicked: view.resetEq()
                }
            }
        }

        Row {
            id: sliders
            width: parent.width
            height: parent.height - 24
            spacing: 2

            Repeater {
                model: 11

                Item {
                    id: slot
                    required property int index
                    readonly property bool focused: view.eqIdx === index
                    readonly property real gain: pane.gainAt(index)
                    width: Math.floor((sliders.width - 20) / 11) + (index === 1 ? 10 : 0)
                    height: sliders.height

                    Column {
                        anchors.right: parent.right
                        width: parent.width - (slot.index === 1 ? 10 : 0)
                        height: parent.height
                        spacing: 2

                        Text {
                            textFormat: Text.PlainText
                            width: parent.width
                            height: 14
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter
                            text: pane.dbText(slot.gain)
                            color: slot.focused ? Theme.good : Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                        }

                        Item {
                            id: track
                            width: parent.width
                            height: parent.height - 32

                            Rectangle {
                                width: 4
                                height: parent.height
                                radius: 2
                                anchors.horizontalCenter: parent.horizontalCenter
                                color: Theme.mantle
                            }

                            Rectangle {
                                anchors.left: parent.left
                                anchors.right: parent.right
                                y: parent.height / 2
                                height: 1
                                color: Theme.muted
                            }

                            Rectangle {
                                width: 4
                                anchors.horizontalCenter: parent.horizontalCenter
                                y: Math.min(parent.height / 2, knob.y + 3)
                                height: Math.max(2, Math.abs(parent.height / 2 - (knob.y + 3)))
                                color: slot.focused ? Theme.good : Theme.border
                            }

                            Rectangle {
                                id: knob
                                width: 10
                                height: 6
                                radius: 1
                                anchors.horizontalCenter: parent.horizontalCenter
                                y: Math.max(0, Math.min(parent.height - height, (12 - slot.gain) / 24 * parent.height - height / 2))
                                color: slot.focused ? Theme.good : Theme.foreground
                            }

                            MouseArea {
                                anchors.fill: parent
                                acceptedButtons: Qt.LeftButton
                                onPressed: function(mouse) {
                                    view.eqIdx = slot.index
                                    view.focusPane("playlist")
                                    view.setEqGain(slot.index, pane.dbFromY(mouse.y, height))
                                }
                                onPositionChanged: function(mouse) {
                                    if (!pressed)
                                        return
                                    view.setEqGain(slot.index, pane.dbFromY(mouse.y, height))
                                }
                                onDoubleClicked: view.setEqGain(slot.index, 0)
                            }

                            WheelHandler {
                                onWheel: function(event) {
                                    view.eqIdx = slot.index
                                    var step = event.modifiers & Qt.ShiftModifier ? 3 : 1
                                    var dir = event.angleDelta.y < 0 ? -1 : 1
                                    view.setEqGain(slot.index, slot.gain + dir * step)
                                }
                            }
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: parent.width
                            height: 14
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter
                            text: pane.marks[slot.index]
                            color: slot.focused ? Theme.border : Theme.muted
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                            elide: Text.ElideRight
                        }
                    }
                }
            }
        }
    }
}
