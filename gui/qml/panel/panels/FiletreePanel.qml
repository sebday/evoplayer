import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "../widgets"
import "../filetree/FiletreeLogic.js" as FiletreeLogic

SectionPanel {
    id: filetreePanel
    required property var dashboard

    label: ""
    notchLegend: true
    legendText: "File tree"
    legendIcon: "󰉋"
    legendBackground: Theme.background
    fillHeight: true

    onVisibleChanged: {
        if (visible)
            dashboard.kickSidePanels()
    }

    ListView {
        id: filetreeListRoot
        Layout.fillWidth: true
        Layout.fillHeight: true
        clip: true
        spacing: Theme.spacing2
        model: dashboard.filetreeRows
        reuseItems: true
        cacheBuffer: height > 0 ? height * 2 : 480

        ScrollBar.vertical: PanelScrollBar { }

        Component.onCompleted: dashboard.attachFiletreeList(filetreeListRoot)

        onHeightChanged: {
            if (height > 8 && width > 8 && dashboard.filetreePanelOpen)
                dashboard.kickSidePanels()
        }

        onWidthChanged: {
            if (height > 8 && width > 8 && dashboard.filetreePanelOpen)
                dashboard.kickSidePanels()
        }

        onMovementStarted: dashboard.pauseFiletreeArtWarm()

        onMovementEnded: {
            dashboard.resumeFiletreeArtWarm()
            dashboard.saveFiletreeScroll()
            if (atYEnd)
                dashboard.loadMoreFiletreeTracks()
        }

        onFlickingChanged: {
            if (!flicking && !moving) {
                dashboard.resumeFiletreeArtWarm()
                dashboard.saveFiletreeScroll()
            }
        }

        onContentYChanged: {
            dashboard.noteFiletreeScrollActivity()
            if (FiletreeLogic.shouldAcceptFiletreeScrollSave({
                scrollLocked: dashboard.filetreeScrollLocked,
                holdY: dashboard.filetreeHoldY,
                restoreY: dashboard.filetreeRestoreY,
                moving: moving,
                flicking: flicking,
                artPatchInFlight: dashboard.filetreeArtOnlyPatch
            }))
                dashboard.saveFiletreeScroll()
        }

        onContentHeightChanged: {
            if (dashboard.filetreeHoldY < 0)
                return
            dashboard.applyFiletreeHoldViewport()
        }

        Text {
            anchors.centerIn: parent
            visible: dashboard.filetreeTreeLoading && dashboard.filetreeRows.length === 0
            text: "loading…"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.listFont
            opacity: Theme.opacityDisabled
        }

        footer: Text {
            width: filetreeListRoot.width
            visible: dashboard.filetreeLoadingMore
            horizontalAlignment: Text.AlignHCenter
            text: "loading more…"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.listFont
            opacity: Theme.opacityDisabled
        }

        delegate: Item {
            required property var modelData
            required property int index
            readonly property var dash: filetreePanel.dashboard
            readonly property bool isDir: modelData.type === "dir"
            readonly property bool folderSelected: isDir
                && String(dash.selectedFiletreeFolderPath) === String(modelData.path || "")
            readonly property int indent: 8 + (Number(modelData.depth || 0) * 14)
            readonly property int trackLeft: indent + 14
            readonly property string folderRowPath: isDir ? String(modelData.path || "") : ""
            readonly property var folderRowEntry: isDir
                ? { type: "dir", path: folderRowPath, name: String(modelData.name || "") }
                : null
            readonly property int folderActionIconCount: folderSelected ? 4 : 2
            readonly property int folderActionReserve: FiletreeLogic.folderActionReserve(
                folderActionIconCount,
                modelData.count !== undefined && modelData.count !== null)
            width: filetreeListRoot.width
            height: isDir ? 34 : 40

            Rectangle {
                id: folderRow
                anchors.fill: parent
                visible: isDir
                enabled: isDir
                radius: Theme.radiusL
                color: folderSelected
                    ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.22)
                    : (treeRowMouse.containsMouse
                        ? Theme.foregroundGhost
                        : "transparent")

                Text {
                    x: indent + 5
                    anchors.verticalCenter: parent.verticalCenter
                    text: modelData.expanded ? "󰅃" : "󰅂"
                    color: Theme.foreground
                    opacity: treeRowMouse.containsMouse ? 0.9 : 0.45
                    font.family: Theme.fontFamily
                    font.pixelSize: dash.listFont
                }

                RowLayout {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.leftMargin: indent + 28
                    anchors.rightMargin: folderActionReserve
                    spacing: Theme.spacingS

                    Text {
                        text: "󰉋"
                        color: Theme.foreground
                        opacity: Theme.opacityMuted
                        font.family: Theme.fontFamily
                        font.pixelSize: dash.listFont
                    }

                    Text {
                        Layout.fillWidth: true
                        text: dash.playlistTabLabel(modelData.name || "")
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dash.listFont
                        elide: Text.ElideRight
                    }
                }

                MouseArea {
                    id: treeRowMouse
                    anchors.fill: parent
                    anchors.leftMargin: indent
                    anchors.rightMargin: folderActionReserve
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: function(mouse) {
                        mouse.accepted = true
                        dash.selectFiletreeFolder(folderRowPath)
                        dash.toggleFiletreeNode(folderRowPath)
                    }
                }
            }

            Item {
                id: folderActions
                visible: isDir
                z: 30
                anchors.right: parent.right
                y: 0
                width: folderActionReserve
                height: 34

                RowLayout {
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.rightMargin: 6
                    spacing: Theme.spacingS

                    Text {
                        visible: modelData.count !== undefined && modelData.count !== null
                        text: String(modelData.count)
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: dash.libraryFont
                        opacity: Theme.opacityDisabled
                    }

                    RowIconButton {
                        dashboard: dash
                        visible: folderSelected
                        icon: "󰉖"
                        tooltip: "open folder in file manager"
                        hitPadding: 0
                        onActivated: dash.openBrowseFolder(folderRowEntry)
                    }

                    RowIconButton {
                        dashboard: dash
                        visible: folderSelected
                        icon: "󰉚"
                        tooltip: "sort files into album folders"
                        hitPadding: 0
                        onActivated: dash.filetreeSortFolder(folderRowEntry)
                    }

                    RowIconButton {
                        dashboard: dash
                        icon: "󰐕"
                        tooltip: "add folder"
                        iconColor: Theme.accent
                        opacityIdle: 0.55
                        opacityHover: 1
                        hitPadding: 0
                        z: 31
                        onActivated: dash.filetreeAppendFolder(folderRowEntry)
                    }

                    RowIconButton {
                        dashboard: dash
                        icon: "󰐊"
                        tooltip: "play"
                        iconColor: Theme.accent
                        opacityIdle: 0.55
                        opacityHover: 1
                        hitPadding: 0
                        z: 32
                        onActivated: dash.filetreeQueueFolder(folderRowEntry)
                    }
                }
            }

            FiletreeTrackRow {
                id: filetreeTrackRow
                z: 2
                visible: !isDir
                enabled: !isDir
                x: trackLeft
                width: parent.width - trackLeft - 6
                rowWidth: width
                height: 40
                dashboard: dash
                deferArtWarm: true
                externalClick: true
                showFolder: true
                hoveredExternal: filetreeTrackMouse.containsMouse
                track: modelData.track || modelData
                trackRevision: dash.filetreeArtRevision + dash.tracksRevision
                selected: dash.isTrackSelected(dash.trackEntryPath(modelData))
                playing: dash.isTrackPlaying(dash.trackEntryPath(modelData))
                showGenre: false
                onPlayRequested: dash.playFiletreeTrack(modelData)
                onLikeToggled: dash.toggleTrackFavorite(
                    modelData.path || (modelData.track && modelData.track.path) || "",
                    modelData.track || modelData)
                onRevealRequested: dash.openTrackInThunar(modelData.path)
                onFolderOpenRequested: dash.openTrackFolder(modelData)
                onAddRequested: dash.appendTrackToCurrent(
                    modelData.track || modelData)
            }

            MouseArea {
                id: filetreeTrackMouse
                visible: !isDir
                z: 1
                x: trackLeft
                width: Math.max(0, parent.width - trackLeft - filetreeTrackRow.controlsReserve - 6)
                height: 40
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                acceptedButtons: Qt.LeftButton

                onClicked: function(mouse) {
                    mouse.accepted = true
                    var path = dash.trackEntryPath(modelData)
                    if (dash.isTrackPlaying(path)) {
                        dash.selectTrackEntry(modelData)
                        return
                    }
                    if (dash.isTrackSelected(path))
                        dash.playFiletreeTrack(modelData)
                    else
                        dash.selectTrackEntry(modelData)
                }

                onDoubleClicked: function(mouse) {
                    mouse.accepted = true
                    var path = dash.trackEntryPath(modelData)
                    if (dash.isTrackPlaying(path)) {
                        dash.showNowplaying()
                        return
                    }
                    if (!dash.isTrackSelected(path))
                        dash.selectTrackEntry(modelData)
                    dash.playFiletreeTrack(modelData)
                }
            }
        }
    }
}
