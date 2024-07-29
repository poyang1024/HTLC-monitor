package activity

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func MonitorTetherContractActivity(ctx context.Context, tetherContractAddress string) error {
	log.Println("開始監控 Tether 合約活動", "address", tetherContractAddress)

	client, err := ethclient.Dial("https://eth.public-rpc.com")
	if err != nil {
		return err
	}

	address := common.HexToAddress(tetherContractAddress)
	transferTopic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				log.Println("獲取當前區塊失敗:", err)
				continue
			}

			if currentBlock > latestBlock {
				query := ethereum.FilterQuery{
					FromBlock: big.NewInt(int64(latestBlock + 1)),
					ToBlock:   big.NewInt(int64(currentBlock)),
					Addresses: []common.Address{address},
					Topics:    [][]common.Hash{{transferTopic}},
				}

				logs, err := client.FilterLogs(ctx, query)
				if err != nil {
					log.Println("過濾日誌失敗:", err)
					continue
				}

				for _, vLog := range logs {
					from := common.HexToAddress(vLog.Topics[1].Hex())
					to := common.HexToAddress(vLog.Topics[2].Hex())
					amount := new(big.Int).SetBytes(vLog.Data)
					log.Printf("Tether 轉賬事件: 從 %s 到 %s，金額: %s USDT\n", from.Hex(), to.Hex(), amount.String())
				}

				latestBlock = currentBlock
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}