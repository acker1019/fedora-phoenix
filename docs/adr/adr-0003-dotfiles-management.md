# ADR 0003: Dotfiles Management

> **Project:** Fedora Phoenix
> **Status:** ⚠️ Amended by [ADR 0007](./adr-0007-artifact-sync-harvesting.md) (Stow strategy superseded)
> **Date:** 2025-12-26
> **Refers to:** [ADR 0002](./adr-0002-block-architecture.md) (Block IV Definition)

---

## 📋 Context (背景)

在 ADR 0002 中，我們定義了 **Block IV (User Space)** 負責還原用戶資料。

### 面臨的挑戰

1. **自行開發的 Symlink 管理過於脆弱**
   - 難以處理衝突解決 (Conflict Resolution)
   - 路徑追蹤複雜
   - 重複造輪子 (Reinventing the wheel)

2. **Dotfiles 包含敏感資訊**
   - 需以 `.tgz` 壓縮檔形式物理注入 (Artifact Injection)
   - 不能直接從公開 Git 倉庫拉取

3. **開發者工作區需要還原**
   - 需要從多個 Git 倉庫還原專案代碼

---

## 🎯 Decision (決策)

我們決定引入 **GNU Stow** 作為 Dotfiles 的部署管理工具，並在核心功能中恢復 **Git Repository Cloning** 能力。

### 1. 採用 GNU Stow 進行部署

我們不再手動維護 Symlink 邏輯，而是將解壓後的 Dotfiles 目錄視為 Stow 的 **"Package Directory"**。

| 屬性 | 說明 |
|------|------|
| **依賴性** | `stow` 必須被加入 Block III 的預裝套件清單 (`packages.dnf`) |
| **執行模式** | 使用 `stow --restow` (或 `-R`) 指令，確保操作的冪等性 |
| **權限** | 必須透過 `RunCommandAsUser` 以目標使用者身份執行，確保連結權限正確 |

### 2. Artifact-based Injection (檔案注入策略)

#### 流程

```text
┌─────────────────────────────────────────────────────────┐
│  1. User manually prepares dotfiles tarball             │
│     (e.g., backup.tgz with secrets)                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  2. Phoenix (Block IV) extracts to ~/dotfiles           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  3. Phoenix invokes Stow to deploy to $HOME             │
└─────────────────────────────────────────────────────────┘
```

### 3. Workspace Provisioning (工作區還原)

恢復 **GitClone** 功能，專門用於還原非 Dotfiles 類的開發專案 (Project Repos)。

**行為**:
1. 檢查目標目錄
2. 若不存在則執行 `git clone`
3. (選用) 若存在則執行 `git pull`

### 4. Configuration Schema (phoenix.yml)

擴充 `phoenix.yml` 以支援上述邏輯：

```yaml
# Dotfiles 設定
stow:
  source_dir: "~/dotfiles"  # 解壓後的 Stow 根目錄
  target_dir: "~"           # 部署目標 (通常是 Home)
  packages:                 # 要啟用的 Stow Packages
    - zsh
    - nvim
    - git
    - secrets               # 包含敏感資料的包

# 開發專案還原
repos:
  - url: "git@github.com:user/project-alpha.git"
    dest: "~/Workspace/project-alpha"
```

---

## 🔧 Core Bricks Updates (功能模組更新)

基於此決策，`internal/ops/user.go` 將包含以下核心函數：

| 函數 | 功能 |
|------|------|
| `ExtractTarball` | 處理 Artifact 解壓 |
| `RunStow` | 封裝 `stow -d Source -t Target -R Package` 指令 |
| `GitClone` | 處理 `git clone <repo> <dest>` |

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **穩定性提升** | 依賴成熟的 GNU Stow 處理複雜的 Symlink 管理與衝突檢測 |
| **除錯容易** | Stow 會明確報錯並拒絕覆蓋非 Symlink 的檔案，避免意外破壞資料 |
| **職責分離** | Phoenix 負責「搬運與執行」，Stow 負責「連結管理」，Git 負責「代碼下載」 |

### ❌ 負面影響 (Cons)

| 風險 | 說明 | 緩解措施 |
|------|------|----------|
| **增加 Runtime 依賴** | 系統必須先安裝 Perl (Stow 的依賴) 和 Stow 本身 | 在 Fedora 上透過 DNF 輕鬆解決 |
| **結構要求嚴格** | Dotfiles 壓縮檔目錄結構必須符合 Stow 的規範 | 使用者需遵循 `package_name/.config/...` 結構 |

---

## 📝 Implementation Notes

### Stow Package 結構範例

```text
~/dotfiles/
├── zsh/
│   └── .zshrc
├── nvim/
│   └── .config/
│       └── nvim/
│           └── init.vim
├── git/
│   └── .gitconfig
└── secrets/
    └── .ssh/
        └── id_rsa
```

### 執行命令範例

```bash
# Extract tarball
tar -xzf backup.tgz -C ~/dotfiles

# Deploy with Stow (as user, not root)
stow -d ~/dotfiles -t ~ -R zsh nvim git secrets
```
