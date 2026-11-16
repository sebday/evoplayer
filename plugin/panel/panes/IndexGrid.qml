import QtQuick
import QtQuick.Controls
import "../compat"

Item {
    id: root

    required property var view

    readonly property bool artists: view.page && view.page.kind === "artists"
    readonly property bool albums: view.page && view.page.kind === "albums"

    function readScroll() { return grid.contentY }
    function writeScroll(y) { grid.contentY = y }
    function scrollMax() { return Math.max(0, grid.contentHeight - grid.height) }

    GridView {
        id: grid
        anchors.top: parent.top
        anchors.topMargin: 8
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        model: root.view.indexRows || []
        currentIndex: root.view.indexIdx

        readonly property int columns: Math.max(1, Math.floor(width / 150))
        cellWidth: Math.floor(width / columns)
        cellHeight: cellWidth + 34
        onColumnsChanged: root.view.indexColumns = columns
        Component.onCompleted: root.view.indexColumns = columns
        highlightFollowsCurrentItem: false
        cacheBuffer: cellHeight

        ScrollBar.vertical: ScrollBar {
            policy: ScrollBar.AsNeeded
            implicitWidth: 7
            contentItem: Rectangle {
                implicitWidth: 3
                radius: width / 2
                color: Theme.muted
                opacity: 0.6
            }
            background: Item {}
        }

        Text {
            anchors.centerIn: parent
            visible: grid.count === 0 && !root.view.pageBusy
            textFormat: Text.PlainText
            text: root.view.indexFilter ? "no matches" : "nothing here yet"
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
        }

        Connections {
            target: root.view
            function onIndexIdxChanged() {
                if (root.view.holdScroll)
                    return
                if (root.view.indexIdx >= 0 && root.view.indexIdx < grid.count)
                    grid.positionViewAtIndex(root.view.indexIdx, GridView.Contain)
            }
            function onPageChanged() {
                if (!root.view.indexPage())
                    return
                Qt.callLater(function() {
                    if (root.view.holdScroll)
                        return
                    if (root.view.indexIdx < grid.count)
                        grid.positionViewAtIndex(root.view.indexIdx, GridView.Contain)
                })
            }
        }

        delegate: IndexTile {
            required property int index
            required property var modelData

            width: grid.cellWidth
            height: grid.cellHeight
            view: root.view
            row: modelData
            round: root.artists
            starred: (root.artists || root.albums) && !!modelData.starred
            selected: root.view.pane === "playlist" && !root.view.textCapture && index === root.view.indexIdx
            onActivated: {
                root.view.indexIdx = index
                root.view.focusPane("playlist")
                root.view.openIndexAt(index)
            }
            onContextActivated: root.view.openFieldTags(root.view.indexKind(), String(modelData.name || ""))
        }
    }
}
