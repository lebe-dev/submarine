# Unvollständige Übersetzung mit einer Spenderdatei zusammenführen

> Die Lücken in einer unvollständigen Übersetzung mit Zeilen aus einer anderen Untertiteldatei füllen, ohne die bereits vertrauenswürdigen Zeilen anzutasten.

## Szenario

Es liegt eine handgefertigte Übersetzung vor, bei der einige Zeilen fehlen — eine Szene wurde übersprungen
oder der Übersetzer hat auf halbem Weg aufgehört. Eine andere Datei (ein "Donor") deckt diese fehlenden Zeilen
ab, ist aber um einen konstanten Betrag verschoben (ein anderes Release) und enthält eine zusätzliche Zeile,
die in der eigenen Datei fehlt. Die eigene Formulierung soll überall dort erhalten bleiben, wo sie vorhanden
ist, und nur die Zeilen des Donors für die Lücken übernommen werden.

Dies ist der Arbeitsablauf, der früher mit Ad-hoc-Python-Skripten durchgeführt wurde. Mit `sm` ist es ein
einziger Befehl, dem zwei schnelle Diagnosen vorangehen.

## Verwendete Befehle

- `sm detect-offset` — die konstante Zeitverschiebung zwischen den beiden Dateien messen
- `sm diff` — genau sehen, welche Zeilen der Donor beitragen würde
- `sm gaps` — die Lücken in der Basisdatei lokalisieren
- `sm merge` ★ — den Merge durchführen und dabei den Offset automatisch korrigieren
- `sm info` / `sm verify` — das Ergebnis bestätigen

## Beispieldateien

Diese zwei Dateien zum Mitmachen erstellen.

`base.srt` — die Übersetzung, bei der die Zeile um 00:00:10 fehlt:

```srt
1
00:00:01,000 --> 00:00:03,000
Good morning.

2
00:00:04,000 --> 00:00:06,000
How are you?

3
00:00:20,000 --> 00:00:22,000
See you tomorrow.
```

`donor.srt` — derselbe Dialog, um +200 ms verschoben, mit einer zusätzlichen Zeile in der Lücke:

```srt
1
00:00:01,200 --> 00:00:03,200
Good morning.

2
00:00:04,200 --> 00:00:06,200
How are you?

3
00:00:10,200 --> 00:00:12,200
I missed you.

4
00:00:20,200 --> 00:00:22,200
See you tomorrow.
```

## Schritt für Schritt

### 1. Den Offset zwischen den Dateien messen

```bash
sm detect-offset base.srt donor.srt
```

```
Offset Detection
================

File A: base.srt
File B: donor.srt

Anchor matches:     3
Median offset (ms): 200
Stddev (ms):        0
```

`Median offset (ms): 200` bedeutet, dass der Donor konstant 200 ms hinter der Basis liegt. Ein
`Stddev (ms)` von `0` bestätigt, dass die Verschiebung gleichmäßig ist — ein einziger Offset, kein
progressiver Drift. (Mit nur drei Zeilen kann auch ein Hinweis "files share little dialogue" erscheinen;
diese Heuristik ist für vollständige Dateien mit Hunderten von Cues gedacht und kann hier ignoriert werden.)

### 2. Sehen, was der Donor hinzufügen würde

```bash
sm diff base.srt donor.srt --by time
```

```
Diff between subtitle files
===========================

--- base.srt
+++ donor.srt

Common:    3
Only in A: 0
Only in B: 1

+ [3] 00:00:10,200 --> 00:00:12,200  I missed you.
```

Drei Zeilen überlappen sich zeitlich (`Common: 3`) und genau eine Zeile existiert nur im Donor
(`Only in B: 1`) — der `I missed you.`-Cue, der die Lücke füllt. `diff --by time` vergleicht nach
Zeitüberlappung, sodass die +200-ms-Verschiebung keine falschen "only in B"-Einträge erzeugt.

### 3. (Optional) Die Lücke in der Basis lokalisieren

```bash
sm gaps base.srt --min-gap 5000
```

```
after index 2: 00:00:06,000 --> 00:00:20,000, duration 14.000s
```

Eine 14-Sekunden-Stille nach Zeile 2 ist der Ort, an den der fehlende Dialog gehört.

### 4. Den Merge mit `--dry-run` vorschauen

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```
Dry-run: merge would be applied

Strategy: fill-gaps
Applied offset: 200 ms

Base subtitles:  3
Donor subtitles: 4

Would add:     1
Would skip (overlapping): 3
Would replace: 0
Total after merge: 4

Output (not written): merged.srt
```

