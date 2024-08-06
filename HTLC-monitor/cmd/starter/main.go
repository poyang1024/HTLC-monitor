package main

import (
	"context"
	"log"
	"HTLC-monitor/internal/workflow"

	"go.temporal.io/sdk/client"
)

func main() {
	c, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("無法創建 Temporal 客戶端:", err)
	}
	defer c.Close()

	// Sepolia 測試網上的 HTLC 合約地址
	HTLCContractAddress := "0x404e1cf018c9974F738644847c6844ebAC8aDe89"
	
	// 起始區塊號碼，這裡設置為 0，表示從最新的區塊開始監控
	// 如果您想從特定區塊開始監控，請將 0 替換為所需的區塊號碼
	startBlock := int64(0)
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        "htlc-monitor-workflow",
		TaskQueue: "htlc-monitor-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflow.HTLCMonitorWorkflow, HTLCContractAddress, startBlock)
	if err != nil {
		log.Fatalln("無法啟動工作流程:", err)
	}

	log.Println("啟動 HTLC 監控工作流程成功。WorkflowID:", we.GetID(), "RunID:", we.GetRunID())
}