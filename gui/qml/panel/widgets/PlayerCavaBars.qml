import QtQuick
import "../compat"
import "."

Item {
    required property var dashboard

    id: cavaBars
    property bool active: false
    property bool scannerMode: false
    property bool scannerReverse: false
    property real scanPhase: 0
    property var envelopes: []
    property real vizMid: 0
    property real vizBarW: 4
    property real vizPitch: 5
    property var levels: []

    readonly property int barCount: 100

    function zeroBarLevels() {
        var out = []
        for (var i = 0; i < barCount; i++)
            out.push(0)
        return out
    }

    property var barLevels: zeroBarLevels()
    property var shapedLevels: zeroBarLevels()

    opacity: active ? 1 : 0

    function requestPaint() {
        vizCanvas.requestPaint()
    }

    function resetBars() {
        barLevels = zeroBarLevels()
        shapedLevels = zeroBarLevels()
        vizCanvas.requestPaint()
    }

    function scannerBoost(barCenterX, scanX, beamW) {
        var dist = Math.abs(barCenterX - scanX) / beamW
        if (dist >= 1)
            return 0
        return Math.pow(1 - dist, 2)
    }

    function applyLevels(raw) {
        if (!active || scannerMode)
            return
        var nextLevels = []
        var n = Math.min(barCount, raw ? raw.length : 0)
        for (var i = 0; i < barCount; i++) {
            var v = i < n ? Number(raw[i]) || 0 : 0
            if (v < 0)
                v = 0
            if (v > 1)
                v = 1
            nextLevels.push(v)
        }
        barLevels = nextLevels
        shapedLevels = shapeCavaLevels(nextLevels)
        vizCanvas.requestPaint()
    }

    // Keep local peaks and a little floor so thin bins stay visible.
    // Do not lift toward the average or normalize to the frame max —
    // that flattened quiet vs loud before the overlay painted.
    function shapeCavaLevels(levels) {
        var n = Math.min(barCount, levels.length)
        var out = []
        for (var j = 0; j < barCount; j++) {
            var bin = j < n ? (Number(levels[j]) || 0) : 0
            if (bin < 0)
                bin = 0
            if (bin > 1)
                bin = 1
            var peak = Math.pow(bin, 0.85)
            var floor = peak > 0 ? 0.05 : 0
            out.push(Math.min(1, Math.max(floor, peak)))
        }
        return out
    }

    function localAccent(index, shaped) {
        var n = shaped.length
        if (!n)
            return 0
        var level = index < n ? Number(shaped[index]) || 0 : 0
        var lo = Math.max(0, index - 2)
        var hi = Math.min(n - 1, index + 2)
        for (var k = lo; k <= hi; k++)
            level = Math.max(level, Number(shaped[k]) || 0)
        return level
    }

    function syncScannerAnimation() {
        if (scannerMode && active) {
            scanAnimForward.stop()
            scanAnimReverse.stop()
            if (scannerReverse)
                scanAnimReverse.start()
            else
                scanAnimForward.start()
        } else {
            scanAnimForward.stop()
            scanAnimReverse.stop()
        }
    }

    onActiveChanged: {
        if (!active) {
            scanAnimForward.stop()
            scanAnimReverse.stop()
            resetBars()
            return
        }
        syncScannerAnimation()
        if (!scannerMode)
            applyLevels(levels)
    }

    onScannerModeChanged: {
        if (scannerMode) {
            scanPhase = scannerReverse ? 1 : 0
            resetBars()
            syncScannerAnimation()
            return
        }
        scanPhase = 0
        scanAnimForward.stop()
        scanAnimReverse.stop()
        resetBars()
        if (active)
            applyLevels(levels)
    }

    onScannerReverseChanged: {
        if (!scannerMode)
            return
        scanPhase = scannerReverse ? 1 : 0
        syncScannerAnimation()
    }

    onScanPhaseChanged: {
        if (scannerMode)
            requestPaint()
    }

    onLevelsChanged: applyLevels(levels)
    onEnvelopesChanged: vizCanvas.requestPaint()

    SequentialAnimation {
        id: scanAnimForward
        loops: Animation.Infinite
        NumberAnimation {
            target: cavaBars
            property: "scanPhase"
            from: 0
            to: 1
            duration: 380
            easing.type: Easing.Linear
        }
    }

    SequentialAnimation {
        id: scanAnimReverse
        loops: Animation.Infinite
        NumberAnimation {
            target: cavaBars
            property: "scanPhase"
            from: 1
            to: 0
            duration: 380
            easing.type: Easing.Linear
        }
    }

    Canvas {
        id: vizCanvas
        anchors.fill: parent
        onWidthChanged: requestPaint()
        onPaint: {
            var ctx = getContext("2d")
            ctx.clearRect(0, 0, width, height)
            if (!cavaBars.active)
                return

            var barCount = cavaBars.barCount
            var pitch = cavaBars.vizPitch > 0 ? cavaBars.vizPitch : width / barCount
            var barW = cavaBars.vizBarW > 0 ? cavaBars.vizBarW : Math.max(2, pitch * 0.75)
            var mid = cavaBars.vizMid > 0 ? cavaBars.vizMid : height / 2
            var envs = cavaBars.envelopes || []
            var barColor = Theme.mixColors(Theme.accent, Theme.foreground, 0.38)
            var accent = Theme.accent
            var fallbackAmp = Math.max(8, height * 0.16)

            if (cavaBars.scannerMode) {
                var scanX = width * cavaBars.scanPhase
                var beamW = Math.max(24, width * 0.12)
                for (var s = 0; s < barCount; s++) {
                    var frameH = s < envs.length ? envs[s] : fallbackAmp
                    var x = s * pitch + (pitch - barW) / 2
                    var center = x + barW * 0.5
                    var boost = cavaBars.scannerBoost(center, scanX, beamW)
                    var level = 0.1 + boost * 0.9
                    var liveH = Math.max(1, frameH * level)
                    var alpha = 0.18 + boost * 0.62
                    ctx.fillStyle = boost > 0.35
                        ? Qt.rgba(accent.r, accent.g, accent.b, alpha)
                        : Qt.rgba(barColor.r, barColor.g, barColor.b, alpha)
                    ctx.fillRect(x, mid - liveH, barW, liveH)
                    ctx.fillRect(x, mid, barW, liveH)
                }
                return
            }

            var shaped = cavaBars.shapedLevels
            for (var i = 0; i < barCount; i++) {
                var frameH2 = i < envs.length ? envs[i] : fallbackAmp
                var level2 = localAccent(i, shaped)
                if (level2 < 0.015)
                    continue
                var liveH2 = Math.max(1, frameH2 * level2)
                var x2 = i * pitch + (pitch - barW) / 2
                ctx.fillStyle = Qt.rgba(barColor.r, barColor.g, barColor.b, 0.42 + level2 * 0.5)
                ctx.fillRect(x2, mid - liveH2, barW, liveH2)
                ctx.fillRect(x2, mid, barW, liveH2)
            }
        }
    }
}
