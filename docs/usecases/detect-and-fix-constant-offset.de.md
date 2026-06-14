# Konstanten Synchronisations-Offset erkennen und beheben

> Die Untertitel sind durchgehend um einen festen Betrag zu früh oder zu spät — die Verschiebung messen und dann die gesamte Datei in einem Schritt korrigieren.

## Szenario

Die Untertitel passen zum Dialog, aber jede Zeile erscheint um denselben Betrag zu früh oder zu spät —
ein klassisches Missverhältnis zwischen der Untertiteldatei und dem Videorelease. Es liegt eine
vertrauenswürdige Referenz vor (eine andere Spur, die synchron ist, oder das eigene Timing des Videos)
und die Datei soll daran angepasst werden.

## Verwendete Befehle

- `sm detect-offset` ★ — die Verschiebung gegenüber einer Referenz messen
- `sm delay` — die Verschiebung auf die gesamte Datei anwenden
- `sm detect-offset` erneut — bestätigen, dass das Ergebnis ~0 ist

## Beispieldateien

`reference.srt` — korrekt getaktet:

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

`subs.srt` — derselbe Dialog, 2,5 Sekunden zu spät:

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

## Schritt für Schritt

### 1. Den Offset messen

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

`Median offset (ms): -2500` bedeutet "Datei A liegt 2500 ms hinter Datei B". `Stddev: 0`
bestätigt einen sauberen konstanten Offset (kein Drift), sodass eine einzige Verschiebung ihn behebt. Um
`subs.srt` wieder zu synchronisieren, wird die Negation dieser Verzögerung angewendet: **−2500 ms**.

### 2. Die Verschiebung mit `--dry-run` vorschauen

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

Die Vorschau zeigt, dass jede Zeile genau auf die Referenz-Timings trifft. Es wird nichts geschrieben.

> **Die Reihenfolge der Flags ist bei einem negativen Offset wichtig.** Ein führendes `-` lässt den Offset
> wie ein Flag aussehen; daher Optionsflags (`--dry-run`, `--range`, `--from-timestamp`) **vor** dem
> Offset platzieren: `sm delay --dry-run subs.srt -2500`. Positive Offsets (`+2500`) sind
> nicht betroffen.

### 3. Anwenden

```bash
sm delay subs.srt -2500
```

```
✓ Time offset applied successfully

Backup: backups/subs.srt.2026-06-14_10-43-26.541735
Offset: -2500 ms
Subtitles adjusted: 3
```

`sm delay` ändert die Datei direkt und schreibt vorher eine Kopie mit Zeitstempel nach `backups/`.

### 4. Bestätigen

```bash
sm detect-offset subs.srt reference.srt
```

```
Median offset (ms): 0
```

Ein Median von `0` bedeutet, dass die Dateien jetzt synchron sind.

## JSON-Ausgabe

```bash
sm --output json delay subs.srt -2500
```

```json
{"ok":true,"data":{"offset_ms":-2500,"subtitles_adjusted":1,"backup_path":"backups/subs.srt.2026-06-14_10-43-26.549470"}}
```

- `offset_ms` — die angewendete Verschiebung.
- `subtitles_adjusted` — wie viele Cues verschoben wurden.
- `backup_path` — wo die Kopie vor der Änderung gespeichert wurde (`N/A (dry-run)` bei `--dry-run`).

Und der Erkennungs-Envelope, der diesen gesamten Ablauf antreibt:

```bash
sm --output json detect-offset subs.srt reference.srt
```

```json
{"ok":true,"data":{"anchor_matches":3,"median_offset_ms":-2500,"stddev_ms":0,"drift_detected":false,"same_video":false}}
```

`drift_detected:false` ist das Signal, dass ein einzelnes `delay` das richtige Werkzeug ist.

## Fallstricke & Randfälle

- **Der Offset muss ein Vorzeichen tragen:** `+100` oder `-500`. Ein bloßes `100` wird abgelehnt.
- **Vorwärtsverschiebung kann auf null treffen.** `sm delay` verweigert einen Offset, der eine Zeile
  vor `00:00:00,000` verschieben würde. Mit `--dry-run` prüfen, wenn ein Fehler wegen negativem Zeitstempel auftritt.
- **Hohes `Stddev` / "drift detected" bedeutet, dies ist der falsche Fix** — die Verzögerung wächst mit
  der Zeit. Stattdessen [rescale](fix-framerate-drift-with-rescale.de.md) verwenden.
- **Nur ein Teil der Datei ist falsch?** Siehe [Nur einen Teil einer Datei neu synchronisieren](resync-a-partial-range.de.md).

## Siehe auch

- [Nur einen Teil einer Datei neu synchronisieren](resync-a-partial-range.de.md)
- [Framerate-Drift beheben (23.976 ↔ 25)](fix-framerate-drift-with-rescale.de.md)
- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md)
- `sm detect-offset --help`, `sm delay --help`, `sm describe`
