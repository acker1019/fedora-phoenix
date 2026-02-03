# ADR 0009: Trisolaran Rebranding & Stargazing Dimension

> **Project:** Fedora Phoenix → Trisolaran
> **Status:** ✅ Accepted
> **Date:** 2026-02-04
> **Refers to:**
> - [ADR 0007](./adr-0007-artifact-sync-harvesting.md) (Supersedes Harvest Strategy)
> - [ADR 0008](./adr-0008-artifact-storage-format.md) (Artifact Format)

---

## 📋 Context (背景)

經歷最近一次系統崩潰後，我們重新審視了這個工具的核心隱喻。

### 原有概念的局限性

**"Phoenix" (鳳凰浴火重生)** 這個名稱暗示的是：
- 從灰燼中重生
- 單向的毀滅與重建過程
- 強調「Provision」階段（重生）

### 實際工作流的本質

然而，這個工具的真實價值在於：

| 階段 | 行為 | 本質 |
|------|------|------|
| **系統運行中** | 持續工作，狀態不斷變化 | 正常水化狀態 |
| **系統重灌前** | 將 Running State 收集成 Artifact | 脫水 (Dehydration) |
| **乾旱期** | 系統重裝、格式化 Root | 三體人的乾旱期 |
| **系統重灌後** | 從 Artifact 還原所有狀態 | 重新水化 (Rehydration) |
| **監控階段** | 觀測系統狀態與漂移 | 持續觀星 (Stargazing) |

這更像是 **「Workspace 虛擬化」** —— 將物理系統視為可拋棄的載體，真正重要的是抽象化的工作狀態。

---

## 🎯 Decision (決策)

我們決定將專案重新命名為 **Trisolaran**，並完全重新設計命令集架構。

### 1. 新專案名稱

```
fedora-phoenix → trisolaran
```

**命名靈感**：三體文明的脫水/重水化生存機制

| 概念 | 三體人 | Trisolaran Tool |
|------|--------|-----------------|
| **正常期** | 有水環境，正常生活 | 系統正常運行 |
| **脫水期** | 將自己脫水成纖維狀態 | 將 Workspace 抽象成 Artifact |
| **乾旱期** | 纖維狀態等待復甦 | 系統重灌（/ 被格式化）|
| **重水化** | 遇水重新水化復活 | 從 Artifact 還原系統 |
| **觀星行為** | 透過觀星預測恆星運行 | 監控系統狀態，預測漂移 |

---

### 2. 命令集完全重新設計

#### 核心三指令 (The Trisolaran Trinity)

```bash
# 1. Rehydration (重水化) - 系統還原
tri rehydra

# 2. Dehydration (脫水) - 狀態收集
tri dehydra

# 3. Stargazing (觀星) - 系統監控
tri stargazing
```

#### 與舊命令的關係

| 舊命令 | 新命令 | 關係 |
|--------|--------|------|
| `phoenix provision` | `tri rehydra` | 概念延續，命名更新 |
| `phoenix harvest` | `tri dehydra` | **完全取代** |
| (無) | `tri stargazing` | **全新維度** |

---

### 3. Dehydra：完全取代 Harvest

| 層面 | Harvest (舊) | Dehydra (新) |
|------|--------------|--------------|
| **概念** | 收割 (Harvest) | 脫水 (Dehydration) |
| **範圍** | 主要針對 Dotfiles | 完整的 Workspace 狀態 |
| **產出** | 零散的檔案變更 | 單一 Artifact (tgz) |
| **使用時機** | 持續性收集 | 準備進入「乾旱期」前 |

**重要**：
- `tri dehydra` **完全取代** `phoenix harvest`，不是單純改名
- 不提供 `harvest` 的 alias 或相容層
- Dehydra 是 Harvest 的 superset，包含其所有功能但概念更完整，不應混淆

#### Dehydra 與 Systemd 整合

**核心機制**：
- `tri dehydra` 完全繼承先前 `harvest` 的定義
- Dehydra 本身只是工具，不負責產生監視程序
- 透過 `tri dehydra as-service` (暫定) 指令註冊 systemd 定時任務

---

### 4. Stargazing：新增的監控維度

三體人透過觀星來預測三顆恆星的運行規律，以判斷是否需要脫水準備。

Trisolaran 的 Stargazing 提供系統監控能力，用於觀測系統狀態與漂移。

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **概念一致性** | "脫水/重水化" 精確描述 Workspace 虛擬化的本質 |
| **功能擴展** | Stargazing 維度提供全新的系統監控能力 |
| **品牌識別度** | Trisolaran 名稱更獨特，避免與其他 Phoenix 專案混淆 |
| **哲學對齊** | 與 "Workspace 虛擬化" 的核心理念完美契合 |
| **清晰切割** | Dehydra 完全取代 Harvest，沒有歷史包袱 |

### ❌ 負面影響 (Cons)

| 劣勢 | 說明 |
|------|------|
| **破壞性變更** | 現有使用者需要完全重新學習命令 |
| **文件更新成本** | 所有文件需要更新專案名稱 |

---

## 💭 Design Philosophy

> **Trisolaran 的使命**：
>
> 讓你的 Workspace 像三體人一樣，能夠脫水成 Artifact，
> 在系統重灌的「乾旱期」中保持靜默，
> 然後在新系統中重新水化，完美復活。
>
> 透過持續觀星 (Stargazing)，我們預測系統的漂移與異常，
> 在下一次「亂紀元」來臨前做好準備。

---

**"In Chaos, We Dehydrate. In Order, We Rehydrate. Forever, We Stargaze."**
