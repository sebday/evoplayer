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
    readonly property bool showPause: queueHeader && view.playing
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

    component SkipButton: Item {
        property bool forward: false
        signal activated()

        width: 32
        height: 56

        Canvas {
            anchors.fill: parent
            onWidthChanged: requestPaint()
            onHeightChanged: requestPaint()
            onPaint: {
                var ctx = getContext("2d")
                ctx.clearRect(0, 0, width, height)
                var s = Math.min(width, height)
                var cy = height / 2
                var th = s * 0.34
                var tw = s * 0.28
                var barW = Math.max(2, s * 0.07)
                var gap = s * 0.06
                ctx.fillStyle = Theme.withOpacity(Theme.foreground, 0.78)
                if (parent.forward) {
                    var left = (width - (tw + gap + barW)) / 2
                    ctx.beginPath()
                    ctx.moveTo(left, cy - th / 2)
                    ctx.lineTo(left, cy + th / 2)
                    ctx.lineTo(left + tw, cy)
                    ctx.closePath()
                    ctx.fill()
                    ctx.fillRect(left + tw + gap, cy - th / 2, barW, th)
                } else {
                    var right = (width - (barW + gap + tw)) / 2
                    ctx.fillRect(right, cy - th / 2, barW, th)
                    ctx.beginPath()
                    ctx.moveTo(right + barW + gap + tw, cy - th / 2)
                    ctx.lineTo(right + barW + gap + tw, cy + th / 2)
                    ctx.lineTo(right + barW + gap, cy)
                    ctx.closePath()
                    ctx.fill()
                }
            }
        }

        MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: parent.activated()
        }
    }

    Item {
        id: banner
        width: hero.width
        height: 252

        BlurBackdrop {
            anchors.fill: parent
            source: hero.info.blur || ""
        }

        Row {
            id: transport
            x: 16
            y: 22
            height: 56
            spacing: 2

            SkipButton { onActivated: hero.view.transport("prev") }

            PlayCircle {
                id: playCircle
                pause: hero.showPause
                onActivated: hero.view.headerPlay()
            }

            SkipButton {
                forward: true
                onActivated: hero.view.transport("next")
            }
        }

        Column {
            id: textCol
            anchors.left: transport.right
            anchors.leftMargin: 16
            anchors.right: artSlot.left
            anchors.rightMargin: 16
            anchors.top: transport.top
            anchors.topMargin: -4
            spacing: 6

            Row {
                spacing: 8

                Rectangle {
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

            Text {
                width: parent.width
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

            Row {
                width: parent.width
                spacing: 6
                visible: hero.info.artist !== "" || hero.info.album !== "" || hero.info.label !== ""
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
                }

                Text {
                    visible: hero.info.artist !== "" && (hero.info.album !== "" || hero.info.label !== "")
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

                LinkText {
                    visible: text !== ""
                    text: hero.info.label
                    font.pixelSize: Theme.fontSizeM
                    width: Math.min(implicitWidth, parent.width * 0.3)
                    onActivated: hero.view.openLabel(text)
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
                visible: text !== ""
                text: hero.info.stats
                color: Theme.withOpacity(hero.onBanner, 0.7)
                elide: Text.ElideRight
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
            }

            Flow {
                width: parent.width
                spacing: 8

                Chip {
                    text: hero.queueHeader ? (hero.view.shuffle ? "shuffle on" : "shuffle off") : "shuffle play"
                    inverted: hero.queueHeader && hero.view.shuffle
                    color: inverted ? Theme.border : (hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1))
                    onActivated: hero.view.headerShuffle()
                }

                Chip {
                    visible: !hero.queueHeader
                    text: "play all"
                    prefix: "▶"
                    color: hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1)
                    onActivated: hero.view.headerPlay()
                }

                Chip {
                    text: "like"
                    inverted: hero.view.liked
                    visible: hero.view.trackPath !== ""
                    color: inverted ? Theme.liked : (hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1))
                    onActivated: hero.view.likePlaying()
                }

                Chip {
                    text: "repeat"
                    inverted: hero.view.repeatOn
                    color: inverted ? Theme.good : (hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1))
                    onActivated: hero.view.toggleRepeat()
                }

                Chip {
                    text: "viz"
                    inverted: hero.view.vizOn
                    color: inverted ? Theme.good : (hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1))
                    onActivated: hero.view.toggleViz()
                }

                Chip {
                    text: "eq"
                    inverted: hero.view.eqHot()
                    color: inverted ? Theme.good : (hovered ? Theme.foregroundSubtle : Theme.withOpacity(Theme.foreground, 0.1))
                    onActivated: hero.view.toggleEq()
                }

                Text {
                    visible: hero.view.channels === 1 || hero.view.channels >= 2
                    height: 22
                    verticalAlignment: Text.AlignVCenter
                    textFormat: Text.PlainText
                    text: hero.view.channels === 1 ? "mono" : "stereo"
                    color: Theme.good
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    font.bold: true
                }
            }
        }

        Row {
            id: seekRow
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            anchors.leftMargin: 18
            anchors.rightMargin: 18
            anchors.bottomMargin: 12
            height: 36
            spacing: 14

            WaveformViz {
                id: wave
                width: seekRow.width - volumeBar.width - seekRow.spacing
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

            Item {
                id: volumeBar
                width: 88
                height: seekRow.height

                Rectangle {
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width
                    height: 3
                    radius: 1
                    color: Theme.withOpacity(Theme.foreground, 0.28)
                }

                Rectangle {
                    width: parent.width * hero.view.volumeFrac()
                    height: 3
                    radius: 1
                    anchors.verticalCenter: parent.verticalCenter
                    color: Theme.good
                }

                MouseArea {
                    anchors.fill: parent
                    anchors.topMargin: -8
                    anchors.bottomMargin: -8
                    onClicked: function(mouse) {
                        hero.view.setVolume(Math.round(100 * mouse.x / width))
                    }
                }
            }
        }

        Item {
            id: artSlot
            anchors.right: parent.right
            anchors.rightMargin: 22
            anchors.top: parent.top
            anchors.topMargin: 18
            anchors.bottom: seekRow.top
            anchors.bottomMargin: 10
            width: banner.width > 640 ? height : 0
            visible: width > 0

            Cover {
                anchors.fill: parent
                visible: hero.info.mosaic.length === 0
                source: hero.info.art || ""
                round: hero.info.round
                name: hero.info.title
                decodeSize: 400
                cached: !hero.queueHeader
            }

            Grid {
                anchors.fill: parent
                visible: hero.info.mosaic.length > 0
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
                enabled: hero.queueHeader
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
                enabled: hero.queueHeader && hero.view.trackPath !== ""
                cursorShape: Qt.PointingHandCursor
                onClicked: {
                    if (hero.view.mode !== "art")
                        hero.view.openArt(hero.view.trackPath)
                }
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
                }
            }
        }
    }

    Column {
        width: hero.width
        spacing: 8
        visible: hero.entity && hero.pageKind !== "album" && (hero.entityData.releases || []).length > 0

        SectionLabel { text: "releases" }

        ListView {
            id: releases
            width: parent.width
            height: 150
            orientation: ListView.Horizontal
            spacing: 14
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            model: hero.entityData.releases || []

            WheelHandler {
                onWheel: function(event) {
                    var d = event.angleDelta.y !== 0 ? event.angleDelta.y : event.angleDelta.x
                    var maxX = Math.max(0, releases.contentWidth - releases.width)
                    releases.contentX = Math.max(0, Math.min(maxX, releases.contentX - d))
                }
            }

            delegate: Item {
                required property var modelData

                width: 112
                height: releases.height

                Cover {
                    id: releaseCover
                    width: 112
                    height: 112
                    source: hero.view.rowArtUrl(modelData)
                    name: String(modelData.name || "")
                    scale: releaseArea.containsMouse ? 1.03 : 1

                    Behavior on scale {
                        NumberAnimation { duration: Theme.motionFast }
                    }
                }

                Text {
                    anchors.top: releaseCover.bottom
                    anchors.topMargin: 5
                    width: parent.width
                    textFormat: Text.PlainText
                    text: String(modelData.name || "")
                    elide: Text.ElideRight
                    color: releaseArea.containsMouse ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.85)
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    font.bold: true
                }

                Text {
                    anchors.bottom: parent.bottom
                    width: parent.width
                    textFormat: Text.PlainText
                    text: [String(modelData.year || ""), hero.pageKind === "artist" ? String(modelData.label || "") : String(modelData.artist || "")]
                        .filter(function(s) { return s !== "" }).join(" · ")
                    elide: Text.ElideRight
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeXs
                }

                MouseArea {
                    id: releaseArea
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: hero.view.openPage("album", modelData.name, modelData.dir)
                }
            }
        }
    }

    Repeater {
        model: hero.entity ? [
            { label: hero.pageKind === "artist" ? "with" : "artists", kind: "artist", rows: hero.entityData.artists || [] },
            { label: "labels", kind: "label", rows: hero.entityData.labels || [] },
            { label: "genres", kind: "genre", rows: hero.pageKind === "genre" ? [] : (hero.entityData.genres || []) }
        ] : []

        Column {
            id: group
            required property var modelData

            width: hero.width
            spacing: 8
            visible: modelData.rows.length > (hero.pageKind === "album" ? 1 : 0)

            SectionLabel { text: modelData.label }

            Flow {
                width: parent.width
                spacing: 6

                Repeater {
                    model: modelData.rows

                    Chip {
                        required property var modelData
                        text: String(modelData.name || "")
                        prefix: group.modelData.kind === "genre" ? "#" : ""
                        suffix: String(modelData.count || "")
                        onActivated: hero.view.openPage(group.modelData.kind, text, "")
                    }
                }
            }
        }
    }

    SectionLabel {
        visible: hero.entity
        topPadding: 4
        text: "tracks"
    }
}
