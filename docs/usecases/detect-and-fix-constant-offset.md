# Detect and fix a constant sync offset

> Your subtitles are consistently a fixed amount early or late — measure the shift, then correct the whole file in one move.

## Scenario

The subtitles match the dialogue but every line appears too soon or too late by the same
amount — a classic mismatch between your subtitle file and your video release. You have a
reference you trust (another track that is in sync, or the video's own timing) and want to
shift your file to match it.

## You'll use

- `sm detect-offset` ★ — measure the shift against a reference
- `sm delay` — apply the shift to the whole file
- `sm detect-offset` again — confirm the result is ~0

## Fixtures

`reference.srt` — correctly timed:

```srt
1
00:00:10,000 --> 00:00:12,000
Good morning.

2
00:00:14,000 --> 00:00:16,000
How are you?

3
00:00:20,000 --> 00:00:22,000
See you tomorrow.
```

`subs.srt` — the same dialogue, running 2.5 seconds late:

```srt
1
00:00:12,500 --> 00:00:14,500
Good morning.

2
00:00:16,500 --> 00:00:18,500
How are you?

3
00:00:22,500 --> 00:00:24,500
See you tomorrow.
```

## Walkthrough

### 1. Measure the offset

```bash
sm detect-offset subs.srt reference.srt
```

```
Offset Detection
================

File A: subs.srt
File B: reference.srt

Anchor matches:     3
Median offset (ms): -2500
Stddev (ms):        0
```

`Median offset (ms): -2500` reads as "file A is 2500 ms behind file B". `Stddev: 0`
confirms a clean constant offset (no drift), so a single shift will fix it. To move
`subs.srt` back into sync you apply the negation of that lag: **−2500 ms**.

### 2. Preview the shift with `--dry-run`

```bash
sm delay --dry-run subs.srt -2500
```

```
Dry-run: time offset would be applied

Offset: -2500 ms
Subtitles affected: 3

Sample (first 3 subtitles):
  [1] 00:00:12,500 --> 00:00:14,500  =>  00:00:10,000 --> 00:00:12,000
  [2] 00:00:16,500 --> 00:00:18,500  =>  00:00:14,000 --> 00:00:16,000
  [3] 00:00:22,500 --> 00:00:24,500  =>  00:00:20,000 --> 00:00:22,000
```

The sample shows every line landing exactly on the reference timings. Nothing is written.

> **Flag order matters with a negative offset.** A leading `-` makes the offset look like
> a flag, so put option flags (`--dry-run`, `--range`, `--from-timestamp`) **before** the
> offset: `sm delay --dry-run subs.srt -2500`. Positive offsets (`+2500`) are not
> affected.

### 3. Apply it

```bash
sm delay subs.srt -2500
```

```
✓ Time offset applied successfully

Backup: backups/subs.srt.2026-06-14_10-43-26.541735
Offset: -2500 ms
Subtitles adjusted: 3
```

`sm delay` mutates the file in place, writing a timestamped copy to `backups/` first.

### 4. Confirm

```bash
sm detect-offset subs.srt reference.srt
```

```
Median offset (ms): 0
```

A median of `0` means the files are now in sync.

## JSON output

```bash
sm --output json delay subs.srt -2500
```

```json
{"ok":true,"data":{"offset_ms":-2500,"subtitles_adjusted":1,"backup_path":"backups/subs.srt.2026-06-14_10-43-26.549470"}}
```

- `offset_ms` — the shift applied.
- `subtitles_adjusted` — how many cues moved.
- `backup_path` — where the pre-change copy was written (`N/A (dry-run)` under `--dry-run`).

And the detection envelope that drives this whole flow:

```bash
sm --output json detect-offset subs.srt reference.srt
```

```json
{"ok":true,"data":{"anchor_matches":3,"median_offset_ms":-2500,"stddev_ms":0,"drift_detected":false,"same_video":false}}
```

`drift_detected:false` is the signal that a single `delay` is the right tool.

## Pitfalls & edge cases

- **Offset must carry a sign:** `+100` or `-500`. A bare `100` is rejected.
- **Shifting earlier can hit zero.** `sm delay` refuses an offset that would push any line
  before `00:00:00,000`. Re-check with `--dry-run` if you see a negative-timestamp error.
- **High `Stddev` / "drift detected" means this is the wrong fix** — the lag grows over
  time. Use [rescale](fix-framerate-drift-with-rescale.md) instead.
- **Only part of the file is off?** See [Re-sync only part of a file](resync-a-partial-range.md).

## See also

- [Re-sync only part of a file](resync-a-partial-range.md)
- [Fix frame-rate drift (23.976 ↔ 25)](fix-framerate-drift-with-rescale.md)
- [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md)
- `sm detect-offset --help`, `sm delay --help`, `sm describe`
