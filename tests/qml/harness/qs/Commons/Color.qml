pragma Singleton
import QtQuick

QtObject {
    property color foreground: "#d3c6aa"
    property color background: "#2d353b"
    property color accent: "#7fbbb3"
    property color urgent: "#e67e80"
    property color muted: "#707880"

    readonly property QtObject popups: QtObject {
        property color background: "#252b30"
    }

    readonly property QtObject bar: QtObject {
        property color background: "#2d353b"
        property color text: "#d3c6aa"
    }
}
