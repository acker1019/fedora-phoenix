# Core Philosophy

> **Fedora Phoenix 的設計原則與核心理念**

---

## 🎯 核心定位

Phoenix 是一個 **Playbook 工具**，而非 Script 工具。

| 特性 | Script | Playbook (Phoenix) |
|------|--------|-------------------|
| **執行模式** | 命令式 (Imperative) | 宣告式 (Declarative) |
| **可重複性** | 只能執行一次 | 可無限次重複執行 |
| **失敗處理** | 需要手動清理 | 修正後直接 rerun |
| **狀態管理** | 無狀態檢查 | Check-Diff-Act |

---

## 📐 設計原則

### 1. Declarative, not Imperative
**宣告期望狀態，而非命令步驟**

```yaml
# ✅ Good: 宣告期望狀態
system:
  packages:
    - vim
    - git

# ❌ Bad: 命令式步驟
# - run: "dnf install vim"
# - run: "dnf install git"
```

使用者只需要描述「想要什麼」，而非「如何做」。

---

### 2. Idempotent, not Transactional
**可重複執行，而非一次性腳本**

```go
// Check-Diff-Act 模式
func EnsurePackage(pkg string) error {
    // Check: 檢查當前狀態
    if isInstalled(pkg) {
        log.Info("Already installed, skipping")
        return nil
    }

    // Act: 執行變更
    return install(pkg)
}
```

每個操作都必須：
- 檢查當前狀態
- 僅在需要時執行
- 可安全重複執行

參見 [ADR-0005: Idempotency Pattern](adr/adr-0005-idempotency-pattern.md)

---

### 3. Rerun, not Rollback
**失敗後修正並重跑，而非回滾**

```text
執行失敗 (網路中斷)
         │
         ▼
   ❌ 不需要 Rollback
         │
         ▼
修正問題 (恢復網路)
         │
         ▼
重新執行 phoenix provision
  → 已完成的步驟自動跳過
  → 僅執行未完成的部分
```

**為什麼不需要 Rollback？**

因為所有操作都是冪等的：
- 已完成的部分不會重複執行
- 未完成的部分會自動補齊
- 最終達到期望狀態

參見 [Anti-Requirements](anti-requirements.md)

---

### 4. Local, not Remote
**本地執行，專注單機場景**

Phoenix 設計為在目標機器上執行，而非遠端控制：

```bash
# ✅ Good: 本地執行
sudo phoenix provision --secrets=secrets.yml

# ❌ Bad: 遠端執行（不支援）
# phoenix provision --host=remote-server
```

**理由**:
- 簡化安全模型（無需處理 SSH keys）
- 減少網路相關錯誤
- 符合「單機 provisioning」的定位

如需遠端部署，使用：
```bash
scp phoenix user@remote:/tmp/
ssh user@remote "sudo /tmp/phoenix provision --secrets=secrets.yml"
```

---

### 5. Simple, not Complex
**保持簡單，避免過度工程**

Phoenix 刻意不實作：
- ❌ 多環境配置管理 (dev/staging/prod)
- ❌ 遠端執行與 SSH 支援
- ❌ 複雜的相依性解析
- ❌ 互動式確認提示
- ❌ Rollback 機制
- ❌ GUI / Web UI

**原則**:
- 單一職責：Provisioning 單機
- 信任系統工具：DNF、Systemd、Git
- 不重新發明輪子

參見 [Anti-Requirements](anti-requirements.md) 完整清單

---

## 🧱 架構哲學

### Engine-Blueprint 分離

```text
┌─────────────────────────────────────────────────────────────┐
│ Engine (phoenix binary)                                     │
│ • Stateless                                                 │
│ • 不包含任何配置                                              │
│ • 可重複使用於不同機器                                         │
└────────────────┬────────────────────────────────────────────┘
                 │
                 │ reads
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Blueprint (phoenix.yml)                                     │
│ • 宣告期望狀態                                                │
│ • 版本控制                                                    │
│ • 機器特定配置                                                │
└─────────────────────────────────────────────────────────────┘
```

優勢：
- Engine 可以升級而不影響配置
- Blueprint 可以獨立測試與分享
- 清楚的職責分離

參見 [ADR-0004: Blueprint Pattern](adr/adr-0004-blueprint-pattern.md)

