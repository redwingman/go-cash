package config_stake

import (
	"pandora-pay/config/config_coins"
	"pandora-pay/config/globals"
)

func GetRequiredStake(blockHeight uint64) (requiredStake uint64) {

	var err error

	var amount uint64
	if blockHeight < 30000 { //~5 weeks
		amount = 200
	} else {
		amount = 5000
	}

	if requiredStake, err = config_coins.ConvertToUnitsUint64(amount); err != nil {
		panic(err)
	}

	return
}

func GetPendingStakeWindow(blockHeight uint64) uint64 {

	if globals.Arguments["--new-devnet"] == true {

		if blockHeight == 0 {
			return 1
		}
		return 10
	}

	if blockHeight < 10000 { //11 days
		return 10 //0.27h
	} else if blockHeight < 50000 { //8 weeks
		return 120 //3.3 h
	}

	return 4000 //4.67 days
}
