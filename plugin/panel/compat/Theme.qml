pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import qs.commons as Shell

Item {
    id: root
    visible: false
    width: 0
    height: 0

    function withOpacity(c, alpha) {
        return Qt.rgba(c.r, c.g, c.b, alpha)
    }

    function mixColors(a, b, t) {
        return Qt.rgba(
            a.r + (b.r - a.r) * t,
            a.g + (b.g - a.g) * t,
            a.b + (b.b - a.b) * t,
            1
        )
    }

    readonly property color foreground: Shell.Theme.foreground
    readonly property color background: Shell.Theme.background
    readonly property color accent: Shell.Theme.accent
    readonly property color urgent: Shell.Theme.urgent
    property color good: Shell.Theme.accent
    property color liked: Shell.Theme.urgent
    property color border: Shell.Theme.accent
    property color muted: Shell.Theme.muted
    readonly property color highlight: Shell.Theme.accent
    readonly property color inactiveBorder: withOpacity(foreground, 0.25)
    readonly property color mantle: Shell.Theme.popups.background
    readonly property string iconThemeName: ""
    readonly property real surfaceOpacity: 0.97
    readonly property real surfaceOpacityInactive: 0.88
    readonly property bool roundingOn: Shell.Theme.cornerRadius > 0
    readonly property bool gapsOn: Shell.Theme.gapsOut > 0
    readonly property int gapsOut: Shell.Theme.gapsOut * 2
    readonly property int shellCornerRadiusPx: Shell.Theme.cornerRadius
    readonly property int panelCornerRadius: Shell.Theme.cornerRadius
    readonly property bool fieldsetRoundingOn: true
    readonly property int fieldsetCornerRadius: Shell.Theme.cornerRadius
    readonly property color overlaySurface: Shell.Theme.popups.background
    readonly property color overlaySurfaceInactive: withOpacity(mantle, surfaceOpacityInactive)
    readonly property color panelBackground: overlaySurface
    readonly property real panelMantleLift: 0.12
    readonly property color panelMantle: mixColors(mantle, foreground, panelMantleLift)
    readonly property color heatmap0: mixColors(mantle, foreground, 0.12)
    readonly property color heatmap1: mixColors(mantle, accent, 0.4)
    readonly property color heatmap2: mixColors(mantle, accent, 0.6)
    readonly property color heatmap3: mixColors(mantle, accent, 0.8)
    readonly property color heatmap4: accent
    readonly property var heatmapColors: [heatmap0, heatmap1, heatmap2, heatmap3, heatmap4]
    readonly property color recapArtistsTint: mixColors(mantle, accent, 0.52)
    readonly property color recapAlbumsTint: mixColors(mantle, urgent, 0.52)
    readonly property color recapTracksTint: mixColors(mantle, highlight, 0.52)
    readonly property var chartPalette: [
        accent,
        mixColors(accent, foreground, 0.4),
        mixColors(mantle, foreground, 0.5),
        mixColors(accent, urgent, 0.35)
    ]

    readonly property int motionFast: 120
    readonly property int motionNormal: 180
    readonly property int motionSlow: 220

    readonly property color fillNeutralSubtle: foregroundFaint
    readonly property color fillAccentSubtle: withOpacity(accent, 0.14)
    readonly property color fillUrgentSubtle: withOpacity(urgent, 0.14)
    readonly property real opacityBodyText: 0.85
    readonly property real opacityDisabled: 0.45
    readonly property real opacityMuted: 0.55
    readonly property real opacityHover: 0.65
    readonly property real opacitySecondary: 0.72
    readonly property real opacityEmphasis2: 0.82
    readonly property real opacityEmphasis: opacityEmphasis2

    readonly property color barIconColor: foreground
    readonly property color barIconColorActive: accent
    readonly property real barIconOpacity: opacityEmphasis2
    readonly property real barIconOpacityActive: 1
    readonly property real barIconOpacityDim: 0.45

    readonly property color fieldsetBorderColor: foregroundBorder
    readonly property int fieldsetBorderWidth: 1
    readonly property int fieldsetLegendInset: spacingS
    readonly property int fieldsetLegendMinHeight: fontSizeS + spacingS

    readonly property int hoverPanelStatColumnSpacing: spacingM
    readonly property int hoverPanelStatRowSpacing: spacingS
    readonly property int hoverPanelStatValueFont: fontSizeL
    readonly property int hoverPanelStatLabelFont: fontSizeS

    readonly property color foregroundGhost: withOpacity(foreground, 0.05)
    readonly property color foregroundWash: withOpacity(foreground, 0.06)
    readonly property color foregroundFaint: withOpacity(foreground, 0.08)
    readonly property color foregroundHoverWash: withOpacity(foreground, 0.1)
    readonly property color foregroundRaised: withOpacity(foreground, 0.12)
    readonly property color foregroundDivider: withOpacity(foreground, 0.14)
    readonly property color foregroundSubtle: withOpacity(foreground, 0.16)
    readonly property color foregroundTrack: withOpacity(foreground, 0.18)
    readonly property color foregroundPickerBorder: withOpacity(foreground, 0.22)
    readonly property color foregroundBorder: withOpacity(foreground, 0.32)

    readonly property int spacing2: 2
    readonly property int spacingS: Shell.Theme.space(6)
    readonly property int spacingM: Shell.Theme.space(8)
    readonly property int spacingL: Shell.Theme.space(10)
    readonly property int settingsNavRowPad: spacingL
    readonly property int panelLabelPadH: spacingS

    readonly property int radiusS: 2
    readonly property int radiusM: 3
    readonly property int radiusL: fieldsetCornerRadius

    readonly property string fontFamily: Shell.Theme.font.family
    readonly property bool fontBold: true
    readonly property int fontPixelSize: Shell.Theme.font.body
    readonly property int fontSizeXxs: Math.max(8, fontPixelSize - 3)
    readonly property int fontSizeXs: Math.max(9, fontPixelSize - 2)
    readonly property int fontSizeS: Shell.Theme.font.bodySmall
    readonly property int fontSizeM: Shell.Theme.font.body
    readonly property int fontSizeL: Shell.Theme.font.subtitle
    readonly property int fontSizeXl: Shell.Theme.font.title
    readonly property int fontSize2xl: Shell.Theme.font.heading
    readonly property int fontSize3xl: Shell.Theme.font.display
    readonly property int fontSize4xl: Shell.Theme.font.displayLarge
    readonly property int fontSize5xl: fontPixelSize + 8
    readonly property int fontSize6xl: fontPixelSize + 9
    readonly property int fontSize7xl: fontSizeS * 2
    readonly property int fontSize8xl: fontPixelSize * 2
    readonly property int fontSize9xl: fontPixelSize + 15
    readonly property int fontSizeHero: fontPixelSize * 3
    readonly property int fontSizeHeroLg: fontPixelSize * 4

    readonly property int hoverPanelSectionSpacing: Shell.Theme.space(10)
    readonly property int panelSectionSpacing: Shell.Theme.space(14)
    readonly property int hoverPanelContentPad: Shell.Theme.space(16)
    readonly property int panelContentPad: Shell.Theme.space(10)
    readonly property int panelDockPad: panelContentPad + spacingS
    readonly property int hoverPanelMargin: Shell.Theme.space(16)
    readonly property int hoverPanelTopPad: hoverPanelMargin - 10
    readonly property int hoverPanelBorderWidth: 2
    readonly property int hoverPanelRevealDuration: motionNormal
    readonly property int hoverPanelRevealOffset: 10
    readonly property int hoverPanelRevealMaxWait: 200
    readonly property int barHoverTopPad: 10
    readonly property int barHoverContentTopPad: barHoverTopPad - 10
    readonly property int overlayWidthDefault: hoverPanelWidthStandard
    readonly property int overlayMargin: hoverPanelMargin
    readonly property int overlayContentInset: hoverPanelMargin + hoverPanelBorderWidth
    readonly property int overlayTopInset: hoverPanelTopPad + hoverPanelBorderWidth
    readonly property int overlaySideInset: overlayContentInset
    readonly property real specialWorkspaceDim: 0.6
    readonly property int screenEdgeInset: barHoverTopPad
    readonly property int hoverPanelWidthStandard: 440
    readonly property int hoverPanelWidthWide: 580
    readonly property int overlayPanelWidth: 600
    readonly property int systemPanelWidth: 800
    readonly property int systemMenuPanelWidth: 480
    readonly property real menuPanelHeightRatio: 0.5
    readonly property real menuPanelWidthRatio: 0.25
    readonly property int settingsSideTabWidth: 152
    readonly property int settingsSideTabIconWidth: 24
    readonly property int systemMenuPanelHeight: 600
    readonly property int settingsPanelWidth: systemPanelWidth
    readonly property int systemMenuWidth: systemMenuPanelWidth
    readonly property int clipboardPanelWidth: Math.round(systemPanelWidth / 2)
    readonly property int barHeight: Shell.Theme.bar.sizeHorizontal
    readonly property int barPaddingX: Shell.Theme.space(16)
    readonly property int barGap: Shell.Theme.space(8)
    readonly property int barSectionGap: Shell.Theme.space(14)
    readonly property int sparklineGap: 6
    readonly property int sparklineChartMargin: 10
    readonly property int hoverPanelChartPadH: sparklineChartMargin + spacingS
    readonly property int sparklineHeight: 12
    readonly property int sparklineWideBarWidth: 8
    readonly property int sparklineCellSize: 7
    readonly property int sparklineBarSpacing: 1
    readonly property int sparklineExpandedHeight: 52
    readonly property int sparklineExpandedBarWidth: 10
    readonly property int sparklineExpandedBarSpacing: 3
    readonly property int notificationWidth: 440
    readonly property int notificationPadding: 14
    readonly property int notificationArtSize: 84
    readonly property int notificationMediaPad: 16
    readonly property int notificationStackSlot: 104

    property string _themeBuf: ""
    property bool _themeOverflow: false

    function pickTomlColor(text, key) {
        var lines = String(text || "").split("\n")
        for (var i = 0; i < lines.length; i++) {
            var line = lines[i].replace(/^\s+|\s+$/g, "")
            if (line.indexOf(key) !== 0)
                continue
            var after = line.charAt(key.length)
            if (after !== " " && after !== "=" && after !== "\t")
                continue
            var eq = line.indexOf("=")
            if (eq < 0)
                continue
            var rest = line.slice(eq + 1).replace(/^\s+|\s+$/g, "")
            if (rest.charAt(0) === "\"")
                rest = rest.slice(1)
            var quote = rest.indexOf("\"")
            if (quote >= 0)
                rest = rest.slice(0, quote)
            if (/^#[0-9A-Fa-f]{6}$/.test(rest))
                return rest
        }
        return ""
    }

    function applyThemeColors(text) {
        var green = pickTomlColor(text, "green")
        var red = pickTomlColor(text, "red")
        var blue = pickTomlColor(text, "blue")
        var accentHex = pickTomlColor(text, "accent")
        var dim = pickTomlColor(text, "muted")
        good = green || Shell.Theme.accent
        liked = red || Shell.Theme.urgent
        border = blue || accentHex || Shell.Theme.accent
        muted = dim || Shell.Theme.muted
    }

    function themeColorsPath() {
        var home = Quickshell.env("HOME") || ""
        if (!home)
            return ""
        return home + "/.themes/current/colors.toml"
    }

    Process {
        id: themeColorsProc
        stdout: SplitParser {
            splitMarker: ""
            onRead: function(chunk) {
                if (root._themeOverflow)
                    return
                root._themeBuf += String(chunk || "")
                if (root._themeBuf.length > 4096) {
                    root._themeOverflow = true
                    root._themeBuf = ""
                    themeColorsProc.signal(15)
                }
            }
        }
        onExited: {
            if (!root._themeOverflow && root._themeBuf)
                root.applyThemeColors(root._themeBuf)
            root._themeBuf = ""
        }
    }

    Component.onCompleted: {
        var colorsPath = themeColorsPath()
        if (!colorsPath)
            return
        _themeBuf = ""
        _themeOverflow = false
        themeColorsProc.command = [
            "dd",
            "if=" + colorsPath,
            "iflag=nofollow,nonblock,count_bytes,fullblock",
            "bs=1",
            "count=4097",
            "status=none"
        ]
        themeColorsProc.running = true
    }
}
