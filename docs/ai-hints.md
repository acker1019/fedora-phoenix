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

**Last Updated:** 2025-12-26
