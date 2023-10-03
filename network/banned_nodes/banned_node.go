package banned_nodes

import (
	"time"
)

type BannedNode struct {
	URL        string
	Timestamp  time.Time
	Expiration time.Time
	Message    string
}
