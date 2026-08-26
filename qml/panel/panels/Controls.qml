import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"

Item {
    required property var dashboard

    id: transportBar
    property bool showTimestamps: true
    implicitHeight: dashboard.controlsHeight

    readonly property int fitBtnCount: 6
    readonly property int fitMaxGap: dashboard.compactLayout ? Theme.spacingL : 24
    readonly property int fitMinGap: 2
    readonly property int fitMinBtn: 18
    readonly property int fitMinProgress: 36
    readonly property int fitMaxProgress: 72
    readonly property int fitProgressMinPanelWidth: 425
    readonly property int fitFullWidth: fitBtnCount * dashboard.transportBtnSize
        + 5 * fitMaxGap
        + fitMinProgress
    readonly property bool fitShowProgress: {
        var w = transportBar.width
        if (w <= 1)
            return true
        return w >= fitProgressMinPanelWidth
    }
    readonly property int fitActiveGaps: fitShowProgress ? 6 : 5

    readonly property int fitBtnSize: {
        var w = transportBar.width
        var maxBtn = dashboard.transportBtnSize
        if (w <= 1)
            return maxBtn
        if (!fitShowProgress) {
            var scale = Math.max(0.5, Math.min(1, w / Math.max(1, fitFullWidth)))
            return Math.max(fitMinBtn, Math.round(maxBtn * scale))
        }
        var remain = w - fitActiveGaps * fitGap - fitMinProgress
        return Math.max(fitMinBtn, Math.min(maxBtn, Math.floor(remain / fitBtnCount)))
    }
    readonly property int fitGap: {
        var w = transportBar.width
        var maxGap = fitMaxGap
        var gaps = Math.max(1, fitActiveGaps)
        if (w <= 1)
            return maxGap
        if (!fitShowProgress)
            return Math.max(fitMinGap, Math.min(maxGap, 8))
        var need = fitBtnCount * dashboard.transportBtnSize + gaps * maxGap + fitMaxProgress
        if (need <= w)
            return maxGap
        return Math.max(fitMinGap, maxGap - Math.ceil((need - w) / gaps))
    }
    readonly property int fitProgressMin: {
        if (!fitShowProgress)
            return 0
        var w = transportBar.width
        if (w <= 1)
            return fitMaxProgress
        var remain = w - fitBtnCount * fitBtnSize - fitActiveGaps * fitGap
        return Math.max(fitMinProgress, Math.min(fitMaxProgress, remain))
    }
    readonly property bool fitShowSpacers: {
        if (!fitShowProgress)
            return true
        if (dashboard.compactLayout)
            return false
        var w = transportBar.width
        if (w <= 1)
            return true
        return fitBtnCount * dashboard.transportBtnSize + (fitActiveGaps + 2) * fitMaxGap + fitMaxProgress <= w
    }
    readonly property real fitIconScale: fitBtnSize / dashboard.transportBtnSize

    RowLayout {
        id: transportRow
        anchors.fill: parent
        anchors.leftMargin: dashboard.compactLayout ? 0 : 6
        anchors.rightMargin: dashboard.compactLayout ? 0 : 6
        spacing: dashboard.compactLayout ? Theme.spacingL : 12

        TransportTimePill {
        dashboard: transportBar.dashboard
            visible: showTimestamps
            label: dashboard.transportSnap.position_label
                || dashboard.player.position_label
                || "0:00"
        }

        Item {
            visible: showTimestamps
            Layout.fillWidth: true
            Layout.minimumWidth: 4
        }

        RowLayout {
            id: controlsRow
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: transportBar.fitGap

            Item {
                visible: transportBar.fitShowSpacers
                Layout.fillWidth: true
            }

            TransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
                icon: "󰒮"
                onActivated: dashboard.skipTrack(false)
            }
            TransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
                icon: dashboard.playerPlaying ? "󰏤" : "󰐊"
                accent: true
                onActivated: dashboard.togglePlayback()
            }
            TransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
                icon: "󰒭"
                onActivated: dashboard.skipTrack(true)
            }

            TransportProgressBar {
        dashboard: transportBar.dashboard
                visible: transportBar.fitShowProgress
                minBarWidth: transportBar.fitProgressMin
                preferredBarWidth: Math.max(transportBar.fitProgressMin, 140)
            }

            TransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
                icon: "󰒟"
                smallGlyph: true
                dimmed: !dashboard.player.shuffle
                onActivated: {
                    if (!dashboard.monitorTransport("playback.shuffle", { on: !dashboard.player.shuffle }))
                        dashboard.runPlayer(["shuffle", "toggle"], dashboard.refreshStatus)
                }
            }

            TransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
                icon: "󰋑"
                iconOffsetY: 2
                smallGlyph: true
                liked: dashboard.favoriteApplyPending && dashboard.favoriteApplyPath === String(dashboard.player.path || "")
                    ? dashboard.favoriteApplyLiked
                    : !!dashboard.player.liked
                onActivated: dashboard.toggleFavorite()
            }
            VolumeTransportBtn {
        dashboard: transportBar.dashboard
                btnSize: transportBar.fitBtnSize
                iconScale: transportBar.fitIconScale
            }

            Item {
                visible: transportBar.fitShowSpacers
                Layout.fillWidth: true
            }
        }

        Item {
            visible: showTimestamps
            Layout.fillWidth: true
            Layout.minimumWidth: 4
        }

        TransportTimePill {
        dashboard: transportBar.dashboard
            visible: showTimestamps
            label: dashboard.transportSnap.duration_label
                || dashboard.player.duration_label
                || "0:00"
        }
    }
}
