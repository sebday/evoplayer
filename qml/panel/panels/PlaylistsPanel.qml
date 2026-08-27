import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"

SectionPanel {
    id: playlistsRoot
    required property var dashboard

    label: ""
    notchLegend: true
    legendText: "Playlists"
    legendIcon: "󰲸"
    legendBackground: Theme.background
    fillHeight: true

    onVisibleChanged: {
        if (visible) {
            dashboard.refreshPlaylistsPanelView()
            dashboard.kickSidePanels()
        }
    }

    ColumnLayout {
        id: playlistPanelColumn
        Layout.fillWidth: true
        Layout.fillHeight: true
        spacing: Theme.spacingS

        RowLayout {
            visible: dashboard.playlistPanelMode === "library"
            Layout.fillWidth: true
            spacing: Theme.spacingS

            Item { Layout.fillWidth: true }

            Text {
                text: "+ add"
                color: Theme.accent
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                font.bold: Theme.fontBold

                MouseArea {
                    anchors.fill: parent
                    anchors.margins: -6
                    cursorShape: Qt.PointingHandCursor
                    onClicked: dashboard.createUserPlaylist("playlist-" + Date.now())
                }
            }
        }

        ListView {
            id: sidePlaylistLibraryList
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: dashboard.playlistPanelMode === "library"
            clip: true
            spacing: Theme.spacing2
            model: dashboard.libraryPlaylists
            boundsBehavior: Flickable.StopAtBounds

            ScrollBar.vertical: PanelScrollBar { }

            Component.onCompleted: dashboard.playlistLibraryListView = sidePlaylistLibraryList

            onHeightChanged: {
                if (height > 8 && width > 8 && dashboard.playlistsPanelOpen)
                    dashboard.kickSidePanels()
            }

            onWidthChanged: {
                if (height > 8 && width > 8 && dashboard.playlistsPanelOpen)
                    dashboard.kickSidePanels()
            }

            onVisibleChanged: {
                if (visible && dashboard.libraryPlaylists.length === 0 && !dashboard.playlistsLoading)
                    dashboard.loadPlaylists(true)
            }

            Text {
                anchors.centerIn: parent
                visible: dashboard.playlistsLoading && dashboard.libraryPlaylists.length === 0
                text: "loading…"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.listFont
                opacity: Theme.opacityDisabled
            }

            delegate: Rectangle {
                required property var modelData
                width: sidePlaylistLibraryList.width
                height: 38
                radius: Theme.radiusL
                color: dashboard.selectedLibraryPlaylist === (modelData.name || "")
                    ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.14)
                    : (sideLibPlaylistMouse.containsMouse
                        ? Theme.foregroundGhost
                        : "transparent")

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 10
                    anchors.rightMargin: 10
                    spacing: Theme.spacingM

                    Text {
                        Layout.fillWidth: true
                        text: dashboard.playlistTabLabel(modelData.name || "")
                        color: dashboard.selectedLibraryPlaylist === (modelData.name || "")
                            ? Theme.accent
                            : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                        font.bold: dashboard.selectedLibraryPlaylist === (modelData.name || "") && Theme.fontBold
                        elide: Text.ElideRight
                    }

                    Text {
                        visible: dashboard.selectedLibraryPlaylist === (modelData.name || "")
                            && dashboard.playlistIsUserEditable(modelData.name || "")
                        text: "󰐊"
                        color: Theme.accent
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                        MouseArea {
                            anchors.fill: parent
                            anchors.margins: -6
                            cursorShape: Qt.PointingHandCursor
                            onClicked: dashboard.openSelectedLibraryPlaylist()
                        }
                    }

                    Text {
                        visible: dashboard.selectedLibraryPlaylist === (modelData.name || "")
                            && dashboard.playlistIsUserEditable(modelData.name || "")
                        text: "󰑗"
                        color: Theme.foreground
                        opacity: 0.7
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                        MouseArea {
                            anchors.fill: parent
                            anchors.margins: -6
                            cursorShape: Qt.PointingHandCursor
                            onClicked: dashboard.renameUserPlaylist(modelData.name, modelData.name + "-renamed")
                        }
                    }

                    Text {
                        visible: dashboard.selectedLibraryPlaylist === (modelData.name || "")
                            && dashboard.playlistIsUserEditable(modelData.name || "")
                        text: "󰆴"
                        color: Theme.urgent
                        opacity: 0.8
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                        MouseArea {
                            anchors.fill: parent
                            anchors.margins: -6
                            cursorShape: Qt.PointingHandCursor
                            onClicked: dashboard.deleteUserPlaylist(modelData.name)
                        }
                    }

                    Text {
                        text: String(modelData.count || 0)
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont
                        opacity: Theme.opacityDisabled
                    }

                    Text {
                        visible: dashboard.playlistCanStar(modelData.name || "")
                        text: modelData.starred === true ? "󰓎" : "󰓒"
                        color: modelData.starred === true ? Theme.accent : Theme.foreground
                        opacity: modelData.starred === true
                            ? 1
                            : (playlistStarMouse.containsMouse ? 0.75 : 0.35)
                        font.family: Theme.fontFamily
                        font.pixelSize: dashboard.listFont

                        MouseArea {
                            id: playlistStarMouse
                            anchors.fill: parent
                            anchors.margins: -6
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: function(mouse) {
                                mouse.accepted = true
                                dashboard.togglePlaylistStar(modelData.name)
                            }
                        }
                    }
                }

                MouseArea {
                    id: sideLibPlaylistMouse
                    anchors.fill: parent
                    anchors.rightMargin: dashboard.playlistCanStar(modelData.name || "") ? 36 : 0
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: dashboard.selectGenrePlaylist(modelData.name)
                    onDoubleClicked: {
                        dashboard.selectGenrePlaylist(modelData.name)
                        dashboard.selectPlaylist(modelData.name)
                    }
                }
            }
        }

        ListView {
            id: sidePlaylistTrackList
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: dashboard.playlistPanelMode === "tracks"
            clip: true
            spacing: Theme.spacing2
            model: dashboard.tracks
            reuseItems: true
            cacheBuffer: height > 0 ? height * 2 : 480

            ScrollBar.vertical: PanelScrollBar { }

            Component.onCompleted: dashboard.playlistTrackList = sidePlaylistTrackList
            Component.onDestruction: {
                if (dashboard.playlistTrackList === sidePlaylistTrackList)
                    dashboard.playlistTrackList = null
            }

            onMovementStarted: dashboard.notePlaylistScrollActivity()

            onMovementEnded: {
                if (atYEnd)
                    dashboard.loadMorePlaylistTracks()
                dashboard.savePlaylistScrollPosition(dashboard.selectedPlaylist)
                dashboard.scheduleVisibleArtWarm()
            }

            onFlickingChanged: {
                if (!flicking && !moving)
                    dashboard.savePlaylistScrollPosition(dashboard.selectedPlaylist)
            }

            onContentYChanged: {
                if (!dashboard.playlistScrollLocked)
                    dashboard.notePlaylistScrollActivity()
            }

            onHeightChanged: {
                if (height > 8 && width > 8 && dashboard.playlistsPanelOpen)
                    dashboard.kickSidePanels()
            }

            Text {
                anchors.centerIn: parent
                visible: dashboard.tracksLoading
                text: "loading…"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.listFont
                opacity: Theme.opacityDisabled
            }

            footer: Text {
                width: sidePlaylistTrackList.width
                visible: dashboard.playlistTracksLoadingMore
                horizontalAlignment: Text.AlignHCenter
                text: "loading more…"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.listFont
                opacity: Theme.opacityDisabled
            }

            delegate: FiletreeTrackRow {
                dashboard: playlistsRoot.dashboard
                required property var modelData
                required property int index
                rowWidth: sidePlaylistTrackList.width
                track: modelData
                trackRevision: playlistsRoot.dashboard.tracksRevision
                selected: playlistsRoot.dashboard.isTrackSelected(modelData.path)
                playing: playlistsRoot.dashboard.isTrackPlaying(modelData.path)
                showGenre: false
                deferArtWarm: true
                onPressed: playlistsRoot.dashboard.selectPlaylistTrack(index)
                onPlayRequested: playlistsRoot.dashboard.playTrackAt(index)
                onLikeToggled: playlistsRoot.dashboard.toggleTrackFavorite(modelData.path || "", modelData)
                onRevealRequested: playlistsRoot.dashboard.openTrackInThunar(modelData.path)
                onFolderOpenRequested: playlistsRoot.dashboard.openTrackFolder(modelData)
                onAddRequested: playlistsRoot.dashboard.appendTrackToCurrent(modelData)
            }
        }
    }
}
