# HTLC 監控系統

這個專案使用 Temporal 工作流引擎來監控以太坊網絡上的 HTLC (Hash Time-Locked Contract) 交易事件。

## 前置需求

- Go 1.16 或更高版本
- Docker 和 Docker Compose (用於運行 Temporal 服務器)
- 以太坊節點訪問 (本例中使用 Sepolia 測試網絡)

## 專案結構
```
HTLC-monitor/
├── cmd/
│   ├── worker/
│   │   └── main.go
│   └── starter/
│       └── main.go
├── internal/
│   ├── workflow/
│   │   └── workflow.go
│   └── activity/
│       └── activity.go
├── go.sum 
└── go.mod

```

## 設置步驟

1. git clone repo:
```sh
git clone [你的存儲庫 URL]
cd HTLC-monitor
```

2. 安裝依賴:
```sh
go mod tidy
```
3. 啟動 Temporal 服務器:
```sh
cd docker-compose
docker-compose up -d
```
4. 在 `internal/activity/activity.go` 中更新 Sepolia 測試網絡 RPC 端點:
```go
client, err := ethclient.Dial("https://rpc.sepolia.org")
```
## 運行系統
0. 換到 HTLC-monitor 資料夾
```sh
cd HTLC-monitor
```
1. 在一個終端中啟動 worker:
```go
go run cmd/worker/main.go
```

2. 在另一個終端中啟動工作流:
```go
go run cmd/starter/main.go
```

## 監控和管理

訪問 Temporal Web UI: http://localhost:8080
在 UI 中，你可以查看工作流狀態、歷史和日誌
或透過 CLI 也可以看到回吐的 `log` 資訊

## 主要程式碼

1. Workflow ( `internal/workflow/workflow.go` ):

- 定義了 `HTLCMonitorWorkflow` 函數
- 處理活動的執行和錯誤重試


2. Activity ( `internal/activity/activity.go` ):

- 定義了 `MonitorContractActivity` 函數
- 實現了 HTLC 合約的監控邏輯


3. Worker ( `cmd/worker/main.go` ):

- 設置和運行 Temporal `worker`
- 註冊工作流和活動


4. Starter ( `cmd/starter/main.go` ):

- 啟動 HTLC 監控工作流



## HTLC 交易流程
```mermaid
graph TD
    %% 初始化階段
    A[1. 交易初始化]:::temporal --> B[2. 生成HTLC參數]:::temporal
    A --> |a| A1[交易系統接收請求]
    A --> |b| A2[新增交易記錄]:::db
    A --> |c| A3[有專屬 ID]:::db
    B --> |a| B1[生成交易專屬 secret]
    B --> |b| B2[建立交易開始狀態]:::db

    %% Asset Contract階段
    B --> C[3. 開始第一個 asset 交易指令 -> 賣家]:::blockchain
    C --> D[4. 監控 asset 合約]:::temporal
    C --> |a| C1[賣家簽署]:::blockchain
    C --> |b| C2[更新狀態記錄 -> asset contract 賣家執行中]:::db
    D --> |a| D1[鏈上驗證及確認合約，並鎖住 asset]:::blockchain
    D1 --> |a| D3[平台將 H（a） 傳給買家]
    D1 --> |b| D4[紀錄 asset 已鎖住的事件並更新狀態]:::db
    D3 --> E[5.買家確認 asset 狀況，並選擇創建 payment 合約]
    D --> |b| D2[更新狀態 -> 賣家 init 合約已確認]:::db
    E --> |b| E2[更新狀態 -> 買家已收到 H（a）]:::db

    %% Payment Contract階段
    E --> F[6. 開始進行 payment 合約]:::blockchain
    F --> G[7. 監控 payment 合約]:::temporal
    G1 --> H[8. 買家驗證第二個payment指令]:::temporal
    F --> |a| F1[買家驗證和確認交易]:::blockchain
    F --> |b| F2[更新買家 init paymeny 合約狀態且 payment 已鎖住]:::db
    F1 --> |a| G1[賣家驗證 payment 狀態，並 comfirm payment 交易]:::blockchain
    G1 --> |b| G2[資料庫更新狀態 -> 賣家已驗證交易]:::db
    H --> |a| H1[接收到 secret 並成功支付]:::blockchain
    H --> |b| H2[更新紀錄為 payment complete]:::db

    %% 完成階段
    H --> I[9. 創建 asset comfirm -> 買家]:::blockchain
    I --> J[10.監控完成 asset 交付]:::temporal
    J --> K[11. 最終確認]:::temporal
    K --> L[12. 通知參與方]:::temporal
    L --> M[13. 清理和歸檔]:::temporal
    I --> |b| I2[更新買家已經 comfirm asset 狀態]:::db
    J --> |a| J1[執行指令]:::blockchain
    J --> |b| J2[確認完成]:::blockchain
    J --> |c| J3[更新狀態]:::db
    K --> |a| K1[驗證步驟]
    K --> |b| K2[最終更新]:::db
    L --> |a| L1[發送通知]
    L --> |b| L2[記錄狀態]:::db
    M --> |a| M1[移動數據]:::db
    M --> |b| M2[清理數據]:::db

    %% 階段標籤
    Init[初始化階段]
    Asset[Asset Contract階段]
    Payment[Payment Contract階段]
    Finish[完成階段]

    %% 階段連接
    Init -.-> A
    Asset -.-> C
    Payment -.-> F
    Finish -.-> I

    classDef db fill:#afd,stroke:#6a6,stroke-width:2px;
    classDef blockchain fill:#fad,stroke:#a66,stroke-width:2px;
    classDef temporal fill:#FFD1A4,stroke:#66a,stroke-width:2px;
    classDef label fill:none,stroke:none;
    
    class Init,Asset,Payment,Finish label;

    style A fill:#f9f,stroke:#333,stroke-width:2px
    style J fill:#bfb,stroke:#333,stroke-width:2px
    style M fill:#fbb,stroke:#333,stroke-width:2px

    %% 圖例
    subgraph 圖例
    DB[寫入資料庫]:::db
    BC[區塊鏈執行合約]:::blockchain
    TM[Temporal監控]:::temporal
    end
```

## 故障排除

1. 如果 UI 顯示工作流已終止，但 CLI 仍在運行:

- 檢查 `MonitorContractActivity` 中的取消處理邏輯
- 確保活動正確響應 `ctx.Done()` 信號


2. 如果看不到詳細日誌:

- 使用 `activity.GetLogger(ctx)` 進行日誌記錄
- 調整 `worker` 的日誌級別


3. 如果活動持續失敗:

- 檢查 `Sepolia` 測試網絡 `RPC 端點` 的可用性
- 查看 `Temporal UI` 中的錯誤消息



## 擴展建議

1. 添加數據持久化:

- 將監控到的 HTLC 事件保存到數據庫中


2. 實現通知系統:

- 為關鍵 HTLC 階段添加警報功能


3. 優化性能:

- 調整輪詢間隔
- 實現更高效的事件過濾機制


4. 增加測試覆蓋:

- 為工作流和活動添加單元測試
- 使用 Temporal 的測試框架進行集成測試


5. 添加更多 HTLC 特定功能:

- 實現超時監控和自動退款機制
- 添加多鏈支持，擴展到其他支持 HTLC 的區塊鏈