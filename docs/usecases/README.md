# Use cases

Task-oriented recipes for real subtitle problems. Each page states a problem, then walks
through the exact `sm` commands that solve it — with copy-paste fixtures, the expected
output, a `--dry-run` preview before any mutation, and a `--output json` example for
scripting.

These are recipes, not a flag reference. For the full option list of any command, use
`sm <command> --help`; for machine-readable schemas, use `sm describe`.

## By task

### Sync & timing

| Use case | Solves | Main command |
|---|---|---|
| [Detect and fix a constant sync offset](detect-and-fix-constant-offset.md) | Subtitles are consistently early or late by a fixed amount | `detect-offset` → `delay` |
| [Fix frame-rate drift (23.976 ↔ 25)](fix-framerate-drift-with-rescale.md) | Sync drifts further off toward the end (fps mismatch) | `rescale` |
| [Re-sync only part of a file](resync-a-partial-range.md) | A shift appears only after a certain point or splice | `delay --range` / `--from-timestamp` |

### Merging & joining

| Use case | Solves | Main command |
|---|---|---|
| [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md) | Complete a partial translation from another file without touching trusted lines | `merge` |
| [Join split parts (CD1/CD2) into one file](join-split-parts-cd1-cd2.md) | A release split into consecutive parts needs stitching | `concat` |
| [Remove duplicates and overlaps after merging](deduplicate-and-clean-after-merge.md) | Near-duplicate lines pile up after an aggressive merge | `dedupe` |

### Inspection & structure

| Use case | Solves | Main command |
|---|---|---|
| [Find missing dialogue with gap analysis](find-missing-dialogue-gaps.md) | Locate silent stretches where dialogue may be missing | `gaps` |
| [Compare two subtitle tracks](compare-two-subtitle-tracks.md) | Understand the content difference between two tracks | `diff` |
| [Normalize structure: sort, renumber, fix overlaps](normalize-structure.md) | A valid file drifted out of order after manual edits | `normalize` |

## Conventions used in these recipes

- **`--out` is the result file; `--output text|json` is the global format flag.** They are
  different — `merge`, `rescale`, and `concat` write to `--out`.
- **Mutating commands write a backup** to `backups/` before changing a file in place, and
  all support `--dry-run` to preview first.
- **Negative offsets:** put option flags before a negative offset
  (`sm delay --dry-run file -500`), since a leading `-` otherwise looks like a flag.

See the project [README](../../README.md) for installation and the full command list.
