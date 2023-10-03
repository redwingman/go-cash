package banned_nodes

import (
	"net/url"
	"pandora-pay/helpers/generics"
	"time"
)

type BannedNodesType struct {
	bannedMap *generics.Map[string, *BannedNode]
}

func (this *BannedNodesType) IsBanned(urlStr string) bool {
	if _, found := this.bannedMap.Load(urlStr); found {
		return true
	}
	return false
}

func (this *BannedNodesType) BanURL(url *url.URL, message string, duration time.Duration) {
	this.Ban(url.String(), message, duration)
}

func (this *BannedNodesType) Ban(urlStr, message string, duration time.Duration) {

	time := time.Now()
	this.bannedMap.Store(urlStr, &BannedNode{
		URL:        urlStr,
		Message:    message,
		Timestamp:  time,
		Expiration: time.Add(duration),
	})
}

var BannedNodes *BannedNodesType

func init() {
	BannedNodes = &BannedNodesType{
		bannedMap: &generics.Map[string, *BannedNode]{},
	}
}
