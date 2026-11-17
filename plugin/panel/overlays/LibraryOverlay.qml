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
            "download": "Sync Likes",
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

    function jobRunning() {
        return !!(view.libraryJob && view.libraryJob.status === "running")
    }

    component LibraryRow: Item {
        id: row

        property string title: ""
        property string detail: ""
        property string action: ""
        property bool toggle: false
        property bool on: false
        property bool ready: true
        signal activated()

        height: 52

        Column {
            anchors.left: parent.left
            anchors.right: chip.left
            anchors.rightMargin: 12
            anchors.verticalCenter: parent.verticalCenter
            spacing: 2

            Text {
                width: parent.width
                textFormat: Text.PlainText
                text: row.title
                color: Theme.foreground
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeM
                font.bold: true
                elide: Text.ElideRight
            }

            Text {
                width: parent.width
                textFormat: Text.PlainText
                text: row.detail
                color: Theme.muted
                font.family: Theme.fontFamily
                font.pixelSize: Theme.fontSizeS
                elide: Text.ElideRight
            }
        }

        Chip {
            id: chip
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            text: row.action
            enabled: row.ready
            toggle: row.toggle
            on: row.on
            onActivated: row.activated()
        }

        Rectangle {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: 1
            color: Theme.foregroundFaint
        }
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
        anchors.topMargin: 8
        anchors.leftMargin: 8
        anchors.rightMargin: 16
        spacing: 0

        LibraryRow {
            width: parent.width
            title: "Rebuild"
            detail: "Rescan the library"
            action: "Rebuild"
            ready: !pane.jobRunning()
            onActivated: view.startLibraryJob("library.cache", { force: true })
        }

        LibraryRow {
            width: parent.width
            title: "Import"
            detail: "Bring in files from incoming"
            action: "Import"
            ready: !pane.jobRunning() && (view.incomingFiles || []).length > 0
            onActivated: view.startLibraryJob("library.import", {})
        }

        LibraryRow {
            width: parent.width
            title: "Sync Likes"
            detail: "Download liked SoundCloud tracks"
            action: "Sync Likes"
            ready: !pane.jobRunning()
            onActivated: view.startLibraryJob("library.soundcloud.download", { import: false })
        }

        LibraryRow {
            width: parent.width
            title: "All Tracks"
            detail: "Include all tracks or only likes in autogen playlists"
            action: view.playlistLikedOnly ? "Only Likes" : "All Tracks"
            toggle: true
            on: !view.playlistLikedOnly
            onActivated: view.togglePlaylistFilter()
        }

        Item {
            width: parent.width
            visible: pane.jobActive()
            height: visible ? 10 : 0
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
                        width: Math.max(0, parent.width - jobName.implicitWidth - (stopChip.visible ? stopChip.width + parent.spacing : 0) - parent.spacing)
                        textFormat: Text.PlainText
                        text: pane.jobDetail()
                        color: view.libraryJob && view.libraryJob.status === "error" ? Theme.liked : Theme.muted
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeS
                        elide: Text.ElideRight
                    }

                    Chip {
                        id: stopChip
                        visible: pane.jobRunning()
                        text: "stop"
                        danger: true
                        onActivated: view.cancelLibraryJob()
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
        height: 168
        view: pane.view
    }

    ListView {
        id: files
        readonly property bool open: (view.incomingFiles || []).length > 0
        visible: open
        anchors.top: tools.bottom
        anchors.topMargin: open ? 10 : 0
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.rightMargin: 8
        anchors.bottom: open ? folders.top : tools.bottom
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
    }

    Item {
        id: folders
        visible: view.incomingGenrePath !== ""
        anchors.left: parent.left
        anchors.right: helpCol.left
        anchors.bottom: eqPanel.top
        anchors.bottomMargin: visible ? 8 : 0
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
