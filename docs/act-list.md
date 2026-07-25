# Acts List (Capability Inventory)

> **Project:** Fedora Phoenix
> **Status:** ✅ Approved for Implementation
> **Context:** 這些 Acts 是構成 `RunProvision` 流程的原子操作

---

## 📊 Overview

本文件定義了 Phoenix Protocol 的所有原子操作 (Acts)，依照 [ADR-0002](./adr/adr-0002-block-architecture.md) 的四大區塊分類。

---

## 🔐 Block I: Identity & Configuration (身分與配置)

負責讀取飛行計畫與鑰匙。

### 1. LoadBlueprint

```go
func LoadBlueprint(path string) (*config.Blueprint, error)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 讀取公開的 `phoenix.yml`，定義系統還原的藍圖 |
| **Logic** | File Read → YAML Unmarshal → Validate Fields |
| **Location** | `internal/config/blueprint.go` |
| **Status** | ✅ Implemented |

---

### 2. LoadSecrets

```go
func LoadSecrets(path string) (*config.Secrets, error)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 讀取私密的 `secrets.yml`，獲取 LUKS 密碼與 Tokens |
| **Logic** | File Read → YAML Unmarshal → Validate |
| **Location** | `internal/config/secrets.go` |
| **Status** | ✅ Implemented |

---

### 3. CleanupSecrets

