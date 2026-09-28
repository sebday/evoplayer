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

        Column {
            id: metaCol
            width: Math.max(80, parent.width - playBtn.width - viz.width - 24)
            spacing: 4

            Text {
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
                textFormat: Text.PlainText
                width: parent.width
                elide: Text.ElideRight
                text: {
                    var rel = view.nowRelease()
                    var t = view.lcd(view.shownPosition)
                    return rel ? rel + "   " + t : t
                }
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

                Repeater {
                    model: [
                        { glyph: "󰒮", action: "prev", lit: false },
                        { glyph: view.playerState() === "playing" ? "󰏤" : "󰐊", action: "toggle", lit: true },
                        { glyph: "󰓛", action: "stop", lit: false },
                        { glyph: "󰒭", action: "next", lit: false }
                    ]

                    Text {
                        required property var modelData
                        textFormat: Text.PlainText
                        text: modelData.glyph
                        color: modelData.lit ? Theme.good : Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: 18
                        anchors.verticalCenter: parent.verticalCenter

                        MouseArea {
                            anchors.fill: parent
                            anchors.margins: -6
                            onClicked: view.transport(modelData.action)
                        }
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
                    text: "48 KHZ"
                    color: Theme.foreground
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
                        onClicked: view.cycleViz(1)
                    }
                }
            }
        }

        WaveformViz {
            id: viz
            width: view.artW
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
