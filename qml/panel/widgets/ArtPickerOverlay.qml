import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Rectangle {
    required property var dashboard

    id: artPickerRoot
    radius: Theme.radiusM
    color: Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.97)
    border.color: Theme.foregroundSubtle
    border.width: 1
    clip: true

    readonly property int gridSpacing: 4
    readonly property int gridPad: 6

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: gridPad
        spacing: Theme.spacingS

        Item {
            Layout.fillWidth: true
            implicitHeight: Math.max(artPillControls.height, 22)

            RowLayout {
                id: artPillControls
                anchors.horizontalCenter: parent.horizontalCenter
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingS

                Text {
                    text: "Apply to"
                    color: Theme.foreground
                    font.family: Theme.fontFamily
                    font.pixelSize: dashboard.libraryFont
                    opacity: Theme.opacityDisabled
                }

                MetaChip {
        dashboard: artPickerRoot.dashboard
                    label: "Image Set"
                    accent: dashboard.artApplyScope === "album"
                    clickable: true
                    onActivated: dashboard.selectArtApplyScope("album")
                }

                MetaChip {
        dashboard: artPickerRoot.dashboard
                    label: "This Track Only"
                    accent: dashboard.artApplyScope === "track"
                    clickable: true
                    onActivated: dashboard.selectArtApplyScope("track")
                }

                MetaChip {
        dashboard: artPickerRoot.dashboard
                    visible: (dashboard.displayedArt || "") !== ""
                    label: "Remove"
                    accent: false
                    clickable: true
                    onActivated: dashboard.clearAlbumArt()
                }
            }

            Item {
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                width: 22
                height: 22

                Text {
                    anchors.centerIn: parent
                    text: "󰅖"
                    color: Theme.foreground
                    font.family: Theme.fontFamily
                    font.pixelSize: dashboard.bodyFont
                    opacity: artCloseMouse.containsMouse ? 0.95 : 0.45
                }

                MouseArea {
                    id: artCloseMouse
                    anchors.fill: parent
                    anchors.margins: -4
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: dashboard.closeArtPicker()
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 28
            radius: 6
            color: Theme.foregroundWash
            border.color: artSearchInput.activeFocus
                ? Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.35)
                : Theme.foregroundDivider
            border.width: 1

            TextInput {
                id: artSearchInput
                anchors.fill: parent
                anchors.leftMargin: 8
                anchors.rightMargin: 8
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
                selectionColor: Theme.accent
                selectedTextColor: Theme.mantle
                verticalAlignment: TextInput.AlignVCenter
                clip: true
                text: dashboard.artPickerSearchText
                onActiveFocusChanged: dashboard.artPickerSearchFocused = activeFocus
                onTextChanged: {
                    if (text === dashboard.artPickerSearchText)
                        return
                    dashboard.queueArtSearch(text)
                }
                onAccepted: dashboard.searchArtPicker(text)
                Keys.onEscapePressed: {
                    if (text !== "") {
                        dashboard.queueArtSearch("")
                        text = ""
                    } else {
                        dashboard.closeArtPicker()
                    }
                }
            }

            Text {
                anchors.verticalCenter: parent.verticalCenter
                anchors.left: parent.left
                anchors.leftMargin: 8
                visible: !artSearchInput.text
                text: dashboard.artPickerQuery !== "" ? dashboard.artPickerQuery : "search discogs…"
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
                opacity: 0.4
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: dashboard.artPickerLoading

            Text {
                anchors.centerIn: parent
                text: "󰇘"
                color: Theme.accent
                font.family: Theme.fontFamily
                font.pixelSize: Math.round(Math.min(parent.width, parent.height) * 0.28)
                opacity: Theme.opacityEmphasis

                SequentialAnimation on opacity {
                    running: dashboard.artPickerLoading
                    loops: Animation.Infinite
                    NumberAnimation { from: 0.28; to: 1.0; duration: 650; easing.type: Easing.InOutSine }
                    NumberAnimation { from: 1.0; to: 0.28; duration: 650; easing.type: Easing.InOutSine }
                }
            }
        }

        Text {
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: !dashboard.artPickerLoading
                && dashboard.artPickerResults.length === 0
                && dashboard.artPendingDropPath === ""
            text: "no results"
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            color: Theme.foreground
            font.family: Theme.fontFamily
            font.pixelSize: dashboard.libraryFont
            opacity: Theme.opacityDisabled
        }

        Item {
            id: artPickerGridHost
            Layout.fillWidth: true
            Layout.fillHeight: true
            visible: !dashboard.artPickerLoading
                && (dashboard.artPickerResults.length > 0 || dashboard.artPendingDropPath !== "")

            readonly property int cellSize: Math.max(
                40,
                Math.floor(Math.min(
                    (width - artPickerRoot.gridSpacing) / 2,
                    (height - artPickerRoot.gridSpacing) / 2)))
            readonly property int gridWidth: cellSize * 2 + artPickerRoot.gridSpacing
            readonly property int gridHeight: cellSize * 2 + artPickerRoot.gridSpacing
            readonly property int xOffset: Math.max(0, Math.floor((width - gridWidth) / 2))
            readonly property int yOffset: Math.max(0, Math.floor((height - gridHeight) / 2))

                Rectangle {
                visible: dashboard.artPendingDropPath !== ""
                x: artPickerGridHost.xOffset
                y: artPickerGridHost.yOffset
                width: artPickerGridHost.gridWidth
                height: artPickerGridHost.gridHeight
                radius: Theme.radiusM
                clip: true
                color: Theme.foregroundWash
                border.color: dropPreviewMouse.containsMouse
                    ? Theme.accent
                    : Theme.foregroundRaised
                border.width: dropPreviewMouse.containsMouse ? 2 : 1

                Image {
                    anchors.fill: parent
                    source: dashboard.artPendingDropPath !== ""
                        ? dashboard.artUrl(dashboard.artPendingDropPath, true)
                        : ""
                    fillMode: Image.PreserveAspectCrop
                    smooth: true
                    asynchronous: true
                }

                MouseArea {
                    id: dropPreviewMouse
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: {
                        var imagePath = dashboard.artPendingDropPath
                        if (!imagePath)
                            return
                        dashboard.artPendingDropPath = ""
                        dashboard.setAlbumArtFromFile(imagePath)
                        dashboard.artPickerOpen = false
                    }
                }
            }

            Repeater {
                model: dashboard.artPendingDropPath !== ""
                    ? 0
                    : Math.min(4, dashboard.artPickerResults.length)

                Rectangle {
                    required property int index
                    readonly property var result: dashboard.artPickerResults[index]
                    x: artPickerGridHost.xOffset + (index % 2) * (artPickerGridHost.cellSize + artPickerRoot.gridSpacing)
                    y: artPickerGridHost.yOffset + Math.floor(index / 2) * (artPickerGridHost.cellSize + artPickerRoot.gridSpacing)
                    width: artPickerGridHost.cellSize
                    height: artPickerGridHost.cellSize
                    radius: Theme.radiusM
                    clip: true
                    color: Theme.foregroundWash
                    border.color: artPickMouse.containsMouse
                        ? Theme.accent
                        : Theme.foregroundRaised
                    border.width: artPickMouse.containsMouse ? 2 : 1

                    Image {
                        anchors.fill: parent
                        source: result.url || ""
                        fillMode: Image.PreserveAspectCrop
                        smooth: true
                        asynchronous: true
                    }

                    Rectangle {
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.bottom: parent.bottom
                        height: 16
                        visible: String(result.source || "") !== ""
                        color: Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.78)

                        Text {
                            anchors.centerIn: parent
                            text: String(result.source || "")
                            color: Theme.foreground
                            font.family: Theme.fontFamily
                            font.pixelSize: dashboard.libraryFont
                            opacity: Theme.opacityBodyText
                        }
                    }

                    MouseArea {
                        id: artPickMouse
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: dashboard.applyAlbumArtFromUrl(result.url)
                    }
                }
            }
        }
    }
}
