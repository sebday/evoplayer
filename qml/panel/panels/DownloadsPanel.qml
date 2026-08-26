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
    legendText: "Downloads"
    legendIcon: "󰇚"
    legendBackground: Theme.background
    fillHeight: true

    ColumnLayout {
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: Theme.spacingM

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            Text {
                text: "YouTube or SoundCloud URL"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                opacity: Theme.opacityMuted
            }

            Text {
                Layout.fillWidth: true
                text: "https://www.youtube.com/watch?v=… or soundcloud.com/…/…"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                opacity: 0.45
            }

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 34
                radius: 6
                color: Theme.foregroundWash
                border.color: Theme.foregroundDivider
                border.width: 1

                TextInput {
                    id: youtubeUrlInput
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
                    enabled: !dashboard.libraryJobBusy
                    onActiveFocusChanged: dashboard.settingsFieldFocused = activeFocus
                }
            }

            Item {
                Layout.fillWidth: true
                implicitHeight: 34
                opacity: youtubeUrlInput.text.trim() !== "" && !dashboard.libraryJobBusy ? 1 : 0.45
                enabled: youtubeUrlInput.text.trim() !== "" && !dashboard.libraryJobBusy

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 4
                    anchors.rightMargin: 4
                    spacing: Theme.spacingS

                    Text {
                        text: "󰇚"
                        color: downloadUrlMouse.containsMouse ? Theme.accent : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                    }

                    Text {
                        Layout.fillWidth: true
                        text: "Download"
                        color: downloadUrlMouse.containsMouse ? Theme.accent : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.libraryFont
                        elide: Text.ElideRight
                        opacity: downloadUrlMouse.containsMouse ? 1 : 0.78
                    }
                }

                MouseArea {
                    id: downloadUrlMouse
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: parent.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
                    enabled: parent.enabled
                    onClicked: {
                        var url = String(youtubeUrlInput.text || "").trim()
                        if (!url)
                            return
                        dashboard.runDaemonLibraryJob("library.download", { url: url, import: false },
                            "download url", { key: "download-url", stayOnScreen: true })
                    }
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 1
            color: Theme.foregroundDivider
            opacity: 0.35
        }

        GridLayout {
            Layout.fillWidth: true
            columns: 2
            columnSpacing: Theme.spacingS
            rowSpacing: Theme.spacingS

            Repeater {
                model: dashboard.libraryActions

                Item {
                    required property var modelData
                    readonly property bool actionActive: dashboard.libraryJobBusy
                        && dashboard.activeLibraryJobKey === modelData.key
                    Layout.fillWidth: true
                    implicitHeight: 34
                    opacity: enabled ? 1 : 0.45
                    enabled: !dashboard.libraryJobBusy

                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: 4
                        anchors.rightMargin: 4
                        spacing: Theme.spacingS

                        Text {
                            text: modelData.icon
                            color: actionActive ? Theme.accent : Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: dashboard.listFont
                            opacity: downloadActionMouse.containsMouse ? 1 : 0.85
                        }

                        Text {
                            Layout.fillWidth: true
                            text: modelData.button || modelData.label
                            color: actionActive ? Theme.accent : Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: dashboard.libraryFont
                            elide: Text.ElideRight
                            opacity: downloadActionMouse.containsMouse ? 1 : 0.78
                        }
                    }

                    MouseArea {
                        id: downloadActionMouse
                        anchors.fill: parent
                        hoverEnabled: true
                        z: 1
                        cursorShape: parent.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
                        enabled: parent.enabled
                        onClicked: dashboard.runLibraryAction(modelData)
                    }
                }
            }
        }

        Item {
            Layout.fillWidth: true
            visible: dashboard.libraryActivityBusy
            implicitHeight: 34

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: 4
                anchors.rightMargin: 4
                spacing: Theme.spacingS

                Text {
                    text: "󰓛"
                    color: Theme.urgent
                    font.family: Theme.fontFamily
                    font.pixelSize: dashboard.listFont
                }

                Text {
                    Layout.fillWidth: true
                    text: "Stop processing"
                    color: Theme.foreground
                    font.family: Theme.fontFamily
                    font.pixelSize: dashboard.libraryFont
                    elide: Text.ElideRight
                }
            }

            MouseArea {
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: dashboard.stopLibraryJob()
            }
        }

        Text {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: String(dashboard.jobLog || "").trim() !== "" || dashboard.libraryActivityBusy
            text: String(dashboard.jobLog || "").trim() !== ""
                ? String(dashboard.jobLog)
                : (dashboard.jobLogInline() || "running…")
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            wrapMode: Text.Wrap
            opacity: 0.82
        }
    }
}
