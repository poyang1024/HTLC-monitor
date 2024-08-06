package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ContractData struct {
	ContractAddress  string    `json:"contractAddress"`
	LastCheckedBlock int64     `json:"lastCheckedBlock"`
	Timestamp        time.Time `json:"timestamp"`
}

func SaveContractDataToJSON(ctx context.Context, data ContractData) error {
	storageDir := "contract_data"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("創建存儲目錄失敗: %v", err)
	}

	filename := fmt.Sprintf("%s_%d.json", data.ContractAddress, time.Now().Unix())
	filepath := filepath.Join(storageDir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 編碼失敗: %v", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("寫入 JSON 文件失敗: %v", err)
	}

	return nil
}

func GetLastCheckedBlock(ctx context.Context, contractAddress string) (int64, error) {
	storageDir := "contract_data"
	files, err := os.ReadDir(storageDir)
	if err != nil {
		return 0, fmt.Errorf("讀取存儲目錄失敗: %v", err)
	}

	var latestFile os.DirEntry
	var latestTime int64

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		if filepath.Base(file.Name())[:len(contractAddress)] != contractAddress {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		if fileInfo.ModTime().Unix() > latestTime {
			latestFile = file
			latestTime = fileInfo.ModTime().Unix()
		}
	}

	if latestFile == nil {
		return 0, fmt.Errorf("沒有找到合約 %s 的數據文件", contractAddress)
	}

	filePath := filepath.Join(storageDir, latestFile.Name())
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("讀取文件 %s 失敗: %v", filePath, err)
	}

	var data ContractData
	if err := json.Unmarshal(fileContent, &data); err != nil {
		return 0, fmt.Errorf("解析 JSON 文件 %s 失敗: %v", filePath, err)
	}

	return data.LastCheckedBlock, nil
}