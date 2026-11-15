import QtQuick
import "../compat"

Item {
    id: circle

    signal activated()

    property bool pause: false
    property color fill: Theme.border

    width: 56
    height: 56

    Rectangle {
        anchors.fill: parent
        radius: width / 2
        color: area.containsMouse ? Theme.mixColors(circle.fill, Theme.foreground, 0.2) : circle.fill
        scale: area.pressed ? 0.94 : 1

        Behavior on scale {
            NumberAnimation { duration: Theme.motionFast }
        }
    }

    Canvas {
        anchors.fill: parent
        property bool pause: circle.pause
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
        id: area
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: circle.activated()
    }
}
