import QtQuick
import QtQuick.Controls
import "../compat"
import "../widgets"

Item {
    id: root

    required property var view

    readonly property bool artists: view.page && view.page.kind === "artists"

    Row {
        id: titleRow
        width: parent.width
        height: 34
        spacing: 14

        Text {
            anchors.verticalCenter: parent.verticalCenter
            textFormat: Text.PlainText
            text: root.artists ? "Artists" : "Labels"
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.bold: true
            font.pixelSize: Theme.fontSize2xl
        }

        Text {
            anchors.verticalCenter: parent.verticalCenter
            textFormat: Text.PlainText
            text: root.view.pageBusy ? "loading…" : String((root.view.indexRows || []).length)
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }
    }

    Rectangle {
        id: filterBox
        anchors.right: parent.right
        anchors.verticalCenter: titleRow.verticalCenter
        width: Math.min(260, parent.width * 0.4)
        height: 26
        radius: height / 2
        color: Theme.mantle
        border.width: 1
        border.color: filterField.activeFocus ? Theme.good : Theme.inactiveBorder

        Text {
            anchors.fill: filterField
            visible: filterField.text === ""
            textFormat: Text.PlainText
            text: root.artists ? "filter artists" : "filter labels"
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            verticalAlignment: Text.AlignVCenter
        }

        TextInput {
            id: filterField
            anchors.fill: parent
            anchors.leftMargin: 12
            anchors.rightMargin: 12
            verticalAlignment: TextInput.AlignVCenter
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
            clip: true
            selectByMouse: true
            selectionColor: Theme.good
            selectedTextColor: Theme.background
            onTextChanged: root.view.setIndexFilter(text)
            onActiveFocusChanged: {
                if (activeFocus) {
                    root.view.pane = "playlist"
                    root.view.textCapture = true
                } else if (root.view.textCapture) {
                    root.view.textCapture = false
                }
            }
            Keys.priority: Keys.BeforeItem
            Keys.onPressed: function(event) {
                if (root.view.dispatch(event))
                    event.accepted = true
            }
        }
    }

    GridView {
        id: grid
        anchors.top: titleRow.bottom
        anchors.topMargin: 12
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
                if (root.view.indexIdx >= 0 && root.view.indexIdx < grid.count)
                    grid.positionViewAtIndex(root.view.indexIdx, GridView.Contain)
            }
            function onIndexFilterFocusChanged() {
                filterField.forceActiveFocus()
            }
            function onPageChanged() {
                if (!root.view.indexPage())
                    return
                if (filterField.text !== root.view.indexFilter)
                    filterField.text = root.view.indexFilter
                Qt.callLater(function() {
                    if (root.view.indexIdx < grid.count)
                        grid.positionViewAtIndex(root.view.indexIdx, GridView.Center)
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
                source: root.view.rowArtUrl(modelData)
                decodeSize: 200
                scale: hot ? 1.04 : 1

                Behavior on scale {
                    NumberAnimation { duration: Theme.motionFast }
                }
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
                cursorShape: Qt.PointingHandCursor
                onClicked: {
                    root.view.indexIdx = index
                    root.view.focusPane("playlist")
                    root.view.openIndexAt(index)
                }
            }
        }
    }
}
