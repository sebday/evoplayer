import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import "../compat"
import "."

Item {
    required property var dashboard

    id: volBtn
    property int btnSize: dashboard.transportBtnSize
    property real iconScale: 1
    readonly property int level: Math.round(dashboard.player.volume !== undefined ? dashboard.player.volume : 100)
    property bool wheelPopupActive: false
    readonly property bool popupVisible: volHover.containsMouse || volSliderPopup.sliderPressed || wheelPopupActive

    Component.onCompleted: if (visible) dashboard.volumeTransportBtn = volBtn
    Component.onDestruction: {
        if (dashboard.volumeTransportBtn === volBtn)
            dashboard.volumeTransportBtn = null
    }
    onVisibleChanged: {
        if (visible)
            dashboard.volumeTransportBtn = volBtn
        else if (dashboard.volumeTransportBtn === volBtn)
            dashboard.volumeTransportBtn = null
    }

    implicitWidth: btnSize
    implicitHeight: btnSize
    Layout.preferredWidth: btnSize
    Layout.preferredHeight: btnSize
    Layout.alignment: Qt.AlignVCenter

    function nudgeVolume(delta) {
        if (!delta)
            return
        wheelPopupActive = true
        popupHideTimer.restart()
        dashboard.adjustVolume(delta)
    }

    function handleWheel(wheel) {
        if (!wheel.angleDelta.y)
            return
        nudgeVolume(wheel.angleDelta.y > 0 ? 5 : -5)
        wheel.accepted = true
    }

    Timer {
        id: popupHideTimer
        interval: 1600
        repeat: false
        onTriggered: volBtn.wheelPopupActive = false
    }

    Rectangle {
        anchors.fill: parent
        radius: btnSize / 2
        color: Theme.foregroundWash
        visible: volHover.containsMouse
    }

    Text {
        id: volIcon
        anchors.centerIn: parent
        text: dashboard.volumeIcon(volBtn.level)
        color: volBtn.level <= 0 ? Theme.foreground : Theme.accent
        opacity: volBtn.level <= 0
            ? (volHover.containsMouse ? 0.58 : 0.45)
            : (volHover.containsMouse ? 1 : 0.9)
        font.family: Theme.fontFamily
        font.pixelSize: Math.max(9, Math.round(dashboard.transportSecondaryIconFont * volBtn.iconScale))
    }

    MouseArea {
        id: volHover
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.LeftButton
        cursorShape: Qt.PointingHandCursor
        onClicked: dashboard.toggleVolumeMute()
        onWheel: function(wheel) { volBtn.handleWheel(wheel) }
    }

    VolumeSliderPopup {
        dashboard: volBtn.dashboard
        id: volSliderPopup
        visible: popupVisible
        z: 10
        level: volBtn.level
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.top
        anchors.bottomMargin: 8
        onVolumeSet: function(v) {
            volBtn.wheelPopupActive = true
            popupHideTimer.restart()
            dashboard.setVolume(v)
        }
        onInteracted: {
            volBtn.wheelPopupActive = true
            popupHideTimer.restart()
        }
        onWheelNudge: function(delta) { volBtn.nudgeVolume(delta) }
    }
}
