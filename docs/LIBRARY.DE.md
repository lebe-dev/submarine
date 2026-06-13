# Submarine als Bibliothek verwenden

Submarine ist primär als Befehlszeilenwerkzeug konzipiert, seine Kernfunktionalität steht jedoch auch als Bibliothek zur Verfügung, die Sie in eigene Go-Projekte integrieren können.

**Sprache:** [EN](LIBRARY.md) | **DE** | [ES](LIBRARY.ES.md)

Diese Anleitung führt Sie durch die Grundlagen der Verwendung der `submarine`-Bibliothek.

## `submarine` zu Ihrem Projekt hinzufügen

`submarine` ist ein Go-Modul. Fügen Sie es mit `go get` zu Ihrem Projekt hinzu:

```bash
go get github.com/lebe-dev/submarine
```

Es erfordert Go 1.26 oder neuer. Die wiederverwendbaren Bibliothekspakete liegen unter `pkg/` (der reine CLI-Code verbleibt in `internal/` und ist nicht importierbar).

## Kernkonzepte

Die Funktionalität der Bibliothek dreht sich um die Schnittstelle `Service` im Paket `pkg/subtitle`, die die wichtigsten Operationen für die Arbeit mit Untertiteldateien definiert. Die Hauptimplementierung ist `SubRipService`, die mit SubRip-Dateien (.srt) arbeitet.

- **`subtitle.Subtitle`**: stellt einen einzelnen Untertiteleintrag dar — seinen Index, Start- und End-Zeitstempel sowie den Text.
- **`subtitle.SubRipService`**: der Einstiegspunkt für die meisten dateibasierten Operationen. Damit können Sie .srt-Dateien lesen, schreiben und ändern.

Validierte Wertetypen (`SubtitleIndex`, `SubtitleTimestamp`, `SubtitleText`) werden über `New…`-Funktionen erzeugt, die einen `error` zurückgeben, wenn die Eingabe ungültig ist (z. B. muss ein Index `>= 1` sein, der Text darf nicht leer sein). Das entspricht den Garantien, die die Rust-Version mit `nutype` durchgesetzt hat.

## Grundlegende Verwendung

Hier ist ein einfaches Beispiel, wie man mit der Bibliothek Untertitel aus einer Datei liest, einen neuen Untertitel hinzufügt und das Ergebnis in eine neue Datei schreibt. Das vollständige, ausführbare Programm befindet sich in [`examples/simple/main.go`](../examples/simple/main.go) — führen Sie es mit `go run ./examples/simple` aus.

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

