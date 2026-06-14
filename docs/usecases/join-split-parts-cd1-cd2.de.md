# Geteilte Teile (CD1/CD2) in eine Datei zusammenfügen

> Ein Release, das auf zwei nacheinander abgespielte Dateien aufgeteilt ist — sie zu einer zusammennähen, wobei der zweite Teil verschoben wird, um dem ersten zu folgen.

## Szenario

Ein altes Release kommt als `CD1` und `CD2`, wobei beide ihre Timecodes bei null beginnen. Bei der
Wiedergabe als ein Video müssen die Cues der zweiten Datei dort beginnen, wo die erste endet. `sm concat`
hängt die Teile an und verschiebt jeden späteren Teil um die laufende Gesamtdauer, nummeriert das
Ergebnis dann neu.

Dies unterscheidet sich von [merge](merge-incomplete-translation-with-donor.de.md): Die Teile **überlappen**
sich zeitlich **nicht** — sie laufen nacheinander.

## Verwendete Befehle

- `sm concat` ★ — Teile anhängen und jeden so verschieben, dass er dem vorherigen folgt
- `sm info` — die verbundene Dauer und Anzahl bestätigen

## Beispieldateien

`part1.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
CD1 first line.

2
00:00:04,000 --> 00:00:06,000
CD1 last line.
```

`part2.srt` — beginnt ebenfalls nahe bei null:

```srt
1
00:00:01,000 --> 00:00:03,000
CD2 first line.

2
00:00:05,000 --> 00:00:07,000
CD2 last line.
```

## Schritt für Schritt

### 1. Vorschau mit `--dry-run`

`--gap` legt die Stille (in ms) fest, die zwischen Teile eingefügt wird:

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

### 2. Anwenden

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

`full.srt` enthält Teil 1 unverändert, dann Teil 2 verschoben, so dass er nach dem Ende von Teil 1 plus
der Lücke beginnt, mit Indizes, die durchgehend neu nummeriert wurden:

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

### 3. Bestätigen

```bash
sm info full.srt
```

**Total subtitles** und **Total duration** prüfen, ob sie den beiden Teilen plus der Lücke entsprechen.

## JSON-Ausgabe

```bash
sm --output json concat part1.srt part2.srt --out full.srt --gap 2000 --dry-run
```

```json
{"ok":true,"data":{"parts":2,"gap_ms":2000,"total_count":4,"output":"full.srt","dry_run":true}}
```

- `parts` — Anzahl der zusammengefügten Eingabedateien.
- `gap_ms` — zwischen Teile eingefügte Stille.
- `total_count` — Cues im Ergebnis.

## Fallstricke & Randfälle

- **Jeder Teil behält seinen eigenen Vorlauf-Offset.** Teil 2 begann bei `00:00:01,000`, daher
  landet er nach Teil 1 (endet `00:00:06,000`) plus der 2-s-Lücke bei `00:00:09,000` — der eigene
  1-s-Vorlauf des Teils wird zusätzlich zur Lücke beibehalten. Die führende Stille eines Teils vorher
  trimmen, wenn die Lücke exakt sein soll.
- **Reihenfolge ist wichtig.** Teile werden von links nach rechts in der angegebenen Reihenfolge verbunden.
- **`--out`, nicht `--output`.** `--out` ist die Ergebnisdatei; `--output text|json` ist das
  globale Formatierungs-Flag.
- **Überlappend statt aufeinanderfolgend?** Wenn die Teile zeitlich Dialog teilen, ist
  [merge](merge-incomplete-translation-with-donor.de.md) die richtige Wahl.

## Siehe auch

- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md)
- [Struktur normalisieren: sortieren, umnummerieren, Überlappungen beheben](normalize-structure.de.md)
- `sm concat --help`, `sm info --help`, `sm describe`