---

## 🔄 冪等性哲學

### Check-Diff-Act 三步驟

所有 Act Functions 必須遵循：

```go
func EnsureXXX(...) error {
    // 1. Check: 獲取當前狀態
    currentState := getCurrentState()

    // 2. Diff: 比對期望狀態
    if currentState == desiredState {
        log.Info("Already in desired state, skipping")
        return nil
    }

    // 3. Act: 執行修正動作
    if err := reconcile(); err != nil {
        return fmt.Errorf("failed to reconcile: %w", err)
    }

    return nil
}
```

### Ensure* 命名慣例

| ❌ Bad (Imperative) | ✅ Good (Declarative) |
|---------------------|----------------------|
| `InstallPackages()` | `EnsurePackages()` |
| `CreateSymlink()` | `EnsureSymlink()` |
| `MountDisk()` | `EnsureDeviceMounted()` |
| `CloneRepo()` | `EnsureGitRepo()` |

命名反映「確保狀態」的意圖，而非「執行動作」。

參見 [ADR-0005: Idempotency Pattern](adr/adr-0005-idempotency-pattern.md)

---

## 🎭 與其他工具的比較

### Phoenix vs Ansible

| 特性 | Ansible | Phoenix |
|------|---------|---------|
| **部署範圍** | 多機器 Fleet | 單機器 |
| **配置語言** | YAML + Jinja2 | Pure YAML |
| **依賴** | Python + Modules | 零依賴 (單 Binary) |
| **執行模式** | SSH 遠端 | 本地執行 |
| **適用場景** | 伺服器管理 | 開發機重建 |

### Phoenix vs Shell Script

| 特性 | Shell Script | Phoenix |
|------|--------------|---------|
| **可重複性** | 通常只能執行一次 | 可無限次執行 |
| **錯誤處理** | 需手動清理 | 自動跳過已完成 |
| **可讀性** | 程序導向 | 宣告式配置 |
| **維護性** | 難以測試 | 結構化設計 |

### Phoenix vs NixOS

| 特性 | NixOS | Phoenix |
|------|-------|---------|
| **範圍** | 整個作業系統 | Provisioning 工具 |
| **學習曲線** | 陡峭 | 平緩 |
| **現有系統** | 需重裝系統 | 可用於現有 Fedora |
| **哲學** | 函數式純粹性 | 實用主義 |

---

## 💭 設計權衡

### 取捨決策

| 犧牲 | 換取 |
|------|------|
| 多機器部署能力 | 簡化的安全模型與實作 |
| 完整的錯誤回滾 | 簡單的冪等性重執行 |
| 跨發行版支援 | 深度整合 Fedora 生態系 |
| GUI 介面 | 單純的 CLI 工具 |
| 遠端執行 | 更好的本地效能 |

這些都是 **刻意的設計決策**，符合 Phoenix 的定位：

> **專注於單機 Fedora Workstation 的快速重建**

---

## 🚫 明確不做的事

參見 [Anti-Requirements](anti-requirements.md) 完整清單：

1. **Rollback / Transaction 機制** - 使用 Rerun 取代
2. **複雜的狀態追蹤** - 信任系統實際狀態
3. **部分執行模式** - 執行單位是完整 Provision
4. **互動式確認** - 完全自動化
5. **多環境配置** - 一機一配置
6. **遠端執行** - 本地執行工具
7. **GUI / Web UI** - CLI Only

---

## 📚 延伸閱讀

- [ADR-0001: Pure Go Strategy](adr/adr-0001-pure-go-strategy.md) - 為何選擇 Go
- [ADR-0002: Block Architecture](adr/adr-0002-block-architecture.md) - 四階段執行流程
- [ADR-0005: Idempotency Pattern](adr/adr-0005-idempotency-pattern.md) - 冪等性實作細節
- [Anti-Requirements](anti-requirements.md) - 明確不做的功能清單
- [Wishlist](wishlist.md) - 未來可能的功能

---

## 💡 記住

當你考慮新增功能或修改設計時，問自己：

> "這個改動是讓 Phoenix 更像 Playbook，還是更像 Script？"

如果是後者，**不要做**。

---

**Phoenix 的使命**：讓 Fedora Workstation 的重建像執行 `git checkout` 一樣簡單可靠。
