package mpris

import (
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/sebday/evoplayer/server/playback"
)

const basePath = "/org/mpris/MediaPlayer2"

type Bridge struct {
	conn     *dbus.Conn
	mu       sync.RWMutex
	status   playback.Status
	snapshot func() playback.Status
	onToggle func() error
	onStop   func()
	onNext   func() error
	onPrev   func() error
	onSeek   func(float64) error
}

func Start(snapshot func() playback.Status, onToggle func() error, onStop func(), onNext, onPrev func() error, onSeek func(float64) error) (*Bridge, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	b := &Bridge{
		conn:     conn,
		snapshot: snapshot,
		onToggle: onToggle,
		onStop:   onStop,
		onNext:   onNext,
		onPrev:   onPrev,
		onSeek:   onSeek,
	}
	if err := conn.Export(b, dbus.ObjectPath(basePath), "org.mpris.MediaPlayer2"); err != nil {
		return nil, err
	}
	if err := conn.Export(b, dbus.ObjectPath(basePath), "org.mpris.MediaPlayer2.Player"); err != nil {
		return nil, err
	}
	if err := conn.Export(&properties{b: b}, dbus.ObjectPath(basePath), "org.freedesktop.DBus.Properties"); err != nil {
		return nil, err
	}
	if err := conn.Export(introspect.Introspectable(introXML), dbus.ObjectPath(basePath), "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	reply, err := conn.RequestName("org.mpris.MediaPlayer2.evoplayer", dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner && reply != dbus.RequestNameReplyAlreadyOwner {
		return nil, fmt.Errorf("mpris name not acquired: %d", reply)
	}
	return b, nil
}

type properties struct {
	b *Bridge
}

func (p *properties) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	all, err := p.GetAll(iface)
	if err != nil {
		return dbus.Variant{}, err
	}
	v, ok := all[prop]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{"Unknown property " + prop})
	}
	return v, nil
}

func (p *properties) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	b := p.b
	switch iface {
	case "org.mpris.MediaPlayer2":
		identity, _ := b.Identity()
		entry, _ := b.DesktopEntry()
		canQuit, _ := b.CanQuit()
		canRaise, _ := b.CanRaise()
		hasTrackList, _ := b.HasTrackList()
		schemes, _ := b.SupportedUriSchemes()
		mimes, _ := b.SupportedMimeTypes()
		return map[string]dbus.Variant{
			"Identity":            dbus.MakeVariant(identity),
			"DesktopEntry":        dbus.MakeVariant(entry),
			"CanQuit":             dbus.MakeVariant(canQuit),
			"CanRaise":            dbus.MakeVariant(canRaise),
			"HasTrackList":        dbus.MakeVariant(hasTrackList),
			"SupportedUriSchemes": dbus.MakeVariant(schemes),
			"SupportedMimeTypes":  dbus.MakeVariant(mimes),
		}, nil
	case "org.mpris.MediaPlayer2.Player":
		playback, _ := b.PlaybackStatus()
		loop, _ := b.LoopStatus()
		rate, _ := b.Rate()
		shuffle, _ := b.Shuffle()
		meta, _ := b.Metadata()
		volume, _ := b.Volume()
		position, _ := b.Position()
		minRate, _ := b.MinimumRate()
		maxRate, _ := b.MaximumRate()
		canNext, _ := b.CanGoNext()
		canPrev, _ := b.CanGoPrevious()
		canPlay, _ := b.CanPlay()
		canPause, _ := b.CanPause()
		canSeek, _ := b.CanSeek()
		canControl, _ := b.CanControl()
		return map[string]dbus.Variant{
			"PlaybackStatus": dbus.MakeVariant(playback),
			"LoopStatus":     dbus.MakeVariant(loop),
			"Rate":           dbus.MakeVariant(rate),
			"Shuffle":        dbus.MakeVariant(shuffle),
			"Metadata":       dbus.MakeVariant(meta),
			"Volume":         dbus.MakeVariant(volume),
			"Position":       dbus.MakeVariant(position),
			"MinimumRate":    dbus.MakeVariant(minRate),
			"MaximumRate":    dbus.MakeVariant(maxRate),
			"CanGoNext":      dbus.MakeVariant(canNext),
			"CanGoPrevious":  dbus.MakeVariant(canPrev),
			"CanPlay":        dbus.MakeVariant(canPlay),
			"CanPause":       dbus.MakeVariant(canPause),
			"CanSeek":        dbus.MakeVariant(canSeek),
			"CanControl":     dbus.MakeVariant(canControl),
		}, nil
	default:
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{"Unknown interface " + iface})
	}
}

func (p *properties) Set(iface, prop string, _ dbus.Variant) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{"Property " + iface + "." + prop + " is read-only"})
}

