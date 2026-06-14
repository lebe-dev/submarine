# Merge an incomplete translation with a donor file

> Fill the holes in a partial translation with lines from another subtitle file, without touching the lines you already trust.

## Scenario

You have a hand-made translation that is missing a few lines — a scene was skipped, or
the translator stopped halfway. Another file (a "donor") covers those missing lines, but
it is shifted by a constant amount (a different release) and contains an extra line that
your file lacks. You want to keep your own wording everywhere it exists and only borrow
the donor's lines for the gaps.

This is the workflow that used to be done by ad-hoc Python scripts. With `sm` it is one
command, preceded by two quick diagnostics.

## You'll use

- `sm detect-offset` — measure the constant time shift between the two files
- `sm diff` — see exactly which lines the donor would contribute
- `sm gaps` — locate the holes in the base file
- `sm merge` ★ — perform the merge, auto-correcting the offset
- `sm info` / `sm verify` — confirm the result

## Fixtures

Create these two files to follow along.

`base.srt` — your translation, missing the line around 00:00:10:

```srt
1
00:00:01,000 --> 00:00:03,000
Good morning.

2
00:00:04,000 --> 00:00:06,000
How are you?

3
00:00:20,000 --> 00:00:22,000
See you tomorrow.
```

`donor.srt` — same dialogue, shifted +200 ms, with one extra line in the hole:

```srt
1
00:00:01,200 --> 00:00:03,200
Good morning.

2
00:00:04,200 --> 00:00:06,200
How are you?

3
00:00:10,200 --> 00:00:12,200
I missed you.

4
00:00:20,200 --> 00:00:22,200
See you tomorrow.
```

## Walkthrough

### 1. Measure the offset between the files

```bash
sm detect-offset base.srt donor.srt
```

```
Offset Detection
================

File A: base.srt
File B: donor.srt

Anchor matches:     3
Median offset (ms): 200
Stddev (ms):        0
```

`Median offset (ms): 200` means the donor runs a constant 200 ms behind the base. A
`Stddev (ms)` of `0` confirms the shift is uniform — a single offset, not progressive
drift. (With only three lines you may also see a "files share little dialogue" note;
that heuristic is meant for full-length files with hundreds of cues and can be ignored
here.)

### 2. See what the donor would add

```bash
sm diff base.srt donor.srt --by time
```

```
Diff between subtitle files
===========================

--- base.srt
+++ donor.srt

Common:    3
Only in A: 0
Only in B: 1

+ [3] 00:00:10,200 --> 00:00:12,200  I missed you.
```

Three lines overlap in time (`Common: 3`) and exactly one line exists only in the donor
(`Only in B: 1`) — the `I missed you.` cue that fills your gap. `diff --by time` matches
by timeline overlap, so the +200 ms shift does not create false "only in B" entries.

### 3. (Optional) Locate the hole in the base

```bash
sm gaps base.srt --min-gap 5000
```

```
after index 2: 00:00:06,000 --> 00:00:20,000, duration 14.000s
```

A 14-second silence after line 2 is where the missing dialogue belongs.

### 4. Preview the merge with `--dry-run`

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```
Dry-run: merge would be applied

Strategy: fill-gaps
Applied offset: 200 ms

Base subtitles:  3
Donor subtitles: 4

Would add:     1
Would skip (overlapping): 3
Would replace: 0
Total after merge: 4

Output (not written): merged.srt
```

`--auto-offset` reuses the detection from step 1 (`Applied offset: 200 ms`). The default
`fill-gaps` strategy adds only the donor line that lands in a gap (`Would add: 1`) and
skips the three that overlap your existing lines (`Would skip (overlapping): 3`). Nothing
is written yet.

### 5. Run it for real

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset
```

```
✓ Merge completed successfully

Backup: N/A (new file)
Applied offset: 200 ms
Added: 1
Skipped (overlapping): 3
Replaced: 0
Total subtitles: 4
Output: merged.srt
```

`merged.srt` now contains your three original lines plus the donor's line, shifted by the
detected offset so it sits naturally in the gap:

```srt
3
00:00:10,400 --> 00:00:12,400
I missed you.
```

When `--out` points at an existing file, `sm` writes a timestamped copy to `backups/`
before overwriting (here it is a new file, so `Backup: N/A`).

#### Choosing a strategy

- `fill-gaps` (default) — keep every base line; add donor lines only where the base is
  silent. Use this to complete a translation without altering it.
- `keep-base` — never replace; identical to fill-gaps for additions but refuses any
  overlap. Safest when the base is authoritative.
- `prefer-donor` — when a donor line overlaps a base line, replace the base text with the
  donor's. Use when the donor is the better translation.

`--overlap-tolerance` (default `250` ms) controls how much timing slack still counts as
"the same line", which absorbs small release differences left over after the offset.

### 6. Verify

```bash
sm info merged.srt
```

```
Subtitle File Information
========================

File: merged.srt

Basic Information:
  Total subtitles: 4
  Total duration:  00:00:21,000 (21.000s)
```

## JSON output

```bash
sm --output json merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```json
{"ok":true,"data":{"base_count":3,"donor_count":4,"added":1,"skipped_overlapping":3,"replaced":0,"applied_offset_ms":200,"total_count":4,"output":"merged.srt","dry_run":true}}
```

- `added` — donor lines inserted into gaps.
- `skipped_overlapping` — donor lines dropped because they overlap a base line.
- `replaced` — base lines overwritten (non-zero only with `--prefer-donor`).
- `applied_offset_ms` — the shift applied to the donor (from `--auto-offset` or `--offset`).
- `total_count` — subtitle count after the merge.

## Pitfalls & edge cases

- **`--out`, not `--output`.** `--out` is the result file; `--output text|json` is the
  global format flag. They are different.
- **Donor lines outside the base's time range are added too**, not just those in interior
  gaps. Inspect with `sm diff --by time` first.
- **Matching is by timestamp, not text**, so short repeated lines (e.g. "Thanks") in the
  donor will not be mistaken for lines elsewhere in the base.
- **Pass a manual offset with `--offset`** if `--auto-offset` misfires (e.g. too little
  shared dialogue): `sm merge base.srt donor.srt --out merged.srt --offset=-212`. Use the
  `=` form for negative values.
- **Drift, not offset?** If `sm detect-offset` reports a large `Stddev` or "drift
  detected", a single offset will not align the files — see
  [Fix frame-rate drift](fix-framerate-drift-with-rescale.md).

## See also

- [Detect and fix a constant sync offset](detect-and-fix-constant-offset.md)
- [Compare two subtitle tracks](compare-two-subtitle-tracks.md)
- [Find missing dialogue with gap analysis](find-missing-dialogue-gaps.md)
- [Remove duplicates and overlaps after merging](deduplicate-and-clean-after-merge.md)
- `sm merge --help`, `sm describe`
