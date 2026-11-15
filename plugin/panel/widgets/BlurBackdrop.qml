import QtQuick
import QtQuick.Effects
import "../compat"

Item {
    id: backdrop

    property string source: ""
    property int corner: 8

    Rectangle {
        anchors.fill: parent
        radius: backdrop.corner
        color: Theme.mantle
    }

    Image {
        id: blurSource
        anchors.fill: parent
        source: backdrop.source
        fillMode: Image.PreserveAspectCrop
        asynchronous: true
        sourceSize.width: 160
        sourceSize.height: 160
        visible: false
    }

    MultiEffect {
        anchors.fill: parent
        source: blurSource
        visible: blurSource.status === Image.Ready && backdrop.source !== ""
        autoPaddingEnabled: false
        blurEnabled: true
        blur: 1.0
        blurMax: 64
        saturation: 0.15
        maskEnabled: true
        maskSource: mask
        maskThresholdMin: 0.5
        maskSpreadAtMin: 1.0
    }

    Item {
        id: mask
        anchors.fill: parent
        layer.enabled: true
        visible: false

        Rectangle {
            anchors.fill: parent
            radius: backdrop.corner
        }
    }

    Rectangle {
        anchors.fill: parent
        radius: backdrop.corner
        gradient: Gradient {
            orientation: Gradient.Horizontal
            GradientStop { position: 0.0; color: Theme.withOpacity(Theme.background, 0.86) }
            GradientStop { position: 0.6; color: Theme.withOpacity(Theme.background, 0.62) }
            GradientStop { position: 1.0; color: Theme.withOpacity(Theme.background, 0.35) }
        }
    }
}
