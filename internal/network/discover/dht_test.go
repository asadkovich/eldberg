package discover

import (
	"github.com/AlexeySadkovich/eldberg/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

func TestTwentyNodes(t *testing.T) {
	port := 6000
	bootPort := port
	dhts := make([]*DHT, 0, 20)

	logger := zap.NewNop().Sugar()

	for i := 0; i < 20; i++ {
		cfg := &mockNodeConfig{
			dhtAddr: "127.0.0.1",
			dhtPort: port,
			bootnodes: []config.Node{
				{Addr: "127.0.0.1", Port: bootPort},
			},
		}
		dht := New(cfg, logger)
		err := dht.Listen()
		assert.NoError(t, err)

		dhts = append(dhts, dht)
		port++
	}

	wg := sync.WaitGroup{}
	for _, dht := range dhts {
		wg.Add(1)
		go func(dht *DHT, wg *sync.WaitGroup) {
			defer wg.Done()
			err := dht.Bootstrap()
			assert.NoError(t, err)
		}(dht, &wg)
	}

	wg.Wait()
	time.Sleep(2 * time.Second)

	for _, dht := range dhts {
		assert.Equal(t, 19, dht.NodesAmount())
	}
}

type mockNodeConfig struct {
	dhtAddr   string
	dhtPort   int
	bootnodes []config.Node
}

func (m *mockNodeConfig) GetNodeConfig() *config.NodeConfig {
	c := new(config.NodeConfig)
	c.Discover.Address = m.dhtAddr
	c.Discover.ListeningPort = m.dhtPort
	c.Discover.BootstrapNodes = m.bootnodes

	return c
}
func (m *mockNodeConfig) GetChainConfig() *config.ChainConfig {
	return nil
}
func (m *mockNodeConfig) GetHolderConfig() *config.HolderConfig {
	return nil
}
