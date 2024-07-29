package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
	"HTLC-monitor/internal/activity"
)

func TetherMonitorWorkflow(ctx workflow.Context, tetherContractAddress string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("開始監控 Tether 合約", "address", tetherContractAddress)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 24 * time.Hour, // 增加超時時間，因為我們可能需要長時間監控
		HeartbeatTimeout:    2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	err := workflow.ExecuteActivity(ctx, activity.MonitorTetherContractActivity, tetherContractAddress).Get(ctx, nil)
	if err != nil {
		logger.Error("Tether 監控活動失敗", "error", err)
		return err
	}

	logger.Info("Tether 合約監控完成")
	return nil
}