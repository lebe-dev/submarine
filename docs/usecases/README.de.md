# Anwendungsfälle

Aufgabenorientierte Rezepte für reale Untertitelprobleme. Jede Seite beschreibt ein Problem und führt dann Schritt für Schritt durch die genauen `sm`-Befehle, die es lösen — mit kopierfertigen Beispieldateien, der erwarteten Ausgabe, einer `--dry-run`-Vorschau vor jeder Änderung und einem `--output json`-Beispiel für Skripte.

Dies sind Rezepte, kein Flaggen-Referenzhandbuch. Die vollständige Optionsliste eines Befehls liefert `sm <command> --help`; maschinenlesbare Schemata liefert `sm describe`.

## Nach Aufgabe

### Synchronisation & Timing

| Anwendungsfall | Löst | Hauptbefehl |
|---|---|---|
| [Konstanten Synchronisations-Offset erkennen und beheben](detect-and-fix-constant-offset.de.md) | Untertitel sind durchgehend zu früh oder zu spät um einen festen Betrag | `detect-offset` → `delay` |
| [Framerate-Drift beheben (23.976 ↔ 25)](fix-framerate-drift-with-rescale.de.md) | Synchronisation driftet zum Ende hin immer weiter weg (fps-Nichtübereinstimmung) | `rescale` |
| [Nur einen Teil einer Datei neu synchronisieren](resync-a-partial-range.de.md) | Eine Verschiebung tritt nur nach einem bestimmten Punkt oder einem Schnitt auf | `delay --range` / `--from-timestamp` |

### Zusammenführen & Verbinden

| Anwendungsfall | Löst | Hauptbefehl |
|---|---|---|
| [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](merge-incomplete-translation-with-donor.de.md) | Ergänzung einer unvollständigen Übersetzung aus einer anderen Datei, ohne vertrauenswürdige Zeilen anzutasten | `merge` |
| [Geteilte Teile (CD1/CD2) in eine Datei zusammenfügen](join-split-parts-cd1-cd2.de.md) | Ein in aufeinanderfolgende Teile aufgeteiltes Release muss zusammengesetzt werden | `concat` |
| [Duplikate und Überlappungen nach dem Zusammenführen entfernen](deduplicate-and-clean-after-merge.de.md) | Fast identische Zeilen häufen sich nach einem aggressiven Merge an | `dedupe` |

### Inspektion & Struktur

| Anwendungsfall | Löst | Hauptbefehl |
|---|---|---|
| [Fehlende Dialoge mit Lückenanalyse finden](find-missing-dialogue-gaps.de.md) | Stille Stellen lokalisieren, an denen möglicherweise Dialog fehlt | `gaps` |
| [Zwei Untertitelspuren vergleichen](compare-two-subtitle-tracks.de.md) | Den inhaltlichen Unterschied zwischen zwei Spuren verstehen | `diff` |
| [Struktur normalisieren: sortieren, umnummerieren, Überlappungen beheben](normalize-structure.de.md) | Eine gültige Datei geriet nach manuellen Bearbeitungen außer Ordnung | `normalize` |

## Konventionen in diesen Rezepten

- **`--out` ist die Ergebnisdatei; `--output text|json` ist das globale Formatierungs-Flag.** Sie
  sind verschieden — `merge`, `rescale` und `concat` schreiben nach `--out`.
- **Verändernde Befehle schreiben ein Backup** nach `backups/`, bevor sie eine Datei direkt
  ändern, und alle unterstützen `--dry-run` zur Vorschau.
- **Negative Offsets:** Optionsflags vor einem negativen Offset platzieren
  (`sm delay --dry-run file -500`), da ein führendes `-` sonst wie ein Flag aussieht.

Siehe das [README](../../README.de.md) des Projekts für Installation und die vollständige Befehlsliste.
