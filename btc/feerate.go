package btc

import (
	"math"

	"github.com/Multy-io/Multy-back/store"
)

const (
	btcToSatoshi = 100000000
)

func (c *Client) GetAllMempool() ([]store.MempoolRecord, error) {
	allMempool := []store.MempoolRecord{}
	mempool, err := c.RpcClient.GetRawMempoolVerbose()
	if err != nil {
		return allMempool, err
	}
	log.Errorf("MEMPOOL SIZE == %v", len(mempool))

	for hash, txInfo := range mempool {

		floatFee := txInfo.Fee / float64(txInfo.Size) * btcToSatoshi

		//It's some kind of Round function to prefent 0 FeeRates while casting from float to int
		intFee := int(math.Floor(floatFee + 0.5))
		// Node has transatctions withch not exist
		if intFee > 0 {
			allMempool = append(allMempool, newMempoolRecord(intFee, hash))
		}
	}
	return allMempool, err

}
