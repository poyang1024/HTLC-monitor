package activity

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func MonitorContractActivity(ctx context.Context, contractAddress string, eventSignature string) error {
	log.Println("開始監控合約活動", "address", contractAddress)

	client, err := ethclient.Dial("https://rpc.sepolia.org")
	if err != nil {
		return fmt.Errorf("連接到 Sepolia 網絡失敗: %v", err)
	}

	address := common.HexToAddress(contractAddress)
	eventTopic := common.HexToHash(eventSignature)

	// 獲取合約創建時間
	deploymentTime, err := getContractDeploymentTime(ctx, client, address)
	if err != nil {
		return fmt.Errorf("獲取合約部署時間失敗: %v", err)
	}
	log.Printf("合約部署時間: %v\n", deploymentTime)

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("獲取最新區塊號碼失敗: %v", err)
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
					Topics:    [][]common.Hash{{eventTopic}},
				}

				logs, err := client.FilterLogs(ctx, query)
				if err != nil {
					log.Println("過濾日誌失敗:", err)
					continue
				}

				for _, vLog := range logs {
					log.Printf("檢測到事件: 區塊號 %d, 交易哈希 %s\n", vLog.BlockNumber, vLog.TxHash.Hex())
					// 在這裡可以添加更多的事件處理邏輯
				}

				latestBlock = currentBlock
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func getContractDeploymentTime(ctx context.Context, client *ethclient.Client, address common.Address) (time.Time, error) {
	// 從最新的區塊向前搜索
	latestBlock, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		return time.Time{}, err
	}

	for blockNumber := latestBlock.NumberU64(); blockNumber > 0; blockNumber-- {
		block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return time.Time{}, err
		}

		for _, tx := range block.Transactions() {
			if tx.To() == nil { // 合約創建交易
				receipt, err := client.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					continue
				}
				if receipt.ContractAddress == address {
					return time.Unix(int64(block.Time()), 0), nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("無法找到合約部署交易")
}