import QtQuick
import "../compat"

Flickable {
    id: flick

    required property var view

    contentWidth: width
    contentHeight: col.height
    clip: true
    boundsBehavior: Flickable.StopAtBounds

    Column {
        id: col
        width: flick.width
        spacing: 2

        Text {
            textFormat: Text.PlainText
            text: "keys"
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
        }

        Repeater {
            model: [
                { key: "↑↓", label: "move selection" },
                { key: "shift+↑↓", label: "reorder playlist" },
                { key: "pgup dn", label: "page scroll" },
                { key: "tab", label: "files / playlist" },
                { key: "shift+tab", label: "playlist / files" },
                { key: "← →", label: "parent / enter folder" },
                { key: "backspace", label: "parent folder" },
                { key: "⏎", label: "play / run" },
                { key: "space", label: "play / pause" },
                { key: ", .", label: "previous / next" },
                { key: "< >", label: "skip 10s" },
                { key: "- =", label: "volume" },
                { key: "/", label: "find" },
                { key: "d", label: "add dir" },
                { key: "f", label: "folder" },
                { key: "l", label: "like" },
                { key: "m", label: "move" },
                { key: "e", label: "edit tags" },
                { key: "tab", label: "next tag field (in editor)" },
                { key: "L", label: "like playing" },
                { key: "a", label: "art" },
                { key: "s", label: "art (track)" },
                { key: "v", label: "visualizer on / off" },
                { key: "V", label: "visualizer prev" },
                { key: "esc", label: "back" },
                { key: "h ?", label: "help" },
                { key: "q", label: "quit" },
                { key: "k", label: "keep discover" },
                { key: "x", label: "dismiss discover" },
                { key: "n", label: "more like this" }
            ]

            Row {
                required property var modelData
                spacing: 12
                width: col.width

                Text {
                    textFormat: Text.PlainText
                    width: 72
                    text: modelData.key
                    color: Theme.border
                    font.family: Theme.fontFamily
                    font.bold: true
                    font.pixelSize: Theme.fontSizeM
                }

                Text {
                    textFormat: Text.PlainText
                    text: modelData.label
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }
            }
        }

        Text {
            textFormat: Text.PlainText
            topPadding: 10
            text: "commands"
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeM
        }

        Repeater {
            model: [
                { key: "download", label: "sync soundcloud likes into .incoming" },
                { key: "download <url>", label: "youtube or soundcloud, then import" },
                { key: "discover", label: "similar tracks for the current song" },
                { key: "discover keep", label: "keep a discover track" },
                { key: "discover dismiss", label: "hide a discover track" },
                { key: "job status", label: "download or import progress" },
                { key: "job stop", label: "cancel the running job" },
                { key: "toggle", label: "play / pause" },
                { key: "next", label: "next track" },
                { key: "prev", label: "previous track" },
                { key: "seek <sec>", label: "jump in the track" },
                { key: "shuffle", label: "shuffle on, off, or toggle" },
                { key: "volume", label: "change or set volume" },
                { key: "load <path>", label: "play a file or folder" },
                { key: "status", label: "what is playing" },
                { key: "queue", label: "append, play, extend, up-next" },
                { key: "browse <path>", label: "list a library folder" },
                { key: "find <query>", label: "search the library" },
                { key: "playlist", label: "list, create, rename, star" },
                { key: "favorite <path>", label: "like a track" },
                { key: "tags", label: "read or edit tags" },
                { key: "art", label: "search or set cover art" },
                { key: "cache", label: "rebuild the library cache" },
                { key: "library import", label: "file .incoming into the library" },
                { key: "scrobble", label: "last.fm now playing and submit" },
                { key: "history report", label: "what you played" },
                { key: "config", label: "library root and settings" },
                { key: "viz", label: "visualizer" }
            ]

            Row {
                required property var modelData
                spacing: 12
                width: col.width

                Text {
                    textFormat: Text.PlainText
                    width: 140
                    text: modelData.key
                    color: Theme.border
                    font.family: Theme.fontFamily
                    font.bold: true
                    font.pixelSize: Theme.fontSizeM
                }

                Text {
                    textFormat: Text.PlainText
                    width: Math.max(40, col.width - 152)
                    text: modelData.label
                    wrapMode: Text.Wrap
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeM
                }
            }
        }
    }
}
