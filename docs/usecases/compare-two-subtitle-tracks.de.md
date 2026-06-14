# Zwei Untertitelspuren vergleichen

> Den tatsächlichen inhaltlichen Unterschied zwischen zwei Untertiteldateien sehen — welche Zeilen in jeder einzigartig sind und welche sich nur in der Formulierung unterscheiden.

## Szenario

Es liegen zwei Versionen einer Spur vor — zwei Übersetzungen, eine bearbeitete und eine originale, oder ein Release und eine Neusynchronisation — und man möchte wissen, wie sie sich tatsächlich unterscheiden: welche Cues geteilt werden, welche nur auf einer Seite vorhanden sind und wo derselbe Moment einen anderen Text trägt.

## Verwendete Befehle

- `sm diff` ★ — Vergleich nach Zeitüberlappung oder nach Text
- `sm verify` — für die verwandte, aber andere Frage "sind diese strukturell parallel?"

## Beispieldateien

`a.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
We need to leave now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

`b.srt` — gleiche Timings, eine Zeile umformuliert:

```srt
1
00:00:01,000 --> 00:00:03,000
We have to go now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

## Schritt für Schritt

### Vergleich nach Text

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

`--by text` vergleicht anhand des normalisierten Textes. Die identische Zeile "Where are the keys?" ist
`Common`; die umformulierte erste Zeile erscheint einmal auf jeder Seite (`Only in A` / `Only in B`),
dargestellt als `-`/`+`-Paar im Unified-Diff-Stil.

### Vergleich nach Zeit

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

`--by time` (die Standardeinstellung) vergleicht anhand der Zeitüberlappung innerhalb von `--tolerance` (Standard
`250` ms). Beide Cues belegen dieselben Fenster, daher ist alles `Common` — obwohl die
Formulierung von Zeile 1 geändert wurde. Dieser Modus findet **strukturelle** Unterschiede (hinzugefügte oder
weggelassene Zeilen, Verschiebungen); `--by text` findet **Formulierungs**unterschiede.

## JSON-Ausgabe

```bash
sm --output json diff a.srt b.srt --by text
```

```json
{"ok":true,"data":{"a_file":"a.srt","b_file":"b.srt","by":"text","tolerance_ms":250,"only_in_a":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We need to leave now.","has_html":false}],"only_in_b":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We have to go now.","has_html":false}],"common_count":1}}
```

- `by` — der verwendete Modus (`time` oder `text`).
- `only_in_a` / `only_in_b` — vollständige Cue-Datensätze, die nur auf einer Seite vorhanden sind.
- `common_count` — Cues, die auf beiden Seiten übereinstimmen.

## `diff` vs. `verify`

`sm diff` beantwortet die Frage "wie unterscheiden sich die Inhalte?" — es verknüpft Cues nach Überlappung oder Text
und meldet die nicht übereinstimmenden. `sm verify` beantwortet eine strengere Frage: "sind diese zwei Dateien
parallel nach Index und Zeitstempel?", mit einer Übereinstimmungsprozentzahl und Bestanden/Nicht bestanden. Für die
obigen Beispieldateien meldet `verify` **SUCCESS** (beide Cues stimmen nach Index und Timing überein), obwohl
ihre Formulierung abweicht — weshalb man genau dann zu `diff` greift, wenn der Inhalt interessiert:

```bash
sm verify a.srt b.srt
```

```
Matched: 2/2 (100.0%)

Verification: SUCCESS
```

## Fallstricke & Randfälle

- **Den Modus zur Frage passend wählen.** `--by time` ignoriert Formulierungen; `--by text` ignoriert
  Timing. Eine umformulierte, aber neu synchronisierte Zeile ist "common" unter `time` und "unique" unter `text`.
- **`--tolerance` betrifft nur `--by time`.** Für Releases mit leichtem Drift vergrößern.
- **Textvergleich ist normalisiert** (Leerzeichen/Groß-Kleinschreibung), sodass kosmetische Unterschiede identisch
  aussehende Zeilen nicht trennen.

## Siehe auch

- [Fehlende Dialoge mit Lückenanalyse finden](find-missing-dialogue-gaps.de.md)
- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md)
- `sm diff --help`, `sm verify --help`, `sm describe`
