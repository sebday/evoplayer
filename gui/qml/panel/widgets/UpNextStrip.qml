import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: upNext
    property var tracks: []
    signal revealFirst()

    readonly property bool activityMode: dashboard.libraryActivityBusy
        || dashboard.externalJobBusy
    readonly property string bannerLabel: activityMode
        ? dashboard.libraryActivityBannerLabel()
        : "Up next"
    readonly property string lineText: activityMode
        ? dashboard.libraryActivityLineText()
        : dashboard.upNextLineText(upNext.tracks)
    readonly property bool cpuHot: dashboard.ffmpegCpuPercent >= 50
        || dashboard.evoplayerCpuPercent >= 50

    visible: !dashboard.scanJobRunning && (activityMode || upNext.tracks.length > 0)
    implicitHeight: 26
    Layout.fillWidth: true
    Layout.minimumWidth: 0

    RowLayout {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: Theme.spacingS

        Text {
            Layout.alignment: Qt.AlignVCenter
            text: upNext.bannerLabel
            color: upNext.activityMode && upNext.cpuHot ? Theme.urgent : Theme.accent
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeXs
            font.bold: Theme.fontBold
            opacity: upNextMouse.containsMouse ? 1 : 0.9
        }

        Rectangle {
            Layout.preferredWidth: 1
            Layout.preferredHeight: 12
            color: Theme.foregroundDivider
        }

        Item {
            id: scrollClip
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.minimumWidth: 0
            clip: true

            Text {
                id: lineMeasure
                visible: false
                text: upNext.lineText
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
            }

            readonly property bool needsScroll: lineMeasure.width > scrollClip.width + 6
            readonly property real loopWidth: lineMeasure.width + 24

            Text {
                id: scrollLine
                anchors.verticalCenter: parent.verticalCenter
                text: scrollClip.needsScroll
                    ? (upNext.lineText + "   ·   " + upNext.lineText)
                    : upNext.lineText
                x: scrollClip.needsScroll ? scrollClip.scrollX : 0
                color: upNext.activityMode && upNext.cpuHot ? Theme.urgent : Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
                opacity: upNextMouse.containsMouse ? 0.92 : 0.78
                elide: Text.ElideRight
            }

            property real scrollX: 0

            SequentialAnimation {
                id: marqueeAnim
                running: scrollClip.needsScroll && scrollClip.width > 0
                loops: Animation.Infinite
                NumberAnimation {
                    target: scrollClip
                    property: "scrollX"
                    from: 0
                    to: -scrollClip.loopWidth
                    duration: Math.max(9000, scrollClip.loopWidth * 28)
                    easing.type: Easing.Linear
                }
                PauseAnimation { duration: 600 }
            }

            onLoopWidthChanged: scrollX = 0
            onNeedsScrollChanged: scrollX = 0

            Rectangle {
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                width: 16
                z: 1
                visible: scrollClip.needsScroll
                gradient: Gradient {
                    orientation: Gradient.Horizontal
                    GradientStop { position: 0.0; color: "transparent" }
                    GradientStop { position: 1.0; color: Theme.background }
                }
            }
        }
    }

    MouseArea {
        id: upNextMouse
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: upNext.activityMode ? Qt.ArrowCursor : Qt.PointingHandCursor
        onClicked: {
            if (upNext.activityMode)
                return
            upNext.revealFirst()
        }
    }
}
