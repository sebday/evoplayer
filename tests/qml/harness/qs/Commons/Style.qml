pragma Singleton
import QtQuick

QtObject {
    property int cornerRadius: 7
    property int gapsOut: 5

    function space(px) {
        return Math.max(0, Math.round(Number(px) || 0))
    }

    readonly property QtObject font: QtObject {
        property string family: "monospace"
        property int bodySmall: 11
        property int body: 12
        property int subtitle: 13
        property int title: 14
        property int heading: 16
        property int display: 24
        property int displayLarge: 28
    }

    readonly property QtObject bar: QtObject {
        property int sizeHorizontal: 26
        property int sizeVertical: 28
    }
}
