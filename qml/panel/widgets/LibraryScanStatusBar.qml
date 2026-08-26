import QtQuick
import QtQuick.Layouts
import Quickshell
import "../compat"
import "."

Item {
    required property var dashboard

    id: scanBar
    implicitHeight: dashboard.genreTabHeight
    Layout.fillWidth: true
    Layout.minimumWidth: 0

    readonly property string phaseText: dashboard.libraryScanPhaseText()
    readonly property string folderText: dashboard.libraryScanFolderText()
    readonly property string cpuText: dashboard.libraryCpuLoadLine()
    readonly property bool cpuHot: dashboard.ffmpegCpuPercent >= 50
        || dashboard.evoplayerCpuPercent >= 50

    RowLayout {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        spacing: Theme.spacingS

        Text {
            Layout.alignment: Qt.AlignVCenter
            text: dashboard.libraryActivityBannerLabel()
            color: scanBar.cpuHot ? Theme.urgent : Theme.accent
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeXs
            font.bold: Theme.fontBold
        }

        Rectangle {
            Layout.preferredWidth: 1
            Layout.preferredHeight: 12
            color: Theme.foregroundDivider
        }

        Text {
            Layout.alignment: Qt.AlignVCenter
            text: scanBar.phaseText
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            opacity: 0.82
        }

        Text {
            visible: scanBar.folderText !== ""
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignVCenter
            Layout.minimumWidth: 0
            text: scanBar.folderText
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            opacity: 0.82
            elide: Text.ElideLeft
        }

        Item {
            visible: scanBar.folderText === ""
            Layout.fillWidth: true
            Layout.minimumWidth: 0
        }

        Text {
            visible: scanBar.cpuText !== ""
            Layout.alignment: Qt.AlignVCenter
            text: scanBar.cpuText
            color: scanBar.cpuHot ? Theme.urgent : Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            opacity: 0.82
        }

        RowIconButton {
            dashboard: scanBar.dashboard
            icon: "󰓛"
            tooltip: "Stop scan"
            iconColor: Theme.urgent
            opacityIdle: 0.72
            opacityHover: 1
            onActivated: dashboard.stopScanJob()
        }
    }
}
