# ADR 0004: Blueprint Pattern

> **Project:** Fedora Phoenix
> **Status:** ✅ Accepted
> **Date:** 2025-12-26
> **Refers to:** [ADR-0001](./adr-0001-pure-go-strategy.md), [ADR-0002](./adr-0002-block-architecture.md)

---

## 📋 Context (背景)

在早期的設計中，還原邏輯（如安裝哪些套件、掛載點路徑）可能被硬編碼 (Hardcoded) 於 Go 程式碼中。

### 面臨的問題

| 問題 | 影響 |
|------|------|
| **硬編碼配置** | 每次微調環境需求（例如更換使用者名稱、新增一個開發工具）都需要重新編譯 Binary |
| **缺乏彈性** | 無法用同一套 Binary 適應不同的機器或場景 |
| **維護困難** | 執行邏輯 (How) 與期望狀態 (What) 耦合在一起 |

我們需要將「執行邏輯 (How)」與「期望狀態 (What)」徹底解耦，以支援同一套 Binary 能夠適應不同的機器或場景。

---

## 🎯 Decision (決策)

我們採用 **「引擎與藍圖分離 (Engine-Blueprint Separation)」** 模式。

### 1. 核心概念：The Phoenix Blueprint (phoenix.yml)

我們定義 `phoenix.yml` 為系統還原的 **「藍圖 (Blueprint)」**。它是一個宣告式 (Declarative) 的 YAML 文件，作為公開的、可版本控制的 (Git-friendly) 單一事實來源 (Single Source of Truth)。

| 屬性 | 說明 |
|------|------|
| **性質** | 公開 (Public) |
| **邊界** | 僅包含環境結構描述，嚴禁包含任何機密資訊 (Secrets/Credentials) |
| **格式** | YAML |
| **版本控制** | ✅ Git-friendly，可追溯環境演變歷史 |

---

### 2. 抽象領域定義 (Abstract Domains)

為了保持架構彈性，`phoenix.yml` 的 schema 不與特定的實作細節綁定，而是描述以下四大抽象領域的期望狀態：

#### 🆔 Target Identity (目標身份)

描述系統應呈現的使用者特徵與身份資訊。這確保了 Engine 知道該「為誰」進行還原。

---

#### 🏗️ Infrastructure Layout (基礎設施佈局)

描述底層儲存與硬體資源的對應關係。這讓 Engine 能夠適應不同的硬體分區規劃，而無需修改程式碼。

---

#### ⚙️ System State Definition (系統狀態定義)

描述 OS 層級應具備的軟體清單與服務狀態。包含「必須存在的軟體 (Inventory)」與「必須鎖定的版本 (Constraints)」，且不限制於特定包管理器。

---

#### 📂 Data Projection Rules (資料投射規則)

描述如何將持久化資料 (Persistent Data) 與 使用者空間 (User Space) 連結。包含外部資料的掛載/連結、設定檔的部署策略，以及外部程式碼倉庫的還原

---

### 3. 執行原則

#### Phoenix Engine (Binary)

| 特性 | 說明 |
|------|------|
| **無狀態執行者** | 不持有任何關於「這台電腦該長怎樣」的知識 |
| **完全依賴 Blueprint** | 所有行為由讀入的 `phoenix.yml` 決定 |

#### Parametrization (參數化)

所有的 Act Functions 必須設計為**接受參數輸入**，禁止在函數內部寫死具體數值。這確保 Binary 保持無狀態，所有行為完全由 Blueprint 驅動

---

## 🏗️ Architecture Pattern

```text
┌──────────────────────────────────────────────────────┐
│              phoenix provision                       │
│              (Reads phoenix.yml)                     │
└────────────────────┬─────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │  config.Blueprint     │
         │  (Parsed YAML Schema) │
         └───────────┬───────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
  ┌─────────┐  ┌─────────┐  ┌─────────┐
  │Block II │  │Block III│  │Block IV │
  │  (Infra)│  │(System) │  │ (User)  │
  └─────────┘  └─────────┘  └─────────┘
       │            │            │
       └────────────┴────────────┘
                    │
                    ▼
           Stateless Execution
```

---

## ⚖️ Consequences (後果)

### ✅ 正面影響 (Pros)

| 優勢 | 說明 |
|------|------|
| **可攜性 (Portability)** | 同一顆 Binary 可以透過切換不同的 `.yml` 檔案，用來還原工作電腦、個人電腦或測試機 |
| **GitOps Ready** | `phoenix.yml` 可以存放在 Dotfiles Git Repo 中。透過 Git History，我們可以追溯環境演變的歷史（例如：「為什麼我在 2025 年移除了 Firefox？」） |
| **安全性隔離** | 透過將非敏感配置與敏感憑證（`secrets.yml`）物理分離，降低了誤將密碼 Commit 進 Git 的風險 |
| **易於測試** | 可以為不同場景準備不同的 blueprint 檔案進行測試 |

### ❌ 負面影響 (Cons)

| 風險 | 說明 | 緩解措施 |
|------|------|----------|
| **Schema 維護成本** | Go Struct (`config.Blueprint`) 必須與 YAML 結構嚴格對應。新增功能時需要同時修改 Go Struct 與 YAML 檔案 | 透過詳細的文件與測試確保一致性 |
| **執行依賴** | Binary 無法單獨運作，必須始終搭配合法的 `phoenix.yml` 才能執行 | 提供範例 blueprint 檔案與驗證工具 |

---

## 📝 Implementation Notes

### Blueprint Schema 原則

`phoenix.yml` 的 schema 定義位於 `internal/config/blueprint.go`，採用 Go Struct + YAML tags 的方式進行序列化。

| 原則 | 說明 |
|------|------|
| **版本化** | Schema 應包含版本標記，便於未來演進與向後相容 |
| **分層清晰** | 對應四大抽象領域進行結構化設計 |
| **可驗證性** | 提供 Validation 機制，在執行前檢查配置完整性 |