func (b *Bridge) Close() {
	if b.conn != nil {
		_ = b.conn.Close()
	}
}

func (b *Bridge) Sync(st playback.Status) {
	b.mu.Lock()
	prev := b.status
	b.status = st
	b.mu.Unlock()
	b.emitPlayerChanges(prev, st)
}

func (b *Bridge) emitPlayerChanges(prev, next playback.Status) {
	statusCh, metadataCh, canCh := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if !statusCh && !metadataCh && !canCh {
		return
	}
	if b.conn == nil {
		return
	}
	changed := map[string]dbus.Variant{}
	if statusCh {
		changed["PlaybackStatus"] = dbus.MakeVariant(playbackStatusOf(next.State))
	}
	if metadataCh {
		changed["Metadata"] = dbus.MakeVariant(metadataFrom(next))
	}
	if canCh {
		can := next.Path != ""
		changed["CanPlay"] = dbus.MakeVariant(can)
		changed["CanPause"] = dbus.MakeVariant(can)
		changed["CanSeek"] = dbus.MakeVariant(can)
	}
	_ = b.conn.Emit(
		dbus.ObjectPath(basePath),
		"org.freedesktop.DBus.Properties.PropertiesChanged",
		"org.mpris.MediaPlayer2.Player",
		changed,
		[]string{},
	)
}

func (b *Bridge) current() playback.Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.status.Path != "" {
		return b.status
	}
	if b.snapshot != nil {
		return b.snapshot()
	}
	return b.status
}

func (b *Bridge) Identity() (string, *dbus.Error)     { return "evoplayer", nil }
func (b *Bridge) DesktopEntry() (string, *dbus.Error) { return "evoplayer", nil }
func (b *Bridge) CanQuit() (bool, *dbus.Error)        { return true, nil }
func (b *Bridge) CanRaise() (bool, *dbus.Error)       { return false, nil }
func (b *Bridge) HasTrackList() (bool, *dbus.Error)   { return false, nil }
func (b *Bridge) SupportedUriSchemes() ([]string, *dbus.Error) {
	return []string{"file"}, nil
}
func (b *Bridge) SupportedMimeTypes() ([]string, *dbus.Error) {
	return []string{"audio/mpeg", "audio/flac", "audio/ogg", "audio/wav"}, nil
}

func (b *Bridge) PlaybackStatus() (string, *dbus.Error) {
	return playbackStatusOf(b.current().State), nil
}

func (b *Bridge) LoopStatus() (string, *dbus.Error) { return "None", nil }
func (b *Bridge) Rate() (float64, *dbus.Error)      { return 1.0, nil }
func (b *Bridge) Shuffle() (bool, *dbus.Error)      { return b.current().Shuffle, nil }
func metadataFrom(st playback.Status) map[string]dbus.Variant {
	meta := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/evoplayer/CurrentTrack")),
	}
	if st.Path != "" {
		meta["xesam:url"] = dbus.MakeVariant("file://" + st.Path)
	}
	if st.Title != "" {
		meta["xesam:title"] = dbus.MakeVariant(st.Title)
	}
	if st.Artist != "" {
		meta["xesam:artist"] = dbus.MakeVariant([]string{st.Artist})
	}
	if st.Album != "" {
		meta["xesam:album"] = dbus.MakeVariant(st.Album)
	}
	if st.Art != "" {
		artURL := st.Art
		if !strings.HasPrefix(artURL, "file://") {
			artURL = "file://" + artURL
		}
		meta["mpris:artUrl"] = dbus.MakeVariant(artURL)
	}
	return meta
}

func (b *Bridge) Metadata() (map[string]dbus.Variant, *dbus.Error) {
	return metadataFrom(b.current()), nil
}

func (b *Bridge) Volume() (float64, *dbus.Error) {
	v := float64(b.current().Volume) / 100.0
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return v, nil
}

func (b *Bridge) Position() (int64, *dbus.Error) {
	return int64(b.current().Position * 1e6), nil
}
func (b *Bridge) MinimumRate() (float64, *dbus.Error) { return 1.0, nil }
func (b *Bridge) MaximumRate() (float64, *dbus.Error) { return 1.0, nil }
func (b *Bridge) CanGoNext() (bool, *dbus.Error)      { return true, nil }
func (b *Bridge) CanGoPrevious() (bool, *dbus.Error)  { return true, nil }
func (b *Bridge) CanPlay() (bool, *dbus.Error)        { return b.current().Path != "", nil }
func (b *Bridge) CanPause() (bool, *dbus.Error)       { return b.current().Path != "", nil }
func (b *Bridge) CanSeek() (bool, *dbus.Error)        { return b.current().Path != "", nil }
func (b *Bridge) CanControl() (bool, *dbus.Error)     { return true, nil }

