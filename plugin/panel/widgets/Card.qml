import QtQuick
import qs.commons as Shell
import "../compat"

// Rounded pane surface. The border takes the accent while the pane has focus.
Rectangle {
    id: root

    default property alias content: contentHost.data

    property bool active: false
    property int pad: 12

    radius: 8
    color: Shell.Theme.mantle
    border.width: 1
    border.color: active ? Theme.withOpacity(Theme.border, 0.65) : Theme.foregroundDivider

    Behavior on border.color { ColorAnimation { duration: Theme.motionFast } }

    data: [
        Item {
            id: contentHost
            anchors.fill: parent
            anchors.margins: root.pad
        }
    ]
}
