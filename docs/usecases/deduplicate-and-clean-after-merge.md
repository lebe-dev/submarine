# Remove duplicates and overlaps after merging

> After an aggressive merge or splice, near-identical lines pile up at almost the same time — collapse them, then tidy the file.

## Scenario

Combining tracks left the file with near-duplicates: the same line appears twice within a
few hundred milliseconds, sometimes once with `<i>` formatting and once without.
`sm dedupe` collapses cues that share the same (normalized) text and overlap in time,
keeping one merged cue; `sm normalize` then re-sorts and renumbers.

## You'll use

- `sm dedupe` ★ — collapse near-duplicate cues
- `sm normalize` — final sort and renumber

## Fixtures

`dup.srt` — each line appears twice, ~50–100 ms apart, one copy with HTML:

```srt
1
00:00:01,000 --> 00:00:03,000
<i>Hello there.</i>

2
00:00:01,100 --> 00:00:03,100
Hello there.

3
00:00:05,000 --> 00:00:07,000
A unique line.

4
00:00:05,050 --> 00:00:07,050
A unique line.
```

## Walkthrough

### 1. Preview with `--dry-run`

`--time-tolerance` is how far apart (in ms) two cues with the same text may still count as
the same line; `--ignore-html` strips tags before comparing, so `<i>Hello there.</i>` and
`Hello there.` match:

```bash
sm dedupe dup.srt --time-tolerance 200 --ignore-html --dry-run
```

```
Dry-run: duplicates would be removed

Time tolerance: 200 ms
Original subtitles: 4
Removed: 2
Merged groups: 2
Final subtitles: 2
```

Two pairs collapse into two cues: `Removed: 2`, `Merged groups: 2`, `Final subtitles: 2`.

### 2. Apply it

```bash
sm dedupe dup.srt --time-tolerance 200 --ignore-html
```

```
✓ Duplicates removed successfully

Backup: backups/dup.srt.2026-06-14_10-36-09.200473
Original subtitles: 4
Removed: 2
Merged groups: 2
Final subtitles: 2
```

Each merged cue keeps the first occurrence's text and spans the union of the two windows
(earliest start, latest end):

```srt
1
00:00:01,000 --> 00:00:03,100
<i>Hello there.</i>

2
00:00:05,000 --> 00:00:07,050
A unique line.
```

### 3. Tidy up

Renumber and re-sort the cleaned file:

```bash
sm normalize dup.srt
```

See [Normalize structure](normalize-structure.md) for details.

## JSON output

```bash
sm --output json dedupe dup.srt --time-tolerance 200 --ignore-html --dry-run
```

```json
{"ok":true,"data":{"file":"dup.srt","original_count":4,"removed":2,"merged":2,"final_count":2,"time_tolerance_ms":200,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `original_count` / `final_count` — cue count before and after.
- `removed` — cues dropped as duplicates.
- `merged` — number of groups collapsed.

## Pitfalls & edge cases

- **`--time-tolerance 0` requires overlapping windows** to merge — only cues that actually
  intersect in time and share text collapse. Widen the tolerance to catch copies that sit
  just apart.
- **Without `--ignore-html`, formatting differences keep cues separate** —
  `<i>Hello</i>` and `Hello` are treated as different text.
- **The first cue's text wins** in a merged group, so an italicized copy may survive over a
  plain one (or vice versa) depending on order. Run [normalize](normalize-structure.md)
  afterward to renumber.

## See also

- [Normalize structure: sort, renumber, fix overlaps](normalize-structure.md)
- [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md)
- `sm dedupe --help`, `sm normalize --help`, `sm describe`
