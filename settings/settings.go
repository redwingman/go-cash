package settings

import (
	"pandora-pay/config/arguments"
	"pandora-pay/gui"
	"pandora-pay/helpers"
	"sync"
)

type settings struct {
	Name         string `json:"name"  msgpack:"name"`
	sync.RWMutex `json:"-"  msgpack:"-"`
}

var Settings *settings

func (self *settings) createEmptySettings() (err error) {
	self.Lock()
	defer self.Unlock()

	self.Name = helpers.RandString(10)

	self.updateSettings()
	if err = self.saveSettings(); err != nil {
		return
	}
	return
}

func (self *settings) updateSettings() {
	gui.GUI.InfoUpdate("Node", self.Name)
}

func Initialize() error {

	Settings = &settings{}
	if err := Settings.loadSettings(); err != nil {
		if err.Error() != "Settings doesn't exist" {
			return err
		}
		if err = Settings.createEmptySettings(); err != nil {
			return err
		}
	}

	var changed bool
	if arguments.Arguments["--node-name"] != nil {
		Settings.Name = arguments.Arguments["--node-name"].(string)
		changed = true
	}

	if changed {
		Settings.updateSettings()
		if err := Settings.saveSettings(); err != nil {
			return err
		}
	}

	gui.GUI.Log("Settings Initialized")
	return nil
}
