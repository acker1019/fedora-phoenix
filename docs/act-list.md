# Acts List (Capability Inventory)

> **Project:** Fedora Trisolaran
> **Status:** ✅ Approved for Implementation
> **Context:** 這些 Acts 是構成 `runRehydra` 流程的原子操作

---

## 📊 Overview

本文件定義了 Trisolaran Protocol 的所有原子操作 (Acts)，依照 [ADR-0002](./adr/adr-0002-block-architecture.md) 的四大區塊分類。

---

## 🔐 Block I: Identity & Configuration (身分與配置)

負責讀取飛行計畫與鑰匙。

### 1. LoadBlueprint

```go
func LoadBlueprint(path string) (*config.Blueprint, error)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 讀取公開的 `trisolaran.yml`，定義系統還原的藍圖 |
| **Logic** | File Read → YAML Unmarshal (strict: `KnownFields(true)`) → Validate Fields |
| **Note** | Strict decoding 會拒絕未知欄位，避免 schema 改名（例如 `packages` → `pkgs`）後，舊檔案裡的欄位被靜默忽略、卻不報錯 |
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

## 🔧 Block II: Infrastructure (基礎設施)

負責底層儲存裝置操作。**失敗即中止 (Fatal)**。

**LUKS 是選配的**：若 blueprint 沒有 `infrastructure` 這行，或 `infrastructure.luks` 底下三個欄位都留空，`UnlockLuks` 與 `MountDevice` 會整段被跳過（見 `config.Blueprint.HasLuks()`），`secrets.luks_password` 也不會被要求。三個欄位（`device`/`mapper_name`/`mount_point`）是全有或全無，只填一部分會在 Blueprint 驗證階段報錯。

### 3. UnlockLuks

```go
func UnlockLuks(devicePath, mapperName, password string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 解鎖 LUKS 加密分區 |
| **Idempotency** | Check if `/dev/mapper/NAME` exists |
| **Optionality** | 僅在 `infrastructure.luks` 有配置時執行，見上方說明 |
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

### 4. MountDevice

```go
func MountDevice(mapperName, mountPoint string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 掛載已解鎖的分區 |
| **Idempotency** | Check if already mounted (`mountpoint -q`) |
| **Optionality** | 僅在 `infrastructure.luks` 有配置時執行，見上方說明 |
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

### 4a. EnsureSystemUpdate

```go
func EnsureSystemUpdate() error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | Block III 的第一件事：把所有已安裝套件更新到最新 |
| **Idempotency** | 無獨立 check 步驟——直接呼叫 `dnf update`，系統已是最新時 dnf 本身就回報無事可做 |
| **Command** | `dnf update -y --refresh` |
| **Location** | `internal/ops/pkg.go` |
| **Optionality** | 預設開啟；`system.skip_update: true` 可跳過 |
| **Status** | ✅ Implemented |

---

### 4b. EnsurePkgRepos

```go
func EnsurePkgRepos(repos []config.PkgRepoConfig) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 確保 `system.pkg_repos` 宣告的 repo 處於期望狀態，讓 `EnsurePackages` 能從非預設來源（如 VS Code、Chrome）安裝套件 |
| **Idempotency** | Repo 檔不存在 → 建立（並視需要 `rpm --import` gpgkey）；已存在 → 只比對並調整 `enabled=` 狀態 |
| **Command** | `rpm --import <gpgkey>`（建立時）；調整既有 repo 的 `enabled=` 則直接改寫 `.repo` 檔本身，不呼叫 `dnf config-manager`（dnf4 的 `--set-enabled`/`--set-disabled` 語法在 dnf5 上不存在） |
| **Location** | `internal/ops/pkg.go` |
| **Note** | 兩種情境共用同一個 struct：只給 `id`（+ `enabled`）代表既有 repo 只需翻轉開關；給 `baseurl`（+ 選填 `gpgkey`）代表 repo 檔不存在、需要建立 |
| **Status** | ✅ Implemented |

---

### 5. EnsurePackages

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

### 6. EnsurePinnedPackages

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

### 7. EnsureServices

```go
func EnsureServices(services []string)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 啟動 Systemd 服務 |
| **Command** | `systemctl enable --now <service>` |
| **Location** | `internal/ops/systemd.go` |

---

### 7a. EnsureTmpfiles

```go
func EnsureTmpfiles(paths []string, userHome string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 宣告 `system.tmpfiles` 裡的路徑，透過 systemd-tmpfiles 在每次開機時移除（例如上次非正常關機留下的 app lock 檔） |
| **Idempotency** | 比對 `/etc/tmpfiles.d/trisolaran.conf` 現有內容跟期望內容是否相同，不同才寫入 |
| **Command** | 寫入 `/etc/tmpfiles.d/trisolaran.conf`，每行 `r <展開後路徑>`；不需要自訂 systemd unit，因為 `systemd-tmpfiles-setup.service` 本來就內建、每次開機都會執行 |
| **Location** | `internal/ops/tmpfiles.go` |
| **Status** | ✅ Implemented |

---

### 8. EnsureUserShell

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

### 8a. EnsureGroups / EnsureUsers

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
| **Note** | `EnsureGroups` 在 `EnsureUsers` 之前執行；若使用者名稱跟某個既有群組同名（例如 blueprint 同時宣告了 `system.groups: [docker]` 與 `system.users: [{name: docker}]`），建立該使用者時會用 `-g <name>` 直接沿用該群組當主要群組，不讓 `useradd` 嘗試自動建立同名 private group（否則會因為名稱衝突而失敗） |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | 只記錄 name，不 hardcode UID/GID——建立後一律由 OS 讀回 |

---

## 👤 Block IV: User Space (用戶空間)

負責使用者資料與環境。**必須透過 `RunCommandAsUser` 執行**以確保權限正確。

### 9. RunCommandAsUser (Core Utility)

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

### 10. EnsureSymlink

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

### 11. artifact.Pack (Dehydration)

```go
func Pack(paths []string, blueprintPath, outputPath string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 掃描 `userspace.dehydration.paths`，鏡射複製進 `fs/`，產生 `filemeta.yml`，打包成 `trisolaran-backup-<date>.tgz` |
| **Command** | `tri dehydra --output=<path>` |
| **Location** | `internal/artifact/pack.go` |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | filemeta.yml 只記錄 owner/group **名稱**與八進位 mode，不記錄 UID/GID |
| **Note** | 若 `blueprintPath` 非空，會把該 blueprint 原檔複製進 archive 根目錄的 `blueprint.yml`（與 `filemeta.yml` 同層，不進 `fs/`），讓 artifact 可以自帶還原時所需的 blueprint |

---

### 12. artifact.Restore (Provision-side Rehydration)

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

### 12a. artifact.ExtractBlueprint

```go
func ExtractBlueprint(archivePath string) ([]byte, error)
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 從 artifact tgz 裡單獨讀出 `blueprint.yml`，不解壓其餘內容 |
| **Behavior** | `tri rehydra` 在 `--blueprint` 未被明確指定、但有解析出 artifact 路徑（明確給的位置參數，或自動偵測到的 `trisolaran-backup.tgz`）時，會優先呼叫這個函式取得 blueprint；`--blueprint` 一經明確指定則永遠優先，忽略 artifact 內建的版本 |
| **Location** | `internal/artifact/blueprint.go` |
| **Refers to** | [ADR-0008](./adr/adr-0008-artifact-storage-format.md) |
| **Note** | 找不到內建 blueprint 時回傳 `ErrBlueprintNotEmbedded`（例如舊格式 artifact），呼叫端會 fallback 回讀取 `--blueprint` 指定（或預設）的檔案 |
| **Status** | ✅ Implemented |

---

### 13. GitClone (Workspace Repos)

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

### 14. EnsureScripts

```go
func EnsureScripts(scripts []string, username string) error
```

| 屬性 | 說明 |
|------|------|
| **Responsibility** | 依序執行 `userspace.scripts` 定義的單行 shell 指令，用於處理無法透過 dnf 安裝的東西（例如需要跑第三方 installer script 的工具） |
| **Execution** | Via `RunCommandAsUser`，在目標使用者自己的 `$HOME` 下執行，不是 root |
| **Ordering** | **必須是 Block IV 的最後一步**（在 `rehydra.go` 裡緊接在 `GitClone` 迴圈之後），因為腳本內容可能會依賴 artifact 還原或 repo clone 的結果 |
| **Idempotency** | 沒有通用的 Check-Diff 機制——每行指令對 Trisolaran 而言是不透明的，重跑是否安全由撰寫該行的人自己負責 |
| **Command** | `sh -c <script>` |
| **Location** | `internal/ops/script.go` |
| **Status** | ✅ Implemented |

---

## 📋 Implementation Status

| Block | Act | Status | Location |
|-------|-----|--------|----------|
| **I** | LoadBlueprint | ✅ Implemented | `internal/config/blueprint.go` |
| **I** | LoadSecrets | ✅ Implemented | `internal/config/secrets.go` |
| **II** | UnlockLuks | ✅ Implemented | `internal/ops/luks.go` |
| **II** | MountDevice | ✅ Implemented | `internal/ops/luks.go` |
| **III** | EnsureSystemUpdate | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsurePkgRepos | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsurePackages | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsurePinnedPackages | ✅ Implemented | `internal/ops/pkg.go` |
| **III** | EnsureServices | ✅ Implemented | `internal/ops/systemd.go` |
| **III** | EnsureTmpfiles | ✅ Implemented | `internal/ops/tmpfiles.go` |
| **III** | EnsureUserShell | ✅ Implemented | `internal/ops/user.go` |
| **III** | EnsureGroups / EnsureUsers | ✅ Implemented | `internal/ops/account.go` |
| **IV** | RunCommandAsUser | ✅ Implemented | `internal/utils/exec.go` |
| **IV** | EnsureSymlink | ✅ Implemented | `internal/ops/user.go` |
| **IV** | artifact.Pack | ✅ Implemented | `internal/artifact/pack.go` |
| **IV** | artifact.Restore | ✅ Implemented | `internal/artifact/unpack.go` |
| **IV** | artifact.ExtractBlueprint | ✅ Implemented | `internal/artifact/blueprint.go` |
| **IV** | GitClone | ✅ Implemented | `internal/ops/user.go` |
| **IV** | EnsureScripts | ✅ Implemented | `internal/ops/script.go` |
