package config

type SeedNode struct {
	Url string `json:"url" msgpack:"url"`
}

var (
	MAIN_NET_SEED_NODES = []*SeedNode{
		{
			"ws://seed.mainnet.pandoracash.com:80/ws",
		},
		{
			"ws://seed.mainnet.pandoracash.com:443/ws",
		},
		{
			"ws://seed.mainnet.pandoracash.com:5102/ws",
		},
		{
			"ws://seed.mainnet.pandoracash.com:5100/ws",
		},
	}

	TEST_NET_SEED_NODES = []*SeedNode{
		{
			"ws://seed.testnet.pandoracash.com:5100/ws",
		},
		{
			"ws://seed.testnet.pandoracash.com:5101/ws",
		},
		{
			"ws://seed.testnet.pandoracash.com:5102/ws",
		},
		{
			"ws://seed.testnet.pandoracash.com:5100/ws",
		},
	}

	DEV_NET_SEED_NODES = []*SeedNode{
		{
			"ws://seed.devnet.pandoracash.com:6100/ws",
		},
		{
			"ws://seed.devnet.pandoracash.com:6101/ws",
		},
		{
			"ws://seed.devnet.pandoracash.com:6102/ws",
		},
		{
			"ws://seed.devnet.pandoracash.com:6100/ws",
		},
	}
)
