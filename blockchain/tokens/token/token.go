package token

import (
	"pandora-pay/helpers"
)

type Token struct {
	Version uint64
	//upgrade different settings
	CanUpgrade bool
	//increase supply
	CanMint bool
	//decrease supply
	CanBurn bool
	//can change key
	CanChangeKey bool
	//can change supply key
	CanChangeSupplyKey bool
	//can pause (suspend transactions)
	CanPause bool
	//freeze supply changes
	CanFreeze bool

	DecimalSeparator byte
	MaxSupply        uint64
	Supply           uint64

	Key       [20]byte
	SupplyKey [20]byte

	Name        string
	Ticker      string
	Description string
}

func (token *Token) Validate() {
	if token.DecimalSeparator > 10 {
		panic("token decimal separator is invalid")
	}
	if len(token.Name) > 15 || len(token.Name) < 3 {
		panic("token name length is invalid")
	}
	if len(token.Ticker) > 7 || len(token.Ticker) < 2 {
		panic("token ticker length is invalid")
	}
	if len(token.Description) > 512 {
		panic("token  description length is invalid")
	}

}

func (token *Token) AddSupply(sign bool, amount uint64) {

	if sign {
		if !token.CanMint {
			panic("Can't mint")
		}
		if token.MaxSupply-token.Supply < amount {
			panic("Supply exceeded max supply")
		}
		helpers.SafeUint64Add(&token.Supply, amount)
	} else {
		if !token.CanBurn {
			panic("Can't burn")
		}
		if token.Supply < amount {
			panic("Supply would become negative")
		}

		helpers.SafeUint64Sub(&token.Supply, amount)
	}

}

func (token *Token) Serialize() []byte {

	writer := helpers.NewBufferWriter()

	writer.WriteUvarint(token.Version)

	writer.WriteBool(token.CanUpgrade)
	writer.WriteBool(token.CanMint)
	writer.WriteBool(token.CanBurn)
	writer.WriteBool(token.CanChangeKey)
	writer.WriteBool(token.CanChangeSupplyKey)
	writer.WriteBool(token.CanPause)
	writer.WriteBool(token.CanFreeze)
	writer.WriteByte(token.DecimalSeparator)

	writer.WriteUvarint(token.MaxSupply)
	writer.WriteUvarint(token.Supply)

	writer.Write(token.Key[:])
	writer.Write(token.SupplyKey[:])

	writer.WriteString(token.Name)
	writer.WriteString(token.Ticker)
	writer.WriteString(token.Description)

	return writer.Bytes()
}

func (token *Token) Deserialize(buf []byte) {

	reader := helpers.NewBufferReader(buf)

	token.Version = reader.ReadUvarint()
	token.CanUpgrade = reader.ReadBool()
	token.CanMint = reader.ReadBool()
	token.CanBurn = reader.ReadBool()
	token.CanChangeKey = reader.ReadBool()
	token.CanChangeSupplyKey = reader.ReadBool()
	token.CanPause = reader.ReadBool()
	token.CanFreeze = reader.ReadBool()
	token.DecimalSeparator = reader.ReadByte()
	token.MaxSupply = reader.ReadUvarint()
	token.Supply = reader.ReadUvarint()
	token.Key = reader.Read20()
	token.SupplyKey = reader.Read20()
	token.Name = reader.ReadString()
	token.Ticker = reader.ReadString()
	token.Description = reader.ReadString()

	return
}
