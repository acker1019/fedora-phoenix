# ADR 0010: TPM Auto-Enrollment for LUKS

> **Project:** Fedora Trisolaran
> **Status:** ✅ Accepted (Implementation Deferred)
> **Date:** 2026-07-26

---

## 📋 Context (背景)

Fedora 安裝器本身是否啟用 LUKS 加密，由使用者在重灌時自行勾選，這件事不歸 Trisolaran 管；`infrastructure.luks`（device / mapper_name / mount_point）也是使用者自行在 blueprint 裡填寫，schema 不需要改變。

目前每次開機解鎖都得手動輸入密碼。既然 `tri rehydra` 本來就會用 `secrets.luks_password` 執行一次解鎖，這組密碼其實可以順便拿去讓開機解鎖自動化。

---

## 🎯 Decision (決策)

`tri rehydra` 偵測到 `infrastructure.luks` 有配置時，除了現有的解鎖流程，額外自動把 `secrets.luks_password` 註冊進 TPM，讓之後開機能自動解鎖，不需要每次都手動輸入密碼。

沿用既有的 Check-Diff-Act 冪等性原則：先確認該 LUKS header 是否已經有 TPM 註冊，有就跳過，沒有才執行註冊。這樣不管是全新重灌的容器（一定沒有註冊過）還是既有容器（可能已經註冊過），行為都正確，不需要額外判斷「這次重灌是否連容器一起重建」。

---

## ⚖️ Consequences (後果)

- 新增一個 Block II 的 Act，性質與現有 `UnlockLuks` 平行，一樣是「失敗即中止」等級。
- `secrets.luks_password` 的用途從「單純這次解鎖用」擴展為「這次解鎖 + 順便註冊給以後開機自動解鎖用」，欄位本身不用變。
- 不影響 `infrastructure.luks` schema，也不影響 LUKS 選配的既有機制（[ADR 見 rehydra.go 的 `HasLuks()` 判斷]）。

---

## 💭 Status Note

本 ADR 記錄方向與決策，實作有意延後。
