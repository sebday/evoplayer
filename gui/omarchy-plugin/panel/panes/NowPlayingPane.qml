import QtQuick
import "../compat"
import "../widgets"

Fieldset {
    id: pane

    required property var view

    number: 1
    legend: "now playing"
    active: false
    hints: view.nowHints()
    hintsRight: true
    implicitHeight: metaCol.implicitHeight + 32

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
                onPlayingChanged: requestPaint()
                onWidthChanged: requestPaint()
                onHeightChanged: requestPaint()
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
                    ctx.fillStyle = Theme.withOpacity(Theme.foreground, 0.72)
                    if (playing) {
                        var barW = s * 0.07
                        var barH = s * 0.28
                        var gap = s * 0.06
                        ctx.fillRect(cx - gap - barW, cy - barH / 2, barW, barH)
                        ctx.fillRect(cx + gap, cy - barH / 2, barW, barH)
                    } else {
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

        Column {
            id: metaCol
            width: Math.min(Math.max(260, titleText.implicitWidth, releaseText.implicitWidth),
                            parent.width * 0.35)
            spacing: 4

            Text {
                id: titleText
                textFormat: Text.PlainText
                width: parent.width
                elide: Text.ElideRight
                text: view.nowTitle()
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.bold: true
                font.pixelSize: Theme.fontSizeL
            }

            Text {
                id: releaseText
                textFormat: Text.PlainText
                width: parent.width
                elide: Text.ElideRight
                text: view.nowRelease()
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
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
                    text: "EQ"
                    color: view.vizOn ? Theme.good : Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    anchors.verticalCenter: parent.verticalCenter
                    MouseArea {
                        anchors.fill: parent
                        onClicked: view.toggleViz()
                    }
                }
            }
        }

        WaveformViz {
            id: viz
            width: Math.max(120, parent.width - transport.width - metaCol.width - 24)
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
