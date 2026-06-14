# Duplikate und Überlappungen nach dem Zusammenführen entfernen

> Nach einem aggressiven Merge oder Schnitt häufen sich fast identische Zeilen zum fast gleichen Zeitpunkt an — sie zusammenfassen und dann die Datei aufräumen.

## Szenario

Das Kombinieren von Spuren hinterließ die Datei mit Fast-Duplikaten: dieselbe Zeile erscheint zweimal
innerhalb weniger hundert Millisekunden, manchmal einmal mit `<i>`-Formatierung und einmal ohne.
`sm dedupe` fasst Cues zusammen, die denselben (normalisierten) Text teilen und sich zeitlich überlappen,
und behält dabei einen zusammengeführten Cue; `sm normalize` sortiert und nummeriert danach neu.

## Verwendete Befehle

- `sm dedupe` ★ — Fast-Duplikat-Cues zusammenfassen
- `sm normalize` — abschließendes Sortieren und Umnummerieren

## Beispieldateien

`dup.srt` — jede Zeile erscheint zweimal, ~50–100 ms auseinander, eine Kopie mit HTML:

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

## Schritt für Schritt

### 1. Vorschau mit `--dry-run`

`--time-tolerance` gibt an, wie weit (in ms) zwei Cues mit demselben Text auseinanderliegen dürfen und
trotzdem als dieselbe Zeile zählen; `--ignore-html` entfernt Tags vor dem Vergleich, sodass `<i>Hello there.</i>` und
`Hello there.` übereinstimmen:

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

Zwei Paare werden zu zwei Cues zusammengefasst: `Removed: 2`, `Merged groups: 2`, `Final subtitles: 2`.

### 2. Anwenden

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

Jeder zusammengeführte Cue behält den Text der ersten Vorkommen und überspannt die Vereinigung beider Fenster
(frühester Start, spätestes Ende):

```srt
1
00:00:01,000 --> 00:00:03,100
<i>Hello there.</i>

2
00:00:05,000 --> 00:00:07,050
A unique line.
```

### 3. Aufräumen

Die bereinigte Datei umnummerieren und neu sortieren:

```bash
sm normalize dup.srt
```

Siehe [Struktur normalisieren](normalize-structure.de.md) für Details.

## JSON-Ausgabe

```bash
sm --output json dedupe dup.srt --time-tolerance 200 --ignore-html --dry-run
```

```json
{"ok":true,"data":{"file":"dup.srt","original_count":4,"removed":2,"merged":2,"final_count":2,"time_tolerance_ms":200,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `original_count` / `final_count` — Cue-Anzahl vor und nach der Bereinigung.
- `removed` — als Duplikate entfernte Cues.
- `merged` — Anzahl der zusammengefassten Gruppen.

## Fallstricke & Randfälle

- **`--time-tolerance 0` erfordert überlappende Fenster** zum Zusammenführen — nur Cues, die sich
  tatsächlich zeitlich überschneiden und denselben Text teilen, werden zusammengefasst. Toleranz vergrößern, um
  Kopien zu erfassen, die knapp auseinanderliegen.
- **Ohne `--ignore-html` halten Formatierungsunterschiede Cues getrennt** —
  `<i>Hello</i>` und `Hello` werden als unterschiedlicher Text behandelt.
- **Der Text des ersten Cues gewinnt** in einer zusammengeführten Gruppe, sodass eine kursive Kopie über einer
  normalen überleben kann (oder umgekehrt), abhängig von der Reihenfolge. Danach [normalize](normalize-structure.de.md)
  ausführen, um neu zu nummerieren.

## Siehe auch

- [Struktur normalisieren: sortieren, umnummerieren, Überlappungen beheben](normalize-structure.de.md)
- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md)
- `sm dedupe --help`, `sm normalize --help`, `sm describe`
