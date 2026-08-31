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
                        dashboard.runDaemonLibraryJob("library.download", { url: url },
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
            visible: (dashboard.downloadFiles || []).length === 0
                && (String(dashboard.jobLog || "").trim() !== "" || dashboard.libraryActivityBusy)
            text: String(dashboard.jobLog || "").trim() !== ""
                ? String(dashboard.jobLog)
                : (dashboard.jobLogInline() || "running…")
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            wrapMode: Text.Wrap
            opacity: 0.82
        }

        Flickable {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: (dashboard.downloadFiles || []).length > 0
            clip: true
            contentWidth: width
            contentHeight: downloadList.implicitHeight
            boundsBehavior: Flickable.StopAtBounds
            interactive: contentHeight > height

            ColumnLayout {
                id: downloadList
                width: parent.width
                spacing: Theme.spacingS

                Repeater {
                    model: dashboard.downloadFiles

                    Rectangle {
                        id: downloadCard
                        required property var modelData
                        readonly property string filePath: String((modelData && modelData.path) || "")
                        readonly property bool needsFolder: !!(modelData && (modelData.needs_folder || modelData.needs_genre))
                        Layout.fillWidth: true
                        implicitHeight: downloadRow.implicitHeight + 12
                        radius: 6
                        color: Theme.foregroundWash

                        ColumnLayout {
                            id: downloadRow
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.top: parent.top
                            anchors.margins: 6
                            spacing: Theme.spacingS

                            RowLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingS

                                Item {
                                    Layout.preferredWidth: 44
                                    Layout.preferredHeight: 44

                                    Image {
                                        id: artImage
                                        anchors.fill: parent
                                        fillMode: Image.PreserveAspectCrop
                                        asynchronous: true
                                        cache: true
                                        source: dashboard.artUrl(modelData && modelData.art, false)
                                        visible: status === Image.Ready
                                    }

                                    Rectangle {
                                        anchors.fill: parent
                                        radius: 4
                                        color: Theme.foregroundDivider
                                        visible: artImage.status !== Image.Ready
                                        Text {
                                            anchors.centerIn: parent
                                            text: "󰝚"
                                            color: Theme.foreground
                                            opacity: 0.45
                                            font.pixelSize: dashboard.listFont
                                        }
                                    }
                                }

                                ColumnLayout {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    Text {
                                        Layout.fillWidth: true
                                        text: String((modelData && modelData.title) || "downloaded")
                                        color: Theme.foreground
                                        font.family: Theme.fontFamily
                                        font.pixelSize: dashboard.libraryFont
                                        elide: Text.ElideRight
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        visible: String((modelData && modelData.artist) || "") !== ""
                                            && String((modelData && modelData.title) || "") !== ""
                                        text: String((modelData && modelData.artist) || "")
                                        color: Theme.foreground
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeS
                                        elide: Text.ElideRight
                                        opacity: 0.7
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: {
                                            var parts = []
                                            var folder = String((modelData && (modelData.folder || modelData.genre)) || "").trim()
                                            var year = String((modelData && modelData.year) || "").trim()
                                            var dur = String((modelData && modelData.duration_label) || "").trim()
                                            if (folder)
                                                parts.push(folder)
                                            if (year)
                                                parts.push(year)
                                            if (dur)
                                                parts.push(dur)
                                            return parts.join("   ·   ")
                                        }
                                        color: Theme.foreground
                                        font.family: Theme.fontFamily
                                        font.pixelSize: Theme.fontSizeS
                                        elide: Text.ElideRight
                                        opacity: 0.55
                                        visible: text !== ""
                                    }
                                }
                            }

                            Text {
                                visible: needsFolder
                                text: "choose a library folder"
                                color: Theme.accent
                                font.family: Theme.fontFamily
                                font.pixelSize: Theme.fontSizeS
                                opacity: 0.85
                            }

                            Flow {
                                Layout.fillWidth: true
                                visible: (dashboard.downloadFolders || []).length > 0
                                spacing: Theme.spacingS
                                z: 2

                                Repeater {
                                    model: dashboard.downloadFolders

                                    delegate: MetaChip {
                                        required property var modelData
                                        dashboard: dashboard
                                        label: String(modelData || "")
                                        clickable: true
                                        accent: String((downloadCard.modelData && (downloadCard.modelData.folder || downloadCard.modelData.genre)) || "") === label
                                        onActivated: dashboard.setIncomingFolder(downloadCard.filePath, label)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