`--auto-offset` verwendet die Erkennung aus Schritt 1 (`Applied offset: 200 ms`). Die Standard-Strategie
`fill-gaps` fügt nur die Donor-Zeile ein, die in eine Lücke fällt (`Would add: 1`), und überspringt die
drei, die die eigenen Zeilen überlappen (`Would skip (overlapping): 3`). Es wird noch nichts geschrieben.

### 5. Ausführen

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset
```

```
✓ Merge completed successfully

Backup: N/A (new file)
Applied offset: 200 ms
Added: 1
Skipped (overlapping): 3
Replaced: 0
Total subtitles: 4
Output: merged.srt
```

`merged.srt` enthält jetzt die drei ursprünglichen Zeilen plus die Zeile des Donors, um den erkannten
Offset verschoben, sodass sie natürlich in der Lücke sitzt:

```srt
3
00:00:10,400 --> 00:00:12,400
I missed you.
```

Wenn `--out` auf eine vorhandene Datei zeigt, schreibt `sm` vor dem Überschreiben eine Kopie mit
Zeitstempel nach `backups/` (hier ist es eine neue Datei, daher `Backup: N/A`).

#### Eine Strategie wählen

- `fill-gaps` (Standard) — jede Basis-Zeile behalten; Donor-Zeilen nur dort hinzufügen, wo die Basis
  stumm ist. Verwenden, um eine Übersetzung zu vervollständigen, ohne sie zu verändern.
- `keep-base` — niemals ersetzen; identisch mit fill-gaps für Hinzufügungen, verweigert aber jede
  Überlappung. Am sichersten, wenn die Basis maßgeblich ist.
- `prefer-donor` — wenn eine Donor-Zeile eine Basis-Zeile überlappt, den Basis-Text durch den des Donors
  ersetzen. Verwenden, wenn der Donor die bessere Übersetzung ist.

`--overlap-tolerance` (Standard `250` ms) steuert, wie viel Timing-Spielraum noch als "dieselbe Zeile"
zählt, was kleine Release-Unterschiede nach dem Offset absorbiert.

### 6. Verifizieren

```bash
sm info merged.srt
```

```
Subtitle File Information
========================

File: merged.srt

Basic Information:
  Total subtitles: 4
  Total duration:  00:00:21,000 (21.000s)
```

## JSON-Ausgabe

```bash
sm --output json merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```json
{"ok":true,"data":{"base_count":3,"donor_count":4,"added":1,"skipped_overlapping":3,"replaced":0,"applied_offset_ms":200,"total_count":4,"output":"merged.srt","dry_run":true}}
```

- `added` — in Lücken eingefügte Donor-Zeilen.
- `skipped_overlapping` — Donor-Zeilen, die wegen Überlappung mit einer Basis-Zeile verworfen wurden.
- `replaced` — überschriebene Basis-Zeilen (ungleich null nur bei `--prefer-donor`).
- `applied_offset_ms` — die auf den Donor angewendete Verschiebung (aus `--auto-offset` oder `--offset`).
- `total_count` — Untertitelanzahl nach dem Merge.

## Fallstricke & Randfälle

- **`--out`, nicht `--output`.** `--out` ist die Ergebnisdatei; `--output text|json` ist das
  globale Formatierungs-Flag. Sie sind unterschiedlich.
- **Donor-Zeilen außerhalb des Zeitbereichs der Basis werden ebenfalls hinzugefügt**, nicht nur die in
  inneren Lücken. Vorher mit `sm diff --by time` prüfen.
- **Abgleich erfolgt nach Zeitstempel, nicht nach Text**, sodass kurze wiederholte Zeilen (z. B. "Thanks")
  im Donor nicht mit Zeilen an anderer Stelle in der Basis verwechselt werden.
- **Einen manuellen Offset mit `--offset` übergeben**, wenn `--auto-offset` fehlschlägt (z. B. zu wenig
  gemeinsamer Dialog): `sm merge base.srt donor.srt --out merged.srt --offset=-212`. Die `=`-Form für
  negative Werte verwenden.
- **Drift, kein Offset?** Wenn `sm detect-offset` ein großes `Stddev` oder "drift detected" meldet, richtet
  ein einzelner Offset die Dateien nicht aus — siehe
  [Framerate-Drift beheben](fix-framerate-drift-with-rescale.de.md).

## Siehe auch

- [Konstanten Synchronisations-Offset erkennen und beheben](detect-and-fix-constant-offset.de.md)
- [Zwei Untertitelspuren vergleichen](compare-two-subtitle-tracks.de.md)
- [Fehlende Dialoge mit Lückenanalyse finden](find-missing-dialogue-gaps.de.md)
- [Duplikate und Überlappungen nach dem Zusammenführen entfernen](deduplicate-and-clean-after-merge.de.md)
- `sm merge --help`, `sm describe`
