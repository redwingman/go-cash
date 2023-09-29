package api_delegator_node

type DelegatorNode struct {
	chainHeight uint64 //use atomic
}

func NewDelegatorNode() (delegator *DelegatorNode) {

	delegator = &DelegatorNode{
		0,
	}

	return
}
