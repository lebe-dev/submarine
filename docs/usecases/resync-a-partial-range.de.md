# Nur einen Teil einer Datei neu synchronisieren

> Mitten in der Datei ist eine Verschiebung aufgetreten — nach einem Schnitt oder einer eingefügten Szene — sodass nur die späteren Cues verschoben werden müssen.

## Szenario

Die erste Hälfte der Datei ist synchron, aber irgendwo in der Mitte (oft dort, wo zwei Quellen
zusammengeschnitten wurden) kippt das Timing und bleibt für den Rest daneben. Die gesamte Datei zu
verschieben würde den bereits korrekten Teil kaputt machen. Nur der betroffene Bereich soll verschoben
werden.

## Verwendete Befehle

- `sm detect-offset` — die Verschiebung im nicht ausgerichteten Abschnitt messen
- `sm delay` ★ — nur einen Bereich von Cues verschieben, nach Index oder Zeitstempel

## Beispieldateien

`joined.srt` — Cues 1–2 sind korrekt, Cues 3–4 kamen nach einem Schnitt 500 ms zu früh:

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

## Schritt für Schritt

### 1. Per Index-Bereich verschieben

Cues 3 bis 4 um +500 ms verschieben, 1–2 unberührt lassen. Zuerst vorschauen:

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

Nur Cues 3–4 bewegen sich (`Subtitles affected: 2`); die Vorschau bestätigt, dass 1 und 2 unverändert sind.

### 2. Oder per Zeitstempel verschieben

Wenn der Moment bekannt ist, aber nicht der Index, `--from-timestamp` verwenden — jeder Cue ab
diesem Zeitpunkt oder später wird verschoben:

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

### 3. Anwenden

```bash
sm delay --range 3-4 joined.srt +500
```

`--dry-run` weglassen, um die Änderung zu schreiben; ein Backup wird vorher nach `backups/` gespeichert.

> **Tipp für negative Offsets:** Die Flags **vor** dem Offset platzieren
> (`sm delay --range 3-4 joined.srt -500`). Ein führendes `-` lässt den Offset wie ein
> Flag aussehen, sodass danach platzierte Flags ignoriert würden.

## JSON-Ausgabe

```bash
sm --output json delay --range 3-4 joined.srt +500
```

```json
{"ok":true,"data":{"offset_ms":500,"subtitles_adjusted":2,"backup_path":"backups/joined.srt.2026-06-14_10-34-15.001252","range_start":3,"range_end":4}}
```

- `subtitles_adjusted` — wie viele Cues verschoben wurden.
- `range_start` / `range_end` — der verschobene Index-Bereich (weggelassen bei Verwendung von
  `--from-timestamp`).

## Fallstricke & Randfälle

- **`--range` und `--from-timestamp` schließen sich gegenseitig aus:**

  ```bash
  sm delay --range 3-4 --from-timestamp 00:00:09,000 joined.srt +500
  ```
  ```
  error: use either --range or --from-timestamp, not both
  ```

- **Ohne eines dieser Flags wird die gesamte Datei verschoben** — das ist der
  [Konstant-Offset](detect-and-fix-constant-offset.de.md)-Fall.
- **`--range` ist inklusiv** und verwendet Untertitel-Indizes (`START-END`), keine Zeitstempel.
- **Nur am schlechten Abschnitt messen.** `sm detect-offset` gegen eine synchrone Referenz ausführen,
  dabei nur den nicht ausgerichteten Teil verwenden, damit der Offset nicht durch die korrekte erste
  Hälfte verwässert wird.

## Siehe auch

- [Konstanten Synchronisations-Offset erkennen und beheben](detect-and-fix-constant-offset.de.md)
- [Geteilte Teile (CD1/CD2) in eine Datei zusammenfügen](join-split-parts-cd1-cd2.de.md)
- `sm delay --help`, `sm detect-offset --help`, `sm describe`
