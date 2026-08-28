import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"

SectionPanel {
    id: filterRoot
    required property var dashboard

    label: ""
    notchLegend: true
    legendText: dashboard.filterLabel || "Filter"
    legendIcon: "󰍉"
    legendBackground: Theme.background
    fillHeight: true

    ListView {
        id: sideFilterTrackList
        Layout.fillWidth: true
        Layout.fillHeight: true
        clip: true
        spacing: Theme.spacing2
        model: dashboard.filterTracks

        Text {
            anchors.centerIn: parent
            visible: dashboard.filterLoading
            text: "loading…"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.listFont
            opacity: Theme.opacityDisabled
        }

        Text {
            anchors.centerIn: parent
            visible: !dashboard.filterLoading && dashboard.filterTracks.length === 0
            text: "no tracks"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.listFont
            opacity: Theme.opacityDisabled
        }

        delegate: FiletreeTrackRow {
            dashboard: filterRoot.dashboard
            required property var modelData
            required property int index
            rowWidth: sideFilterTrackList.width
            track: modelData
            trackRevision: filterRoot.dashboard.tracksRevision
            selected: filterRoot.dashboard.isTrackSelected(modelData.path)
            playing: filterRoot.dashboard.isTrackPlaying(modelData.path)
            showGenre: false
            onPressed: filterRoot.dashboard.selectFilterTrack(index)
            onPlayRequested: filterRoot.dashboard.playFilterTrackAt(index)
            onLikeToggled: filterRoot.dashboard.toggleTrackFavorite(modelData.path || (modelData.track && modelData.track.path) || "", modelData.track || modelData)
            onRevealRequested: filterRoot.dashboard.openTrackInThunar(modelData.path)
            onFolderOpenRequested: filterRoot.dashboard.openTrackFolder(modelData)
            onAddRequested: filterRoot.dashboard.appendTrackToCurrent(modelData)
        }
    }
}
