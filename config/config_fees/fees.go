package config_fees

var (
	FEE_PER_BYTE             = uint64(15)
	FEE_PER_BYTE_ZETHER      = uint64(30)
	FEE_PER_BYTE_EXTRA_SPACE = uint64(2000)
)

func ComputeTxFee(size, feePerByte, extraSpace, feePerByeExtraSpace uint64) uint64 {
	return size*feePerByte + extraSpace*feePerByeExtraSpace
}
