import QtQuick
import "../compat"

Item {
    id: pane

    required property var view

    readonly property var marks: ["pre", "31", "62", "125", "250", "500", "1k", "2k", "4k", "8k", "16k"]
    readonly property int capH: 28

    function dbFromY(y, h) {
        var travel = Math.max(1, h - capH)
        var t = (y - capH / 2) / travel
        if (t < 0)
            t = 0
        if (t > 1)
            t = 1
        return Math.round(12 - t * 24)
    }

    function capY(gain, h) {
        var travel = Math.max(0, h - capH)
        var t = (12 - gain) / 24
        if (t < 0)
            t = 0
        if (t > 1)
            t = 1
        return t * travel
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

    component EqButton: Rectangle {
        id: btn
        property string label: ""
        property bool lit: false
        property bool armed: false
        property bool dim: false
        signal clicked()

        width: Math.max(24, btnLabel.implicitWidth + 16)
        height: 20
        radius: 4
        anchors.verticalCenter: parent.verticalCenter
        color: armed ? Theme.fillUrgentSubtle : (lit ? Theme.fillAccentSubtle : "transparent")
        border.width: 1
        border.color: armed ? Theme.liked : (lit ? Theme.good : Theme.inactiveBorder)

        Text {
            id: btnLabel
            textFormat: Text.PlainText
            anchors.centerIn: parent
            text: btn.label
            color: btn.armed ? Theme.liked : (btn.lit ? Theme.good : (btn.dim ? Theme.muted : Theme.foreground))
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }

        MouseArea {
            anchors.fill: parent
            onClicked: btn.clicked()
        }
    }

    Item {
        id: head
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        height: 22

        Row {
            anchors.left: parent.left
            height: parent.height
            spacing: 8

            EqButton {
                label: view.eqEnabled ? "on" : "off"
                lit: view.eqEnabled
                dim: !view.eqEnabled
                onClicked: view.toggleEqEnabled()
            }

            EqButton {
                label: "flat"
                onClicked: view.resetEq()
            }
        }

        Row {
            anchors.right: parent.right
            height: parent.height
            spacing: 8

            EqButton {
                label: "save"
                armed: view.eqSaveArmed
                onClicked: view.eqSaveArmed = !view.eqSaveArmed
            }

            Repeater {
                model: 3

                EqButton {
                    required property int index
                    label: String(index + 1)
                    armed: view.eqSaveArmed
                    lit: view.eqPresetActive(index + 1)
                    dim: !(view.eqPresets || [])[index]
                    onClicked: view.pickEqPreset(index + 1)
                }
            }
        }
    }

    Row {
        id: sliders
        anchors.top: head.bottom
        anchors.topMargin: 6
        anchors.bottom: parent.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        spacing: 2

        Repeater {
            model: 11

            Item {
                id: slot
                required property int index
                readonly property bool focused: view.eqIdx === index
                readonly property real gain: pane.gainAt(index)
                readonly property color capColor: focused ? Theme.good : Theme.foregroundBorder
                width: Math.floor((sliders.width - 20) / 11) + (index === 1 ? 10 : 0)
                height: sliders.height

                Item {
                    anchors.fill: parent
                    anchors.leftMargin: slot.index === 1 ? 10 : 0

                    Text {
                        id: db
                        textFormat: Text.PlainText
                        anchors.top: parent.top
                        anchors.left: parent.left
                        anchors.right: parent.right
                        height: 14
                        horizontalAlignment: Text.AlignHCenter
                        text: pane.dbText(slot.gain)
                        color: slot.focused ? Theme.good : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                    }

                    Text {
                        id: freq
                        textFormat: Text.PlainText
                        anchors.bottom: parent.bottom
                        anchors.left: parent.left
                        anchors.right: parent.right
                        height: 14
                        horizontalAlignment: Text.AlignHCenter
                        text: pane.marks[slot.index]
                        color: slot.focused ? Theme.border : Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        elide: Text.ElideRight
                    }

                    Item {
                        id: well
                        anchors.top: db.bottom
                        anchors.bottom: freq.top
                        anchors.topMargin: 4
                        anchors.bottomMargin: 4
                        anchors.left: parent.left
                        anchors.right: parent.right

                        Rectangle {
                            width: 8
                            height: parent.height
                            radius: 3
                            anchors.horizontalCenter: parent.horizontalCenter
                            color: Theme.background
                            border.width: 1
                            border.color: Theme.foregroundTrack
                        }

                        Rectangle {
                            width: 2
                            height: parent.height - 8
                            anchors.centerIn: parent
                            color: Theme.foregroundDivider
                        }

                        Repeater {
                            model: 3
                            Rectangle {
                                required property int index
                                width: 8
                                height: 1
                                x: well.width / 2 - 14
                                y: (index / 2) * Math.max(0, well.height - pane.capH) + pane.capH / 2
                                color: Theme.muted
                            }
                        }

                        Rectangle {
                            id: cap
                            width: Math.max(16, Math.min(parent.width - 2, 26))
                            height: pane.capH
                            radius: 3
                            anchors.horizontalCenter: parent.horizontalCenter
                            y: pane.capY(slot.gain, well.height)
                            color: slot.capColor
                            border.width: 1
                            border.color: slot.focused ? Theme.good : Theme.foreground

                            Rectangle {
                                anchors.left: parent.left
                                anchors.right: parent.right
                                anchors.leftMargin: 3
                                anchors.rightMargin: 3
                                anchors.verticalCenter: parent.verticalCenter
                                height: 2
                                radius: 1
                                color: Theme.background
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            acceptedButtons: Qt.LeftButton
                            onPressed: function(mouse) {
                                view.eqIdx = slot.index
                                view.focusPane("playlist")
                                view.setEqGain(slot.index, pane.dbFromY(mouse.y, well.height))
                            }
                            onPositionChanged: function(mouse) {
                                if (!pressed)
                                    return
                                view.setEqGain(slot.index, pane.dbFromY(mouse.y, well.height))
                            }
                            onDoubleClicked: view.setEqGain(slot.index, 0)
                        }

                        WheelHandler {
                            onWheel: function(event) {
                                view.eqIdx = slot.index
                                var step = (event.modifiers & Qt.ShiftModifier) ? 3 : 1
                                var dir = event.angleDelta.y < 0 ? -1 : 1
                                view.setEqGain(slot.index, slot.gain + dir * step)
                            }
                        }
                    }
                }
            }
        }
    }
}
