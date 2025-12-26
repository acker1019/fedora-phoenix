# ADR 0007: Artifact Synchronization & Continuous Harvesting Strategy

> **Project:** Fedora Phoenix
> **Status:** ✅ Accepted
> **Date:** 2025-12-27
> **Refers to:**
> - [ADR 0003](./adr-0003-dotfiles-management.md) (Supersedes Stow Logic)
> - [ADR 0006](./adr-0006-session-state.md) (Uses Session for Paths)

---

## 📋 Context (背景)

在 ADR 0003 中，我們曾考慮使用 GNU Stow (Symlink) 來管理 Dotfiles。然而，經過深入分析與實務考量，Symlink 方案在「高韌性系統」的需求下存在明顯缺陷。

### Symlink 的局限性

| 問題 | 說明 |
|------|------|
| **軟體相容性 (Resilience)** | 部分應用程式（如 SSH, GPG, Snap/Flatpak apps）對 Symlink 支援度不佳，或對檔案屬性檢查極為嚴格，導致設定失效或程式崩潰 |
| **權限模糊 (Permission Ambiguity)** | Symlink 本身權限通常是 777，真實權限取決於 Target。這使得 Git Repo 中的檔案權限必須與系統運行的嚴格要求（如 0600）完全一致，這在協作或備份時不便 |
| **單向性** | Symlink 假設 Git Repo 永遠是真理，忽略了使用者在系統上直接修改（Running State）作為「新真理」的可能性 |

### 新需求：Harvesting (收割)

我們需要一種機制，能夠：

- **雙向同步**: 既能 Provision (Repo → System)，也能 Harvest (System → Repo)
- **精確權限**: 能夠捕捉並還原 Git 無法記錄的 Unix File Mode (e.g., 0600, 0640)
- **自動化**: 透過 Daemon 自動監控漂移 (Drift)，且不消耗過多系統資源

---

## 🎯 Decision (決策)

我們決定 **廢除 GNU Stow (Symlink) 策略**，改採 「**Master-Replica 實體複製**」 模式，並引入 Abstract State Store 與 Systemd Daemon。

### 1. Synchronization Strategy: The "Master-Replica" Pattern

```text
┌─────────────────────────────────────────────────────────────┐
│ Master: Git Repo (Artifact)                                 │
│ • Source of Truth for Content                               │
│ • Version Controlled                                        │
└────────────────┬────────────────────────────────────────────┘
                 │
                 │ Physical Copy
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Replica: System Home Directory                              │
│ • Applications read REAL files (not symlinks)               │
│ • Users can modify directly                                 │
└─────────────────────────────────────────────────────────────┘
```

**Method**: Physical File Copy (實體複製)。應用程式讀寫的是真實存在於 Home 目錄下的檔案，而非連結。

---

### 2. Change Detection: Content Hash

為了避免 `touch` 或複製過程導致的時間戳 (ModTime) 變動觸發假警報，同步判定一律基於內容雜湊。

| 屬性 | 值 |
|------|---|
| **Algorithm** | SHA256 (或其他強雜湊演算法) |
| **Logic** | `if Hash(Src) != Hash(Dest) { Sync Required }` |

---

### 3. Permission Tracking: The Sidecar State Store

由於 Git 只能追蹤 `+x` (Exec bit)，無法紀錄 Owner/Group 或具體的 Read/Write bits，我們必須實作一個獨立的 **狀態儲存機制 (State Persistence Mechanism)**。

#### Requirement (需求定義)

| 需求 | 說明 |
|------|------|
| **Persistence** | 必須將每個受控檔案的 Metadata（包含 Permission Mode, Owner UID/GID, Content Hash）持久化儲存 |
| **Portability** | 該儲存媒體必須隨附於 Dotfiles Repo 中，作為還原時的「權限真理來源 (Source of Truth for Permissions)」|
| **Precision** | 必須能夠精確記錄八進位權限 (e.g., 0600, 0640) |

---

### 4. Daemonization: Systemd User Service

Phoenix 不自行實作 Process Forking，而是作為 Systemd 的控制器。

| 項目 | 值 |
|------|---|
| **Service Name** | `phoenix-harvest.service` (User Scope) |
| **Command** | `phoenix harvest --watch` |
| **Behavior** | 使用 `time.Ticker` 進行週期性輪詢 (Polling) |

---

## 💡 Implementation Guidelines (實作指引)

### 1. Provision Logic (Deploy)

**行為**:
將 Repo 檔案複製到 System 後，必須讀取 State Store 中的 Metadata，並強制執行 `chmod`/`chown` 套用紀錄中的權限。

