.pragma library

function enabled(envValue) {
    return String(envValue || "").toLowerCase() === "cliamp"
}

function cliampBin(envValue) {
    if (envValue && String(envValue).trim() !== "")
        return String(envValue).trim()
    return "cliamp"
}

function cliampSocketPath(home, envValue) {
    if (envValue && String(envValue).trim() !== "")
        return String(envValue).trim()
    return String(home || "") + "/.config/cliamp/cliamp.sock"
}

function isLibraryMethod(method) {
    var m = String(method || "")
    if (!m)
        return false
    return m.indexOf("library.") === 0
        || m.indexOf("job.") === 0
        || m.indexOf("scrobble.") === 0
        || m.indexOf("queue.") === 0
}

function isPlaybackMethod(method) {
    var m = String(method || "")
    if (!m)
        return false
    return m.indexOf("playback.") === 0
        || m === "state.get"
        || m === "subscribe"
        || m.indexOf("viz.") === 0
}
