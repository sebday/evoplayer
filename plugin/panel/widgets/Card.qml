import QtQuick
import "../compat"

// Rounded pane surface. The border takes the accent while the pane has focus.
Rectangle {
    id: root

    default property alias content: contentHost.data

    property bool active: false
    property int pad: 12

    radius: 0
    color: "transparent"
    border.width: 0

    data: [
        Item {
            id: contentHost
            anchors.fill: parent
            anchors.margins: root.pad
        }
    ]
}
