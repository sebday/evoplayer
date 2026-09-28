import QtQuick
import "../compat"
import "../overlays"
import "../widgets"

Fieldset {
    id: pane

    required property var view

    number: 3
    legend: view.playlistLegend()
    active: view.pane === "playlist"
    hints: view.playlistHints()

    Item {
        anchors.fill: parent

        Text {
            id: errLine
            textFormat: Text.PlainText
            visible: view.err !== ""
            width: parent.width
            height: visible ? 18 : 0
            text: view.err
            color: Theme.liked
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        Text {
            id: noteLine
            textFormat: Text.PlainText
            anchors.top: errLine.bottom
            visible: view.mode === "discover" && view.discoverNote !== ""
            width: parent.width
            height: visible ? 18 : 0
            text: view.discoverNote
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        ListView {
            id: list
            anchors.top: errLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            visible: view.mode === "queue"
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.shownPlaylist ? view.shownTracks : view.queue
            onHeightChanged: view.playlistPage = Math.max(1, Math.floor(height / 22))

            Text {
                textFormat: Text.PlainText
                visible: list.count === 0
                text: "no tracks"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            Connections {
                target: view
                function onPlaylistIdxChanged() {
                    if (view.mode !== "queue" || view.playlistIdx < 0)
                        return
                    list.currentIndex = view.playlistIdx
                    if (view.playlistIdx < list.count)
                        list.positionViewAtIndex(view.playlistIdx, ListView.Contain)
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: list.width
                height: 22

                readonly property bool playing: String(modelData.path || "") !== "" && String(modelData.path) === view.trackPath
                readonly property bool selected: view.pane === "playlist" && view.mode === "queue" && index === view.playlistIdx

                Rectangle {
                    anchors.fill: parent
                    radius: 2
                    color: selected ? Theme.good : "transparent"
                }

                Row {
                    anchors.fill: parent
                    anchors.leftMargin: 2
                    anchors.rightMargin: 2
                    spacing: 8

                    Row {
                        id: titleRow
                        spacing: 4
                        width: parent.width - meta.implicitWidth - 8
                        anchors.verticalCenter: parent.verticalCenter

                        Text {
                            textFormat: Text.PlainText
                            width: 12
                            height: 22
                            verticalAlignment: Text.AlignVCenter
                            text: playing ? ">" : " "
                            color: selected ? Theme.background : Theme.good
                            font.family: Theme.fontFamily
                            font.bold: true
                            font.pixelSize: Theme.fontSizeM
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: Math.max(40, titleRow.width - 16)
                            height: 22
                            verticalAlignment: Text.AlignVCenter
                            text: view.trackLabel(modelData)
                            elide: Text.ElideRight
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.foreground)
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeM
                        }
                    }

                    Row {
                        id: meta
                        spacing: 6
                        anchors.verticalCenter: parent.verticalCenter

                        Text {
                            textFormat: Text.PlainText
                            width: 14
                            horizontalAlignment: Text.AlignHCenter
                            text: modelData.liked ? "♥" : ""
                            color: selected ? Theme.background : Theme.good
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeM
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: 36
                            horizontalAlignment: Text.AlignRight
                            text: view.trackClock(modelData)
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.muted)
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                        }

                        Text {
                            textFormat: Text.PlainText
                            width: 32
                            text: String(modelData.year || "")
                            color: selected ? Theme.background : (playing ? Theme.good : Theme.muted)
                            font.family: Theme.fontFamily
                            font.pixelSize: Theme.fontSizeS
                        }
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    onClicked: view.clickPlaylist(index)
                    onDoubleClicked: view.playSelected()
                }
            }
        }

        ListView {
            id: discoverList
            anchors.top: noteLine.bottom
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            visible: view.mode === "discover"
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            highlightMoveDuration: 0
            model: view.discoverTracks

            Text {
                textFormat: Text.PlainText
                visible: discoverList.count === 0 && view.discoverNote === ""
                text: "nothing new"
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
            }

            Connections {
                target: view
                function onDiscoverIdxChanged() {
                    if (view.mode !== "discover" || view.discoverIdx < 0)
                        return
                    discoverList.currentIndex = view.discoverIdx
                    if (view.discoverIdx < discoverList.count)
                        discoverList.positionViewAtIndex(view.discoverIdx, ListView.Contain)
                }
            }

            delegate: Item {
                required property int index
                required property var modelData

                width: discoverList.width
                height: 22

                readonly property bool selected: view.pane === "playlist" && view.mode === "discover" && index === view.discoverIdx
                readonly property string mark: view.discoverMark(modelData)

                Rectangle {
                    anchors.fill: parent
                    radius: 2
                    color: selected ? Theme.good : "transparent"
                }

                Row {
                    anchors.fill: parent
                    anchors.leftMargin: 2
                    anchors.rightMargin: 2
                    spacing: 8

                    Text {
                        textFormat: Text.PlainText
                        width: Math.max(40, parent.width - 70)
                        text: view.trackLabel(modelData)
                        elide: Text.ElideRight
                        color: selected ? Theme.background : (mark !== "" ? Theme.good : Theme.foreground)
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeM
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: 18
                        horizontalAlignment: Text.AlignHCenter
                        text: mark
                        color: selected ? Theme.background : Theme.good
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    Text {
                        textFormat: Text.PlainText
                        width: 36
                        horizontalAlignment: Text.AlignRight
                        text: view.trackClock(modelData)
                        color: selected ? Theme.background : Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    onClicked: view.clickDiscover(index)
                    onDoubleClicked: view.previewDiscover()
                }
            }
        }

        HelpOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "help"
            view: pane.view
        }

        TagEditor {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "tags"
            view: pane.view
        }

        MovePicker {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "move"
            view: pane.view
        }

        ArtPickerOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "art"
            view: pane.view
        }

        SettingsOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "settings"
            view: pane.view
        }

        DownloadOverlay {
            anchors.fill: parent
            anchors.topMargin: errLine.height
            visible: view.mode === "download"
            view: pane.view
        }
    }
}
