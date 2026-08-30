import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: tip
    property bool show: false
    property string text: ""
    property bool placeBelow: false

    z: 200
    width: 1
    height: 1

    function computePlaceBelow() {
        var item = parent
        while (item) {
            if (item.clip === true) {
                var p = mapToItem(item, 0, 0)
                return p.y < 28
            }
            item = item.parent
        }
        var g = mapToItem(root, 0, 0)
        return g.y < 28
    }

    onShowChanged: {
        if (show)
            placeBelow = computePlaceBelow()
    }

    HoverPanelLabelPill {
        visible: show && text !== ""
        anchors.horizontalCenter: parent.horizontalCenter
        y: tip.placeBelow ? (parent.parent ? parent.parent.height : 22) + 6 : -(height + 5)
        text: parent.text
        fontSize: Theme.fontSizeXs
        textOpacity: 0.9
        fieldsetLegend: false
        fill: Qt.rgba(Theme.mantle.r, Theme.mantle.g, Theme.mantle.b, 0.96)
    }
}