```go
func CleanupSecrets(path string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 執行「讀後即焚」策略，刪除實體檔案 |
| **Logic** | Secure Overwrite → `os.Remove(path)` (Best effort) |
| **Location** | `internal/config/secrets.go` |
| **Status** | ✅ Implemented |

---

## 🔧 Block II: Infrastructure (基礎設施)

負責底層儲存裝置操作。**失敗即中止 (Fatal)**。

### 4. UnlockLuks

```go
func UnlockLuks(devicePath, mapperName, password string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 解鎖 LUKS 加密分區 |
| **Idempotency** | Check if `/dev/mapper/NAME` exists |
| **Security** | ⚠️ Password must be piped via Stdin, NOT command arguments |
| **Command** | `cryptsetup open ... --type luks -` |
| **Location** | `internal/ops/luks.go` |
| **Status** | ✅ Implemented |

#### Logic Flow

```text
1. Check: /dev/mapper/{mapperName} exists?
   ├─ Yes → Skip (Idempotent)
   └─ No  → Continue
2. Exec: cryptsetup open {devicePath} {mapperName} --type luks
3. Pipe password via Stdin
```

---

### 5. MountDevice

```go
func MountDevice(mapperName, mountPoint string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 掛載已解鎖的分區 |
| **Idempotency** | Check if already mounted (`mountpoint -q`) |
| **Location** | `internal/ops/luks.go` |
| **Status** | ✅ Implemented |

#### Logic Flow

```text
1. Ensure mountPoint exists (mkdir -p)
2. Check: mountpoint -q {mountPoint}
   ├─ Yes → Skip (Already mounted)
   └─ No  → Continue
3. Exec: mount /dev/mapper/{mapperName} {mountPoint}
```

---

## ⚙️ Block III: System State (系統狀態)

負責作業系統層級的設定。以 **Root** 身份執行。

### 6. EnsurePackages

```go
func EnsurePackages(pkgs []string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 安裝一般套件 (Always Latest) |
| **Idempotency** | Filter installed packages using `rpm -q` for speed |
| **Command** | `dnf install -y <pkg>` for missing ones |
| **Location** | `internal/ops/pkg.go` |
| **Status** | ✅ Implemented |

---

### 7. EnsurePinnedPackages

```go
func EnsurePinnedPackages(pkgs []string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 安裝並鎖定特定版本的套件 (Version Locking) |
| **Prerequisite** | Ensure `python3-dnf-plugin-versionlock` is installed |
| **Location** | `internal/ops/pkg.go` |

#### Logic Flow

```text
1. Ensure: python3-dnf-plugin-versionlock installed
2. For each pkg:
   ├─ dnf install -y <pkg-nvr> (Force specific version)
   └─ dnf versionlock add <pkg-nvr>
```

---

### 8. EnsureServices

```go
func EnsureServices(services []string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 啟動 Systemd 服務 |
| **Command** | `systemctl enable --now <service>` |
| **Location** | `internal/ops/systemd.go` |

---

### 9. EnsureUserShell

```go
func EnsureUserShell(username, targetShell string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 更改使用者的預設 Shell |
| **Idempotency** | Read `/etc/passwd` to check current shell |
| **Command** | `usermod -s <targetShell> <username>` (if mismatch) |
| **Location** | `internal/ops/user.go` |

---

### 9a. EnsureGroups / EnsureUsers

```go
func EnsureGroups(groups []config.GroupConfig) error
func EnsureUsers(users []config.UserConfig) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 建立 `system.groups` / `system.users` 定義的群組與使用者，並確保群組成員關係一致 |
| **Idempotency** | `user.LookupGroup`/`user.Lookup` 檢查是否已存在；比對現有群組成員後只補缺少的部分 |
| **Command** | `groupadd`, `useradd -m`, `usermod -a -G` |
| **Location** | `internal/ops/account.go` |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | 只記錄 name，不 hardcode UID/GID——建立後一律由 OS 讀回 |

---

## 👤 Block IV: User Space (用戶空間)

負責使用者資料與環境。**必須透過 `RunCommandAsUser` 執行**以確保權限正確。

### 10. RunCommandAsUser (Core Utility)

```go
func RunCommandAsUser(username, name string, args ...string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | ⭐ **Block IV 的核心引擎** - 以目標使用者身份執行命令 |
| **Location** | `internal/utils/exec.go` |

#### Logic Flow

```text
1. Lookup uid, gid of the user
2. Set cmd.SysProcAttr.Credential to switch context
3. Exec command
```

---

### 11. EnsureSymlink

```go
func EnsureSymlink(src, dest, username string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 建立資料夾連結（主要用於 Workspace 資料掛載） |
| **Execution** | Via `RunCommandAsUser` |
| **Command** | `ln -sfn <src> <dest>` |
| **Location** | `internal/ops/user.go` |

#### Logic Flow

```text
1. Check if dest exists
2. If not exists OR is wrong link:
   └─ Exec: ln -sfn {src} {dest} (via RunCommandAsUser)
```

---

### 12. artifact.Pack (Harvest)

```go
func Pack(paths []string, outputPath string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 掃描 `userspace.harvest.paths`，鏡射複製進 `fs/`，產生 `filemeta.yml`，打包成 `phoenix-backup-<date>.tgz` |
| **Command** | `phoenix harvest --output=<path>` |
| **Location** | `internal/artifact/pack.go` |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | filemeta.yml 只記錄 owner/group **名稱**與八進位 mode，不記錄 UID/GID |

---

### 13. artifact.Restore (Provision-side Rehydration)

```go
func Restore(archivePath string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 解壓 Artifact，讀取 `filemeta.yml`，將 `fs/` 內容還原到系統絕對路徑，套用 mode/owner/group，並以 SHA256 驗證完整性 |
| **Location** | `internal/artifact/unpack.go` |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | owner/group 一律用名稱在還原當下重新查詢 UID/GID，不信任 Artifact 內的數字 |

---

### 14. GitClone (Workspace Repos)

```go
func GitClone(url, dest, username string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 下載開發專案代碼 |
| **Execution** | Via `RunCommandAsUser` |
| **Idempotency** | Check if dest exists |
| **Location** | `internal/ops/user.go` |
| **Refers to** | [ADR-0003](./adr/adr-0003-dotfiles-management.md) |

#### Logic Flow

```text
1. Check if dest exists:
   ├─ No  → Exec: git clone {url} {dest} (via RunCommandAsUser)
   └─ Yes → (Optional) git pull OR skip
```

---

## 📋 Implementation Status

| Block | Act | Status | Location |
|-------|-----|--------|----------|
| **I** | LoadBlueprint | ✅ Implemented | `internal/config/blueprint.go` |
| **I** | LoadSecrets | ✅ Implemented | `internal/config/secrets.go` |
| **I** | CleanupSecrets | ✅ Implemented | `internal/config/secrets.go` |
| **II** | UnlockLuks | ✅ Implemented | `internal/ops/luks.go` |
| **II** | MountDevice | ✅ Implemented | `internal/ops/luks.go` |
| **III** | EnsurePackages | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsurePinnedPackages | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsureServices | ✅ Implemented | `internal/ops/systemd.go` |
| **III** | EnsureUserShell | ✅ Implemented | `internal/ops/user.go` |
| **III** | EnsureGroups / EnsureUsers | ✅ Implemented | `internal/ops/account.go` |
| **IV** | RunCommandAsUser | ✅ Implemented | `internal/utils/exec.go` |
| **IV** | EnsureSymlink | ✅ Implemented | `internal/ops/user.go` |
| **IV** | artifact.Pack | ✅ Implemented | `internal/artifact/pack.go` |
| **IV** | artifact.Restore | ✅ Implemented | `internal/artifact/unpack.go` |
| **IV** | GitClone | ✅ Implemented | `internal/ops/user.go` |
