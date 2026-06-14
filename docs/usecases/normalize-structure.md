# Normalize structure: sort, renumber, fix overlaps

> After manual or scripted edits, put a valid SRT back in order — sorted by time, renumbered from 1, with overlaps trimmed.

## Scenario

Hand-editing or a quick script left the file out of order: indices no longer ascend,
cues are not sorted by start time, and a couple of neighbours overlap. The file still
parses, but it is messy. `sm normalize` rewrites it into canonical shape.

## You'll use

- `sm normalize` ★ — sort, renumber, and optionally fix overlaps
- `sm doctor` — for the different job of repairing structural parse errors

## Fixtures

`scram.srt` — out of order, with the first cue overlapping the second:

```srt
3
00:00:08,000 --> 00:00:10,000
Third in time.

1
00:00:01,000 --> 00:00:04,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.
```

Cue `1` ends at `00:00:04,000` but cue `2` starts at `00:00:03,000` — a 1-second overlap.

## Walkthrough

### 1. Preview with `--dry-run`

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```
Dry-run: normalization would be applied

Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

`Overlaps fixed: 1` is the one neighbour pair that will be trimmed. Nothing is written.

### 2. Apply it

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps
```

```
✓ Subtitles normalized successfully

Backup: backups/scram.srt.2026-06-14_10-36-09.226844
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

The file is rewritten in place (a backup is saved to `backups/` first). Cues are now
sorted, renumbered from 1, and the overlap is resolved by clipping cue 1's end to cue 2's
start:

```srt
1
00:00:01,000 --> 00:00:03,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.

3
00:00:08,000 --> 00:00:10,000
Third in time.
```

### Defaults

`--sort` and `--renumber` are **on by default**, so a bare `sm normalize scram.srt`
already reorders and renumbers — but it leaves overlaps alone:

```bash
sm normalize scram.srt --dry-run
```

```
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: no
```

Add `--fix-overlaps` only when you want neighbour timings trimmed.

## JSON output

```bash
sm --output json normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```json
{"ok":true,"data":{"file":"scram.srt","total_count":3,"sorted":true,"renumbered":true,"overlaps_fixed":1,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `sorted` / `renumbered` — whether each pass ran.
- `overlaps_fixed` — count of neighbour overlaps trimmed.
- `total_count` — cues after normalization.

## `normalize` vs `doctor`

- `sm doctor` **diagnoses and repairs structural problems** — malformed timecodes,
  broken cue blocks, parse errors. Reach for it when a file will not load cleanly.
- `sm normalize` assumes the file is already **valid** and reorders, renumbers, and
  de-overlaps it. Run `doctor` first if parsing fails, then `normalize` to tidy.

## Pitfalls & edge cases

- **`--fix-overlaps` changes timings**, not just order — it clips a cue's end to the next
  cue's start. Preview with `--dry-run`.
- **Disable a default explicitly** with `--sort=false` / `--renumber=false` if you want to
  renumber without resorting, or vice versa.
- **Renumbering after a merge or manual splice** is the common case — combine with
  [dedupe](deduplicate-and-clean-after-merge.md) when near-duplicates also crept in.

## See also

- [Remove duplicates and overlaps after merging](deduplicate-and-clean-after-merge.md)
- [Join split parts (CD1/CD2) into one file](join-split-parts-cd1-cd2.md)
- `sm normalize --help`, `sm doctor --help`, `sm describe`
