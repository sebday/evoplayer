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
    legendText: "SoundCloud"
    legendIcon: "󰕧"
    legendBackground: Theme.background
    fillHeight: false

    readonly property string oauthSource: String(dashboard.settingsScOAuthSource || "")
    readonly property string oauthLabel: {
        if (oauthSource === "brave")
            return "brave cookie"
        if (oauthSource === "chromium")
            return "chromium cookie"
        if (oauthSource === "pass")
            return "pass"
        return "not found"
    }

    ColumnLayout {
        Layout.fillWidth: true
        spacing: Theme.spacingL

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            Text {
                text: "User"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                opacity: Theme.opacityMuted
            }

            Rectangle {
                Layout.fillWidth: true
                implicitHeight: 34
                radius: 6
                color: Theme.foregroundWash
                border.color: Theme.foregroundDivider
                border.width: 1

                TextInput {
                    id: settingsScUserInput
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
                    text: dashboard.settingsScUser
                    enabled: dashboard.settingsInputsEnabled
                    onActiveFocusChanged: dashboard.settingsFieldFocused = activeFocus
                    onEditingFinished: {
                        if (dashboard.settingsReady)
                            dashboard.setPlayerSetting("soundcloud.user", text)
                    }

                    Connections {
                        target: dashboard
                        function onSettingsScUserChanged() {
                            if (!settingsScUserInput.activeFocus)
                                settingsScUserInput.text = dashboard.settingsScUser
                        }
                    }
                }
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            Text {
                text: "OAuth"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                opacity: Theme.opacityMuted
            }

            Text {
                Layout.fillWidth: true
                text: oauthLabel
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
            }

            Text {
                Layout.fillWidth: true
                text: "Uses a logged-in SoundCloud session in Brave, or pass show omarchy/soundcloud/oauth-token"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                wrapMode: Text.Wrap
                opacity: 0.55
            }
        }
    }
}
