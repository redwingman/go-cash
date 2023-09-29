package settings

import (
	"bytes"
	"errors"
	"pandora-pay/gui"
	"pandora-pay/helpers/msgpack"
	"pandora-pay/store"
	"pandora-pay/store/store_db/store_db_interface"
)

func (settings *settings) saveSettings() error {

	return store.StoreSettings.DB.Update(func(writer store_db_interface.StoreDBTransactionInterface) (err error) {

		writer.Put("saved", []byte{2})

		marshal, err := msgpack.Marshal(settings)
		if err != nil {
			return
		}

		writer.Put("settings", marshal)
		writer.Put("saved", []byte{1})

		return
	})

}

func (self *settings) loadSettings() error {
	return store.StoreSettings.DB.View(func(reader store_db_interface.StoreDBTransactionInterface) (err error) {

		saved := reader.Get("saved")
		if saved == nil {
			return errors.New("Settings doesn't exist")
		}
		if bytes.Equal(saved, []byte{1}) {
			gui.GUI.Log("Settings Loading... ")

			unmarshal := reader.Get("settings")
			if err = msgpack.Unmarshal(unmarshal, self); err != nil {
				return err
			}

			self.updateSettings()
			gui.GUI.Log("Settings Loaded! " + self.Name)

		} else {
			err = errors.New("Error loading wallet ?")
		}

		return
	})
}
