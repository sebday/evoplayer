import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.Pipewire
import qs.Commons
import qs.Ui
import "panel/compat/PluginIds.js" as PluginIds
import "media/Model.js" as Model

BarWidget {
  id: root
  moduleName: PluginIds.pluginId

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
  }

  function refresh() {
    if (panelLoader.item && panelLoader.item.refresh) panelLoader.item.refresh()
  }

  function togglePanel() {
    if (panelLoader.item && panelLoader.item.toggle) panelLoader.item.toggle()
  }

  function summonEvoplayer() {
    Quickshell.execDetached(["xdg-terminal-exec", "--", "evoplayer"])
  }

  readonly property var mediaService: bar?.shell?.firstPartyServiceFor("omarchy.media")
  readonly property bool bravePlaying: !!(mediaService && mediaService.bravePlaying)
  readonly property string browserTitle: bravePlaying && mediaService.activePlayer
    ? String(mediaService.activePlayer.trackTitle || "")
    : ""
  readonly property string browserArtist: bravePlaying && mediaService.activePlayer
    ? String(mediaService.activePlayer.trackArtist || "")
    : ""

  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false

  function open() {
    if (panelLoader.item && panelLoader.item.openFromHotkey) panelLoader.item.openFromHotkey()
  }

  function close() {
    if (panelLoader.item && panelLoader.item.close) panelLoader.item.close()
  }

  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false

  function closeForPopoutSwitch() {
    if (panelLoader.item) panelLoader.item.closeForPopoutSwitch()
  }

  readonly property bool iconError: panelLoader.item ? panelLoader.item.iconError === true : false
  readonly property bool iconBusy: panelLoader.item ? panelLoader.item.iconBusy === true : false
  readonly property bool iconMuted: panelLoader.item ? panelLoader.item.iconMuted === true : false
  readonly property string tooltip: Model.barTooltip(
    panelLoader.item ? panelLoader.item.playerPlaying === true : false,
    panelLoader.item ? panelLoader.item.trackTitle : "",
    root.hasOutput && root.outputMuted
  )

  readonly property var sink: Pipewire.defaultAudioSink
  readonly property var source: Pipewire.defaultAudioSource
  readonly property var nodes: Pipewire.nodes ? Pipewire.nodes.values : []
  property string volumeSinkName: ""
  property real wheelAccumulator: 0

  readonly property var volumeSink: {
    if (volumeSinkName === "" || !sink) return sink
    if (volumeSinkName === String(sink.name)) return sink
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i]
      if (n && n.isSink && !n.isStream && String(n.name) === volumeSinkName && n.audio)
        return n
    }
    return sink
  }

  readonly property real outputVolume: volumeSink && volumeSink.audio ? volumeSink.audio.volume : 0
  readonly property bool outputMuted: volumeSink && volumeSink.audio ? volumeSink.audio.muted : false
  readonly property bool inputMuted: source && source.audio ? source.audio.muted : false
  readonly property bool hasOutput: !!(volumeSink && volumeSink.audio)
  readonly property bool hasInput: !!(source && source.audio)
  readonly property bool anyAudible: (hasOutput && !outputMuted) || (hasInput && !inputMuted)

  function resolveVolumeSink() {
    if (!volumeSinkProc.running) volumeSinkProc.running = true
  }

  function setOutputVolume(v) {
    if (!volumeSink || !volumeSink.audio) return outputVolume
    var volume = Math.max(0, Math.min(1, v))
    volumeSink.audio.volume = volume
    return volume
  }

  function showVolumeOsd(volume) {
    if (!bar || !bar.shell) return
    bar.shell.summon("omarchy.osd", JSON.stringify({
      icon: Model.outputIcon(root.sink, volume, root.outputMuted),
      value: Math.round(volume * 100)
    }))
  }

  function toggleAllMuted() {
    var mute = anyAudible
    if (hasOutput) volumeSink.audio.muted = mute
    if (hasInput) source.audio.muted = mute
  }

  implicitWidth: chrome.implicitWidth
  implicitHeight: button.implicitHeight
  width: implicitWidth
  height: implicitHeight

  onBarChanged: injectPanel()
  onSettingsChanged: injectPanel()
  onSinkChanged: resolveVolumeSink()

  PwObjectTracker {
    objects: {
      var list = []
      if (root.sink) list.push(root.sink)
      if (root.source) list.push(root.source)
      if (root.volumeSink && list.indexOf(root.volumeSink) < 0)
        list.push(root.volumeSink)
      return list
    }
  }

  Process {
    id: volumeSinkProc
    command: ["omarchy-audio-output-sink"]
    stdout: StdioCollector {
      waitForEnd: true
      onStreamFinished: root.volumeSinkName = String(text).trim()
    }
  }

  Timer {
    interval: 15000
    running: true
    repeat: true
    triggeredOnStart: true
    onTriggered: root.resolveVolumeSink()
  }

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("media/Panel.qml")
    visible: false
    onLoaded: {
      root.injectPanel()
      Qt.callLater(root.injectPanel)
    }
  }

  Row {
    id: chrome
    spacing: root.bravePlaying && root.browserTitle !== "" ? Style.space(8) : 0
    height: root.height

    Item {
      id: scrollClip
      implicitWidth: root.bravePlaying && root.browserTitle !== "" ? Math.min(180, labelText.implicitWidth) : 0
      width: implicitWidth
      height: button.implicitHeight
      clip: true
      visible: width > 0

      Text {
        id: labelText
        textFormat: Text.PlainText
        text: root.browserTitle + (root.browserArtist ? "  ·  " + root.browserArtist : "")
        color: root.bar ? root.bar.barForeground : "white"
        font.family: root.bar ? root.bar.fontFamily : Style.font.family
        font.pixelSize: Style.font.body
        anchors.verticalCenter: parent.verticalCenter

        property bool needsScroll: implicitWidth > scrollClip.width && scrollClip.width > 0

        NumberAnimation on x {
          running: labelText.needsScroll && root.bravePlaying && root.bar && !root.bar.vertical
          loops: Animation.Infinite
          duration: Math.max(6000, labelText.implicitWidth * 25)
          from: scrollClip.width
          to: -labelText.implicitWidth
          easing.type: Easing.Linear
        }
      }
    }

    BarIconButton {
      id: button
      width: implicitWidth
      height: implicitHeight
      bar: root.bar
      text: Model.outputIcon(root.sink, root.outputVolume, root.outputMuted)
      active: root.iconError
      useActiveColor: root.iconError
      dimmed: !root.hasOutput
      tooltipText: Model.plain(root.tooltip)
      opacity: !hasVisualContent || concealed ? 0 : (root.iconError ? 1 : (root.iconBusy ? pulseOpacity : 1))
      property real pulseOpacity: 1

      SequentialAnimation on pulseOpacity {
        running: root.iconBusy && !root.iconError
        loops: Animation.Infinite
        NumberAnimation { from: 1.0; to: 0.42; duration: 880; easing.type: Easing.InOutSine }
        NumberAnimation { from: 0.42; to: 1.0; duration: 880; easing.type: Easing.InOutSine }
      }

      onPressed: function(b) {
        if (!root.bar) return
        if (b === Qt.RightButton) {
          root.toggleAllMuted()
          return
        }
        if (b === Qt.MiddleButton) {
          root.summonEvoplayer()
          return
        }
        root.togglePanel()
      }

      onWheelMoved: function(delta) {
        if (!root.hasOutput) return
        var wheel = Util.wheelSteps(root.wheelAccumulator, delta)
        root.wheelAccumulator = wheel.remainder
        if (wheel.steps === 0) return
        var volume = root.setOutputVolume(root.outputVolume + wheel.steps * 0.05)
        root.showVolumeOsd(volume)
      }
    }
  }
}
