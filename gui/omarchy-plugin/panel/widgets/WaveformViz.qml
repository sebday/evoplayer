import QtQuick
import "../compat"
import "."

Item {
    id: root

    property var peaks: []
    property var levels: []
    property real progress: 0
    property bool live: false
    property string positionText: ""
    property string durationText: ""

    Canvas {
        id: wave
        anchors.fill: parent
        onPaint: {
            var ctx = getContext("2d")
            ctx.clearRect(0, 0, width, height)
            var raw = root.peaks || []
            var n = raw.length
            if (n > 0 && width > 2 && height > 2) {
                var mid = height / 2
                var bars = Math.max(8, Math.floor(width / 4))
                var slot = width / bars
                var barW = Math.max(1, slot * 0.7)
                var shade = Theme.mixColors(Theme.background, Theme.muted, 0.35)
                for (var i = 0; i < bars; i++) {
                    var src = Math.min(n - 1, Math.floor(i * n / bars))
                    var v = Number(raw[src]) || 0
                    if (v < 0)
                        v = 0
                    if (v > 1)
                        v = 1
                    var h = Math.max(1, v * height * 0.46)
                    ctx.fillStyle = Theme.muted
                    ctx.fillRect(i * slot, mid - h, barW, h)
                    ctx.fillStyle = shade
                    ctx.fillRect(i * slot, mid, barW, h)
                }
            }
            var p = root.progress
            if (p < 0)
                p = 0
            if (p > 1)
                p = 1
            if (width > 2) {
                ctx.fillStyle = Theme.foreground
                ctx.fillRect(Math.round(p * (width - 1)), 0, 1, height)
            }
        }
    }

    PlayerCavaBars {
        anchors.fill: parent
        levels: root.live ? root.levels : []
        opacity: root.live ? 0.9 : 0
        barColor: Theme.good
    }

    Rectangle {
        anchors.verticalCenter: parent.verticalCenter
        width: parent.width
        height: 1
        z: 1
        color: Theme.foreground
    }

    Text {
        anchors.left: parent.left
        anchors.leftMargin: 4
        anchors.bottom: parent.verticalCenter
        anchors.bottomMargin: 3
        z: 2
        textFormat: Text.PlainText
        text: root.positionText
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeXs
    }

    Text {
        anchors.right: parent.right
        anchors.rightMargin: 4
        anchors.bottom: parent.verticalCenter
        anchors.bottomMargin: 3
        z: 2
        textFormat: Text.PlainText
        text: root.durationText
        color: Theme.foreground
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeXs
    }

    signal seekRequested(real fraction)

    MouseArea {
        anchors.fill: parent
        z: 3
        cursorShape: Qt.PointingHandCursor
        onClicked: function(mouse) {
            if (width <= 0)
                return
            var fraction = mouse.x / width
            if (fraction < 0)
                fraction = 0
            if (fraction > 1)
                fraction = 1
            root.seekRequested(fraction)
        }
    }

    onPeaksChanged: wave.requestPaint()
    onProgressChanged: wave.requestPaint()
    onWidthChanged: wave.requestPaint()
    onHeightChanged: wave.requestPaint()
}
