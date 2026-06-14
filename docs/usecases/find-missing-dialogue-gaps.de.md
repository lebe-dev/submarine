# Fehlende Dialoge mit Lückenanalyse finden

> Stille Stellen in einer Untertiteldatei aufspüren, an denen Dialog möglicherweise weggelassen wurde.

## Szenario

Eine Übersetzung wirkt unvollständig — eine Szene scheint keine Untertitel zu haben. Anstatt das Video
durchzuscrubben, können die Zeitlücken zwischen aufeinanderfolgenden Cues aufgelistet und die verdächtigen
untersucht werden; dann kann eine Spenderdatei geprüft werden, ob dort tatsächlich Dialog fehlt.

## Verwendete Befehle

- `sm gaps` ★ — Stillen länger als ein Schwellenwert auflisten
- `sm info` — die durchschnittliche Lücke der Datei ermitteln, um einen Schwellenwert zu wählen
- `sm diff` — prüfen, ob eine Spenderdatei Dialog in der Lücke hat

## Schritt für Schritt

Dieses Beispiel verwendet eine mit dem Projekt mitgelieferte Beispieldatei,
`test-data/valid/complex.srt`.

### 1. Die Lücken auflisten

```bash
sm gaps test-data/valid/complex.srt --min-gap 1500
```

```
after index 2: 00:00:05,135 --> 00:00:07,790, duration 2.655s
```

Jede Zeile meldet eine Stille: `after index 2` ist der vorausgehende Cue, dann das Fenster
(`Ende von Cue 2 → Start des nächsten`) und seine Länge. Eine 2,655-Sekunden-Lücke nach Zeile 2 ist
die zu untersuchende Stelle.

### 2. `--min-gap` wählen

`--min-gap` ist die Untergrenze (in Millisekunden, Standard `1000`), unterhalb derer Lücken ignoriert werden.
Etwas oberhalb der typischen Pause der Datei einstellen, damit nur ungewöhnliche Stillen auftauchen. `sm info`
liefert diese Ausgangsbasis:

```bash
sm info test-data/valid/complex.srt
```

Auf **Average gap between subtitles** achten. Wenn die durchschnittliche Pause einer Datei ~9 s beträgt, zeigt
sich eine ausgelassene Szene als viel größere Lücke — mit `--min-gap 15000` beginnen, um nur die echten Lücken
hervorzuheben, und dann verringern, wenn kleinere Auslassungen vermutet werden.

### 3. Mit einer Spenderdatei abgleichen

Eine Lücke allein beweist nicht, dass Dialog fehlt — es kann schlicht keine Sprache vorhanden sein. Gegen
eine vollständigere Spur vergleichen, um sicherzugehen:

```bash
sm diff test-data/valid/complex.srt donor.srt --by time
```

Zeilen, die als **Only in B** gemeldet werden und in die Lücke fallen, sind Dialog, der in der Datei fehlt —
genau die Eingabe für [einen Donor-Merge](merge-incomplete-translation-with-donor.de.md).

## JSON-Ausgabe

```bash
sm --output json gaps test-data/valid/complex.srt --min-gap 1500
```

```json
{"ok":true,"data":{"file":"test-data/valid/complex.srt","min_gap_ms":1500,"count":1,"gaps":[{"after_index":2,"start":"00:00:05,135","end":"00:00:07,790","duration_ms":2655}]}}
```

- `count` — Anzahl der Lücken ab `min_gap_ms`.
- `gaps[].after_index` — der Cue-Index, dem die Lücke folgt.
- `gaps[].start` / `end` — das stille Fenster.
- `gaps[].duration_ms` — seine Länge in Millisekunden.

## Fallstricke & Randfälle

- **Eine Lücke ist kein Beweis für eine fehlende Zeile.** Pausen, Szenenwechsel und Musik sind normal —
  mit `sm diff` abgleichen, bevor geschlussfolgert wird, dass Dialog fehlt.
- **Der Schwellenwert ist in Millisekunden.** `--min-gap 1500` ist 1,5 s, nicht 1500 s.
- **Überlappende Cues erzeugen keine Lücke** (das Fenster wäre negativ), sodass sie nie im Bericht
  erscheinen; diese zuerst mit [normalize](normalize-structure.de.md) bereinigen.

## Siehe auch

- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md)
- [Zwei Untertitelspuren vergleichen](compare-two-subtitle-tracks.de.md)
- `sm gaps --help`, `sm info --help`, `sm describe`
