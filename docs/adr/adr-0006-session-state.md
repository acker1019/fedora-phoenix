# ADR 0006: Session State Management

> **Project:** Fedora Phoenix
> **Status:** ✅ Accepted
> **Date:** 2025-12-27
> **Refers to:** [ADR 0002](./adr-0002-block-architecture.md) (Block Architecture)

---

## 📋 Context (背景)

在 Provisioning 執行流程中，許多資訊需要在不同 Block 之間共享：

### 共享資料需求

| 資料類型 | 發現位置 | 使用位置 | 範例 |
|---------|---------|---------|------|
| **Configuration** | Block I (開頭) | 所有 Blocks | Blueprint, Secrets |
| **User Identity** | Block I (開頭) | Block III, IV | Username, UID, GID, Home |
| **Infrastructure State** | Block II (LUKS) | Block IV (Path Expansion) | Mount Point, Mapper Name |
| **Expanded Paths** | Block IV (Path Expansion) | Block IV (後續操作) | Stow Source/Target Dirs |
| **Temporary Variables** | Flags | Block IV | Dotfiles Archive Path |

### 傳統解決方案的問題

| 方式 | 問題 |
|------|------|
| **多個局部變數** | 變數散落各處，難以追蹤完整狀態 |
| **全域變數** | 污染全域空間，測試困難，隱式依賴 |
| **函數參數傳遞** | 導致函數簽名過長，耦合度高 |
| **依賴注入 (DI)** | 需要 Container，增加複雜度 |
| **`map[string]any`** | 無型別檢查，容易拼錯 key，IDE 無法自動補全 |

---

## 🎯 Decision (決策)

實作一個 **Local Session Instance**，將所有執行期狀態集中管理，並透過明確的參數傳遞給需要的 Acts。

### 1. 架構設計：Local Instance with Explicit Passing

```go
// internal/session/session.go
package session

import "github.com/acker1019/fedora-phoenix/internal/config"

// Session holds all runtime state for a single provision execution.
// This is created locally in runProvision() and passed to Acts as needed.
type Session struct {
    // Configuration (loaded from files)
    Blueprint *config.Blueprint
    Secrets   *config.Secrets

    // User Identity (discovered at runtime)
    Username string
    UID      int
    GID      int
    UserHome string

    // Infrastructure State (from Block II)
    LuksMapperName string
    LuksMountPoint string
    LuksUnlocked   bool
    LuksMounted    bool

    // Expanded Paths (from Block IV)
    StowSourceDir string
    StowTargetDir string

    // Temporary Variables
    DotfilesArchive string
}
```

### 2. 使用方式：創建 Local Instance

```go
// internal/cmd/provision.go
func runProvision() {
    // Create session instance
    sess := &session.Session{}

    // Populate fields as data is discovered
    sess.Username, sess.UID, sess.GID, _ = utils.GetRealUser()
    sess.UserHome, _ = utils.EnsureUserHome(sess.Username, sess.UID, sess.GID)

    sess.Blueprint, _ = config.LoadBlueprint(blueprintPath)
    sess.Secrets, _ = config.LoadSecrets(secretsPath)

    sess.DotfilesArchive = dotfilesArchive

    // Pass to Acts as needed (future enhancement)
    // ops.SomeAct(sess)
}
```

### 3. 核心原則

| 原則 | 說明 | 範例 |
|------|------|------|
| **Explicit Declaration** | 所有欄位明確定義，不使用 `map[string]any` | `Username string` ✅ / `data["username"]` ❌ |
| **Public Fields** | 直接存取成員，不需要 Getter/Setter | `sess.UID` ✅ / `sess.GetUID()` ❌ |
| **Local Instance** | 在 `runProvision()` 中創建，不是全域變數 | `sess := &session.Session{}` ✅ |
| **Explicit Passing** | 未來可選擇性傳給需要的 Acts | `ops.SomeAct(sess)` (可選) |
| **Type Safety** | 編譯期型別檢查，IDE 自動補全 | Go struct 天然支援 |

---

## 💡 Usage Patterns (使用模式)

### Pattern 1: 初始化 Session (runProvision 開頭)

```go
// internal/cmd/provision.go
func runProvision() {
    // Create local session instance
    sess := &session.Session{}

    // All state will be stored here
}
```

### Pattern 2: 填充 User Identity (Block I)

```go
// 3. Real User Detection
realUser, realUID, realGID, err := utils.GetRealUser()
if err != nil {
    panic(err)
}
sess.Username = realUser
sess.UID = realUID
sess.GID = realGID
sess.UserHome, err = utils.EnsureUserHome(sess.Username, sess.UID, sess.GID)
```

### Pattern 3: 填充 Configuration (Block I)

```go
// Load Blueprint and Secrets
sess.Blueprint, err = config.LoadBlueprint(blueprintPath)
sess.Secrets, err = config.LoadSecrets(secretsPath)
sess.DotfilesArchive = dotfilesArchive
```

### Pattern 4: 使用 Session 中的資料 (Block II)

