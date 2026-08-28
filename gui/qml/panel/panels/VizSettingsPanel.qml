import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"

SectionPanel {
    id: root

    required property var dashboard

    label: ""
    notchLegend: true
    legendText: "Visualizer"
    legendIcon: "󰕾"
    legendBackground: Theme.background
    fillHeight: false

    ColumnLayout {
        Layout.fillWidth: true
        spacing: Theme.spacingM

        Text {
            Layout.fillWidth: true
            text: "Live waveform overlay on the now-playing view. Uses the same CAVA-style analyzer as the bar volume viz."
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeXs
            opacity: Theme.opacityMuted
            wrapMode: Text.WordWrap
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "Frame rate"
                valueSuffix: " fps"
                minimum: 20
                maximum: 60
                step: 1
                value: dashboard.settingsVizFrameRate
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("frame_rate", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "How often bar levels update."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "Sensitivity"
                minimum: 0
                maximum: 400
                step: 5
                value: dashboard.settingsVizSensitivity
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("sensitivity", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Overall bar height multiplier."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "Autosens"
                minimum: 0
                maximum: 5
                step: 1
                value: dashboard.settingsVizAutosens
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("autosens", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Auto-boosts quiet tracks over time. 0 disables."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "Noise reduction"
                minimum: 0
                maximum: 100
                step: 1
                value: dashboard.settingsVizNoiseReduction
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("noise_reduction", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Suppresses background noise and idle hiss."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 4

            Text {
                text: "Monstercat"
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
                    id: monstercatInput
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
                    text: String(dashboard.settingsVizMonstercat)
                    enabled: dashboard.settingsInputsEnabled
                    onActiveFocusChanged: dashboard.settingsFieldFocused = activeFocus
                    onEditingFinished: {
                        if (!dashboard.settingsReady)
                            return
                        var n = parseFloat(String(text).trim())
                        if (isNaN(n))
                            return
                        dashboard.setVizSetting("monstercat", n)
                    }

                    Connections {
                        target: dashboard
                        function onSettingsVizMonstercatChanged() {
                            if (!monstercatInput.activeFocus)
                                monstercatInput.text = String(dashboard.settingsVizMonstercat)
                        }
                    }
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Peak fall smoothing — bars linger and drop more slowly."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "Low cutoff"
                valueSuffix: " Hz"
                minimum: 20
                maximum: 500
                step: 10
                value: dashboard.settingsVizLowCutoff
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("low_cutoff", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Ignore bass below this frequency."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            SliderSetting {
                Layout.fillWidth: true
                label: "High cutoff"
                valueSuffix: " Hz"
                minimum: 1000
                maximum: 20000
                step: 500
                value: Math.min(dashboard.settingsVizHighCutoff, maximum)
                enabled: dashboard.settingsInputsEnabled
                onValueCommitted: function(v) {
                    dashboard.setVizSetting("high_cutoff", v)
                }
            }

            Text {
                Layout.fillWidth: true
                text: "Ignore treble above this frequency."
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
                wrapMode: Text.WordWrap
            }
        }
    }
}
