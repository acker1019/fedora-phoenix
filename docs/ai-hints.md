# AI Hints for Fedora Phoenix

> **Purpose:** This document contains guidelines and conventions for AI assistants working on the Fedora Phoenix project.

---

## 📝 Documentation Guidelines

### ADR Files (Architecture Decision Records)

- **File Naming Convention**: `adr-XXXX-short-title.md`
  - Example: `adr-0001-pure-go-strategy.md`
  - Example: `adr-0002-block-architecture.md`

- **Title Format**: Use concise English titles that match the filename
  - Format: `# ADR XXXX: Title`
  - Example: `# ADR 0001: Pure Go Strategy`

- **Related Documents Section**: ❌ **DO NOT** include "Related Documents" sections in ADR files
  - ADRs should be self-contained
  - Cross-references should be inline using markdown links

### Acts List (`act-list.md`)

- **Related Documents Section**: ❌ **DO NOT** include "Related Documents" section
  - The act-list is a reference document
  - ADR references should be inline within individual act descriptions

---

## 🎨 Content Philosophy

### Avoid Over-Specification

When writing documentation (especially ADRs):

- **Focus on "Why" not "What"**: Document design principles, not implementation details
- **Avoid concrete field examples**: Don't list specific YAML fields or struct members
- **Use abstract descriptions**: Describe concepts at a high level
- **Minimize future maintenance**: Concrete examples require updates when implementation changes

**Example:**

❌ **Bad** (Too Specific):
```yaml
infrastructure:
  luks:
    device: "/dev/nvme0n1p4"
    mapper_name: "company_data"
```

✅ **Good** (Abstract):
> Describes底層儲存與硬體資源的對應關係，讓 Engine 能夠適應不同的硬體分區規劃。

### Wishlist Items

When adding items to [wishlist.md](wishlist.md):

- **Concept Only**: Record only the high-level concept and purpose
- **No Implementation Details**: Do NOT elaborate on:
  - Specific CLI flags or commands
  - Configuration schema or YAML examples
  - Code structure or function signatures
  - Step-by-step implementation plans
- **Minimalist Approach**: Each wishlist item should contain:
  - 概述 (Overview): 1-2 sentences describing what the feature does
  - 優先級 (Priority): Low/Medium/High
  - 相關 ADR (Related ADRs): Optional references

**Example:**

❌ **Bad** (Too Detailed):

```markdown
## 🔐 LUKS-Encrypted Swap

### Implementation
- Mode 1: Dedicated partition
- Mode 2: Swap file
- Schema: system.swap.luks.device, system.swap.luks.size
```

✅ **Good** (Concept Only):

```markdown
## 🔐 LUKS-Encrypted Swap

### 概述
支援設定 LUKS 加密的 Swap 分割區。

### 優先級
**Medium** - 安全性需求
```

---

## 🧪 Testing Policy

- **Never proactively bring up testing** — do not suggest, offer, or mention writing/running tests unless the user explicitly asks for it.

---

## 🔀 Commit Workflow

Never proactively ask or suggest whether to commit. Only take any commit-related action (staging, committing, drafting messages) when the user explicitly asks for a commit.

When the user does ask for a commit, there are two scenarios:

### Scenario A: User already staged files

Trigger: user says something like "commit the staged files" — they ran `git add` themselves first.

- Inspect the staged diff and commit it as a **single commit**.
- If the user described the commit content in Chinese, translate it into an **English** commit message.
- If the user gave no description, write the commit message yourself based on the diff and surrounding context.
- Execute the commit directly — no need to ask for confirmation in this scenario.

### Scenario B: Nothing staged yet

Trigger: user asks for a commit but hasn't staged anything.

- Inspect the full diff and split it into multiple commits by **atomicity of change** (each commit should represent one coherent, self-contained change).
- Draft the commit message for each proposed commit (English).
- **List the proposed split + messages for the user to review — do NOT commit yet.**
- Only execute the commits after the user confirms (they may edit the split or messages first).

---

## 🔧 Code Conventions

### Logging

- Use the project's logging package: `internal/logging`
- Create package-level logger: `var log = logging.WithSource("package-name")`
- Replace `fmt.Printf/Println` with appropriate log methods:
  - `log.Infof()` for informational messages
  - `log.Warnf()` for warnings
  - `log.Errorf()` for errors (but use `fmt.Errorf()` for error construction)


### 測試產物生成

- 所有測試需要編譯 binary 時，一律生成於 `/bin/` 目錄。
  - 例如：`go build -o /bin/test-binary ...`

### Naming Conventions


- **AI 生成報告/分析檔案命名**：
  - 任何 AI 協助生成的報告或分析檔案，無論副檔名與檔名為何，一律採用 `*.report.*` 格式。
  - 例如：`summary.report.md`、`security-check.report.txt`、`2025-12-27.report.json`

- **Blueprint over Manifest**: Use "blueprint" terminology for configuration files
  - Struct name: `config.Blueprint` (not `config.Manifest`)
  - File: `internal/config/blueprint.go`
  - Variable: `blueprint` (not `manifest`)

---

## 📚 Project-Specific Terms

| Prefer | Avoid | Context |
|--------|-------|---------|
| Blueprint | Manifest | Configuration schema |
| Acts | Functions/Operations | Atomic operations |
| Block | Module/Component | Architecture layers |
| Phoenix Protocol | System | Overall framework name |

---

## ✅ Quality Checklist

Before completing documentation work:

- [ ] File names follow the `adr-XXXX-title.md` convention
- [ ] No "Related Documents" sections in ADRs or act-list
- [ ] Cross-references use updated filenames
- [ ] Content focuses on principles over implementation details
- [ ] All code uses the logging package (no fmt.Printf for logs)
- [ ] "Blueprint" terminology used consistently

---

**Last Updated:** 2026-07-25 (commit workflow added)
