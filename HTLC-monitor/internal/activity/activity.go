package activity

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.temporal.io/sdk/activity"
)

const assetChainABI = `[
    {"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint256","name":"Id","type":"uint256"},{"indexed":false,"internalType":"bool","name":"assetIncepted","type":"bool"}],"name":"AssetIncepted","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint256","name":"Id","type":"uint256"},{"indexed":false,"internalType":"bool","name":"assetConfirmed","type":"bool"}],"name":"AssetConfirmed","type":"event"}
]`

func MonitorContractActivity(ctx context.Context, contractAddress string, txHash string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("開始監控 AssetChain 合約活動", "地址", contractAddress, "起始交易哈希", txHash)

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(dialCtx, "https://rpc.sepolia.org")
	if err != nil {
		return fmt.Errorf("連接到 Sepolia 網絡失敗: %v", err)
	}
	defer client.Close()

	address := common.HexToAddress(contractAddress)
	hash := common.HexToHash(txHash)

	// 獲取交易收據
	receiptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	receipt, err := client.TransactionReceipt(receiptCtx, hash)
	if err != nil {
		return fmt.Errorf("獲取交易收據失敗: %v", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("交易 %s 執行失敗", txHash)
	}

	logger.Info("交易已確認", "區塊", receipt.BlockNumber.Uint64())

	contractAbi, err := abi.JSON(strings.NewReader(assetChainABI))
	if err != nil {
		return fmt.Errorf("解析 ABI 失敗: %v", err)
	}

	assetInceptedSig := []byte("AssetIncepted(uint256,bool)")
	assetConfirmedSig := []byte("AssetConfirmed(uint256,bool)")
	logAssetInceptedSig := crypto.Keccak256Hash(assetInceptedSig)
	logAssetConfirmedSig := crypto.Keccak256Hash(assetConfirmedSig)

	logger.Info("事件簽名", "AssetIncepted", logAssetInceptedSig.Hex(), "AssetConfirmed", logAssetConfirmedSig.Hex())

	// 從交易所在的區塊開始監控
	startBlock := receipt.BlockNumber.Uint64()
	latestBlock := startBlock

	logger.Info("開始監控", "起始區塊", startBlock)

	// 首先檢查起始交易是否觸發了事件
	if err := checkTransactionLogs(receipt, contractAbi, logAssetInceptedSig, logAssetConfirmedSig); err != nil {
		logger.Error("檢查起始交易日誌時發生錯誤", "error", err)
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, latestBlock)

			blockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			currentBlock, err := client.BlockNumber(blockCtx)
			cancel()
			if err != nil {
				logger.Error("獲取當前區塊失敗", "error", err)
				continue
			}

			logger.Info("檢查新區塊", "當前區塊", currentBlock, "上次檢查的區塊", latestBlock)

			if currentBlock > latestBlock {
				err := filterLogs(ctx, client, contractAbi, address, latestBlock+1, currentBlock, logAssetInceptedSig, logAssetConfirmedSig)
				if err != nil {
					logger.Error("過濾日誌時發生錯誤", "error", err)
					continue
				}

				latestBlock = currentBlock
				err = saveLastCheckedBlock(contractAddress, latestBlock)
				if err != nil {
					logger.Error("保存最後檢查的區塊失敗", "error", err)
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func checkTransactionLogs(receipt *types.Receipt, contractAbi abi.ABI, logAssetInceptedSig, logAssetConfirmedSig common.Hash) error {
	for _, vLog := range receipt.Logs {
		if vLog.Topics[0] == logAssetInceptedSig || vLog.Topics[0] == logAssetConfirmedSig {
			log.Printf("在起始交易中檢測到事件: 區塊號: %d, 交易哈希: %s, 主題[0]: %s\n", 
				vLog.BlockNumber, vLog.TxHash.Hex(), vLog.Topics[0].Hex())
			
			switch vLog.Topics[0] {
			case logAssetInceptedSig:
				event := struct {
					Id          *big.Int
					AssetIncepted bool
				}{}
				err := contractAbi.UnpackIntoInterface(&event, "AssetIncepted", vLog.Data)
				if err != nil {
					log.Printf("解析 AssetIncepted 事件失敗: %v\n", err)
					log.Printf("事件數據: %x\n", vLog.Data)
					continue
				}
				log.Printf("檢測到 AssetIncepted 事件: Id %s, AssetIncepted %v\n", event.Id.String(), event.AssetIncepted)

			case logAssetConfirmedSig:
				event := struct {
					Id             *big.Int
					AssetConfirmed bool
				}{}
				err := contractAbi.UnpackIntoInterface(&event, "AssetConfirmed", vLog.Data)
				if err != nil {
					log.Printf("解析 AssetConfirmed 事件失敗: %v\n", err)
					log.Printf("事件數據: %x\n", vLog.Data)
					continue
				}
				log.Printf("檢測到 AssetConfirmed 事件: Id %s, AssetConfirmed %v\n", event.Id.String(), event.AssetConfirmed)
			}
		}
	}
	return nil
}

func filterLogs(ctx context.Context, client *ethclient.Client, contractAbi abi.ABI, address common.Address, fromBlock, toBlock uint64, logAssetInceptedSig, logAssetConfirmedSig common.Hash) error {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{address},
		Topics: [][]common.Hash{{
			logAssetInceptedSig,
			logAssetConfirmedSig,
		}},
	}

	filterCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	logs, err := client.FilterLogs(filterCtx, query)
	if err != nil {
		return fmt.Errorf("過濾日誌失敗: %v", err)
	}

	log.Printf("查詢到 %d 條日誌\n", len(logs))

	for _, vLog := range logs {
		log.Printf("處理日誌: 區塊號: %d, 交易哈希: %s, 主題[0]: %s\n", 
			vLog.BlockNumber, vLog.TxHash.Hex(), vLog.Topics[0].Hex())

		switch vLog.Topics[0].Hex() {
		case logAssetInceptedSig.Hex():
			event := struct {
				Id            *big.Int
				AssetIncepted bool
			}{}
			err := contractAbi.UnpackIntoInterface(&event, "AssetIncepted", vLog.Data)
			if err != nil {
				log.Printf("解析 AssetIncepted 事件失敗: %v\n", err)
				log.Printf("事件數據: %x\n", vLog.Data)
				continue
			}
			log.Printf("檢測到 AssetIncepted 事件: Id %s, AssetIncepted %v\n", event.Id.String(), event.AssetIncepted)

		case logAssetConfirmedSig.Hex():
			event := struct {
				Id             *big.Int
				AssetConfirmed bool
			}{}
			err := contractAbi.UnpackIntoInterface(&event, "AssetConfirmed", vLog.Data)
			if err != nil {
				log.Printf("解析 AssetConfirmed 事件失敗: %v\n", err)
				log.Printf("事件數據: %x\n", vLog.Data)
				continue
			}
			log.Printf("檢測到 AssetConfirmed 事件: Id %s, AssetConfirmed %v\n", event.Id.String(), event.AssetConfirmed)
		
		default:
			log.Printf("未知的事件類型: %s\n", vLog.Topics[0].Hex())
		}
	}

	return nil
}

func saveLastCheckedBlock(contractAddress string, blockNumber uint64) error {
	// 這裡應該實現將最後檢查的區塊號保存到持久化存儲的邏輯
	// 為了示例，這裡只是打印一條日誌
	log.Printf("保存最後檢查的區塊號: %d for 合約 %s", blockNumber, contractAddress)
	return nil
}

func StopMonitoring(ctx context.Context, contractAddress string) error {
	// 這裡可以實現任何需要在停止監控時執行的清理邏輯
	log.Printf("停止監控合約 %s", contractAddress)
	return nil
}