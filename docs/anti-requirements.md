# Anti-Requirements (反向需求清單)

> **目的**: 明確說明 Phoenix **不需要**、**不應該** 實作的功能
> **受眾**: AI Agents、未來的貢獻者

---

## ❌ 不需要 Rollback / Transaction 機制

### 錯誤理解

認為 Phoenix 需要像資料庫 transaction 一樣的 rollback 機制，在失敗時需要「還原」到執行前的狀態。

### 正確理解

Phoenix 採用 **Playbook 模式**，不是 Script 模式：
- 所有操作都是 **冪等的 (Idempotent)**，可以安全地重複執行
- 失敗時的正確做法是：**修正問題後重新執行 (Rerun)**，而不是 Rollback

### 為什麼不需要 Rollback

```text
Playbook Philosophy:

執行失敗 (例如網路中斷)
         │
         ▼
   ❌ 不需要 Rollback
   (會破壞已完成的正確狀態)
         │
         ▼
修正問題 (例如恢復網路)
         │
         ▼
重新執行 phoenix provision
  • 已完成的步驟會被跳過 (Check-Diff-Act)
  • 僅執行未完成的部分
  • 最終達到期望狀態
```

### 實際範例

```bash
# 第一次執行，在安裝套件時失敗 (網路問題)
$ sudo phoenix provision --secrets=secrets.yml
✓ LUKS unlocked
✓ Device mounted
✗ Package installation failed: network timeout

# 修正網路後，直接重新執行
$ sudo phoenix provision --secrets=secrets.yml
✓ LUKS already unlocked, skipping
✓ Device already mounted, skipping
✓ Installing remaining packages...  ← 接續未完成的部分
✓ Complete
```

### 正確的錯誤處理策略

Phoenix 應該專注於：

1. **明確的錯誤訊息**
   ```
   ❌ Failed to unlock LUKS device /dev/sda2
   Reason: cryptsetup returned exit code 2
   Hint: Check if the password is correct
   ```

2. **快速失敗 (Fail Fast)**
   - 遇到無法恢復的錯誤時立即停止
   - 不要嘗試「猜測」使用者的意圖

3. **Idempotent Operations**
   - 所有 Acts 都遵循 Check-Diff-Act 模式 (參見 ADR-0005)
   - 確保重新執行是安全的

---

## 📝 核心哲學

**Phoenix 是 Playbook，不是 Transaction**

- **Declarative**, not Imperative (宣告式，非命令式)
- **Idempotent**, not Transactional (冪等性，非事務性)
- **Rerun**, not Rollback (重新執行，非回滾)

當 AI Agent 或開發者想要實作「錯誤處理」功能時，應該問自己：
> "這個功能是為了支援 Playbook 模式的冪等性重執行，還是試圖模仿 Transaction 模式的 rollback？"

如果是後者，**不要實作**。
