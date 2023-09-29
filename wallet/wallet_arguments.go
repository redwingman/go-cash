package wallet

import (
	"encoding/base64"
	"errors"
	"pandora-pay/config/arguments"
	"pandora-pay/wallet/wallet_address"
	"strconv"
	"strings"
)

func (self *wallet) ProcessWalletArguments() (err error) {

	if mnemonic := arguments.Arguments["--wallet-import-secret-mnemonic"]; mnemonic != nil {
		if err = self.ImportMnemonic(mnemonic.(string)); err != nil {
			return
		}
	}

	if entropy := arguments.Arguments["--wallet-import-secret-entropy"]; entropy != nil {
		var bytes []byte
		if bytes, err = base64.StdEncoding.DecodeString(entropy.(string)); err != nil {
			return
		}
		if err = self.ImportEntropy(bytes); err != nil {
			return
		}
	}

	if str := arguments.Arguments["--wallet-encrypt"]; str != nil {
		v := strings.Split(str.(string), ",")

		var diff int
		if diff, err = strconv.Atoi(v[1]); err != nil {
			return
		}

		if err = self.Encryption.Encrypt(v[0], diff); err != nil {
			return
		}
	}

	if password := arguments.Arguments["--wallet-decrypt"]; password != nil {
		if err = self.loadWallet(password.(string), true); err != nil {
			return
		}
	}

	if arguments.Arguments["--wallet-remove-encryption"] == true {
		if err = self.Encryption.RemoveEncryption(); err != nil {
			return
		}
	}

	if str := arguments.Arguments["--wallet-export-shared-staked-address"]; str != nil {
		v := strings.Split(str.(string), ",")

		var addr *wallet_address.WalletAddress

		if v[0] == "auto" {
			if addr, err = self.GetFirstStakedAddress(true); err != nil {
				return
			}
		} else {
			var index int
			if index, err = strconv.Atoi(v[0]); err != nil {
				return
			} else {
				if addr, err = self.GetWalletAddress(index, true); err != nil {
					return
				}
			}
			if addr == nil {
				if addr, err = self.GetWalletAddressByEncodedAddress(v[0], true); err != nil {
					return
				}
			}
		}

		if addr == nil {
			return errors.New("Address specified by --wallet-export-shared-staked-address was not found")
		}
		if _, err = self.exportSharedStakedAddress(addr, v[2], false); err != nil {
			return
		}

	}

	return
}
