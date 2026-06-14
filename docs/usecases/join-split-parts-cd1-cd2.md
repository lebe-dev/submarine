# Join split parts (CD1/CD2) into one file

> A release split across two files that play back-to-back — stitch them into one, with the second part shifted to follow the first.

## Scenario

An old release ships as `CD1` and `CD2`, each starting its timecodes from zero. Played as
one video, the second file's cues need to begin where the first ends. `sm concat` appends
the parts and shifts each later part by the running total, then renumbers the result.

This differs from [merge](merge-incomplete-translation-with-donor.md): the parts do **not**
overlap in time — they run consecutively.

## You'll use

- `sm concat` ★ — append parts and shift each to follow the previous
- `sm info` — confirm the joined duration and count

## Fixtures

`part1.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
CD1 first line.

2
00:00:04,000 --> 00:00:06,000
CD1 last line.
```

`part2.srt` — also starting near zero:

```srt
1
00:00:01,000 --> 00:00:03,000
CD2 first line.

2
00:00:05,000 --> 00:00:07,000
CD2 last line.
```

## Walkthrough

### 1. Preview with `--dry-run`

`--gap` sets the silence (in ms) inserted between parts:

```bash
sm concat part1.srt part2.srt --out full.srt --gap 2000 --dry-run
```

```
Dry-run: subtitles would be concatenated

Parts: part1.srt, part2.srt
Gap: 2000 ms
Total subtitles: 4
Output: full.srt
```

### 2. Apply it

```bash
sm concat part1.srt part2.srt --out full.srt --gap 2000
```

```
✓ Subtitles concatenated successfully

Parts: part1.srt, part2.srt
Gap: 2000 ms
Total subtitles: 4
Output: full.srt
Backup: N/A (new file)
```

`full.srt` carries part 1 unchanged, then part 2 shifted to start after part 1's end plus
the gap, with indices renumbered straight through:

```srt
1
00:00:01,000 --> 00:00:03,000
CD1 first line.

2
00:00:04,000 --> 00:00:06,000
CD1 last line.

3
00:00:09,000 --> 00:00:11,000
CD2 first line.

4
00:00:13,000 --> 00:00:15,000
CD2 last line.
```

### 3. Confirm

```bash
sm info full.srt
```

Check **Total subtitles** and **Total duration** match the two parts plus the gap.

## JSON output

```bash
sm --output json concat part1.srt part2.srt --out full.srt --gap 2000 --dry-run
```

```json
{"ok":true,"data":{"parts":2,"gap_ms":2000,"total_count":4,"output":"full.srt","dry_run":true}}
```

- `parts` — number of input files joined.
- `gap_ms` — silence inserted between parts.
- `total_count` — cues in the result.

## Pitfalls & edge cases

- **Each part keeps its own leading offset.** Part 2 began at `00:00:01,000`, so after
  part 1 (ends `00:00:06,000`) plus the 2 s gap it lands at `00:00:09,000` — the part's
  own 1 s lead-in is preserved on top of the gap. Trim a part's leading silence beforehand
  if you want the gap to be exact.
- **Order matters.** Parts are joined left to right in the order you list them.
- **`--out`, not `--output`.** `--out` is the result file; `--output text|json` is the
  global format flag.
- **Overlapping, not consecutive?** If the parts share dialogue in time, you want
  [merge](merge-incomplete-translation-with-donor.md) instead.

## See also

- [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md)
- [Normalize structure: sort, renumber, fix overlaps](normalize-structure.md)
- `sm concat --help`, `sm info --help`, `sm describe`
