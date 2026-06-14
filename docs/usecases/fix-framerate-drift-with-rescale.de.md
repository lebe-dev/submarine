# Framerate-Drift beheben (23.976 ↔ 25)

> Die Synchronisation ist am Anfang gut, driftet aber zum Ende hin immer weiter ab — ein Framerate-Missverhältnis, kein konstanter Offset. Die Zeitachse strecken statt verschieben.

## Szenario

Untertitel, die für ein 25-fps-Release (PAL) erstellt wurden, werden gegen eine 23.976-fps-Kodierung
(NTSC/Film) abgespielt, oder umgekehrt. Frühe Zeilen sehen fast richtig aus, aber der Fehler wächst
linear, bis die letzten Zeilen Sekunden zu weit weg sind. Ein einzelnes `delay` kann dies nicht beheben —
die Zeitachse muss *skaliert* werden.

## Verwendete Befehle

- `sm detect-offset` — bestätigen, dass es Drift ist, kein konstanter Offset
- `sm rescale` ★ — die Zeitachse per Faktor, fps-Paar oder zwei Ankerpunkten strecken/stauchen

## Beispieldateien

`movie.srt` — drei Cues über 20 Minuten verteilt:

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

## Schritt für Schritt

### 1. Bestätigen, dass es Drift ist, kein Offset

Die nicht synchrone Datei gegen eine synchrone Referenz vergleichen:

```bash
sm detect-offset stretched.srt movie.srt
```

```
Median offset (ms): -25620
Stddev (ms):        20745

Note: drift detected (offset varies across the timeline); consider 'sm rescale'.
```

Das Erkennungsmerkmal ist ein **großes `Stddev`** und der explizite **"drift detected"**-Hinweis: der
gemessene Offset ist nahe dem Start klein und am Ende riesig. Diese Varianz ist das Kennzeichen eines
Framerate-Missverhältnisses — `rescale` verwenden, nicht `delay`.

### 2. Per fps-Paar skalieren

Wenn Quell- und Ziel-Frameraten bekannt sind, können sie direkt angegeben werden:

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

Die Korrektur wächst mit der Zeit — 0,4 s am Anfang, ~51 s bei der 20-Minuten-Marke — was den linearen
Drift genau aufhebt. `sm` leitet den Faktor (25 / 23.976) selbst her.

### 3. Oder per explizitem Faktor

Wenn der Multiplikator bereits bekannt ist:

```bash
sm rescale movie.srt --factor 1.0427 --out scaled.srt --dry-run
```

```
Mode: factor
Factor: 1.0427
```

### 4. Oder per zwei Ankerpunkten

Wenn die fps nicht bekannt sind, aber zwei Cues und die Timecodes, die sie *haben sollten*, identifiziert
werden können, zwei `--anchor IDX=HH:MM:SS,mmm`-Punkte angeben. `sm` löst die lineare Transformation,
die jeden Cue auf sein Ziel abbildet (Berechnung von Skalierungsfaktor und einem kleinen Offset):

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

Cues 1 und 3 landen genau auf ihren angeforderten Timecodes; alles dazwischen wird interpoliert.

### 5. Anwenden

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

## JSON-Ausgabe

```bash
sm --output json rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt --dry-run
```

```json
{"ok":true,"data":{"file":"movie.srt","mode":"fps","factor":1.0427093760427095,"offset_ms":0,"total_count":3,"output":"ntsc.srt","backup_path":"N/A (dry-run)","dry_run":true}}
```

- `mode` — `factor`, `fps` oder `anchor`.
- `factor` / `offset_ms` — die aufgelöste lineare Transformation `t' = factor·t + offset`.
- `output` — die Ergebnisdatei.

## Fallstricke & Randfälle

- **Genau einen Modus angeben.** `--factor`, das `--from-fps`/`--to-fps`-Paar oder zwei
  `--anchor`-Werte — eine Kombination oder Auslassung führt zum Fehler *"specify one of …"*.
- **`--out`, nicht `--output`.** `--out` ist die Ergebnisdatei; `--output text|json` ist das
  globale Formatierungs-Flag.
- **Anker-Zeitstempel enthalten ein Komma** (`HH:MM:SS,mmm`). Jedes `--anchor`-Argument in
  Anführungszeichen setzen, damit die Shell es unverändert weitergibt.
- **Wenn `Stddev` nahe null ist**, handelt es sich nicht um Drift — [delay](detect-and-fix-constant-offset.de.md) verwenden.

## Siehe auch

- [Konstanten Synchronisations-Offset erkennen und beheben](detect-and-fix-constant-offset.de.md)
- [Nur einen Teil einer Datei neu synchronisieren](resync-a-partial-range.de.md)
- `sm rescale --help`, `sm detect-offset --help`, `sm describe`
