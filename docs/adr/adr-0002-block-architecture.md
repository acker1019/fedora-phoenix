# ADR 0002: Block Architecture

> **Project:** Fedora Phoenix
> **Status:** ✅ Accepted (Revised)
> **Date:** 2025-12-26
> **Refers to:** [ADR 0001](./adr-0001-pure-go-strategy.md) (Pure Go Strategy)

---

## 📝 前言

這份文檔將作為我們架構的「地基圖」，定義了這個專案的各個**功能區塊 (Functional Blocks)** 及其邊界。之後丟給 Claude 寫 code 時，他只要依照這個地圖填空即可。

---

## 📋 Context (背景)

我們確立了使用單一 Go Binary 進行系統還原（參見 ADR 0001）。

### 面臨的挑戰

1. **避免巨型腳本**：為了避免 `main.go` 變成無法維護的巨型腳本，我們需要將還原邏輯進行功能性分塊 (Functional Partitioning)
2. **孵化期設計原則**：目前處於「孵化期 (Incubation Phase)」，設計必須保持簡單直觀
3. **拒絕過度設計**：避免複雜的 Plugin 系統或 Dependency Injection

---

## 🎯 Decision (決策)

我們採用 **「線性分層區塊 (Linear Layered Blocks)」** 架構。

整個還原過程被視為一條單向的生產線，由四個獨立的邏輯區塊組成。每個區塊負責特定的領域 (Domain)，並且有明確的先後依賴關係。

---

## 🏗️ 架構設計

### 1. 架構圖 (Architecture Diagram)

```text
┌─────────────────────────────────────────────────────────┐
│                    phoenix provision                    │
│                    (Cobra Command)                      │
└───────────────────────┬─────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
        ▼               ▼               ▼
┌───────────┐    ┌──────────────┐    ┌───────────┐    ┌─────────┐
│  Block I  │──▶│   Block II   │──▶│ Block III │──▶│Block IV │
│ Identity  │    │Infrastructure│    │  System   │    │  User   │
└───────────┘    └──────────────┘    └───────────┘    └─────────┘
```

### 2. 四大功能區塊 (The Four Blocks)

我們將 `internal/ops` 內部的原子操作 (Acts) 歸類為以下四個高層次模組：

---

#### 🔐 Block I: Identity & Credentials (身分與憑證)

| 屬性 | 說明 |
|------|------|
| **職責** | 負責「你是誰」與「你的鑰匙」 |
| **輸入** | 外部 YAML 檔案 (via CLI Flag) |
| **行為** | 解析設定、驗證結構、將敏感資訊載入記憶體、銷毀硬碟上的暫存檔 |
| **關鍵產出** | `SecretsBook` (Struct) |

**實作位置**: `internal/config/secrets.go`

---

#### 🔧 Block II: Infrastructure & Security (基建與安全)

| 屬性 | 說明 |
|------|------|
| **職責** | 負責「硬碟與資料存取」（最底層的物理操作） |
| **行為** | 與 Kernel Crypto API 互動 (LUKS Unlock)、掛載檔案系統 (Mount)、確保硬體層級的存取權 (e.g., Fprintd device) |
| **特性** | ⚠️ **失敗即致命 (Fatal)**：此塊失敗則後續無法進行 |

**實作位置**: `internal/ops/infra.go` (待建立)

---

#### ⚙️ Block III: System State (系統狀態)

| 屬性 | 說明 |
|------|------|
| **職責** | 負責「OS 本身的環境」（不包含個人資料的純淨系統層） |
| **行為** | 套件管理 (Package Management / DNF)、服務管理 (Service Management / Systemd)、全域設定 (Timezone, Hostname, NetworkManager) |
| **特性** | ✨ **冪等性 (Idempotency)** 的核心區域 |

**實作位置**: `internal/ops/system.go` (待建立)

---

#### 👤 Block IV: User Space & Data (用戶空間與資料)

| 屬性 | 說明 |
|------|------|
| **職責** | 負責「讓系統變成你的」（將 Block II 的資料與 Block III 的環境結合） |
| **行為** | 建立連結 (Symlinking)：將 Mount point 的資料映射回 `$HOME`、權限修復 (Permission Fix)、個人化 (Shell change, Dotfiles injection) |
| **權限策略** | 🛡️ **嚴格禁止以 Root 身份產生檔案** |

**實作位置**: `internal/ops/user.go` (待建立)

---

## 🔑 執行策略 (Execution Strategy)

### 權限模型：「上帝視角，凡人代理」

> **核心理念**: "Correctness over Shortcuts"（正確性優於捷徑）

我們採用 **"Impersonation" (冒充/代理)** 模式：

#### 策略說明

| 組件 | 權限 | 說明 |
|------|------|------|
| **Main Process** | Root (sudo) | 握有最高權力，負責 Block I, II, III |
| **Context Detection** | - | 啟動時透過 `os.Getenv("SUDO_USER")` 獲取目標使用者資訊 (UID, GID, HomeDir) |
| **User Acts (Block IV)** | User | 透過 `syscall.SysProcAttr` 或 `os.Chown` 執行 |

#### 精細化執行 (Granular Execution)

```go
// Block II & III: 直接以 Root 執行
ops.UnlockLuks(...)
ops.MountDevice(...)
ops.EnsurePackages(...)

// Block IV: 必須以 User 身份執行
ops.RunAsUser(uid, gid, func() {
    exec.Command("git", "clone", ...).Run()
})

// 或者 Root 建立後立即 chown
os.Symlink(...)
os.Chown(path, uid, gid)  // 原子性修復權限
```

### 模組化入口設計

雖然目前是線性執行，但每個 Block 應設計為獨立函數：

```go
ops.ProvisionIdentity(secretsPath)  // Block I
ops.ProvisionInfra(secrets)          // Block II
ops.ProvisionSystem()                // Block III
ops.ProvisionUser(userCtx)           // Block IV
```

**未來擴展**: 透過 Cobra Flags (e.g., `--skip-infra`, `--only-user`) 即可輕鬆調度，無需重構核心邏輯。

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **清晰的關注點分離** | 每個 Block 職責明確，易於測試與維護 |
| **防禦性程式設計** | 在操作當下就確保權限正確，而非事後補救 |
| **模組化可擴展** | 未來可輕鬆支援部分執行模式 |
| **孵化期友善** | 架構簡單直觀，沒有過度設計 |

### ❌ 負面影響 (Cons)

| 風險 | 說明 | 緩解措施 |
|------|------|----------|
| **權限髒檔案風險** | 如果在 `chown` 執行前程式崩潰，可能留下 root 權限的檔案 | 採用原子性操作，盡早執行 `chown` |
| **目前缺乏部分執行** | 不支援「只跑 Block III」這種模式 | 在單一 Binary 還原場景下可接受 |

---

## 📋 Action Items (下一步)

- [ ] 依照四大區塊，將 `internal/ops` 程式碼進行歸類整理
  - `ops/identity.go` → Block I (或使用現有的 `config/secrets.go`)
  - `ops/infra.go` → Block II
  - `ops/system.go` → Block III
  - `ops/user.go` → Block IV
- [ ] 實作 `RunAsUser` helper function
- [ ] 實作 Block IV 中關鍵的 Permission Fix 邏輯
- [ ] 為每個 Block 設計獨立的入口函數
