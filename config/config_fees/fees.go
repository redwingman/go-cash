package config_fees

var (
	FEE_PER_BYTE             = uint64(50)
	FEE_PER_BYTE_ZETHER      = uint64(130)
	FEE_PER_BYTE_EXTRA_SPACE = uint64(2200)
)

func ComputeTxFee(size, feePerByte, extraSpace, feePerByeExtraSpace uint64) uint64 {
	return size*feePerByte + extraSpace*feePerByeExtraSpace
}
