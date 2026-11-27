pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import qs.commons as Shell

Item {
    id: root
    visible: false
    width: 0
    height: 0

    function withOpacity(c, alpha) {
        return Qt.rgba(c.r, c.g, c.b, alpha)
    }

    function mixColors(a, b, t) {
        return Qt.rgba(
            a.r + (b.r - a.r) * t,
            a.g + (b.g - a.g) * t,
            a.b + (b.b - a.b) * t,
            1
        )
    }

    readonly property color foreground: Shell.Theme.foreground
    readonly property color background: Shell.Theme.background
    readonly property color accent: Shell.Theme.accent
    readonly property color urgent: Shell.Theme.urgent
    property color good: Shell.Theme.accent
    property color liked: Shell.Theme.urgent
    property color border: Shell.Theme.accent
    property color muted: Shell.Theme.muted
    readonly property color inactiveBorder: withOpacity(foreground, 0.25)
    readonly property color mantle: Shell.Theme.popups.background
    readonly property int fieldsetCornerRadius: Shell.Theme.cornerRadius
    readonly property color overlaySurface: Shell.Theme.popups.background

    readonly property int motionFast: 120

    readonly property color fillAccentSubtle: withOpacity(accent, 0.14)
    readonly property color fillUrgentSubtle: withOpacity(urgent, 0.14)

    readonly property color foregroundGhost: withOpacity(foreground, 0.05)
    readonly property color foregroundWash: withOpacity(foreground, 0.06)
    readonly property color foregroundFaint: withOpacity(foreground, 0.08)
    readonly property color foregroundSubtle: withOpacity(foreground, 0.16)
    readonly property color foregroundTrack: withOpacity(foreground, 0.18)
    readonly property color foregroundDivider: withOpacity(foreground, 0.14)
    readonly property color foregroundBorder: withOpacity(foreground, 0.32)

    readonly property string fontFamily: Shell.Theme.font.family
    readonly property int fontPixelSize: Shell.Theme.font.body
    readonly property int fontSizeXs: Math.max(9, fontPixelSize - 2)
    readonly property int fontSizeS: Shell.Theme.font.bodySmall
    readonly property int fontSizeM: Shell.Theme.font.body
    readonly property int fontSizeL: Shell.Theme.font.subtitle
    readonly property int fontSize3xl: Shell.Theme.font.display

    property string _themeBuf: ""
    property bool _themeOverflow: false

    function pickTomlColor(text, key) {
        var lines = String(text || "").split("\n")
        for (var i = 0; i < lines.length; i++) {
            var line = lines[i].replace(/^\s+|\s+$/g, "")
            if (line.indexOf(key) !== 0)
                continue
            var after = line.charAt(key.length)
            if (after !== " " && after !== "=" && after !== "\t")
                continue
            var eq = line.indexOf("=")
            if (eq < 0)
                continue
            var rest = line.slice(eq + 1).replace(/^\s+|\s+$/g, "")
            if (rest.charAt(0) === "\"")
                rest = rest.slice(1)
            var quote = rest.indexOf("\"")
            if (quote >= 0)
                rest = rest.slice(0, quote)
            if (/^#[0-9A-Fa-f]{6}$/.test(rest))
                return rest
        }
        return ""
    }

    function applyThemeColors(text) {
        var green = pickTomlColor(text, "green")
        var red = pickTomlColor(text, "red")
        var blue = pickTomlColor(text, "blue")
        var accentHex = pickTomlColor(text, "accent")
        var dim = pickTomlColor(text, "muted")
        good = green || Shell.Theme.accent
        liked = red || Shell.Theme.urgent
        border = blue || accentHex || Shell.Theme.accent
        muted = dim || Shell.Theme.muted
    }

    function themeColorsPath() {
        var home = Quickshell.env("HOME") || ""
        if (!home)
            return ""
        return home + "/.themes/current/colors.toml"
    }

    Process {
        id: themeColorsProc
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                if (root._themeOverflow)
                    return
                root._themeBuf += String(chunk || "")
                if (root._themeBuf.length > 4096) {
                    root._themeOverflow = true
                    root._themeBuf = ""
                    themeColorsProc.signal(15)
                }
            }
        }
        onExited: {
            if (!root._themeOverflow && root._themeBuf)
                root.applyThemeColors(root._themeBuf)
            root._themeBuf = ""
        }
    }

    Component.onCompleted: {
        var colorsPath = themeColorsPath()
        if (!colorsPath)
            return
        _themeBuf = ""
        _themeOverflow = false
        themeColorsProc.command = [
            "dd",
            "if=" + colorsPath,
            "iflag=nofollow,nonblock,count_bytes,fullblock",
            "bs=1",
            "count=4097",
            "status=none"
        ]
        themeColorsProc.running = true
    }
}
