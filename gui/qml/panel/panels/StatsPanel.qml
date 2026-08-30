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
    legendText: "Stats"
    legendIcon: "󰄨"
    legendBackground: Theme.background
    fillHeight: true

    onVisibleChanged: {
        if (visible)
            dashboard.loadStatsReport()
    }

    ColumnLayout {
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: Theme.spacing2

        GridLayout {
            Layout.fillWidth: true
            columns: 3
            columnSpacing: Theme.spacingS
            rowSpacing: Theme.spacingS
            visible: !dashboard.statsLoading

            HistoryRecapStatBox {
                label: "Artists"
                value: String(dashboard.statsRecapStat("artists").count)
                deltaPct: dashboard.statsRecapStat("artists").deltaPct
                trendUp: dashboard.statsRecapStat("artists").trendUp
                peak: dashboard.statsRecapStat("artists").peak
                allTime: dashboard.statsRecapStat("artists").allTime
                topName: dashboard.statsRecapStat("artists").topName
                topCount: dashboard.statsRecapStat("artists").topCount
                tintColor: Theme.recapArtistsTint
                clickable: dashboard.statsRecapStat("artists").topName !== ""
                onClicked: {
                    var name = dashboard.statsRecapStat("artists").topName
                    if (name)
                        dashboard.openFilter("artist", name, name)
                }
            }

            HistoryRecapStatBox {
                label: "Albums"
                value: String(dashboard.statsRecapStat("albums").count)
                deltaPct: dashboard.statsRecapStat("albums").deltaPct
                trendUp: dashboard.statsRecapStat("albums").trendUp
                peak: dashboard.statsRecapStat("albums").peak
                allTime: dashboard.statsRecapStat("albums").allTime
                topName: dashboard.statsRecapStat("albums").topName
                topCount: dashboard.statsRecapStat("albums").topCount
                tintColor: Theme.recapAlbumsTint
                clickable: dashboard.statsRecapStat("albums").topName !== ""
                onClicked: {
                    var name = dashboard.statsRecapStat("albums").topName
                    if (name)
                        dashboard.openFilter("album", name, name)
                }
            }

            HistoryRecapStatBox {
                label: "Tracks"
                value: String(dashboard.statsRecapStat("tracks").count)
                deltaPct: dashboard.statsRecapStat("tracks").deltaPct
                trendUp: dashboard.statsRecapStat("tracks").trendUp
                peak: dashboard.statsRecapStat("tracks").peak
                allTime: dashboard.statsRecapStat("tracks").allTime
                topName: dashboard.statsRecapStat("tracks").topName
                topCount: dashboard.statsRecapStat("tracks").topCount
                tintColor: Theme.recapTracksTint
                clickable: dashboard.statsRecapStat("tracks").topName !== ""
                onClicked: {
                    var stat = dashboard.statsRecapStat("tracks")
                    if (stat.topTitle)
                        dashboard.openFilter("track", stat.topTitle, stat.topName)
                    else if (stat.topName)
                        dashboard.openFilter("search", stat.topName, stat.topName)
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.topMargin: Theme.spacing2
            spacing: Theme.spacingS
            visible: !dashboard.statsLoading && dashboard.statsTopTracks().length > 0

            Text {
                Layout.preferredWidth: 22
                text: "#"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
            }

            Text {
                Layout.fillWidth: true
                text: "Track"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
            }

            Text {
                Layout.preferredWidth: 36
                horizontalAlignment: Text.AlignRight
                text: "Plays"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
                opacity: Theme.opacityMuted
            }
        }

        ListView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            spacing: Theme.spacing2
            visible: !dashboard.statsLoading
            model: dashboard.statsTopTracks()

            delegate: Rectangle {
                required property var modelData
                required property int index
                width: parent.width
                height: 42
                radius: Theme.radiusL
                color: statsRowMouse.containsMouse ? Theme.foregroundGhost : "transparent"
                clip: true

                Rectangle {
                    anchors.left: parent.left
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    width: parent.width * ((Number(modelData.count) || 0)
                        / Math.max(1, dashboard.statsTopTrackMaxCount()))
                    color: Theme.withOpacity(Theme.highlight, statsRowMouse.containsMouse ? 0.18 : 0.1)
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 10
                    anchors.rightMargin: 10
                    spacing: Theme.spacingS
                    z: 1

                    Text {
                        Layout.preferredWidth: 22
                        text: String(index + 1)
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.libraryFont
                        opacity: Theme.opacityDisabled
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0

                        Text {
                            Layout.fillWidth: true
                            text: dashboard.statsTrackTitle(modelData)
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: dashboard.listFont
                            elide: Text.ElideRight
                        }

                        Text {
                            Layout.fillWidth: true
                            visible: dashboard.statsTrackArtist(modelData) !== ""
                            text: dashboard.statsTrackArtist(modelData)
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeXs
                            opacity: Theme.opacityMuted
                            elide: Text.ElideRight
                        }
                    }

                    Text {
                        Layout.preferredWidth: 36
                        horizontalAlignment: Text.AlignRight
                        text: String(modelData.count || 0)
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.libraryFont
                        font.bold: Theme.fontBold
                    }
                }

                MouseArea {
                    id: statsRowMouse
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: dashboard.openStatsTrackFilter(modelData)
                }
            }
        }

        Text {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: !dashboard.statsLoading && dashboard.statsTopTracks().length === 0
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            text: "no tracks"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.listFont
            opacity: Theme.opacityDisabled
        }
    }
}
