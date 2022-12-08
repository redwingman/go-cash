package config

type SeedNode struct {
	Url string `json:"url" msgpack:"url"`
}

var (
	MAIN_NET_SEED_NODES = []*SeedNode{
		{
			"wss://seed.pandoracash.com:2052/ws",
		},
		{
			"wss://seed.pandoracash.com:2087/ws",
		},
		{
			"wss://seed.pandoracash.com:2096/ws",
		},
		{
			"wss://seed.pandoracash.com:8443/ws",
		},
	}

	TEST_NET_SEED_NODES = []*SeedNode{
		{
			"wss://seed.testnet.pandoracash.com:2052/ws",
		},
		{
			"wss://seed.testnet.pandoracash.com:2087/ws",
		},
		{
			"wss://seed.testnet.pandoracash.com:2096/ws",
		},
		{
			"wss://seed.testnet.pandoracash.com:8443/ws",
		},
	}

	DEV_NET_SEED_NODES = []*SeedNode{
		{
			"wss://seed.devnet.pandoracash.com:2052/ws",
		},
		{
			"wss://seed.devnet.pandoracash.com:2087/ws",
		},
		{
			"wss://seed.devnet.pandoracash.com:2096/ws",
		},
		{
			"wss://seed.devnet.pandoracash.com:8443/ws",
		},
	}
)
