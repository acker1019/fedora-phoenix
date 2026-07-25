# Wishlist (未來功能清單)

> **目的**: 記錄「想做但不急」的功能
> **優先級**: 低 (MVP 之後再考慮)

---

## 🎯 Dry-Run Mode

### 概述
在不實際執行變更的情況下，預覽 Trisolaran 會做什麼。

### 使用場景
```bash
# 預覽會執行哪些操作
sudo tri rehydra --secrets=secrets.yml --dry-run

# 預期輸出:
# 🔍 DRY-RUN MODE (no changes will be made)
#
# Would perform the following actions:
# ✓ LUKS device /dev/sda2 is already unlocked
# ✓ Device already mounted at /mnt/company_data
# → Would install 3 packages: vim, git, tmux
# → Would enable 2 services: sshd, docker
# → Would clone 1 repository to ~/projects/dotfiles
#
# Summary: 0 skipped, 6 actions would be executed
```

### 實作考量
- 需要在每個 Act 中區分「Check」和「Act」步驟
- Dry-run 只執行 Check，不執行 Act
- 輸出應該清楚標示哪些是「已存在」、哪些是「將會執行」

### 優先級
**Low** - MVP 後再實作

### 相關 ADR
- 需要確保與 ADR-0005 (Idempotency Pattern) 的 Check-Diff-Act 結構相容

---

## 🔐 LUKS-Encrypted Swap

### 概述
支援設定 LUKS 加密的 Swap 分割區。

### 優先級
**Medium** - 安全性需求

### 相關 ADR
- ADR-0002 (Block Architecture) - Block II: Infrastructure

---

## 🔒 TPM Management

### 概述
支援 TPM (Trusted Platform Module) 相關操作與管理。

### 優先級
**Medium** - 安全性與硬體整合需求

### 相關 ADR
- ADR-0002 (Block Architecture) - Block II: Infrastructure
