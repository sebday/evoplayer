import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"

SectionPanel {
    required property var dashboard

    label: ""
    notchLegend: true
    legendText: "Library"
    legendIcon: "󰉖"
    legendBackground: Theme.background
    fillHeight: false

    onVisibleChanged: {
        if (visible)
            dashboard.loadLibraryStats()
    }

    ColumnLayout {
        Layout.fillWidth: true
        spacing: Theme.spacingS

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingS

                Rectangle {
                    Layout.fillWidth: true
                    implicitHeight: 34
                    radius: 6
                    color: Theme.foregroundWash
                    border.color: Theme.foregroundDivider
                    border.width: 1

                    TextInput {
                        id: settingsMusicLibInput
                        anchors.fill: parent
                        anchors.leftMargin: 8
                        anchors.rightMargin: 8
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.libraryFont
                        selectionColor: Theme.accent
                        selectedTextColor: Theme.mantle
                        verticalAlignment: TextInput.AlignVCenter
                        clip: true
                        text: dashboard.settingsMusicLibrary
                        enabled: dashboard.settingsInputsEnabled
                        onActiveFocusChanged: dashboard.settingsFieldFocused = activeFocus
                        onEditingFinished: {
                            if (dashboard.settingsReady)
                                dashboard.setMusicLibrary(text)
                        }

                        Connections {
                            target: dashboard
                            function onSettingsMusicLibraryChanged() {
                                if (!settingsMusicLibInput.activeFocus)
                                    settingsMusicLibInput.text = dashboard.settingsMusicLibrary
                            }
                        }
                    }
                }

                Item {
                    Layout.preferredWidth: 34
                    Layout.preferredHeight: 34

                    Text {
                        anchors.centerIn: parent
                        text: "󰉖"
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeXl
                        opacity: settingsLibPickMouse.enabled
                            ? (settingsLibPickMouse.containsMouse ? 1 : 0.72)
                            : 0.35
                    }

                    MouseArea {
                        id: settingsLibPickMouse
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        enabled: dashboard.settingsInputsEnabled
                        onClicked: dashboard.pickMusicLibrary()
                    }
                }
            }
        }

        Text {
            Layout.fillWidth: true
            text: String(dashboard.libraryStats.tracks || 0) + " tracks · "
                + String(dashboard.libraryStats.genres || 0) + " genres"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeXs
            opacity: Theme.opacityMuted
            elide: Text.ElideRight
        }
    }
}
