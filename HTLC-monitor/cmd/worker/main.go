package main

import (
	"log"
	"HTLC-monitor/internal/workflow"
	"HTLC-monitor/internal/activity"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	c, err := client.NewClient(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("無法創建 Temporal 客戶端:", err)
	}
	defer c.Close()

	w := worker.New(c, "tether-monitor-task-queue", worker.Options{})

	w.RegisterWorkflow(workflow.TetherMonitorWorkflow)
	w.RegisterActivity(activity.MonitorContractActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("無法啟動 worker:", err)
	}
}