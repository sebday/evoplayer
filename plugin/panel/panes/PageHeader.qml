import QtQuick
import QtQuick.Effects
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

    Item {
        id: banner
        width: hero.width
        height: 196

        Rectangle {
            id: bannerBase
            anchors.fill: parent
            radius: 8
            color: Theme.mantle
        }

        Image {
            id: blurSource
            anchors.fill: parent
            source: hero.info.blur || ""
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
            sourceSize.width: 160
            sourceSize.height: 160
            visible: false
        }

        MultiEffect {
            anchors.fill: parent
            source: blurSource
            visible: blurSource.status === Image.Ready && hero.info.blur !== ""
            autoPaddingEnabled: false
            blurEnabled: true
            blur: 1.0
            blurMax: 64
            saturation: 0.15
            maskEnabled: true
            maskSource: bannerMask
            maskThresholdMin: 0.5
            maskSpreadAtMin: 1.0
        }

        Item {
            id: bannerMask
            anchors.fill: parent
            layer.enabled: true
            visible: false

            Rectangle {
                anchors.fill: parent
                radius: 8
            }
        }

        Rectangle {
            anchors.fill: parent
            radius: 8
            gradient: Gradient {
                orientation: Gradient.Horizontal
                GradientStop { position: 0.0; color: Theme.withOpacity(Theme.background, 0.86) }
                GradientStop { position: 0.6; color: Theme.withOpacity(Theme.background, 0.62) }
                GradientStop { position: 1.0; color: Theme.withOpacity(Theme.background, 0.35) }
            }
        }

        Item {
            id: playCircle
            anchors.left: parent.left
            anchors.leftMargin: 22
            anchors.top: parent.top
            anchors.topMargin: 24
            width: 56
            height: 56

            Rectangle {
                anchors.fill: parent
                radius: width / 2
                color: playArea.containsMouse ? Theme.mixColors(Theme.border, Theme.foreground, 0.2) : Theme.border
                scale: playArea.pressed ? 0.94 : 1

                Behavior on scale {
                    NumberAnimation { duration: Theme.motionFast }
                }
            }

            Canvas {
                id: playGlyph
                anchors.fill: parent
                property bool pause: hero.showPause
                onPauseChanged: requestPaint()
                onWidthChanged: requestPaint()
                onPaint: {
                    var ctx = getContext("2d")
                    ctx.clearRect(0, 0, width, height)
                    ctx.fillStyle = Theme.background
                    var s = width
                    var cx = s / 2
                    var cy = s / 2
                    if (pause) {
                        var w = s * 0.1
                        var h = s * 0.34
                        ctx.fillRect(cx - w * 1.4, cy - h / 2, w, h)
                        ctx.fillRect(cx + w * 0.4, cy - h / 2, w, h)
                        return
                    }
                    ctx.beginPath()
                    ctx.moveTo(cx - s * 0.12, cy - s * 0.19)
                    ctx.lineTo(cx - s * 0.12, cy + s * 0.19)
                    ctx.lineTo(cx + s * 0.2, cy)
                    ctx.closePath()
                    ctx.fill()
                }
            }

            MouseArea {
                id: playArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: hero.view.headerPlay()
            }
        }

        Column {
            id: textCol
            anchors.left: playCircle.right
            anchors.leftMargin: 18
            anchors.right: artSlot.left
            anchors.rightMargin: 18
            anchors.top: playCircle.top
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
                    visible: hero.view.page !== null
                    height: 18
                    verticalAlignment: Text.AlignVCenter
                    text: "‹ back"
                    font.pixelSize: Theme.fontSizeS
                    onActivated: hero.view.pageBack()
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

            Row {
                spacing: 8
                topPadding: 4

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
            }
        }

        Item {
            id: artSlot
            anchors.right: parent.right
            anchors.rightMargin: 22
            anchors.verticalCenter: parent.verticalCenter
            height: parent.height - 40
            width: banner.width > 520 ? height : 0
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
