pragma Singleton
import QtQuick

QtObject {
    function clampAlpha(n) {
        return Math.max(0, Math.min(1, Number(n) || 0))
    }
}
