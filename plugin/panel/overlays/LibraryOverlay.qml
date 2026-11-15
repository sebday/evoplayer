import QtQuick
import "../compat"
import "../widgets"

Item {
    id: pane

    required property var view

    function jobLine() {
        var job = view.libraryJob
        if (!job || !job.status || job.status === "idle")
            return ""
        var names = {
            "cache": "Rebuild",
            "import": "Import",
            "art-maintain": "Artwork",
            "download": "Sync likes",
            "download-url": "Download"
        }
        var line = (names[String(job.name || "")] || String(job.name || "job")) + " · " + String(job.status || "")
        var progress = job.progress
        if (progress && Number(progress.total) > 0)
            line += " · " + (Number(progress.done) || 0) + "/" + Number(progress.total)
        if (progress && progress.phase)
            line += " · " + progress.phase
        if (job.status === "error" && job.error)
            return String(job.error)
        return line
    }

    function jobLog() {
        var log = String(view.libraryJob && view.libraryJob.log || "")
        if (!log)
            return ""
        var lines = log.split("\n")
        for (var i = lines.length - 1; i >= 0; i--) {
            if (String(lines[i]).trim() !== "")
                return String(lines[i])
        }
        return ""
    }

    Rectangle {
        anchors.fill: parent
        color: Theme.background

        MouseArea {
            anchors.fill: parent
        }
    }

    Item {
        id: helpCol
        anchors.top: parent.top
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        width: Math.min(440, Math.max(300, parent.width * 0.38))

        Rectangle {
            anchors.left: parent.left
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            width: 1
            color: Theme.foregroundFaint
        }

        HelpOverlay {
            anchors.fill: parent
            anchors.leftMargin: 18
            anchors.rightMargin: 16
            anchors.topMargin: 8
            anchors.bottomMargin: 8
            view: pane.view
        }
    }

    Column {
        id: tools
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.top: parent.top
        anchors.leftMargin: 8
        anchors.rightMargin: 16
        spacing: 8

        Row {
            width: parent.width
            height: 22
            spacing: 8

            Repeater {
                model: [
                    { title: "Rebuild", method: "library.cache", params: { force: true } },
                    { title: "Import", method: "library.import", params: {} },
                    { title: "Artwork", method: "library.art.maintain", params: {} },
                    { title: "Sync likes", method: "library.soundcloud.download", params: { import: false } }
                ]

                Chip {
                    required property var modelData
                    text: modelData.title
                    onActivated: {
                        if (view.libraryJob && view.libraryJob.status === "running")
                            return
                        view.startLibraryJob(modelData.method, modelData.params)
                    }
                }
            }

            Chip {
                visible: view.libraryJob && view.libraryJob.status === "running"
                text: "stop"
                onActivated: view.cancelLibraryJob()
            }
        }

        Text {
            width: parent.width
            visible: text !== ""
            textFormat: Text.PlainText
            text: pane.jobLine()
            color: view.libraryJob && view.libraryJob.status === "error" ? Theme.liked : Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        Text {
            width: parent.width
            visible: text !== ""
            textFormat: Text.PlainText
            text: pane.jobLog()
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
            elide: Text.ElideRight
        }

        Text {
            width: parent.width
            textFormat: Text.PlainText
            wrapMode: Text.WordWrap
            text: "Left in incoming until the genre matches a library folder. Set a folder, then run Import."
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }
    }

    EqOverlay {
        id: eqPanel
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.bottom: parent.bottom
        height: {
            var rows = (view.incomingFiles || []).length
            var listH = rows === 0 ? 36 : rows * 60
            var foldersH = view.incomingGenrePath !== "" ? 40 : 0
            var room = parent.height - tools.height - 10 - foldersH
            var listCap = Math.min(listH, Math.max(0, room - 180))
            return Math.max(180, room - listCap) / 2
        }
        view: pane.view
    }

    ListView {
        id: files
        anchors.top: tools.bottom
        anchors.topMargin: 10
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.rightMargin: 8
        anchors.bottom: folders.top
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        model: view.incomingFiles || []
        currentIndex: view.incomingIdx

        delegate: SoundRow {
            id: fileRow
            required property int index
            required property var modelData
            width: files.width
            artSide: 46
            titlePt: 12
            art: view.rowCoverUrl(modelData)
            upper: String(modelData.artist || "")
            upperNote: modelData.needs_genre ? "no library folder" : "still in incoming"
            title: String(modelData.title || "")
            genre: String(modelData.genre || "")
            year: modelData.year ? String(modelData.year) : ""
            clock: view.trackClock(modelData)
            badge: modelData.needs_genre ? (modelData.genre ? String(modelData.genre) : "no genre") : ""
            selected: index === view.incomingIdx
            current: String(modelData.path || "") === view.trackPath
            audible: view.playing
            onClicked: view.playIncoming(index)
            onActivated: view.playIncoming(index)
            onPlayRequested: view.playIncoming(index)

            Chip {
                text: view.incomingGenrePath === String(fileRow.modelData.path || "") ? "close" : "genre"
                inverted: fileRow.filled
                onActivated: view.toggleIncomingGenre(String(fileRow.modelData.path || ""))
            }
        }

        footer: Text {
            width: files.width
            visible: files.count === 0
            leftPadding: 8
            topPadding: 12
            textFormat: Text.PlainText
            text: "Incoming is empty."
            color: Theme.muted
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeS
        }
    }

    Item {
        id: folders
        visible: view.incomingGenrePath !== ""
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.bottom: eqPanel.top
        anchors.leftMargin: 8
        anchors.rightMargin: 16
        height: visible ? 40 : 0

        Flickable {
            anchors.fill: parent
            contentWidth: folderRow.implicitWidth
            flickableDirection: Flickable.HorizontalFlick
            clip: true

            Row {
                id: folderRow
                height: folders.height
                spacing: 6

                Text {
                    height: folderRow.height
                    verticalAlignment: Text.AlignVCenter
                    textFormat: Text.PlainText
                    text: "Folder"
                    color: Theme.muted
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeS
                    font.bold: true
                    font.capitalization: Font.AllUppercase
                    font.letterSpacing: 1
                }

                Repeater {
                    model: view.incomingFolders || []

                    Chip {
                        required property var modelData
                        text: String(modelData)
                        onActivated: view.setIncomingGenre(view.incomingGenrePath, String(modelData))
                    }
                }
            }
        }
    }
}
