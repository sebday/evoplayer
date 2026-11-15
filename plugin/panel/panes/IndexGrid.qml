import QtQuick
import QtQuick.Controls
import "../compat"
import "../widgets"

Item {
    id: root

    required property var view

    readonly property bool artists: view.page && view.page.kind === "artists"

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

        delegate: Item {
            required property int index
            required property var modelData

            width: grid.cellWidth
            height: grid.cellHeight

            readonly property bool selected: root.view.pane === "playlist" && !root.view.textCapture && index === root.view.indexIdx
            readonly property bool hot: cellArea.containsMouse || selected

            Rectangle {
                anchors.fill: parent
                anchors.margins: 4
                radius: 8
                color: selected ? Theme.withOpacity(Theme.border, 0.16) : (cellArea.containsMouse ? Theme.foregroundWash : "transparent")
                border.width: selected ? 1 : 0
                border.color: Theme.border
            }

            Cover {
                id: face
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.top: parent.top
                anchors.topMargin: 12
                width: parent.width - 36
                height: width
                round: root.artists
                corner: 6
                name: String(modelData.name || "")
                source: root.view.rowThumbUrl(modelData)
                decodeSize: Math.ceil(Math.max(96, grid.cellWidth - 36) * Screen.devicePixelRatio)
                scale: hot ? 1.04 : 1

                Behavior on scale {
                    NumberAnimation { duration: Theme.motionFast }
                }
            }

            Text {
                visible: root.artists && !!modelData.starred
                anchors.right: face.right
                anchors.top: face.top
                anchors.rightMargin: 6
                anchors.topMargin: 4
                textFormat: Text.PlainText
                text: "♥"
                color: Theme.liked
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            Text {
                id: nameText
                anchors.top: face.bottom
                anchors.topMargin: 8
                anchors.horizontalCenter: parent.horizontalCenter
                width: parent.width - 16
                horizontalAlignment: Text.AlignHCenter
                textFormat: Text.PlainText
                text: String(modelData.name || "")
                elide: Text.ElideRight
                color: hot ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.85)
                font.family: Theme.fontFamily
                font.bold: true
                font.pixelSize: Theme.fontSizeM
            }

            Text {
                anchors.top: nameText.bottom
                anchors.topMargin: 2
                anchors.horizontalCenter: parent.horizontalCenter
                textFormat: Text.PlainText
                text: root.view.counted(modelData.count, "track")
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeXs
            }

            MouseArea {
                id: cellArea
                anchors.fill: parent
                hoverEnabled: true
                acceptedButtons: Qt.LeftButton | Qt.RightButton
                cursorShape: Qt.PointingHandCursor
                onClicked: function(mouse) {
                    if (mouse.button === Qt.RightButton) {
                        root.view.openFieldTags(root.view.indexKind(), String(modelData.name || ""))
                        return
                    }
                    root.view.indexIdx = index
                    root.view.focusPane("playlist")
                    root.view.openIndexAt(index)
                }
            }
        }
    }
}
