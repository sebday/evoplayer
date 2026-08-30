import QtQuick
import QtQuick.Controls
import "../compat"

ScrollBar {
    id: root

    policy: ScrollBar.AsNeeded
    implicitWidth: 6

    contentItem: Rectangle {
        implicitWidth: 6
        radius: 3
        color: root.pressed
            ? Theme.accent
            : (root.hovered
                ? Qt.rgba(Theme.foreground.r, Theme.foreground.g, Theme.foreground.b, 0.55)
                : Qt.rgba(Theme.foreground.r, Theme.foreground.g, Theme.foreground.b, 0.3))
        opacity: root.active ? 1 : 0.35
    }

    background: Item {
        implicitWidth: 6
    }
}