func (b *Bridge) Quit() *dbus.Error  { return nil }
func (b *Bridge) Raise() *dbus.Error { return nil }

func (b *Bridge) goToggle() {
	if b.onToggle == nil {
		return
	}
	go func() { _ = b.onToggle() }()
}

func (b *Bridge) Play() *dbus.Error {
	st := b.current()
	if st.State != "playing" {
		b.goToggle()
	}
	return nil
}

func (b *Bridge) Pause() *dbus.Error {
	st := b.current()
	if st.State == "playing" {
		b.goToggle()
	}
	return nil
}

func (b *Bridge) PlayPause() *dbus.Error {
	b.goToggle()
	return nil
}

func (b *Bridge) Stop() *dbus.Error {
	if b.onStop != nil {
		go b.onStop()
	}
	return nil
}

func (b *Bridge) Next() *dbus.Error {
	if b.onNext != nil {
		go func() { _ = b.onNext() }()
	}
	return nil
}

func (b *Bridge) Previous() *dbus.Error {
	if b.onPrev != nil {
		go func() { _ = b.onPrev() }()
	}
	return nil
}

func (b *Bridge) Seek(offset int64) *dbus.Error {
	if b.onSeek == nil {
		return nil
	}
	st := b.current()
	go func() { _ = b.onSeek(st.Position + float64(offset)/1e6) }()
	return nil
}

func (b *Bridge) SetPosition(_ dbus.ObjectPath, position int64) *dbus.Error {
	if b.onSeek == nil {
		return nil
	}
	go func() { _ = b.onSeek(float64(position) / 1e6) }()
	return nil
}

func (b *Bridge) OpenUri(uri string) *dbus.Error {
	_ = uri
	return nil
}

const introXML = `<node>
  <interface name="org.freedesktop.DBus.Introspectable">
    <method name="Introspect"><arg direction="out" type="s" name="xml"/></method>
  </interface>
  <interface name="org.freedesktop.DBus.Properties">
    <method name="Get"><arg direction="in" type="s" name="interface_name"/><arg direction="in" type="s" name="property_name"/><arg direction="out" type="v" name="value"/></method>
    <method name="GetAll"><arg direction="in" type="s" name="interface_name"/><arg direction="out" type="a{sv}" name="properties"/></method>
    <method name="Set"><arg direction="in" type="s" name="interface_name"/><arg direction="in" type="s" name="property_name"/><arg direction="in" type="v" name="value"/></method>
    <signal name="PropertiesChanged"><arg type="s" name="interface_name"/><arg type="a{sv}" name="changed_properties"/><arg type="as" name="invalidated_properties"/></signal>
  </interface>
  <interface name="org.mpris.MediaPlayer2">
    <method name="Quit"/>
    <method name="Raise"/>
    <property name="Identity" type="s" access="read"/>
    <property name="DesktopEntry" type="s" access="read"/>
    <property name="CanQuit" type="b" access="read"/>
    <property name="CanRaise" type="b" access="read"/>
    <property name="HasTrackList" type="b" access="read"/>
    <property name="SupportedUriSchemes" type="as" access="read"/>
    <property name="SupportedMimeTypes" type="as" access="read"/>
  </interface>
  <interface name="org.mpris.MediaPlayer2.Player">
    <method name="Next"/>
    <method name="Previous"/>
    <method name="Pause"/>
    <method name="PlayPause"/>
    <method name="Stop"/>
    <method name="Play"/>
    <method name="Seek"><arg direction="in" type="x" name="Offset"/></method>
    <method name="SetPosition"><arg direction="in" type="o" name="TrackId"/><arg direction="in" type="x" name="Position"/></method>
    <method name="OpenUri"><arg direction="in" type="s" name="Uri"/></method>
    <property name="PlaybackStatus" type="s" access="read"/>
    <property name="LoopStatus" type="s" access="readwrite"/>
    <property name="Rate" type="d" access="readwrite"/>
    <property name="Shuffle" type="b" access="readwrite"/>
    <property name="Metadata" type="a{sv}" access="read"/>
    <property name="Volume" type="d" access="readwrite"/>
    <property name="Position" type="x" access="read"/>
    <property name="MinimumRate" type="d" access="read"/>
    <property name="MaximumRate" type="d" access="read"/>
    <property name="CanGoNext" type="b" access="read"/>
    <property name="CanGoPrevious" type="b" access="read"/>
    <property name="CanPlay" type="b" access="read"/>
    <property name="CanPause" type="b" access="read"/>
    <property name="CanSeek" type="b" access="read"/>
    <property name="CanControl" type="b" access="read"/>
  </interface>
</node>`
