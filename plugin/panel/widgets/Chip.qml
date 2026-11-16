import QtQuick
import "../compat"

Rectangle {
    id: chip

    signal activated()
    signal contextActivated()

    property string text: ""
    property string prefix: ""
    property string suffix: ""
    property bool inverted: false
    property bool toggle: false
    property bool on: false
    property int maxWidth: 220

    readonly property bool hovered: enabled && area.containsMouse

    height: 22
    width: Math.min(maxWidth, label.implicitWidth + 18)
    radius: height / 2
    color: {
        if (!enabled)
            return Theme.foregroundFaint
        if (toggle && on)
            return Theme.fillAccentSubtle
        if (inverted)
            return Theme.withOpacity(Theme.background, hovered ? 0.32 : 0.18)
        return hovered ? Theme.foregroundSubtle : Theme.foregroundFaint
    }
    border.width: 1
    border.color: {
        if (!enabled)
            return "transparent"
        if (toggle)
            return on ? Theme.good : Theme.inactiveBorder
        if (inverted)
            return "transparent"
        return hovered ? Theme.foregroundBorder : "transparent"
    }

    Text {
        id: label
        anchors.centerIn: parent
        width: Math.min(implicitWidth, chip.maxWidth - 18)
        textFormat: Text.PlainText
        elide: Text.ElideRight
        text: (chip.prefix ? chip.prefix + " " : "") + chip.text + (chip.suffix ? "  " + chip.suffix : "")
        color: {
            if (!chip.enabled)
                return Theme.muted
            if (chip.toggle)
                return chip.on ? Theme.good : Theme.muted
            if (chip.inverted)
                return Theme.background
            return chip.hovered ? Theme.foreground : Theme.withOpacity(Theme.foreground, 0.78)
        }
        font.family: Theme.fontFamily
        font.pixelSize: Theme.fontSizeS
    }

    MouseArea {
        id: area
        anchors.fill: parent
        enabled: chip.enabled
        hoverEnabled: true
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        cursorShape: chip.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
        onClicked: function(mouse) {
            if (!chip.enabled)
                return
            if (mouse.button === Qt.RightButton)
                chip.contextActivated()
            else
                chip.activated()
        }
    }
}
