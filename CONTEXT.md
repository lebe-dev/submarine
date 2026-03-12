# CONTEXT.md — Machine-readable invariants for AI agents

## Tool

- Binary: `sm`
- Version: check with `sm describe` or `sm --version`
- Output modes: `--output text` (default), `--output json`

## File Format (SubRip .srt)

- Encoding: UTF-8
- Indices: 1-based, sequential, gaps allowed
- Timestamps: `HH:MM:SS,mmm --> HH:MM:SS,mmm` (comma separates milliseconds, not dot)
- Text: may contain HTML tags (`<i>`, `<b>`, `<u>`)
- Entries separated by blank lines

## JSON Output Contract

All commands (except `compare`) support `--output json`.

Success envelope:
```json
{"ok": true, "data": { ... }}
```

Error envelope:
```json
{"ok": false, "error": {"code": "error_code", "message": "...", "hint": "..."}}
```

`hint` is optional and may be `null`. Error `code` values are stable and machine-parseable.

## Command Properties

Use `sm describe` to get the full schema. Key properties:

| Command | Mutates | `--dry-run` | JSON |
|---|---|---|---|
| get | no | no | yes |
| info | no | no | yes |
| export | no | no | yes |
| verify | no | no | yes |
| translation-status | no | no | yes |
| describe | no | no | always JSON |
| set | yes | yes | yes |
| add | yes | yes | yes |
| delay | yes | yes | yes |
| import | yes | yes | yes |
| doctor --fix | yes | no* | yes |
| mass-rename | yes | yes | yes |
| compare | no | no | no (TUI only) |

*`doctor` without `--fix` is diagnostic-only (effectively a dry-run).

## Workflow Invariants

### Translation workflow

1. `sm export <ref.srt> <range> --format anchored` — extract subtitles for translation
2. Translate the anchored file (preserving `[INDEX]` markers)
3. `sm import <target.srt> <translated.txt> --format anchored --reference <ref.srt> --dry-run` — preview
4. `sm import <target.srt> <translated.txt> --format anchored --reference <ref.srt> --force` — apply
5. `sm verify <ref.srt> <target.srt> --range <range>` — validate timestamps match

### Error recovery

- `parse_error` → run `sm doctor --fix <file>` first, then retry
- `timestamp_conflict` / `timestamp_overlap` → check subtitle ordering
- `file_not_found` → verify path; relative paths are resolved from cwd

### Safe mutation pattern

Always use `--dry-run` before mutating commands to preview changes. All mutating commands create backups automatically.

## Idempotency

- **Anchored import**: overwrites existing subtitles at matching indices (idempotent)
- **CSV import**: appends new subtitles to the end (not idempotent)
- **set**: overwrites specified fields (idempotent)
- **delay**: applies offset relative to current timestamps (not idempotent)

## Input Constraints

- Subtitle text: no ASCII control characters except `\n`, `\r`, `\t`
- File paths: no path traversal (`..`), no percent-encoded sequences (`%20`)
- Indices: positive integers >= 1
- Timestamps: non-negative, end > start within a subtitle
- Range format: `START-END` where START <= END (e.g., `1-50`)
