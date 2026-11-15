import QtQuick
import "../compat"
import "../widgets"

Item {
    id: pane

    required property var view

    function jobActive() {
        var job = view.libraryJob
        return !!(job && job.status && job.status !== "idle")
    }

    function jobTitle() {
        var names = {
            "cache": "Rebuild",
            "import": "Import",
            "art-maintain": "Artwork",
            "download": "Sync likes",
            "download-url": "Download"
        }
        var name = String(view.libraryJob && view.libraryJob.name || "")
        return names[name] || name || "Job"
    }

    function jobDetail() {
        var job = view.libraryJob || {}
        if (job.status === "error" && job.error)
            return String(job.error)
        var parts = []
        var status = String(job.status || "")
        if (status)
            parts.push(status.charAt(0).toUpperCase() + status.slice(1))
        var progress = job.progress
        if (progress && Number(progress.total) > 0)
            parts.push((Number(progress.done) || 0) + " / " + Number(progress.total))
        if (progress && progress.phase)
            parts.push(String(progress.phase))
        return parts.join("   ·   ")
    }

    function jobLines() {
        var log = String(view.libraryJob && view.libraryJob.log || "")
        var lines = log.split("\n")
        var out = []
        for (var i = 0; i < lines.length; i++) {
            var line = String(lines[i]).replace(/^\s+|\s+$/g, "")
            if (line)
                out.push(line)
        }
        if (out.length > 4)
            out = out.slice(out.length - 4)
        return out
    }

    function logColor(line) {
        var text = String(line || "")
        if (text.indexOf("✗") === 0)
            return Theme.liked
        if (text.indexOf("✓") === 0)
            return Theme.good
        if (text.indexOf("!") === 0)
            return Theme.border
        return Theme.muted
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
        width: Math.min(280, Math.max(220, parent.width * 0.26))

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
                    { title: "Restart", method: "restart", params: {} },
                    { title: "Import", method: "library.import", params: {} },
                    { title: "Sync likes", method: "library.soundcloud.download", params: { import: false } }
                ]

                Chip {
                    required property var modelData
                    text: modelData.title
                    onActivated: {
                        if (modelData.method === "restart") {
                            view.restartPlayer()
                            return
                        }
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

        Rectangle {
            width: parent.width
            visible: pane.jobActive()
            radius: 8
            color: Theme.foregroundFaint
            height: jobCol.implicitHeight + 16

            Column {
                id: jobCol
                x: 12
                y: 8
                width: parent.width - 24
                spacing: 2

                Row {
                    width: parent.width
                    spacing: 10

                    Text {
                        id: jobName
                        textFormat: Text.PlainText
                        text: pane.jobTitle()
                        color: Theme.foreground
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        font.bold: true
                    }

                    Text {
                        width: Math.max(0, parent.width - jobName.implicitWidth - parent.spacing)
                        textFormat: Text.PlainText
                        text: pane.jobDetail()
                        color: view.libraryJob && view.libraryJob.status === "error" ? Theme.liked : Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        elide: Text.ElideRight
                    }
                }

                Repeater {
                    model: pane.jobLines()

                    Text {
                        required property string modelData
                        width: jobCol.width
                        textFormat: Text.PlainText
                        text: modelData
                        color: pane.logColor(modelData)
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        elide: Text.ElideRight
                    }
                }
            }
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
        anchors.bottomMargin: 16
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
