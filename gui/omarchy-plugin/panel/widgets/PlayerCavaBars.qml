import QtQuick
import "../compat"

Canvas {
    id: root

    property var levels: []
    property color barColor: Theme.good

    onLevelsChanged: requestPaint()
    onBarColorChanged: requestPaint()
    onWidthChanged: requestPaint()
    onHeightChanged: requestPaint()

    onPaint: {
        var ctx = getContext("2d")
        ctx.clearRect(0, 0, width, height)
        var raw = levels || []
        var n = raw.length
        if (n < 1 || width < 2 || height < 2)
            return
        var mid = height / 2
        var slot = width / n
        var barW = Math.max(1, slot * 0.62)
        ctx.fillStyle = barColor
        for (var i = 0; i < n; i++) {
            var v = Number(raw[i]) || 0
            if (v < 0)
                v = 0
            if (v > 1)
                v = 1
            v = Math.pow(v, 0.85)
            var h = Math.max(1, v * height * 0.46)
            ctx.fillRect(i * slot + (slot - barW) / 2, mid - h, barW, h * 2)
        }
    }
}
