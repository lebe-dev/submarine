# Re-sync only part of a file

> A shift appeared partway through — after a splice or an inserted scene — so only the later cues need moving.

## Scenario

The first half of the file is in sync, but somewhere in the middle (often where two
sources were spliced) the timing slips and stays off for the rest. Shifting the whole file
would break the part that is already correct. You want to move only the affected range.

## You'll use

- `sm detect-offset` — measure the slip on the misaligned section
- `sm delay` ★ — shift only a range of cues, by index or by timestamp

## Fixtures

`joined.srt` — cues 1–2 are fine, cues 3–4 came in 500 ms early after a splice:

```srt
1
00:00:01,000 --> 00:00:03,000
Part one line A.

2
00:00:04,000 --> 00:00:06,000
Part one line B.

3
00:00:10,000 --> 00:00:12,000
Part two line A.

4
00:00:13,000 --> 00:00:15,000
Part two line B.
```

## Walkthrough

### 1. Shift by index range

Move cues 3 through 4 by +500 ms, leaving 1–2 untouched. Preview first:

```bash
sm delay --range 3-4 --dry-run joined.srt +500
```

```
Dry-run: time offset would be applied

Offset: 500 ms
Range: 3-4
Subtitles affected: 2

Sample (first 3 subtitles):
  [1] 00:00:01,000 --> 00:00:03,000  =>  00:00:01,000 --> 00:00:03,000
  [2] 00:00:04,000 --> 00:00:06,000  =>  00:00:04,000 --> 00:00:06,000
  [3] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,500 --> 00:00:12,500
```

Only cues 3–4 move (`Subtitles affected: 2`); the sample confirms 1 and 2 are unchanged.

### 2. Or shift by timestamp

When you know the moment but not the index, use `--from-timestamp` — every cue at or after
that time is shifted:

```bash
sm delay --from-timestamp 00:00:09,000 --dry-run joined.srt +500
```

```
Dry-run: time offset would be applied

Offset: 500 ms
From timestamp: 00:00:09,000
Subtitles affected: 2

Sample (first 3 subtitles):
  [1] 00:00:01,000 --> 00:00:03,000  =>  00:00:01,000 --> 00:00:03,000
  [2] 00:00:04,000 --> 00:00:06,000  =>  00:00:04,000 --> 00:00:06,000
  [3] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,500 --> 00:00:12,500
```

### 3. Apply it

```bash
sm delay --range 3-4 joined.srt +500
```

Drop `--dry-run` to write the change; a backup is saved to `backups/` first.

> **Tip for negative offsets:** put the flags **before** the offset
> (`sm delay --range 3-4 joined.srt -500`). A leading `-` makes the offset look like a
> flag, so flags placed after it would be ignored.

## JSON output

```bash
sm --output json delay --range 3-4 joined.srt +500
```

```json
{"ok":true,"data":{"offset_ms":500,"subtitles_adjusted":2,"backup_path":"backups/joined.srt.2026-06-14_10-34-15.001252","range_start":3,"range_end":4}}
```

- `subtitles_adjusted` — how many cues moved.
- `range_start` / `range_end` — the index range that was shifted (omitted when you use
  `--from-timestamp`).

## Pitfalls & edge cases

- **`--range` and `--from-timestamp` are mutually exclusive:**

  ```bash
  sm delay --range 3-4 --from-timestamp 00:00:09,000 joined.srt +500
  ```
  ```
  error: use either --range or --from-timestamp, not both
  ```

- **With neither flag, the whole file shifts** — that is the
  [constant-offset](detect-and-fix-constant-offset.md) case.
- **`--range` is inclusive** and uses subtitle indices (`START-END`), not timestamps.
- **Measure on the bad section only.** Run `sm detect-offset` against an in-sync reference
  using just the misaligned part so the offset is not diluted by the correct first half.

## See also

- [Detect and fix a constant sync offset](detect-and-fix-constant-offset.md)
- [Join split parts (CD1/CD2) into one file](join-split-parts-cd1-cd2.md)
- `sm delay --help`, `sm detect-offset --help`, `sm describe`
