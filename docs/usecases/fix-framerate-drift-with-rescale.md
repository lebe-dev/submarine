# Fix frame-rate drift (23.976 ↔ 25)

> The sync is fine at the start but drifts further off toward the end — a frame-rate mismatch, not a constant offset. Stretch the timeline instead of shifting it.

## Scenario

Subtitles authored for a 25 fps (PAL) release are played against a 23.976 fps (NTSC/film)
encode, or vice versa. Early lines look almost right, but the error grows linearly until
the last lines are seconds off. No single `delay` can fix this — the timeline needs to be
*scaled*.

## You'll use

- `sm detect-offset` — confirm it is drift, not a constant offset
- `sm rescale` ★ — stretch/compress the timeline by factor, fps pair, or two anchors

## Fixtures

`movie.srt` — three cues spread across 20 minutes:

```srt
1
00:00:10,000 --> 00:00:12,000
Opening line.

2
00:10:00,000 --> 00:10:02,000
Midpoint line.

3
00:20:00,000 --> 00:20:02,000
Closing line.
```

## Walkthrough

### 1. Confirm it's drift, not offset

Compare the out-of-sync file against an in-sync reference:

```bash
sm detect-offset stretched.srt movie.srt
```

```
Median offset (ms): -25620
Stddev (ms):        20745

Note: drift detected (offset varies across the timeline); consider 'sm rescale'.
```

The giveaway is a **large `Stddev`** and the explicit **"drift detected"** note: the
measured offset is tiny near the start and huge near the end. That variance is the
signature of a frame-rate mismatch — reach for `rescale`, not `delay`.

### 2. Rescale by fps pair

If you know the source and target frame rates, name them directly:

```bash
sm rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt --dry-run
```

```
Dry-run: rescale would be applied

Mode: fps
Factor: 1.0427093760427095
Offset: 0 ms
Subtitles affected: 3
Output: ntsc.srt

Sample (first 3 subtitles):
  [1] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,427 --> 00:00:12,513
  [2] 00:10:00,000 --> 00:10:02,000  =>  00:10:25,626 --> 00:10:27,711
  [3] 00:20:00,000 --> 00:20:02,000  =>  00:20:51,251 --> 00:20:53,337
```

The correction grows with time — 0.4 s at the start, ~51 s at the 20-minute mark — exactly
undoing linear drift. `sm` derives the factor (25 / 23.976) for you.

### 3. Or by explicit factor

When you already know the multiplier:

```bash
sm rescale movie.srt --factor 1.0427 --out scaled.srt --dry-run
```

```
Mode: factor
Factor: 1.0427
```

### 4. Or by two anchor points

When you do not know the fps but can identify two cues and the timecodes they *should*
have, give two `--anchor IDX=HH:MM:SS,mmm` points. `sm` solves the linear transform that
maps each cue to its target (it computes both a scale factor and a small offset):

```bash
sm rescale movie.srt --anchor "1=00:00:09,592" --anchor "3=00:19:10,096" --out anchored.srt --dry-run
```

```
Dry-run: rescale would be applied

Mode: anchor
Factor: 0.9584067226890757
Offset: 8 ms
Subtitles affected: 3
Output: anchored.srt

Sample (first 3 subtitles):
  [1] 00:00:10,000 --> 00:00:12,000  =>  00:00:09,592 --> 00:00:11,509
  [2] 00:10:00,000 --> 00:10:02,000  =>  00:09:35,052 --> 00:09:36,969
  [3] 00:20:00,000 --> 00:20:02,000  =>  00:19:10,096 --> 00:19:12,013
```

Cues 1 and 3 land exactly on their requested timecodes; everything between is
interpolated.

### 5. Apply it

```bash
sm rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt
```

```
✓ Rescale applied successfully

Mode: fps
Factor: 1.0427093760427095
Offset: 0 ms
Subtitles rescaled: 3
Output: ntsc.srt
Backup: N/A (new file)
```

## JSON output

```bash
sm --output json rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt --dry-run
```

```json
{"ok":true,"data":{"file":"movie.srt","mode":"fps","factor":1.0427093760427095,"offset_ms":0,"total_count":3,"output":"ntsc.srt","backup_path":"N/A (dry-run)","dry_run":true}}
```

- `mode` — `factor`, `fps`, or `anchor`.
- `factor` / `offset_ms` — the resolved linear transform `t' = factor·t + offset`.
- `output` — the result file.

## Pitfalls & edge cases

- **Provide exactly one mode.** `--factor`, the `--from-fps`/`--to-fps` pair, or two
  `--anchor` values — combining or omitting them errors with *"specify one of …"*.
- **`--out`, not `--output`.** `--out` is the result file; `--output text|json` is the
  global format flag.
- **Anchor timestamps contain a comma** (`HH:MM:SS,mmm`). Quote each `--anchor` argument so
  your shell passes it intact.
- **If `Stddev` is near zero**, it is not drift — use [delay](detect-and-fix-constant-offset.md).

## See also

- [Detect and fix a constant sync offset](detect-and-fix-constant-offset.md)
- [Re-sync only part of a file](resync-a-partial-range.md)
- `sm rescale --help`, `sm detect-offset --help`, `sm describe`
