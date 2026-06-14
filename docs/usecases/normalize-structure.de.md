# Struktur normalisieren: sortieren, umnummerieren, Überlappungen beheben

> Nach manuellen oder skriptbasierten Bearbeitungen eine gültige SRT-Datei wieder in Ordnung bringen — nach Zeit sortiert, ab 1 nummeriert, mit getrimmten Überlappungen.

## Szenario

Manuelle Bearbeitung oder ein schnelles Skript hat die Datei außer Ordnung gebracht: Indizes steigen nicht
mehr an, Cues sind nicht nach Startzeit sortiert, und ein paar Nachbarn überlappen sich. Die Datei wird
noch geparst, ist aber unordentlich. `sm normalize` schreibt sie in kanonische Form.

## Verwendete Befehle

- `sm normalize` ★ — sortieren, umnummerieren und optional Überlappungen beheben
- `sm doctor` — für die andere Aufgabe, strukturelle Parse-Fehler zu reparieren

## Beispieldateien

`scram.srt` — außer Ordnung, mit dem ersten Cue, der den zweiten überlappt:

```srt
3
00:00:08,000 --> 00:00:10,000
Third in time.

1
00:00:01,000 --> 00:00:04,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.
```

Cue `1` endet bei `00:00:04,000`, aber Cue `2` beginnt bei `00:00:03,000` — eine 1-Sekunden-Überlappung.

## Schritt für Schritt

### 1. Vorschau mit `--dry-run`

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```
Dry-run: normalization would be applied

Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

`Overlaps fixed: 1` ist das eine Nachbarpaar, das getrimmt wird. Es wird nichts geschrieben.

### 2. Anwenden

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps
```

```
✓ Subtitles normalized successfully

Backup: backups/scram.srt.2026-06-14_10-36-09.226844
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

Die Datei wird direkt überschrieben (ein Backup wird vorher nach `backups/` gespeichert). Cues sind jetzt
sortiert, ab 1 nummeriert, und die Überlappung wird aufgelöst, indem das Ende von Cue 1 auf den Start von
Cue 2 geclippt wird:

```srt
1
00:00:01,000 --> 00:00:03,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.

3
00:00:08,000 --> 00:00:10,000
Third in time.
```

### Standardeinstellungen

`--sort` und `--renumber` sind **standardmäßig aktiviert**, sodass ein bloßes `sm normalize scram.srt`
bereits neu ordnet und umnummeriert — Überlappungen aber unberührt lässt:

```bash
sm normalize scram.srt --dry-run
```

```
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: no
```

`--fix-overlaps` nur hinzufügen, wenn Nachbar-Timings getrimmt werden sollen.

## JSON-Ausgabe

```bash
sm --output json normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```json
{"ok":true,"data":{"file":"scram.srt","total_count":3,"sorted":true,"renumbered":true,"overlaps_fixed":1,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `sorted` / `renumbered` — ob jeder Durchlauf ausgeführt wurde.
- `overlaps_fixed` — Anzahl der getrimmten Nachbar-Überlappungen.
- `total_count` — Cues nach der Normalisierung.

## `normalize` vs. `doctor`

- `sm doctor` **diagnostiziert und repariert strukturelle Probleme** — fehlerhafte Timecodes,
  kaputte Cue-Blöcke, Parse-Fehler. Darauf zurückgreifen, wenn eine Datei nicht sauber geladen werden kann.
- `sm normalize` setzt voraus, dass die Datei bereits **gültig** ist, und ordnet sie neu, nummeriert sie um
  und entfernt Überlappungen. Zuerst `doctor` ausführen, wenn das Parsen fehlschlägt, dann `normalize` zum
  Aufräumen.

## Fallstricke & Randfälle

- **`--fix-overlaps` ändert Timings**, nicht nur die Reihenfolge — es clippt das Ende eines Cues auf den
  Start des nächsten. Mit `--dry-run` vorschauen.
- **Eine Standardeinstellung explizit deaktivieren** mit `--sort=false` / `--renumber=false`, wenn
  umnummeriert werden soll, ohne neu zu sortieren, oder umgekehrt.
- **Umnummerieren nach einem Merge oder manuellen Schnitt** ist der häufige Fall — mit
  [dedupe](deduplicate-and-clean-after-merge.de.md) kombinieren, wenn auch Fast-Duplikate eingeschlichen sind.

## Siehe auch

- [Duplikate und Überlappungen nach dem Zusammenführen entfernen](deduplicate-and-clean-after-merge.de.md)
- [Geteilte Teile (CD1/CD2) in eine Datei zusammenfügen](join-split-parts-cd1-cd2.de.md)
- `sm normalize --help`, `sm doctor --help`, `sm describe`
