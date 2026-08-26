import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: volPopup
    property int level: 100
    property alias sliderPressed: sliderArea.pressed

    signal volumeSet(int percent)
    signal interacted()
    signal wheelNudge(int delta)

    width: 44
    height: 152

    Rectangle {
        anchors.fill: parent
        radius: 6
        color: Theme.background
        border.color: Theme.foregroundTrack
        border.width: 1
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 10
        spacing: Theme.spacingM

        Text {
            Layout.alignment: Qt.AlignHCenter
            Layout.fillWidth: true
            horizontalAlignment: Text.AlignHCenter
            text: volPopup.level + "%"
            color: Theme.accent
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            font.bold: Theme.fontBold
        }

        Item {
            id: sliderHost
            Layout.fillWidth: true
            Layout.preferredHeight: 96

            Rectangle {
                id: volTrack
                anchors.centerIn: parent
                width: 4
                height: parent.height
                radius: Theme.radiusS
                color: Theme.foregroundDivider
            }

            Rectangle {
                anchors.horizontalCenter: volTrack.horizontalCenter
                anchors.bottom: volTrack.bottom
                width: volTrack.width
                height: volTrack.height * (volPopup.level / 100)
                radius: Theme.radiusS
                color: Theme.accent
            }

            Rectangle {
                id: volThumb
                anchors.horizontalCenter: volTrack.horizontalCenter
                anchors.bottom: volTrack.bottom
                anchors.bottomMargin: (volTrack.height - height) * (volPopup.level / 100)
                width: 12
                height: 12
                radius: 6
                color: Theme.accent
                border.color: Theme.panelBackground
                border.width: 2
            }

            MouseArea {
                id: sliderArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.SizeVerCursor

                function volumeAt(mouseY) {
                    var ratio = 1 - Math.max(0, Math.min(1, mouseY / height))
                    return Math.round(ratio * 100)
                }

                onPressed: function(mouse) {
                    volPopup.interacted()
                    volPopup.volumeSet(volumeAt(mouse.y))
                }

                onPositionChanged: function(mouse) {
                    if (pressed) {
                        volPopup.interacted()
                        volPopup.volumeSet(volumeAt(mouse.y))
                    }
                }

                onWheel: function(wheel) {
                    if (!wheel.angleDelta.y)
                        return
                    volPopup.wheelNudge(wheel.angleDelta.y > 0 ? 5 : -5)
                    wheel.accepted = true
                }
            }
        }
    }
}
