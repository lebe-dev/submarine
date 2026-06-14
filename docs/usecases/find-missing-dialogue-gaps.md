# Find missing dialogue with gap analysis

> Spot the silent stretches in a subtitle file where dialogue may have been dropped.

## Scenario

A translation feels incomplete — a scene seems to have no subtitles. Rather than scrub
the video, you can list the time gaps between consecutive cues and inspect the suspicious
ones, then check a donor file to see whether dialogue really is missing there.

## You'll use

- `sm gaps` ★ — list silences longer than a threshold
- `sm info` — get the file's average gap to pick a threshold
- `sm diff` — check whether a donor has dialogue in the gap

## Walkthrough

This example uses a fixture shipped with the project,
`test-data/valid/complex.srt`.

### 1. List the gaps

```bash
sm gaps test-data/valid/complex.srt --min-gap 1500
```

```
after index 2: 00:00:05,135 --> 00:00:07,790, duration 2.655s
```

Each line reports a silence: `after index 2` is the cue it follows, then the window
(`end of cue 2 → start of the next`) and its length. A 2.655-second gap after line 2 is
the place to investigate.

### 2. Choosing `--min-gap`

`--min-gap` is the floor (in milliseconds, default `1000`) below which gaps are ignored.
Set it just above the file's typical pause so only the unusual silences surface. `sm info`
gives you that baseline:

```bash
sm info test-data/valid/complex.srt
```

Look at **Average gap between subtitles**. If a file's average pause is ~9 s, a dropped
scene shows up as a much larger gap — start with `--min-gap 15000` to highlight only the
real holes, then lower it if you suspect smaller omissions.

### 3. Confirm against a donor

A gap alone does not prove dialogue is missing — there may simply be no speech. Compare
against a fuller track to be sure:

```bash
sm diff test-data/valid/complex.srt donor.srt --by time
```

Lines reported as **Only in B** that fall inside the gap are dialogue your file lacks —
exactly the input for [a donor merge](merge-incomplete-translation-with-donor.md).

## JSON output

```bash
sm --output json gaps test-data/valid/complex.srt --min-gap 1500
```

```json
{"ok":true,"data":{"file":"test-data/valid/complex.srt","min_gap_ms":1500,"count":1,"gaps":[{"after_index":2,"start":"00:00:05,135","end":"00:00:07,790","duration_ms":2655}]}}
```

- `count` — number of gaps at or above `min_gap_ms`.
- `gaps[].after_index` — the cue index the gap follows.
- `gaps[].start` / `end` — the silent window.
- `gaps[].duration_ms` — its length in milliseconds.

## Pitfalls & edge cases

- **A gap is not proof of a missing line.** Pauses, scene changes, and music are normal —
  cross-check with `sm diff` before concluding dialogue was dropped.
- **Threshold is in milliseconds.** `--min-gap 1500` is 1.5 s, not 1500 s.
- **Overlapping cues produce no gap** (the window would be negative), so they never appear
  in the report; clean those up with [normalize](normalize-structure.md).

## See also

- [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md)
- [Compare two subtitle tracks](compare-two-subtitle-tracks.md)
- `sm gaps --help`, `sm info --help`, `sm describe`
