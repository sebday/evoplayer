import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: thumbRoot
    property int side: 56
    property bool showPickerOverlay: false
    property bool fillPane: false
    property string art: ""
    property bool useRevision: false
    property bool artRetryQueued: false

    function nudgeVolume(delta) {
        if (!delta)
            return
        if (dashboard.volumeTransportBtn)
            dashboard.volumeTransportBtn.nudgeVolume(delta)
        else
            dashboard.adjustVolume(delta)
    }

    function handleWheel(wheel) {
        if (!wheel.angleDelta.y)
            return
        nudgeVolume(wheel.angleDelta.y > 0 ? 5 : -5)
        wheel.accepted = true
    }

    implicitWidth: fillPane ? 0 : side
    implicitHeight: fillPane ? 0 : side
    width: fillPane ? undefined : side
    height: fillPane ? undefined : side

    property string imageUrl: {
        if (!thumbRoot.art)
            return ""
        if (thumbRoot.useRevision)
            dashboard.artRevision // bind revision bumps
        return dashboard.artUrl(thumbRoot.art, thumbRoot.useRevision)
    }

    readonly property string displaySource: thumbRoot.imageUrl

    Rectangle {
        id: coverFrame
        anchors.fill: parent
        radius: fillPane ? Theme.fieldsetCornerRadius : 3
        clip: true
        color: Theme.foregroundFaint

        Image {
            id: coverImage
            anchors.fill: parent
            visible: thumbRoot.art !== ""
            opacity: status === Image.Ready ? 1 : 0
            source: thumbRoot.displaySource
            fillMode: Image.PreserveAspectCrop
            smooth: true
            asynchronous: true
            onStatusChanged: {
                if (status === Image.Error && thumbRoot.art !== "" && !thumbRoot.artRetryQueued) {
                    thumbRoot.artRetryQueued = true
                    var p = String((dashboard.player && dashboard.player.path) || "")
                    if (p)
                        dashboard.applyDisplayArtForPath(p, function() {
                            thumbRoot.artRetryQueued = false
                        })
                    else
                        thumbRoot.artRetryQueued = false
                }
                if (status === Image.Ready)
                    thumbRoot.artRetryQueued = false
            }
        }

        Text {
            anchors.centerIn: parent
            visible: thumbRoot.art === "" || coverImage.status !== Image.Ready
            text: "󰎈"
            color: Theme.accent
            font.family: Theme.fontFamily
            font.pixelSize: Math.round((fillPane ? Math.min(thumbRoot.width, thumbRoot.height) : thumbRoot.side) * 0.38)
            opacity: 0.5
        }

        Rectangle {
            anchors.fill: parent
            visible: coverDrop.containsDrag
            color: Qt.rgba(Theme.accent.r, Theme.accent.g, Theme.accent.b, 0.18)
            border.color: Theme.accent
            border.width: 2
            radius: Theme.radiusM

            Text {
                anchors.centerIn: parent
                text: "drop image"
                color: Theme.accent
                font.family: Theme.fontFamily
                font.pixelSize: dashboard.libraryFont
                opacity: Theme.opacityEmphasis
            }
        }

        DropArea {
            id: coverDrop
            anchors.fill: parent
            keys: ["text/uri-list"]

            onEntered: function(drag) {
                var ok = false
                if (dashboard.artTargetPath && drag.hasUrls) {
                    for (var i = 0; i < drag.urls.length; i++) {
                        if (dashboard.isImagePath(dashboard.localPathFromUrl(drag.urls[i]))) {
                            ok = true
                            break
                        }
                    }
                }
                drag.accepted = ok
            }

            onDropped: function(drop) {
                if (!drop.hasUrls || !dashboard.artTargetPath)
                    return
                for (var j = 0; j < drop.urls.length; j++) {
                    var p = dashboard.localPathFromUrl(drop.urls[j])
                    if (dashboard.isImagePath(p)) {
                        dashboard.openArtPickerForDrop(p)
                        break
                    }
                }
            }
        }

        MouseArea {
            z: 1
            anchors.fill: parent
            enabled: !dashboard.artPickerOpen
            hoverEnabled: true
            cursorShape: (dashboard.artTargetPath || "") !== ""
                ? Qt.PointingHandCursor : Qt.ArrowCursor
            onClicked: function(mouse) {
                if (!(dashboard.artTargetPath || ""))
                    return
                mouse.accepted = true
                dashboard.openArtPicker()
            }
            onWheel: function(wheel) { thumbRoot.handleWheel(wheel) }
        }

        ArtPickerOverlay {
        dashboard: thumbRoot.dashboard
            z: 2
            anchors.fill: parent
            visible: thumbRoot.showPickerOverlay && dashboard.artPickerOpen
        }
    }
}
