pragma Singleton

import Quickshell
import QtQuick
import "../../media/Model.js" as Model

QtObject {
    function fileUrl(path) {
        return Model.fileUrl(path)
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
