import QtQuick
import QtQuick.Layouts
import "../compat"
import "../panels"

ColumnLayout {
    id: root

    required property var dashboard

    spacing: Theme.spacingM

    LibrarySettingsPanel {
        dashboard: root.dashboard
        Layout.fillWidth: true
    }

    SoundCloudSettingsPanel {
        dashboard: root.dashboard
        Layout.fillWidth: true
    }
}
