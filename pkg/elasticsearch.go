package pkg

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"time"
)

type esClient struct {
	url string
}

func newEsClient(url string) *esClient {
	return &esClient{url}
}

func (c *esClient) get(path string, out interface{}) error {
	resp, err := httpRequest(c.url, "GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *esClient) put(path string, in interface{}) error {
	return postBody("PUT", c.url, path, in)
}

func (c *esClient) post(path string, in interface{}) error {
	return postBody("POST", c.url, path, in)
}

//==============================================================================

type clusterHealth struct {
	Status             string
	NumberOfDataNodes  int
	InitializingShards int
	RelocatingShards   int
}

type clusterSettings struct {
	Transient  map[string]interface{} `json:"transient,omitempty"`
	Persistent map[string]interface{} `json:"persistent,omitempty"`
}

//==============================================================================

func (c *esClient) clusterHealth() (health *clusterHealth, err error) {
	if err = c.get("_cluster/health", health); err != nil {
		return nil, err
	}
	return health, nil
}

func (c *esClient) updateSettings(settings interface{}) error {
	return c.put("_settings", settings)
}

func (c *esClient) updateClusterSettings(settings *clusterSettings) error {
	return c.put("_cluster/settings", settings)
}

//==============================================================================

func (c *esClient) waitForShardRecovery() error {
	var (
		errorGracePeriod       = 2 * time.Minute
		timesToVerify          = 5
		timesToVerifyRed       = 20
		timesToVerifyUncertain = 20
	)

	var (
		timesVerified          = 0
		timesVerifiedRed       = 0
		timesVerifiedUncertain = 0
	)

	return waitFor(30*time.Minute, 5*time.Second, func(d time.Duration) (bool, error) {
		health, err := c.clusterHealth()
		switch {
		case err != nil:
			if d > errorGracePeriod {
				return false, errors.New("Timed out waiting on shard recovery")
			}
		case health.InitializingShards > 0 || health.RelocatingShards > 0:
			return false, nil
		case health.Status == "green" || health.Status == "yellow":
			timesVerifiedRed = 0
			timesVerifiedUncertain = 0
			timesVerified++
			return timesVerified >= timesToVerify, nil
		case health.Status == "red" && health.InitializingShards == 0:
			timesVerified = 0
			timesVerifiedUncertain = 0
			timesVerifiedRed++
			return timesVerifiedRed >= timesToVerifyRed, nil
		}
		timesVerified = 0
		timesVerifiedRed = 0
		timesVerifiedUncertain++
		return timesVerifiedUncertain >= timesToVerifyUncertain, nil
	})
}

func (c *esClient) setMinMasterNodes(min int) error {
	settings := map[string]int{"discovery.zen.minimum_master_nodes": min}
	return c.updateSettings(settings)
}

func (c *esClient) setAwarenessAttrs(attrs []string) error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.awareness.attributes": strings.Join(attrs, ","),
		},
	})
}

func (c *esClient) clearAwarenessAttrs() error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.awareness.attributes": "",
		},
	})
}

func (c *esClient) disableShardRebalancing() error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.cluster_concurrent_rebalance": 0,
		},
	})
}

func (c *esClient) enableShardRebalancing() error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.cluster_concurrent_rebalance": 2,
		},
	})
}

func (c *esClient) disableShardAllocation() error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.enable": "new_primaries",
		},
	})
}

func (c *esClient) enableShardAllocation() error {
	return c.updateClusterSettings(&clusterSettings{
		Persistent: map[string]interface{}{
			"cluster.routing.allocation.enable": "all",
		},
	})
}

func (c *esClient) flushTranslog() error {
	return c.post("_flush/synced", nil)
}
