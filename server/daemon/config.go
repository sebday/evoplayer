package daemon

import (
	"strings"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/secrets"
)

func (d *Daemon) configJSONView() (config.JSONView, error) {
	view, err := config.JSON(d.Env.MusicConfig, d.Env.MusicRoot)
	if err != nil {
		return view, err
	}
	view.Soundcloud.OAuthSource = secrets.SoundcloudOAuth().Source
	return view, nil
}

func (d *Daemon) handleConfigSet(req ipc.Request) (interface{}, error) {
	var p struct {
		Section string `json:"section"`
		Key     string `json:"key"`
		Value   string `json:"value"`
	}
	if err := ipc.DecodeParams(req.Params, &p); err != nil {
		return nil, ipc.ErrInvalidParams("%v", err)
	}
	section := strings.TrimSpace(p.Section)
	key := strings.TrimSpace(p.Key)
	value := strings.TrimSpace(p.Value)
	if section == "" || key == "" {
		return nil, ipc.ErrInvalidParams("section and key required")
	}
	if section == "paths" && key == "root" {
		value = strings.TrimRight(value, "/")
		if err := config.ValidateMusicRoot(value); err != nil {
			return nil, err
		}
		if err := d.Env.EnsureDirs(); err != nil {
			return nil, err
		}
		if err := config.Set(d.Env.MusicConfig, section, key, value); err != nil {
			return nil, err
		}
		d.Env.MusicRoot = value
		return d.configJSONView()
	}
	if err := config.Set(d.Env.MusicConfig, section, key, value); err != nil {
		return nil, err
	}
	return d.configJSONView()
}