func main() {
	// Der Service ist dateibasiert; jeder Dateiname wird relativ zu diesem
	// Basisverzeichnis aufgelöst. Wir verwenden ein temporäres Verzeichnis,
	// damit das Beispiel in sich abgeschlossen ist.
	baseDir, err := os.MkdirTemp("", "submarine-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	service := subtitle.NewSubRipService(baseDir)

	// 1. Eine Beispieldatei erstellen und Untertitel daraus laden.
	srtContent := "1\n00:00:03,000 --> 00:00:04,000\nThis is a sample subtitle.\n\n" +
		"2\n00:00:05,000 --> 00:00:06,000\nThis is another one.\n"
	if err := os.WriteFile(filepath.Join(baseDir, "sample.srt"), []byte(srtContent), 0o644); err != nil {
		log.Fatal(err)
	}

	subtitles, err := service.GetAll("sample.srt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded %d subtitles\n", len(subtitles))
	for _, sub := range subtitles {
		fmt.Println(strings.TrimSpace(sub.String()))
	}

	// 2. Einen Untertitel programmatisch erstellen.
	index, err := subtitle.NewSubtitleIndex(3)
	if err != nil {
		log.Fatal(err)
	}
	start, err := subtitle.NewSubtitleTimestamp(7000 * time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	end, err := subtitle.NewSubtitleTimestamp(8000 * time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	text, err := subtitle.NewSubtitleText("This is a new subtitle, created in code.")
	if err != nil {
		log.Fatal(err)
	}
	newSubtitle, err := subtitle.NewSubtitle(index, start, end, text)
	if err != nil {
		log.Fatal(err)
	}
	subtitles = append(subtitles, newSubtitle)

	// 3. Die geänderte Liste in eine neue Datei speichern.
	if err := service.WriteAll("output.srt", subtitles); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Subtitles saved to output.srt")
}
```

### Schritt-für-Schritt-Erklärung

1. **`SubRipService` instanziieren**:
   Der Service benötigt ein Basisverzeichnis. Alle Dateinamen, die Sie angeben, werden relativ zu diesem Pfad aufgelöst.

   ```go
   service := subtitle.NewSubRipService(baseDir)
   ```

2. **Untertitel lesen**:
   `GetAll` liest eine .srt-Datei und gibt ein `[]subtitle.Subtitle` zurück.

   ```go
   subtitles, err := service.GetAll("sample.srt")
   ```

3. **Einen neuen Untertitel erstellen**:
   Sie bauen einen `Subtitle` aus validierten Wertetypen. Jeder `New…`-Konstruktor gibt einen `error` zurück, wenn die Eingabe ungültig ist (Indizes müssen positiv sein, der Text darf nicht leer sein, das Ende muss nach dem Start liegen).

   ```go
   index, err := subtitle.NewSubtitleIndex(3)
   start, err := subtitle.NewSubtitleTimestamp(7000 * time.Millisecond)
   end, err := subtitle.NewSubtitleTimestamp(8000 * time.Millisecond)
   text, err := subtitle.NewSubtitleText("This is a new subtitle, created in code.")
   newSubtitle, err := subtitle.NewSubtitle(index, start, end, text)
   ```

4. **Untertitel schreiben**:
   `WriteAll` nimmt einen Slice von `Subtitle`s und schreibt sie in eine Datei, die überschrieben wird, falls sie existiert.

   ```go
   err := service.WriteAll("output.srt", subtitles)
   ```

## Weiterführende Erkundung

Für fortgeschrittene Nutzung erkunden Sie die weiteren Methoden der Schnittstelle `subtitle.Service`:

- `GetByID(filename, id)` — einen einzelnen Untertitel anhand seines Index abrufen (gibt `(nil, nil)` zurück, wenn nicht gefunden).
- `Set(filename, id, update)` — einen vorhandenen Untertitel aktualisieren.
- `Add(filename, start, end, text)` — einen neuen Untertitel an eine Datei anhängen.

Neben `pkg/subtitle` decken die übrigen Bibliothekspakete die restlichen Funktionen des Werkzeugkastens ab:

| Paket | Zweck |
|---|---|
| `pkg/subtitle` | Kerndomänenmodell, SRT-Parsing/-Schreiben (`SubRipService`) |
| `pkg/backup` | Zeitgestempelte Datei-Backups (`SubRipBackupService`) |
| `pkg/doctor` | Fehlerhafte SRT-Dateien diagnostizieren und reparieren |
| `pkg/importer` | Untertitel aus CSV- und Anchored-Formaten importieren |
| `pkg/rename` | Vorlagenbasiertes Massenumbenennen von Untertiteldateien |
| `pkg/verify` | Zwei Dateien auf Index-/Zeitstempel-Abweichungen vergleichen (`CompareSubtitles`) |
| `pkg/translationstatus` | Übersetzungsfortschritt gegen eine Referenz (`CheckTranslationStatus`) |

## Hinweis zum Logging

Die Bibliothek protokolliert über das Standardpaket `log/slog`. Standardmäßig schreibt Gos `slog` Einträge der Stufe `Info` nach stderr, sodass beim Aufruf von Bibliotheksfunktionen Log-Zeilen erscheinen können. Um sie zu steuern oder stummzuschalten, installieren Sie Ihren eigenen Standard-Handler, z. B.:

```go
import "log/slog"

// Nur Warnungen und höher anzeigen.
slog.SetLogLoggerLevel(slog.LevelWarn)
```
