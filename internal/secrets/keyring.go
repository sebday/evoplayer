package secrets

import (
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
)

func keyringPassword(browser string) []byte {
	if secret, err := secretServicePassword(browser); err == nil && len(secret) > 0 {
		return secret
	}
	if secret := secretToolPassword(browser); len(secret) > 0 {
		return secret
	}
	return []byte("peanuts")
}

func secretServicePassword(browser string) ([]byte, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	svc := conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")
	var unused dbus.Variant
	var session dbus.ObjectPath
	if err := svc.Call("org.freedesktop.Secret.Service.OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&unused, &session); err != nil {
		return nil, err
	}
	defer conn.Object("org.freedesktop.secrets", session).Call("org.freedesktop.Secret.Session.Close", 0)

	wantLabel := browser + " Safe Storage"
	app := strings.ToLower(browser)
	searches := []map[string]string{
		{"application": app},
		{"application": browser},
		{"xdg:schema": "chrome_libsecret_os_crypt_password_v2", "application": app},
	}
	for _, attrs := range searches {
		if secret := searchSecretItems(conn, svc, session, attrs, wantLabel, app); len(secret) > 0 {
			return secret, nil
		}
	}

	var alias dbus.ObjectPath
	if err := svc.Call("org.freedesktop.Secret.Service.ReadAlias", 0, "default").Store(&alias); err != nil || alias == "" {
		return nil, err
	}
	if secret := collectionSecret(conn, alias, session, wantLabel, app); len(secret) > 0 {
		return secret, nil
	}
	return nil, nil
}

func searchSecretItems(conn *dbus.Conn, svc dbus.BusObject, session dbus.ObjectPath, attrs map[string]string, wantLabel, app string) []byte {
	var unlocked, locked []dbus.ObjectPath
	if err := svc.Call("org.freedesktop.Secret.Service.SearchItems", 0, attrs).Store(&unlocked, &locked); err != nil {
		return nil
	}
	if len(locked) > 0 {
		var unlockedLocked []dbus.ObjectPath
		var prompt dbus.ObjectPath
		_ = svc.Call("org.freedesktop.Secret.Service.Unlock", 0, locked).Store(&unlockedLocked, &prompt)
		unlocked = append(unlocked, unlockedLocked...)
	}
	for _, path := range unlocked {
		if secret := itemSecret(conn, path, session, wantLabel, app, true); len(secret) > 0 {
			return secret
		}
	}
	return nil
}

func collectionSecret(conn *dbus.Conn, collection, session dbus.ObjectPath, wantLabel, app string) []byte {
	col := conn.Object("org.freedesktop.secrets", collection)
	var itemsVal dbus.Variant
	if err := col.Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.Secret.Collection", "Items").Store(&itemsVal); err != nil {
		return nil
	}
	items, _ := itemsVal.Value().([]dbus.ObjectPath)
	for _, path := range items {
		if secret := itemSecret(conn, path, session, wantLabel, app, false); len(secret) > 0 {
			return secret
		}
	}
	return nil
}

type dbusSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func itemSecret(conn *dbus.Conn, path, session dbus.ObjectPath, wantLabel, app string, fromSearch bool) []byte {
	item := conn.Object("org.freedesktop.secrets", path)
	if !fromSearch {
		var labelVal dbus.Variant
		_ = item.Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.Secret.Item", "Label").Store(&labelVal)
		label, _ := labelVal.Value().(string)
		var attrsVal dbus.Variant
		_ = item.Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.Secret.Item", "Attributes").Store(&attrsVal)
		attrs, _ := attrsVal.Value().(map[string]string)
		if !keyringItemMatch(label, attrs, wantLabel, app) {
			return nil
		}
	}
	var secret dbusSecret
	if err := item.Call("org.freedesktop.Secret.Item.GetSecret", 0, session).Store(&secret); err != nil {
		return nil
	}
	return secret.Value
}

func keyringItemMatch(label string, attrs map[string]string, wantLabel, app string) bool {
	if strings.EqualFold(strings.TrimSpace(label), wantLabel) {
		return true
	}
	if attrs == nil {
		return false
	}
	if strings.EqualFold(attrs["application"], app) {
		return true
	}
	return strings.Contains(strings.ToLower(label), strings.ToLower(app)) && strings.Contains(strings.ToLower(label), "safe storage")
}

func secretToolPassword(browser string) []byte {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return nil
	}
	app := strings.ToLower(browser)
	lookups := [][]string{
		{"application", app},
		{"application", browser},
		{"xdg:schema", "chrome_libsecret_os_crypt_password_v2", "application", app},
	}
	for _, args := range lookups {
		out, err := exec.Command("secret-tool", append([]string{"lookup"}, args...)...).Output()
		if err != nil {
			continue
		}
		if secret := strings.TrimSpace(string(out)); secret != "" {
			return []byte(secret)
		}
	}
	return nil
}
