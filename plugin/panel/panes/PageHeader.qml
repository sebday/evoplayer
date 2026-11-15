import QtQuick
import "../compat"
import "../widgets"

Column {
    id: hero

    required property var view

    readonly property var info: view.headerInfo()
    readonly property bool entity: view.entityPage()
    readonly property var entityData: entity && view.pageData ? view.pageData : ({})
    readonly property bool queueHeader: info.kind === "queue"
    readonly property bool liveTrack: {
        if (queueHeader)
            return false
        var path = String(view.trackPath || "")
        if (!path)
            return false
        var rows = view.playlistRows() || []
        for (var i = 0; i < rows.length; i++) {
            if (String(rows[i] && rows[i].path || "") === path)
                return true
        }
        return false
    }
    readonly property string liveArt: view.artSource()
    readonly property int waveH: 72
    readonly property color onBanner: Theme.foreground
    readonly property string pageKind: view.page ? String(view.page.kind) : ""

    spacing: 14
    bottomPadding: 12

    component SectionLabel: Text {
        textFormat: Text.PlainText
        color: Theme.muted
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
        font.bold: true
        font.capitalization: Font.AllUppercase
        font.letterSpacing: 1
    }

    Item {
        id: banner
        width: hero.width
        height: {
            var textH = textCol.implicitHeight + textCol.anchors.topMargin
            if (hero.queueHeader || hero.liveTrack)
                return textH + 8 + hero.waveH + 14
            return Math.max(188, textH + 28)
        }

        BlurBackdrop {
            anchors.fill: parent
            source: hero.liveTrack ? (hero.liveArt || hero.info.blur || "") : (hero.info.blur || "")
        }

        Column {
            id: textCol
            anchors.left: parent.left
            anchors.leftMargin: 18
            anchors.right: artSlot.left
            anchors.rightMargin: 16
            anchors.top: parent.top
            anchors.topMargin: 18
            spacing: 6

            Row {
                id: captionRow
                width: parent.width
                spacing: 10

                Rectangle {
                    id: captionBadge
                    anchors.verticalCenter: parent.verticalCenter
                    height: 18
                    width: captionText.implicitWidth + 14
                    radius: 3
                    color: Theme.withOpacity(Theme.foreground, 0.12)

                    Text {
                        id: captionText
                        anchors.centerIn: parent
                        textFormat: Text.PlainText
                        text: hero.info.caption
                        color: hero.onBanner
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeXs
                        font.bold: true
                        font.capitalization: Font.AllUppercase
                        font.letterSpacing: 1
                    }
                }

                Text {
                    visible: hero.queueHeader && text !== ""
                    anchors.verticalCenter: parent.verticalCenter
                    width: Math.max(0, captionRow.width - captionBadge.width - captionRow.spacing)
                    height: implicitHeight
                    verticalAlignment: Text.AlignVCenter
                    textFormat: Text.PlainText
                    text: hero.info.stats
                    color: Theme.withOpacity(hero.onBanner, 0.7)
                    elide: Text.ElideRight
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                }

                LinkText {
                    visible: hero.view.page !== null || (hero.view.libraryView === "lists" && hero.view.shownPlaylist !== "")
                    height: 18
                    verticalAlignment: Text.AlignVCenter
                    text: "‹ back"
                    font.pixelSize: Theme.fontSizeS
                    onActivated: {
                        if (hero.view.page)
                            hero.view.pageBack()
                        else
                            hero.view.backToPlaylists()
                    }
                }
            }

            Row {
                id: titleRow
                width: parent.width
                spacing: 12

                Text {
                    id: titleText
                    width: Math.min(implicitWidth, titleRow.width - (artistFav.visible ? artistFav.width + titleRow.spacing : 0))
                    textFormat: Text.PlainText
                    text: hero.info.title
                    color: hero.onBanner
                    elide: Text.ElideRight
                    wrapMode: Text.Wrap
                    maximumLineCount: 2
                    font.family: Theme.fontFamily
                    font.bold: true
                    font.pixelSize: Theme.fontSize3xl
                }

                Text {
                    id: artistFav
                    visible: hero.pageKind === "artist"
                    topPadding: 8
                    textFormat: Text.PlainText
                    text: "♥"
                    color: hero.info.starred ? Theme.liked : Theme.withOpacity(hero.onBanner, 0.45)
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeL

                    MouseArea {
                        anchors.fill: parent
                        anchors.margins: -8
                        cursorShape: Qt.PointingHandCursor
                        onClicked: hero.view.toggleArtistFav()
                    }
                }
            }

            Row {
                width: parent.width
                spacing: 6
                visible: hero.info.artist !== "" || hero.info.album !== ""
                clip: true

                LinkText {
                    visible: text !== ""
                    text: hero.info.artist
                    baseColor: hero.onBanner
                    hoverColor: Theme.border
                    font.pixelSize: Theme.fontSizeM
                    font.bold: true
                    width: Math.min(implicitWidth, parent.width * 0.5)
                    onActivated: hero.view.openArtist(text)
                    onContextActivated: hero.view.openFieldTags("artist", text)
                }

                Text {
                    visible: hero.info.artist !== "" && hero.info.album !== ""
                    textFormat: Text.PlainText
                    text: "·"
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }

                LinkText {
                    visible: text !== ""
                    text: hero.info.album
                    font.pixelSize: Theme.fontSizeM
                    width: Math.min(implicitWidth, parent.width * 0.3)
                    onActivated: hero.view.openAlbum(text, hero.info.path)
                }

                Text {
                    visible: text !== ""
                    textFormat: Text.PlainText
                    text: hero.info.year
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }
            }

            Text {
                width: parent.width
                textFormat: Text.PlainText
                visible: !hero.queueHeader && text !== ""
                text: hero.info.stats
                color: Theme.withOpacity(hero.onBanner, 0.7)
                elide: Text.ElideRight
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
            }

            Flow {
                width: parent.width
                spacing: 8
                visible: !hero.queueHeader

                Chip {
                    text: "play all"
                    prefix: "▶"
                    color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
                    onActivated: hero.view.headerPlay()
                }

                Repeater {
                    model: hero.entity && hero.pageKind !== "genre" ? (hero.entityData.genres || []) : []

                    Chip {
                        required property var modelData
                        text: String(modelData.name || "")
                        prefix: "#"
                        suffix: String(modelData.count || "")
                        onActivated: hero.view.openPage("genre", text, "")
                        onContextActivated: hero.view.openFieldTags("genre", text)
                    }
                }
            }
        }

        Row {
            id: seekRow
            visible: hero.queueHeader || hero.liveTrack
            anchors.left: parent.left
            anchors.right: artSlot.visible ? artSlot.left : parent.right
            anchors.leftMargin: 18
            anchors.rightMargin: 16
            anchors.top: textCol.bottom
            anchors.topMargin: 8
            height: hero.waveH

            WaveformViz {
                id: wave
                width: seekRow.width
                height: seekRow.height
                peaks: hero.view.peaks
                levels: hero.view.service ? hero.view.service.vizLevels : []
                progress: hero.view.progressFrac()
                live: hero.view.vizOn && hero.view.playing
                positionText: hero.view.clock(hero.view.shownPosition) || "0:00"
                durationText: hero.view.clock(hero.view.player.duration) || "0:00"
                onSeekRequested: function(fraction) {
                    var dur = Number(hero.view.player.duration) || 0
                    hero.view.seekTo(dur * fraction)
                }
            }
        }

        Item {
            id: artSlot
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            anchors.rightMargin: hero.queueHeader ? 0 : 18
            width: banner.width > 520 ? height : 0
            visible: width > 0

            Cover {
                anchors.fill: parent
                visible: hero.liveTrack || hero.info.mosaic.length === 0
                source: hero.liveTrack ? (hero.liveArt || hero.info.art || "") : (hero.info.art || "")
                round: hero.liveTrack ? false : hero.info.round
                name: hero.liveTrack ? (hero.view.player.title || hero.info.title) : hero.info.title
                decodeSize: 400
                cached: !hero.queueHeader && !hero.liveTrack
            }

            Grid {
                anchors.fill: parent
                visible: !hero.liveTrack && hero.info.mosaic.length > 0
                columns: 2
                spacing: 2

                Repeater {
                    model: hero.info.mosaic.length > 0 ? 4 : 0

                    Cover {
                        required property int index
                        width: (artSlot.width - 2) / 2
                        height: width
                        corner: 3
                        source: hero.info.mosaic[index % hero.info.mosaic.length] || ""
                        decodeSize: 200
                    }
                }
            }

            DropArea {
                id: drop
                anchors.fill: parent
                enabled: hero.queueHeader || hero.liveTrack
                onDropped: function(dropEvent) {
                    var raw = ""
                    if (dropEvent.hasUrls && dropEvent.urls && dropEvent.urls.length)
                        raw = String(dropEvent.urls[0])
                    else if (dropEvent.text)
                        raw = String(dropEvent.text)
                    if (hero.view.dropArt(raw))
                        dropEvent.accept(Qt.CopyAction)
                }
            }

            Rectangle {
                anchors.fill: parent
                anchors.margins: -3
                visible: drop.containsDrag
                color: "transparent"
                radius: 6
                border.width: 2
                border.color: Theme.good
            }

            MouseArea {
                anchors.fill: parent
                enabled: (hero.queueHeader || hero.liveTrack) && hero.view.trackPath !== ""
                cursorShape: Qt.PointingHandCursor
                onClicked: hero.view.openArt(hero.view.trackPath)
            }
        }
    }

    Column {
        width: hero.width
        spacing: 8
        visible: hero.entity && (hero.entityData.credits || []).length > 0

        SectionLabel { text: "members" }

        Flow {
            width: parent.width
            spacing: 6

            Repeater {
                model: hero.entityData.credits || []

                Chip {
                    required property var modelData
                    text: String(modelData)
                    onActivated: hero.view.openArtist(text)
                    onContextActivated: hero.view.openFieldTags("artist", text)
                }
            }
        }
    }

}
