package dpos

import (
	"pandora-pay/helpers"
)

type DelegatedStakePending struct {

	//pending stake
	StakePending uint64

	//height when the stake pending was last updated
	StakePendingHeight uint64
}

func (delegatedStakePending *DelegatedStakePending) Serialize(writer *helpers.BufferWriter) {

	writer.WriteUvarint(delegatedStakePending.StakePending)
	writer.WriteUvarint(delegatedStakePending.StakePendingHeight)

}

func (delegatedStakePending *DelegatedStakePending) Deserialize(reader *helpers.BufferReader) {

	delegatedStakePending.StakePending = reader.ReadUvarint()
	delegatedStakePending.StakePendingHeight = reader.ReadUvarint()

}
