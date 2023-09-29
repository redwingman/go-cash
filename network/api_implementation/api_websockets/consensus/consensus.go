package consensus

import (
	"pandora-pay/helpers/generics"
	"pandora-pay/helpers/recovery"
)

type Consensus struct {
	forks *Forks
}

func (consensus *Consensus) execute() {
	//discover forks
	processForksThread := newConsensusProcessForksThread(consensus.forks)
	recovery.SafeGo(processForksThread.execute)
}

func NewConsensus() *Consensus {

	consensus := &Consensus{
		&Forks{
			hashes: &generics.Map[string, *Fork]{},
		},
	}

	consensus.execute()

	return consensus
}
