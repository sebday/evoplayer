pragma Singleton

import Quickshell
import QtQuick

QtObject {
    function fileUrl(path) {
        var value = String(path || "").trim()
        if (!value)
            return ""
        if (value.indexOf("file://") === 0)
            return value
        if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(value))
            return value
        var parts = value.split("/")
        var encoded = []
        for (var i = 0; i < parts.length; i++) {
            if (parts[i] === "" && i === 0)
                encoded.push("")
            else if (parts[i] !== "")
                encoded.push(encodeURIComponent(parts[i]))
        }
        return "file://" + encoded.join("/")
    }

    function shellQuote(value) {
        var s = String(value || "")
        return "'" + s.replace(/'/g, "'\\''") + "'"
    }

    function evoplayerBinPath(home) {
        var env = Quickshell.env("EVOPLAYER_BIN")
        if (env && String(env).trim() !== "")
            return String(env).trim()
        return String(home || Quickshell.env("HOME") || "") + "/.local/lib/evoplayer/evoplayer"
    }

    function evoplayerCommand(home, args) {
        var cmd = [evoplayerBinPath(home)]
        if (Array.isArray(args)) {
            for (var i = 0; i < args.length; i++)
                cmd.push(String(args[i]))
        } else if (args !== undefined && args !== null && String(args) !== "") {
            cmd.push(String(args))
        }
        return cmd
    }

    function screenForOutput(outputName, fallbackOutput) {
        var screens = Quickshell.screens
        if (!screens || screens.length === 0)
            return null
        var output = String(outputName || "").trim()
        if (!output)
            output = String(fallbackOutput || "").trim()
        if (output) {
            for (var i = 0; i < screens.length; i++) {
                var screen = screens[i]
                if (screen && String(screen.name) === output)
                    return screen
            }
        }
        return screens[0] || null
    }
}
