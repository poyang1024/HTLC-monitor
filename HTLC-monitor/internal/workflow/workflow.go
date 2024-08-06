package workflow

import (
	"time"

	"HTLC-monitor/internal/activity"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func HTLCMonitorWorkflow(ctx workflow.Context, contractAddress string, startBlock int64) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("開始 HTLC 監控工作流程", "contractAddress", contractAddress)

	// 設置活動選項
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    0, // 無限重試
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 創建一個通道來接收停止信號
	stopChan := workflow.GetSignalChannel(ctx, "stop")

	var lastCheckedBlock int64
	err := workflow.ExecuteActivity(ctx, activity.GetLastCheckedBlock, contractAddress).Get(ctx, &lastCheckedBlock)
	if err != nil {
		logger.Error("獲取最後檢查的區塊失敗", "error", err)
		lastCheckedBlock = startBlock
	}

	// 定期保存合約數據
	saveDataTimer := workflow.NewTimer(ctx, 5*time.Minute)

	// 啟動監控活動
	future := workflow.ExecuteActivity(ctx, activity.MonitorContractActivity, contractAddress, lastCheckedBlock)

	// 等待活動完成、收到停止信號或定時器觸發
	for {
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(future, func(f workflow.Future) {
			err := f.Get(ctx, nil)
			if err != nil {
				logger.Error("監控活動失敗", "error", err)
			}
			// return
		})
		selector.AddReceive(stopChan, func(c workflow.ReceiveChannel, more bool) {
			logger.Info("收到停止信號，正在停止監控活動")
			workflow.ExecuteActivity(ctx, activity.StopMonitoring, contractAddress)
			// return
		})
		selector.AddFuture(saveDataTimer, func(f workflow.Future) {
			// 保存合約數據
			contractData := activity.ContractData{
				ContractAddress:  contractAddress,
				LastCheckedBlock: lastCheckedBlock,
				Timestamp:        workflow.Now(ctx),
			}
			err := workflow.ExecuteActivity(ctx, activity.SaveContractDataToJSON, contractData).Get(ctx, nil)
			if err != nil {
				logger.Error("保存合約數據失敗", "error", err)
			}
			// 重置定時器
			saveDataTimer = workflow.NewTimer(ctx, 5*time.Minute)
		})

		selector.Select(ctx)

		// 檢查是否應該退出循環
		if ctx.Err() != nil {
			break
		}
	}

	logger.Info("HTLC 監控工作流程結束")
	return nil
}