```go
// Store infrastructure info
sess.LuksMapperName = sess.Blueprint.Infrastructure.Luks.MapperName
sess.LuksMountPoint = sess.Blueprint.Infrastructure.Luks.MountPoint

// Use session data
err = ops.UnlockLuks(
    sess.Blueprint.Infrastructure.Luks.Device,
    sess.LuksMapperName,
    sess.Secrets.LuksPassword,
)
sess.LuksUnlocked = true
```

### Pattern 5: 路徑展開 (Block IV)

```go
// Expand paths using session data
sess.StowSourceDir = utils.ExpandPath(
    sess.Blueprint.UserSpace.Stow.SourceDir,
    sess.UserHome,
)
sess.StowTargetDir = utils.ExpandPath(
    sess.Blueprint.UserSpace.Stow.TargetDir,
    sess.UserHome,
)
```

---

## 🔄 Lifecycle (生命週期)

```text
┌─────────────────────────────────────────────────────────────┐
│ runProvision() starts                                       │
│ sess := &session.Session{}                                  │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ User Detection                                              │
│ • sess.Username = utils.GetRealUser()                       │
│ • sess.UID = ...                                            │
│ • sess.GID = ...                                            │
│ • sess.UserHome = utils.EnsureUserHome(...)                 │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Block I: Load Configuration                                 │
│ • sess.Blueprint = config.LoadBlueprint(...)                │
│ • sess.Secrets = config.LoadSecrets(...)                    │
│ • sess.DotfilesArchive = dotfilesArchive                    │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Block II: Infrastructure                                    │
│ • sess.LuksMapperName = sess.Blueprint...                   │
│ • sess.LuksMountPoint = sess.Blueprint...                   │
│ • ops.UnlockLuks(..., sess.Secrets.LuksPassword)            │
│ • sess.LuksUnlocked = true                                  │
│ • ops.MountDevice(...)                                      │
│ • sess.LuksMounted = true                                   │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Block III: System State                                     │
│ • 使用 sess.Blueprint.System.* 資料                          │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Block IV: User Space                                        │
│ • sess.StowSourceDir = utils.ExpandPath(...)                │
│ • sess.StowTargetDir = utils.ExpandPath(...)                │
│ • 使用 sess.* 資料執行所有操作                                │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ runProvision() ends                                         │
│ sess 自動釋放 (Go stack unwinding)                           │
└─────────────────────────────────────────────────────────────┘
```

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **集中管理** | 所有執行期狀態集中在一個結構中，易於追蹤 |
| **型別安全** | 編譯期檢查，IDE 自動補全 |
| **可追蹤性** | 明確知道有哪些欄位，Find All References 可直接找到使用位置 |
| **無全域污染** | Local instance，不污染全域命名空間 |
| **測試友善** | 容易建立測試用的 Session instance |
| **明確資料流** | 透過 `sess.*` 清楚看到資料來源 |

### ❌ 負面影響 (Cons)

| 劣勢 | 說明 | 緩解措施 |
|------|------|----------|
| **需要明確傳遞** | 未來若 Acts 需要 Session，需要修改簽名 | 目前 Acts 不需要，保持簡單 |
| **程式碼量** | 需要寫 `sess.` 前綴 | 換來明確性，值得 |

---

## 📝 Implementation Guidelines

### Rule 1: Session 僅存在於 runProvision()

```go
// ✅ Good: Local instance in runProvision
func runProvision() {
    sess := &session.Session{}
    // Use sess throughout this function
}

// ❌ Bad: Global variable
var globalSession *session.Session  // 不要這樣做
```

### Rule 2: 只儲存跨 Block 共享的資料

```go
// ✅ Good: 跨 Block 使用的資料
sess.UserHome = "/home/ack"
sess.Blueprint = blueprint

// ❌ Bad: 僅在局部使用的臨時變數
tempList := []string{...}  // 不要加入 Session
```

### Rule 3: 優先使用 sess.Field，而非局部變數

```go
// ✅ Good: 使用 Session 欄位
sess.StowSourceDir = utils.ExpandPath(...)
ops.RunStow(sess.StowSourceDir, ...)

// ❌ Bad: 創建重複的局部變數
stowSourceDir := utils.ExpandPath(...)  // 與 sess.StowSourceDir 重複
```

### Rule 4: 按照資料發現順序填充

```go
// ✅ Good: 按照執行順序填充
sess.Username = ...        // 最早發現
sess.UserHome = ...        // 接著發現
sess.Blueprint = ...       // Block I
sess.LuksUnlocked = true   // Block II 完成後

// ❌ Bad: 提前填充未知資料
sess.LuksUnlocked = false  // 不需要初始化為 false (zero value)
```

---

## 🔗 References

- [ADR 0002: Block Architecture](./adr-0002-block-architecture.md) - 定義了四個 Block 的執行流程
- [Go Best Practices: Pass by Value](https://go.dev/doc/effective_go#pointers_vs_values) - 何時使用指標
