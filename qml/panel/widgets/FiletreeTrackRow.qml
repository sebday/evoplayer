import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Rectangle {
    required property var dashboard

    id: browseRow
    property var track: ({})
    property int trackRevision: 0
    property bool selected: false
    property bool playing: false
    property int rowWidth: 0
    property string genreLabel: ""
    property bool showGenre: true
    property bool showFolder: false
    property bool showActionButtons: true
    property bool externalClick: false
    property bool deferArtWarm: false
    signal pressed()
    signal playRequested()
    signal likeToggled()
    signal revealRequested()
    signal folderOpenRequested()
    signal addRequested()

    readonly property int rowFont: dashboard && dashboard.listFont !== undefined
        ? dashboard.listFont
        : Theme.fontSizeL

    readonly property bool trackLiked: {
        if (!dashboard)
            return false
        var _rev = browseRow.trackRevision
        var _likes = dashboard.likedByPath
        var _playerLiked = dashboard.player && dashboard.player.liked
        var path = String((browseRow.track && browseRow.track.path) || "")
        return dashboard.trackIsLiked(path, browseRow.track && browseRow.track.liked)
    }
    readonly property string artistAlbumLine: {
        var artist = String(browseRow.track.artist || "").trim()
        var album = String(browseRow.track.album || "").trim()
        var title = String(browseRow.track.title || "").trim()
        if (album === title)
            album = ""
        if (artist && album)
            return artist + " - " + album
        return artist || album
    }
    readonly property string rowArtPath: {
        if (!dashboard)
            return ""
        var _rev = browseRow.trackRevision
        var _tracksRev = browseRow.externalClick
            ? dashboard.filetreeArtRevision
            : dashboard.tracksRevision
        return dashboard.trackArtForRow(browseRow.track)
    }

    readonly property string rowArtSource: {
        var path = String((browseRow.track && browseRow.track.path) || "")
        return browseRow.rowArtPath ? dashboard.rowArtUrl(browseRow.rowArtPath, path) : ""
    }

    Component.onCompleted: {
        if (browseRow.deferArtWarm || !dashboard)
            return
        var path = String((browseRow.track && browseRow.track.path) || "")
        if (path && !browseRow.rowArtPath)
            dashboard.queueRowArtWarm(path)
    }

    readonly property int genreReserve: browseRow.showGenre && browseRow.genreLabel !== "" ? 108 : 0
    readonly property int likeReserve: 30
    readonly property int folderReserve: browseRow.showFolder ? 30 : 0
    readonly property int actionButtonReserve: browseRow.showActionButtons && browseRow.selected
        ? (22 * 5 + Theme.spacingM * 4)
        : 0
    readonly property int controlsReserve: browseRow.likeReserve + browseRow.actionButtonReserve
        + browseRow.genreReserve + browseRow.folderReserve
    property bool hoveredExternal: false
    readonly property bool hovered: browseRow.externalClick
        ? browseRow.hoveredExternal
        : browseRowMouse.containsMouse

    width: rowWidth
    height: 40
    radius: Theme.radiusL
    color: playing
        ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.08)
        : (selected
            ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.22)
            : (browseRowMouse.containsMouse
                ? Theme.foregroundGhost
                : "transparent"))

    function activateRow() {
        if (browseRow.selected)
            browseRow.playRequested()
        else
            browseRow.pressed()
    }

    RowLayout {
        id: browseRowLayout
        z: 0
        anchors.fill: parent
        anchors.leftMargin: 8
        anchors.rightMargin: 8
        spacing: Theme.spacingM

        Item {
            Layout.preferredWidth: 36
            Layout.preferredHeight: 36

            Rectangle {
                anchors.fill: parent
                radius: Theme.radiusL
                clip: true
                color: Theme.foregroundFaint
                visible: browseArt.opacity < 0.01
            }

            Text {
                anchors.centerIn: parent
                visible: browseArt.opacity < 0.01
                text: "󰎈"
                color: Theme.accent
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeL
                opacity: 0.35
            }

            Image {
                id: browseArt
                anchors.fill: parent
                opacity: status === Image.Ready ? 1 : 0
                source: browseRow.rowArtSource
                sourceSize: Qt.size(72, 72)
                fillMode: Image.PreserveAspectCrop
                smooth: true
                mipmap: true
                asynchronous: true
                cache: true
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.preferredHeight: 40
            Layout.minimumWidth: 48

            ColumnLayout {
                anchors.fill: parent
                spacing: 0

                Text {
                    Layout.fillWidth: true
                    text: browseRow.track.title || ""
                    color: Theme.accent
                    font.family: Theme.fontFamily
                        font.pixelSize: browseRow.rowFont
                        elide: Text.ElideRight
                        opacity: browseRow.selected || browseRow.playing ? 1 : 0.9
                    }

                    Text {
                        Layout.fillWidth: true
                        visible: browseRow.artistAlbumLine !== ""
                        text: browseRow.artistAlbumLine
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: browseRow.rowFont
                    elide: Text.ElideRight
                    opacity: Theme.opacityMuted
                }
            }
        }

        MetaChip {
            dashboard: browseRow.dashboard
            visible: browseRow.showGenre && browseRow.genreLabel !== ""
            label: browseRow.genreLabel
            accent: true
            clickable: false
            maxLabelWidth: 96
            Layout.alignment: Qt.AlignVCenter
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            visible: browseRow.showFolder && !(browseRow.showActionButtons && browseRow.selected)
            icon: "󰉖"
            onActivated: browseRow.folderOpenRequested()
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            z: 10
            visible: browseRow.showActionButtons && browseRow.selected
            icon: "󰆏"
            tooltip: "copy artist - title"
            onActivated: dashboard.copyTrackArtistTitle(browseRow.track)
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            z: 10
            visible: browseRow.showActionButtons && browseRow.selected
            icon: "󰉖"
            tooltip: "open folder"
            onActivated: browseRow.folderOpenRequested()
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            z: 10
            visible: browseRow.showActionButtons && browseRow.selected
            icon: "󰐕"
            tooltip: "add to current"
            onActivated: browseRow.addRequested()
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            z: 12
            visible: browseRow.showActionButtons && browseRow.selected
            icon: "󰐊"
            iconColor: Theme.accent
            tooltip: "play"
            hitPadding: 0
            opacityIdle: 0.55
            opacityHover: 1
            onActivated: browseRow.playRequested()
        }

        RowIconButton {
            dashboard: browseRow.dashboard
            z: 11
            hitPadding: 0
            icon: "󰋑"
            tooltip: browseRow.trackLiked ? "unlike" : "like"
            iconColor: browseRow.trackLiked ? Theme.urgent : Theme.foreground
            opacityIdle: browseRow.trackLiked ? 1 : 0.28
            opacityHover: browseRow.trackLiked ? 1 : 0.55
            onActivated: browseRow.likeToggled()
        }
    }

    MouseArea {
        id: browseRowMouse
        z: 1
        visible: !browseRow.externalClick
        anchors.fill: parent
        anchors.rightMargin: browseRow.controlsReserve
        acceptedButtons: Qt.LeftButton
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor

        onClicked: function(mouse) {
            mouse.accepted = true
            browseRow.pressed()
        }

        onDoubleClicked: function(mouse) {
            mouse.accepted = true
            if (!browseRow.selected)
                browseRow.pressed()
            browseRow.playRequested()
        }
    }

    MouseArea {
        z: 2
        visible: !browseRow.externalClick
        anchors.fill: parent
        anchors.rightMargin: browseRow.likeReserve
        acceptedButtons: Qt.RightButton
        onClicked: browseRow.revealRequested()
    }
}
