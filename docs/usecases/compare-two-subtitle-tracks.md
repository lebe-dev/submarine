# Compare two subtitle tracks

> See the real content difference between two subtitle files — which lines are unique to each, and which only differ in wording.

## Scenario

You have two versions of a track — two translations, an edited and an original, or a
release and a re-sync — and you want to know how they actually differ: which cues are
shared, which are unique to one side, and where the same moment carries different text.

## You'll use

- `sm diff` ★ — compare by timeline overlap or by text
- `sm verify` — for the related but different "are these structurally parallel?" check

## Fixtures

`a.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
We need to leave now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

`b.srt` — same timings, one line reworded:

```srt
1
00:00:01,000 --> 00:00:03,000
We have to go now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

## Walkthrough

### Compare by text

```bash
sm diff a.srt b.srt --by text
```

```
Diff between subtitle files
===========================

--- a.srt
+++ b.srt

Common:    1
Only in A: 1
Only in B: 1

- [1] 00:00:01,000 --> 00:00:03,000  We need to leave now.
+ [1] 00:00:01,000 --> 00:00:03,000  We have to go now.
```

`--by text` matches on normalized text. The identical "Where are the keys?" line is
`Common`; the reworded first line appears once on each side (`Only in A` / `Only in B`),
rendered as a unified-diff-style `-`/`+` pair.

### Compare by time

```bash
sm diff a.srt b.srt --by time
```

```
Diff between subtitle files
===========================

--- a.srt
+++ b.srt

Common:    2
Only in A: 0
Only in B: 0
```

`--by time` (the default) matches on timeline overlap within `--tolerance` (default
`250` ms). Both cues occupy the same windows, so everything is `Common` — even though the
wording of line 1 changed. Use this mode to find **structural** differences (added or
dropped lines, shifts); use `--by text` to find **wording** differences.

## JSON output

```bash
sm --output json diff a.srt b.srt --by text
```

```json
{"ok":true,"data":{"a_file":"a.srt","b_file":"b.srt","by":"text","tolerance_ms":250,"only_in_a":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We need to leave now.","has_html":false}],"only_in_b":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We have to go now.","has_html":false}],"common_count":1}}
```

- `by` — the mode used (`time` or `text`).
- `only_in_a` / `only_in_b` — full cue records present on just one side.
- `common_count` — cues matched on both sides.

## `diff` vs `verify`

`sm diff` answers "how do the contents differ?" — it pairs cues by overlap or text and
reports the unmatched ones. `sm verify` answers a stricter question: "are these two files
parallel by index and timestamp?", reporting a match percentage and pass/fail. For the
fixtures above, `verify` reports **SUCCESS** (both cues line up by index and timing) even
though their wording differs — which is exactly why you reach for `diff` when you care
about content:

```bash
sm verify a.srt b.srt
```

```
Matched: 2/2 (100.0%)

Verification: SUCCESS
```

## Pitfalls & edge cases

- **Pick the mode for the question.** `--by time` ignores wording; `--by text` ignores
  timing. A reworded-but-resynced line is "common" under `time` and "unique" under `text`.
- **`--tolerance` only affects `--by time`.** Widen it for releases that drift slightly.
- **Text matching is normalized** (whitespace/case), so cosmetic differences do not split
  otherwise-identical lines.

## See also

- [Find missing dialogue with gap analysis](find-missing-dialogue-gaps.md)
- [Merge an incomplete translation with a donor file](merge-incomplete-translation-with-donor.md)
- `sm diff --help`, `sm verify --help`, `sm describe`
