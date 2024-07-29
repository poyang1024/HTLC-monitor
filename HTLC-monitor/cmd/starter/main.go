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

	// USDT 合約地址（以太坊主網）
	tetherContractAddress := "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        "tether-monitor-workflow",
		TaskQueue: "tether-monitor-task-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflow.TetherMonitorWorkflow, tetherContractAddress)
	if err != nil {
		log.Fatalln("無法啟動工作流程:", err)
	}

	log.Println("啟動 Tether 監控工作流程成功。WorkflowID:", we.GetID(), "RunID:", we.GetRunID())
}