import QtQuick

Item {
    id: root

    property string text: ""
    property color color: "#9ece6a"

    readonly property var glyphs: ({
        "0": ["01110", "10001", "10011", "10101", "11001", "10001", "01110"],
        "1": ["00100", "01100", "00100", "00100", "00100", "00100", "01110"],
        "2": ["01110", "10001", "00001", "00010", "00100", "01000", "11111"],
        "3": ["01110", "10001", "00001", "00110", "00001", "10001", "01110"],
        "4": ["00010", "00110", "01010", "10010", "11111", "00010", "00010"],
        "5": ["11111", "10000", "11110", "00001", "00001", "10001", "01110"],
        "6": ["00110", "01000", "10000", "11110", "10001", "10001", "01110"],
        "7": ["11111", "00001", "00010", "00100", "01000", "01000", "01000"],
        "8": ["01110", "10001", "10001", "01110", "10001", "10001", "01110"],
        "9": ["01110", "10001", "10001", "01111", "00001", "00010", "01100"],
        ":": ["0", "1", "0", "0", "1", "0", "0"]
    })

    function glyphOf(ch) {
        var g = glyphs[ch]
        return g && g.length === 7 ? g : null
    }

    readonly property int cols: {
        var s = text || ""
        var n = 0
        var drawn = 0
        for (var i = 0; i < s.length; i++) {
            var g = glyphOf(s.charAt(i))
            if (!g)
                continue
            if (drawn > 0)
                n += 1
            n += g[0].length
            drawn++
        }
        return Math.max(1, n)
    }

    readonly property int naturalWidth: Math.max(cols, Math.floor((height > 0 ? height : 7) / 7) * cols)

    readonly property int cell: {
        var byH = Math.max(1, Math.floor(height / 7))
        var byW = Math.floor(width / cols)
        return Math.max(1, Math.min(byH, byW > 0 ? byW : byH))
    }

    Canvas {
        id: face
        anchors.fill: parent

        onPaint: {
            var ctx = getContext("2d")
            ctx.clearRect(0, 0, width, height)
            var cell = root.cell
            if (cell < 1 || width < 2 || height < 2)
                return
            var s = root.text || ""
            var gridW = root.cols * cell
            var gridH = 7 * cell
            var x0 = Math.floor((width - gridW) / 2)
            var y0 = Math.floor((height - gridH) / 2)
            var gap = cell >= 5 ? 1 : 0
            var px = cell - gap
            ctx.fillStyle = root.color
            var cursor = 0
            var drawn = 0
            for (var i = 0; i < s.length; i++) {
                var g = root.glyphOf(s.charAt(i))
                if (!g)
                    continue
                if (drawn > 0)
                    cursor += 1
                var gw = g[0].length
                for (var row = 0; row < 7; row++) {
                    var bits = g[row]
                    for (var col = 0; col < gw; col++) {
                        if (bits.charAt(col) !== "1")
                            continue
                        ctx.fillRect(x0 + (cursor + col) * cell, y0 + row * cell, px, px)
                    }
                }
                cursor += gw
                drawn++
            }
        }
    }

    onTextChanged: face.requestPaint()
    onColorChanged: face.requestPaint()
    onWidthChanged: face.requestPaint()
    onHeightChanged: face.requestPaint()
}
