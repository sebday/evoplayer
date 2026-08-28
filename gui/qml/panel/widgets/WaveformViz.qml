import QtQuick
import "../compat"
import "."

Item {
    id: root

    required property var dashboard

    property int _vizLayoutW: 0
    property int _vizLayoutH: 0

    readonly property bool vizActive: dashboard.active
        && dashboard.nowplayingTabActive
        && dashboard.playerPlaying

    readonly property bool overlayActive: root.vizActive
        || (dashboard.nowplayingTabActive && dashboard.waveformLoading)

    readonly property var vizLevels: dashboard.playerMonitor
        ? (dashboard.playerMonitor.vizLevels || [])
        : []

    readonly property int vizSequence: dashboard.playerMonitor
        ? (dashboard.playerMonitor.vizSequence || 0)
        : 0

    property int _acceptedVizSequence: 0

    Connections {
        target: dashboard.playerMonitor
        function onVizRevisionChanged() {
            if (!dashboard.playerMonitor)
                return
            var seq = dashboard.playerMonitor.vizSequence || 0
            if (seq > 0 && seq <= root._acceptedVizSequence)
                return
            if (seq > 0)
                root._acceptedVizSequence = seq
            cavaOverlay.applyLevels(dashboard.playerMonitor.vizLevels || [])
        }
    }

    function scheduleVizUpdate() {
        vizRebuildTimer.restart()
    }

    Timer {
        id: vizRebuildTimer
        interval: 0
        repeat: false
        onTriggered: root.recomputeVizEnvelopes()
    }

    readonly property int vizBarCount: 100
    property var vizEnvelopes: []
    property real vizMid: 0
    property real vizVPad: 0
    property real vizDrawH: 0
    property real vizBarW: 4
    property real vizPitch: 5

    function recomputeVizEnvelopes() {
        var w = width
        var h = height
        if (w <= 0 || h <= 0) {
            vizEnvelopes = []
            return
        }
        var vPad = Math.max(12, h * 0.12)
        var drawH = Math.max(8, h - vPad * 2)
        var mid = vPad + drawH / 2
        var halfH = drawH * 0.5
        var maxAmp = halfH * 0.93
        var pitch = w / vizBarCount
        var barW = Math.max(2, pitch * 0.75)
        var samples = dashboard.waveformSamples
        var n = samples.length
        var barCount = vizBarCount
        var envelopes = []

        vizMid = mid
        vizVPad = vPad
        vizDrawH = drawH
        vizBarW = barW
        vizPitch = pitch

        if (n === 0) {
            for (var e = 0; e < barCount; e++)
                envelopes.push(maxAmp * 0.35)
            vizEnvelopes = envelopes
            waveCanvas.requestPaint()
            cavaOverlay.requestPaint()
            return
        }

        var peaks = []
        if (n <= barCount) {
            for (var bi = 0; bi < n; bi++)
                peaks.push(Number(samples[bi]) || 0)
            while (peaks.length < barCount)
                peaks.push(0)
        } else {
            var step = n / barCount
            for (var b = 0; b < barCount; b++) {
                var start = Math.floor(b * step)
                var end = Math.floor((b + 1) * step)
                var peak = 0
                var sum = 0
                var cnt = 0
                for (var j = start; j < end && j < n; j++) {
                    var v = Number(samples[j]) || 0
                    peak = Math.max(peak, v)
                    sum += v
                    cnt++
                }
                var avg = cnt ? sum / cnt : 0
                peaks.push(peak * 0.48 + avg * 0.52)
            }
        }

        var sorted = peaks.slice(0, barCount)
        sorted.sort(function(a, b) { return a - b })
        var floorVal = sorted[Math.max(0, Math.floor(barCount * 0.04))]
        var ceilVal = sorted[Math.min(barCount - 1, Math.floor(barCount * 0.994))]
        var span = Math.max(1, ceilVal - floorVal)
        var refPeak = Math.max(1, sorted[barCount - 1])
        var win = 5
        var halfWin = Math.floor(win / 2)

        for (var i = 0; i < barCount; i++) {
            var lo = peaks[i]
            var hi = peaks[i]
            var w0 = Math.max(0, i - halfWin)
            var w1 = Math.min(barCount - 1, i + halfWin)
            for (var k = w0; k <= w1; k++) {
                lo = Math.min(lo, peaks[k])
                hi = Math.max(hi, peaks[k])
            }
            var localSpan = hi - lo
            var local = localSpan > 0.5
                ? (peaks[i] - lo) / localSpan
                : (((i * 11 + Math.floor(peaks[i] * 3)) % 29) / 29)
            var global = Math.max(0, Math.min(1, (peaks[i] - floorVal) / span))
            var envelope = 0.72 + 0.28 * Math.pow(peaks[i] / refPeak, 0.45)
            var blended = 0.38 * Math.pow(local, 0.9) + 0.62 * Math.pow(global, 0.8)
            var signal = 0.34 + 0.66 * blended
            var amp = Math.pow(signal, 1.38) * maxAmp * envelope
            envelopes.push(Math.max(1.2, amp))
        }
        vizEnvelopes = envelopes
        waveCanvas.requestPaint()
        cavaOverlay.requestPaint()
    }

    onWidthChanged: {
        if (Math.abs(width - _vizLayoutW) > 2) {
            _vizLayoutW = width
            recomputeVizEnvelopes()
        }
    }
    onHeightChanged: {
        if (Math.abs(height - _vizLayoutH) > 2) {
            _vizLayoutH = height
            recomputeVizEnvelopes()
        }
    }

    Connections {
        target: dashboard
        function onWaveformSamplesChanged() {
            root.scheduleVizUpdate()
        }
    }

    Component.onCompleted: dashboard.syncVizSubscription()

    Timer {
        id: progressPaintTimer
        interval: 120
        running: root.vizActive
        repeat: true
        onTriggered: waveCanvas.syncPlayProgress()
    }

    Connections {
        target: dashboard.playerMonitor
        enabled: dashboard.playerMonitor !== null
        function onPlayerChanged() {
            waveCanvas.syncPlayProgress()
        }
    }

    Canvas {
        id: waveCanvas
        anchors.fill: parent
        property real playProgress: 0

        function syncPlayProgress() {
            var snap = dashboard.transportSnap
            var dur = Number(snap.duration) || Number(dashboard.player.duration) || 0
            var pos = Number(snap.position) || Number(dashboard.player.position) || 0
            var next = dur > 0 ? Math.max(0, Math.min(1, pos / dur)) : Number(dashboard.progress) || 0
            if (Math.abs(next - playProgress) < 0.0005)
                return
            playProgress = next
            requestPaint()
        }

        Component.onCompleted: syncPlayProgress()
        onPlayProgressChanged: requestPaint()
        onPaint: {
            var ctx = getContext("2d")
            ctx.clearRect(0, 0, width, height)
            var vPad = root.vizVPad
            var drawH = root.vizDrawH
            var mid = root.vizMid
            var prog = playProgress
            var samples = dashboard.waveformSamples
            var n = samples.length
            var barCount = root.vizBarCount
            var pitch = root.vizPitch
            var barW = root.vizBarW
            var envelopes = root.vizEnvelopes

            if (n === 0) {
                var trackH = 3
                var trackY = mid - trackH / 2
                ctx.fillStyle = Theme.foregroundHoverWash
                ctx.fillRect(0, trackY, width, trackH)
                if (prog > 0) {
                    ctx.fillStyle = Theme.accent
                    ctx.fillRect(0, trackY, width * prog, trackH)
                }
                return
            }

            var playX = width * prog
            var fg = Theme.foreground
            var accent = Theme.accent

            for (var i = 0; i < barCount; i++) {
                var amp = i < envelopes.length ? envelopes[i] : 1.2
                var x = i * pitch + (pitch - barW) / 2
                var played = (x + barW * 0.5) <= playX

                if (played) {
                    ctx.fillStyle = accent
                    ctx.globalAlpha = 0.98
                    ctx.fillRect(x, mid - amp, barW, amp)
                    ctx.globalAlpha = 0.38
                    ctx.fillRect(x, mid, barW, amp)
                } else {
                    ctx.fillStyle = Qt.rgba(fg.r, fg.g, fg.b, 1)
                    ctx.globalAlpha = 0.58
                    ctx.fillRect(x, mid - amp, barW, amp)
                    ctx.globalAlpha = 0.24
                    ctx.fillRect(x, mid, barW, amp)
                }
            }
            ctx.globalAlpha = 1

            ctx.strokeStyle = accent
            ctx.lineWidth = 1
            ctx.beginPath()
            ctx.moveTo(0, mid + 0.5)
            ctx.lineTo(width, mid + 0.5)
            ctx.stroke()

            if (prog > 0) {
                ctx.fillStyle = Qt.rgba(accent.r, accent.g, accent.b, 0.95)
                ctx.fillRect(Math.max(0, playX - 1), vPad, 2, drawH)
            }
        }
    }

    PlayerCavaBars {
        id: cavaOverlay
        dashboard: root.dashboard
        z: 1
        anchors.fill: parent
        envelopes: root.vizEnvelopes
        vizMid: root.vizMid
        vizBarW: root.vizBarW
        vizPitch: root.vizPitch
        active: root.overlayActive
        scannerMode: dashboard.waveformLoading
        scannerReverse: dashboard.scannerReverse
        levels: root.vizLevels
    }

    TransportTimePill {
        dashboard: root.dashboard
        z: 3
        dark: true
        compact: true
        visible: String(dashboard.player.path || "") !== "" && dashboard.nowplayingContentVisible
        label: dashboard.player.position_label || "0:00"
        anchors.left: parent.left
        anchors.leftMargin: 8
        anchors.verticalCenter: parent.verticalCenter
    }

    TransportTimePill {
        dashboard: root.dashboard
        z: 3
        dark: true
        compact: true
        visible: String(dashboard.player.path || "") !== "" && dashboard.nowplayingContentVisible
        label: dashboard.player.duration_label || "0:00"
        anchors.right: parent.right
        anchors.rightMargin: 8
        anchors.verticalCenter: parent.verticalCenter
    }

    MouseArea {
        z: 2
        anchors.fill: parent
        acceptedButtons: Qt.LeftButton
        cursorShape: Qt.PointingHandCursor
        onPressed: function(mouse) {
            dashboard.previewSeekFromX(mouse.x, width)
        }
        onPositionChanged: function(mouse) {
            if (pressed && mouse.buttons & Qt.LeftButton)
                dashboard.previewSeekFromX(mouse.x, width)
        }
        onReleased: function(mouse) {
            if (mouse.button === Qt.LeftButton)
                dashboard.commitSeekFromX(mouse.x, width)
        }
        onWheel: function(wheel) {
            if (!wheel.angleDelta.y)
                return
            dashboard.adjustVolume(wheel.angleDelta.y > 0 ? 5 : -5)
            wheel.accepted = true
        }
    }
}