**關鍵點**:
即使 Hash 一致，也必須驗證權限是否符合 State Store 的紀錄，防止使用者手動修改導致權限錯誤。

---

### 2. Harvest Logic (Collect)

**行為**:
1. 掃描 `phoenix.yml` 定義的檔案清單
2. 若 `System Hash != Repo Hash`，將檔案 **反向複製** 回 Repo
3. 若 `System Mode != State Store Mode`，更新 State Store 中的紀錄

**結果**:
Harvest 僅更新 Repo 中的檔案實體與 Metadata 紀錄，**不執行 Git Commit**。使用者需自行審核 `git status` 並提交。

---

### 3. Daemon Management

Phoenix Binary 需包含管理 Systemd Unit 的邏輯：

| 命令 | 功能 |
|------|------|
| `phoenix harvest --daemon` | 生成 Unit file 並 `enable --now` |
| `phoenix harvest --shutdown` | 停止服務並清理 Unit file |
| `phoenix harvest --check` | 檢查服務狀態 |

---

## 🖇️ Alternatives Considered (替代方案考量)

### ❌ Option 1: Inotify (Event-Driven Monitoring)

曾考慮使用 Linux 原生 `inotify` (透過 Go `fsnotify` 庫) 來實作即時檔案監控。

#### 拒絕原因 (Rejection Rationale)

| 問題 | 說明 |
|------|------|
| **資源競爭 (Resource Exhaustion)** | 開發者環境中已有大量工具 (VS Code, Webpack) 競爭 `fs.inotify.max_user_watches` 額度。Phoenix 若進行遞迴監控，極易耗盡額度 |
| **實作複雜度 (Recursion Complexity)** | Go 的 `fsnotify` 不支援遞迴監控，需自行實作目錄遍歷與動態追蹤，易產生 Bug |
| **驚群效應 (Thundering Herd)** | 大規模 Git 操作會瞬間觸發數千個事件，造成 CPU 負載 |
| **非必要性 (Overkill)** | Dotfiles 變更頻率低，不需要毫秒級同步 |

---

### ✅ Option 2: Periodic Polling (Time-Based) - [SELECTED]

使用 `time.Ticker` 每隔固定時間 (e.g., 5 分鐘) 掃描一次檔案雜湊。

#### 優點

| 優勢 | 說明 |
|------|------|
| **Stateless** | 兩次掃描間不佔用系統資源 (File Handles) |
| **Natural Debounce** | 自動過濾掉短時間內的多次修改，只取最終狀態 (Running State is Truth) |
| **Robust** | 不受 `max_user_watches` 限制，穩定性高 |

---

## 🔗 Schema Changes (phoenix.yml)

### 廢除 stow 區塊，新增 sync 區塊

```yaml
# 廢除舊的 stow 配置
# user_space:
#   stow:
#     source_dir: "~/dotfiles"
#     target_dir: "~"
#     packages: [...]

# 新的 sync 配置
sync:
  # 定義同步規則與範圍
  base_dir: "~/dotfiles"
  items:
    # 單一檔案
    - src: "zsh/.zshrc"
      dest: "~/.zshrc"

    # 整個目錄 (Recursive)
    - src: "ssh/"
      dest: "~/.ssh/"
      # 註：這裡可以定義 Provision 時的「預設/Fallback」權限，
      # 但具體的運行權限應由 State Store 管理
      default_chmod: "0600"
```

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **應用程式相容性** | 所有應用程式都能正常讀取實體檔案，無 Symlink 問題 |
| **雙向同步** | 支援 Provision 和 Harvest 雙向流程 |
| **精確權限控制** | 透過 State Store 記錄 Git 無法追蹤的權限資訊 |
| **自動化監控** | Daemon 自動偵測漂移，無需手動介入 |
| **資源友善** | Polling 機制不消耗 inotify 資源 |

### ❌ 負面影響 (Cons)

| 劣勢 | 說明 | 緩解措施 |
|------|------|----------|
| **磁碟空間** | 實體複製佔用額外空間 | Dotfiles 通常很小，影響可忽略 |
| **同步延遲** | Polling 有 5 分鐘延遲 | 對於 Dotfiles 使用場景可接受 |
| **實作複雜度** | 需要實作 State Store 與 Daemon 管理 | 換來更強的功能與相容性 |

---

## 📝 Related ADRs

- [ADR 0003: Dotfiles Management](./adr-0003-dotfiles-management.md) - **被本 ADR 取代** (Symlink → Physical Copy)
- [ADR 0006: Session State](./adr-0006-session-state.md) - Session 提供 UserHome 和展開後的路徑
