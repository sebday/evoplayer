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
                { key: "pgup dn", label: "page scroll" },
                { key: "← →", label: "back a page, open a playlist" },
                { key: "backspace", label: "back a page" },
                { key: "space", label: "play / pause" },
                { key: ", .", label: "previous / next" },
                { key: "< >", label: "skip 10s" },
                { key: "- =", label: "volume" },
                { key: "/", label: "find" },
                { key: "l", label: "like" },
                { key: "e", label: "edit tags" },
                { key: "esc", label: "back (pages, search, overlays)" },
                { key: "h", label: "help" },
                { key: "q", label: "quit" }
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
    }
}
