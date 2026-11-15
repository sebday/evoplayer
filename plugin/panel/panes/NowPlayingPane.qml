import QtQuick
import "../compat"
import "../widgets"

Card {
    id: pane

    required property var view

    implicitHeight: metaCol.implicitHeight + pad * 2

    Row {
        anchors.fill: parent
        spacing: 12

        Row {
            id: transport
            height: parent.height
            spacing: 0

            Item {
                id: prevBtn
                height: parent.height
                width: height * 0.62

                Canvas {
                    anchors.fill: parent
                    onWidthChanged: requestPaint()
                    onHeightChanged: requestPaint()
                    onPaint: {
                        var ctx = getContext("2d")
                        ctx.clearRect(0, 0, width, height)
                        var s = Math.min(width, height)
                        var cy = height / 2
                        var th = s * 0.38
                        var tw = s * 0.30
                        var barW = Math.max(2, s * 0.075)
                        var gap = s * 0.07
                        var left = (width - (barW + gap + tw)) / 2
                        ctx.fillStyle = Theme.withOpacity(Theme.foreground, 0.72)
                        ctx.fillRect(left, cy - th / 2, barW, th)
                        ctx.beginPath()
                        ctx.moveTo(left + barW + gap + tw, cy - th / 2)
                        ctx.lineTo(left + barW + gap + tw, cy + th / 2)
                        ctx.lineTo(left + barW + gap, cy)
                        ctx.closePath()
                        ctx.fill()
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: view.transport("prev")
                }
            }

            Item {
            id: playBtn
            height: parent.height
            width: height

            Canvas {
                id: playMark
                anchors.fill: parent
                property bool playing: view.playerState() === "playing"
                property real breath: 0
                onPlayingChanged: requestPaint()
                onBreathChanged: if (playing) requestPaint()
                onWidthChanged: requestPaint()
                onHeightChanged: requestPaint()

                SequentialAnimation on breath {
                    running: playMark.playing && view.active
                    loops: Animation.Infinite
                    NumberAnimation { from: 0; to: 1; duration: 1400; easing.type: Easing.InOutSine }
                    NumberAnimation { from: 1; to: 0; duration: 1400; easing.type: Easing.InOutSine }
                }

                onPaint: {
                    var ctx = getContext("2d")
                    ctx.clearRect(0, 0, width, height)
                    var s = Math.min(width, height)
                    var cx = width / 2
                    var cy = height / 2
                    var r = s * 0.46
                    ctx.strokeStyle = Theme.muted
                    ctx.lineWidth = Math.max(2, s * 0.035)
                    ctx.beginPath()
                    ctx.arc(cx, cy, r, 0, Math.PI * 2)
                    ctx.stroke()
                    if (playing) {
                        var scale = 0.86 + 0.14 * breath
                        ctx.fillStyle = Theme.withOpacity(Theme.border, 0.45 + 0.55 * breath)
                        var barW = s * 0.07 * scale
                        var barH = s * 0.28 * scale
                        var gap = s * 0.06 * scale
                        ctx.fillRect(cx - gap - barW, cy - barH / 2, barW, barH)
                        ctx.fillRect(cx + gap, cy - barH / 2, barW, barH)
                    } else {
                        ctx.fillStyle = Theme.withOpacity(Theme.foreground, 0.72)
                        ctx.beginPath()
                        ctx.moveTo(cx - s * 0.1, cy - s * 0.16)
                        ctx.lineTo(cx - s * 0.1, cy + s * 0.16)
                        ctx.lineTo(cx + s * 0.16, cy)
                        ctx.closePath()
                        ctx.fill()
                    }
                }
            }

            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: view.transport("toggle")
            }
            }

            Item {
                id: nextBtn
                height: parent.height
                width: height * 0.62

                Canvas {
                    anchors.fill: parent
                    onWidthChanged: requestPaint()
                    onHeightChanged: requestPaint()
                    onPaint: {
                        var ctx = getContext("2d")
                        ctx.clearRect(0, 0, width, height)
                        var s = Math.min(width, height)
                        var cy = height / 2
                        var th = s * 0.38
                        var tw = s * 0.30
                        var barW = Math.max(2, s * 0.075)
                        var gap = s * 0.07
                        var left = (width - (tw + gap + barW)) / 2
                        ctx.fillStyle = Theme.withOpacity(Theme.foreground, 0.72)
                        ctx.beginPath()
                        ctx.moveTo(left, cy - th / 2)
                        ctx.lineTo(left, cy + th / 2)
                        ctx.lineTo(left + tw, cy)
                        ctx.closePath()
                        ctx.fill()
                        ctx.fillRect(left + tw + gap, cy - th / 2, barW, th)
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: view.transport("next")
                }
            }
        }

        Cover {
            id: nowArt
            visible: pane.width > 760
            anchors.verticalCenter: parent.verticalCenter
            height: parent.height
            width: visible ? height : 0
            corner: 4
            cached: false
            decodeSize: 160
            name: String(view.player.artist || view.player.title || "")
            source: view.artSource()

            MouseArea {
                anchors.fill: parent
                enabled: view.trackPath !== ""
                cursorShape: Qt.PointingHandCursor
                onClicked: view.showNowPlaying()
            }
        }

        Column {
            id: metaCol
            width: Math.min(Math.max(260, titleText.implicitWidth, releaseRow.implicitWidth),
                            parent.width * 0.35)
            spacing: 4

            LinkText {
                id: titleText
                width: parent.width
                text: view.trackPath ? String(view.player.title || view.trackPath.slice(view.trackPath.lastIndexOf("/") + 1)) : "nothing playing"
                baseColor: Theme.foreground
                hoverColor: Theme.border
                font.bold: true
                font.pixelSize: Theme.fontSizeL
                onActivated: view.showNowPlaying()
            }

            Row {
                id: releaseRow
                width: parent.width
                spacing: 6
                clip: true

                LinkText {
                    id: nowArtist
                    visible: text !== ""
                    width: Math.min(implicitWidth, releaseRow.width * 0.55)
                    text: String(view.player.artist || "").replace(/^\s+|\s+$/g, "")
                    baseColor: Theme.withOpacity(Theme.foreground, 0.85)
                    hoverColor: Theme.border
                    font.pixelSize: Theme.fontSizeS
                    font.bold: true
                    onActivated: view.openArtist(text)
                }

                LinkText {
                    visible: text !== ""
                    width: Math.min(implicitWidth, releaseRow.width * 0.4)
                    text: String(view.player.album || "").replace(/^\s+|\s+$/g, "")
                    font.pixelSize: Theme.fontSizeS
                    onActivated: view.openAlbum(text, view.trackPath)
                }

                LinkText {
                    visible: text !== "" && text.toLowerCase() !== String(view.player.album || "").replace(/^\s+|\s+$/g, "").toLowerCase()
                    width: Math.min(implicitWidth, releaseRow.width * 0.3)
                    text: String(view.player.label || "").replace(/^\s+|\s+$/g, "")
                    font.pixelSize: Theme.fontSizeS
                    onActivated: view.openLabel(text)
                }

                Text {
                    visible: text !== ""
                    textFormat: Text.PlainText
                    text: String(view.player.year || "").replace(/^\s+|\s+$/g, "")
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                }
            }

            Row {
                width: parent.width
                spacing: 12
                height: 14

                Item {
                    id: volumeBar
                    width: 110
                    height: 10
                    anchors.verticalCenter: parent.verticalCenter

                    Rectangle {
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width
                        height: 3
                        radius: 1
                        color: Theme.muted
                    }
                    Rectangle {
                        width: parent.width * view.volumeFrac()
                        height: 3
                        radius: 1
                        anchors.verticalCenter: parent.verticalCenter
                        color: Theme.good
                    }
                    MouseArea {
                        anchors.fill: parent
                        anchors.margins: -6
                        onClicked: function(mouse) {
                            view.setVolume(Math.round(100 * mouse.x / width))
                        }
                    }
                }
            }

            Row {
                spacing: 16

                Text {
                    textFormat: Text.PlainText
                    visible: view.trackPath !== ""
                    text: "♥"
                    color: view.liked ? Theme.liked : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                    anchors.verticalCenter: parent.verticalCenter

                    MouseArea {
                        anchors.fill: parent
                        anchors.margins: -4
                        onClicked: view.likePlaying()
                    }
                }

                Text {
                    textFormat: Text.PlainText
                    text: "SHUFFLE"
                    color: view.shuffle ? Theme.good : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter

                    MouseArea {
                        anchors.fill: parent
                        onClicked: view.toggleShuffle()
                    }
                }

                Text {
                    textFormat: Text.PlainText
                    text: "REP"
                    color: view.repeatOn ? Theme.good : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter

                    MouseArea {
                        anchors.fill: parent
                        onClicked: view.toggleRepeat()
                    }
                }

                Text {
                    textFormat: Text.PlainText
                    visible: view.channels === 1 || view.channels >= 2
                    text: view.channels === 1 ? "MONO" : "STEREO"
                    color: Theme.good
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter
                }
                Text {
                    textFormat: Text.PlainText
                    text: "VIZ"
                    color: view.vizOn ? Theme.good : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter
                    MouseArea {
                        anchors.fill: parent
                        onClicked: view.toggleViz()
                    }
                }
                Text {
                    textFormat: Text.PlainText
                    text: "EQ"
                    color: view.eqHot() ? Theme.good : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter
                    MouseArea {
                        anchors.fill: parent
                        onClicked: view.toggleEq()
                    }
                }
            }
        }

        WaveformViz {
            id: viz
            width: Math.max(120, parent.width - transport.width - nowArt.width - metaCol.width - (nowArt.visible ? 36 : 24))
            height: parent.height
            peaks: view.peaks
            levels: view.service ? view.service.vizLevels : []
            progress: view.progressFrac()
            live: view.vizOn && view.playing
            positionText: view.clock(view.shownPosition) || "0:00"
            durationText: view.clock(view.player.duration) || "0:00"
            onSeekRequested: function(fraction) {
                var dur = Number(view.player.duration) || 0
                view.seekTo(dur * fraction)
            }
        }
    }
}
