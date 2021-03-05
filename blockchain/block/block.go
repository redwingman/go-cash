package block

import (
	"pandora-pay/blockchain/accounts"
	"pandora-pay/blockchain/accounts/account/dpos"
	"pandora-pay/blockchain/tokens"
	"pandora-pay/config"
	"pandora-pay/config/reward"
	"pandora-pay/cryptography"
	"pandora-pay/cryptography/ecdsa"
	"pandora-pay/helpers"
)

type Block struct {
	BlockHeader
	MerkleHash helpers.Hash

	PrevHash       helpers.Hash
	PrevKernelHash helpers.Hash

	Timestamp uint64

	StakingAmount uint64

	DelegatedPublicKey [33]byte //33 byte public key. It IS NOT included in the kernel hash
	Forger             [20]byte // 20 byte public key hash
	Signature          [65]byte // 65 byte signature
}

func (blk *Block) Validate() {
	blk.BlockHeader.Validate()
}

func (blk *Block) IncludeBlock(acs *accounts.Accounts, toks *tokens.Tokens) {

	acc := acs.GetAccountEvenEmpty(blk.Forger)

	//for genesis block
	if blk.Height == 0 && !acc.HasDelegatedStake() {
		acc.DelegatedStakeVersion = 1
		acc.DelegatedStake = new(dpos.DelegatedStake)
		acc.DelegatedStake.DelegatedPublicKey = blk.DelegatedPublicKey
	}

	reward := reward.GetRewardAt(blk.Height)
	acc.DelegatedStake.AddDelegatedStake(true, reward, blk.Height)
	acs.UpdateAccount(blk.Forger, acc)

	tok := toks.GetToken(config.NATIVE_TOKEN_FULL)
	tok.AddSupply(true, reward)
	toks.UpdateToken(config.NATIVE_TOKEN_FULL, tok)

}

func (blk *Block) RemoveBlock(acs *accounts.Accounts, toks *tokens.Tokens) {

	acc := acs.GetAccount(blk.Forger)

	reward := reward.GetRewardAt(blk.Height)
	acc.DelegatedStake.AddDelegatedStake(false, reward, blk.Height)
	acs.UpdateAccount(blk.Forger, acc)

	tok := toks.GetToken(config.NATIVE_TOKEN_FULL)

	tok.AddSupply(false, reward)
	toks.UpdateToken(config.NATIVE_TOKEN_FULL, tok)
}

func (blk *Block) ComputeHash() helpers.Hash {
	return cryptography.SHA3Hash(blk.Serialize())
}

func (blk *Block) ComputeKernelHashOnly() helpers.Hash {
	out := blk.SerializeBlock(true, false)
	return cryptography.SHA3Hash(out)
}

func (blk *Block) ComputeKernelHash() helpers.Hash {

	hash := blk.ComputeKernelHashOnly()

	if blk.Height == 0 {
		return hash
	}

	return cryptography.ComputeKernelHash(hash, blk.StakingAmount)
}

func (blk *Block) SerializeForSigning() helpers.Hash {
	return cryptography.SHA3Hash(blk.SerializeBlock(false, false))
}

func (blk *Block) VerifySignature() bool {
	hash := blk.SerializeForSigning()
	return ecdsa.VerifySignature(blk.DelegatedPublicKey[:], hash[:], blk.Signature[0:64])
}

func (blk *Block) SerializeBlock(kernelHash bool, inclSignature bool) []byte {

	writer := helpers.NewBufferWriter()

	blk.BlockHeader.Serialize(writer)

	if !kernelHash {
		writer.Write(blk.MerkleHash[:])
		writer.Write(blk.PrevHash[:])
	}

	writer.Write(blk.PrevKernelHash[:])

	if !kernelHash {

		writer.WriteUvarint(blk.StakingAmount)
		writer.Write(blk.DelegatedPublicKey[:])
	}

	writer.WriteUvarint(blk.Timestamp)

	writer.Write(blk.Forger[:])

	if inclSignature {
		writer.Write(blk.Signature[:])
	}

	return writer.Bytes()
}

func (blk *Block) Serialize() []byte {
	return blk.SerializeBlock(false, true)
}

func (blk *Block) Deserialize(reader *helpers.BufferReader) {

	blk.BlockHeader.Deserialize(reader)
	blk.MerkleHash = reader.ReadHash()
	blk.PrevHash = reader.ReadHash()
	blk.PrevKernelHash = reader.ReadHash()
	blk.StakingAmount = reader.ReadUvarint()
	blk.DelegatedPublicKey = reader.Read33()
	blk.Timestamp = reader.ReadUvarint()
	blk.Forger = reader.Read20()
	blk.Signature = reader.Read65()

}
