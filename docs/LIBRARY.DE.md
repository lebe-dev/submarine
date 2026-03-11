# Submarine als Bibliothek verwenden

Submarine ist primär als Befehlszeilenwerkzeug konzipiert, seine Kernfunktionalität steht jedoch auch als Bibliothek zur Verfügung, die Sie in eigene Rust-Projekte integrieren können. Diese Anleitung führt Sie durch die Grundlagen der Verwendung der `submarine-rs`-Bibliothek.

**Sprache:** [EN](LIBRARY.md) | **DE** | [ES](LIBRARY.ES.md)

## `submarine-rs` zum Projekt hinzufügen

Um `submarine-rs` als Bibliothek zu verwenden, fügen Sie es zuerst in Ihrer `Cargo.toml` hinzu.

```toml
[dependencies]
submarine-rs = { git = "https://github.com/lebe-dev/submarine" }
```

Hinweis: Da `submarine-rs` noch nicht auf crates.io veröffentlicht ist, müssen Sie es direkt aus dem Git-Repository einbinden.

## Grundkonzepte

Die Funktionalität der Bibliothek ist auf den Trait `SubtitleService` ausgerichtet, der die primären Operationen für die Arbeit mit Untertiteldateien definiert. Die Hauptimplementierung dieses Traits ist `SubRipService`, der mit SubRip-Dateien (.srt) arbeitet.

- **`Subtitle`**: Dieses Struct repräsentiert einen einzelnen Untertiteleintrag, einschließlich Index, Start- und End-Zeitstempel sowie Text.
- **`SubRipService`**: Dies ist der Einstiegspunkt für die meisten dateibasierten Operationen. Er ermöglicht das Lesen, Schreiben und Ändern von .srt-Dateien.

## Grundlegende Verwendung

Hier ist ein einfaches Beispiel, wie man die Bibliothek verwendet, um Untertitel aus einer Datei zu lesen, einen neuen Untertitel hinzuzufügen und das Ergebnis in eine neue Datei zu schreiben.

Der vollständige Code ist auch in `examples/simple.rs` zu finden.

```rust
use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use std::fs;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Der Service der Bibliothek ist dateibasiert.
    // Wir erstellen eine Service-Instanz, die im Verzeichnis 'examples' arbeitet.
    let service = SubRipService::new("examples");
    let sample_filename = "sample.srt";
    let output_filename = "output.srt";
    let sample_filepath = format!("examples/{}", sample_filename);
    let output_filepath = format!("examples/{}", output_filename);

    // Beispiel 1: Beispieldatei erstellen und Untertitel daraus laden
    println!("--- Untertitel aus Datei laden ---");
    let srt_content = "1\n00:00:03,000 --> 00:00:04,000\nThis is a sample subtitle.\n\n2\n00:00:05,000 --> 00:00:06,000\nThis is another one.\n";
    fs::write(&sample_filepath, srt_content)?;

    let mut subtitles = service.get_all(sample_filename)?;

    println!("Geladene {} Untertitel aus {}:", subtitles.len(), sample_filename);
    for sub in &subtitles {
        println!("{}", sub.to_string().trim());
        println!("---");
    }

    // Beispiel 2: Untertitel programmatisch erstellen und zum Vektor hinzufügen
    println!("\n--- Neuen Untertitel erstellen und hinzufügen ---");
    let new_subtitle = Subtitle::new(
        SubtitleIndex::try_new(3)?,
        SubtitleTimestamp::try_new(Duration::milliseconds(7000))?,
        SubtitleTimestamp::try_new(Duration::milliseconds(8000))?,
        SubtitleText::try_new("This is a new subtitle, created in code.".to_string())?,
    )?;
    println!("Neuer Untertitel erstellt:\n{}", new_subtitle);
    subtitles.push(new_subtitle);

    // Beispiel 3: Die modifizierte Untertitelliste in eine neue Datei speichern
    println!("\n--- Untertitel in Datei speichern ---");
    service.write_all(output_filename, &subtitles)?;
    println!("Untertitel gespeichert in {}", output_filepath);

    // Inhalt der Ausgabedatei überprüfen
    let output_content = fs::read_to_string(&output_filepath)?;
    println!("\n--- Inhalt von {} ---", output_filename);
    println!("{}", output_content.trim());
    println!("---");

    // Erstellte Dateien aufräumen
    fs::remove_file(&sample_filepath)?;
    fs::remove_file(&output_filepath)?;
    println!("\nTemporäre Dateien bereinigt.");

    Ok(())
}
```

### Schritt-für-Schritt-Erklärung

1. **`SubRipService` instanziieren**:
   `SubRipService` benötigt ein Basisverzeichnis. Alle angegebenen Dateinamen sind relativ zu diesem Pfad.

   ```rust
   use lib::subtitle::service::SubRipService;
   let service = SubRipService::new("examples");
   ```

2. **Untertitel lesen**:
   Die Methode `get_all` liest eine .srt-Datei und gibt bei Erfolg einen `Vec<Subtitle>` zurück.

   ```rust
   let subtitles = service.get_all("sample.srt")?;
   ```

3. **Neuen Untertitel erstellen**:
   Sie können neue `Subtitle`-Instanzen programmatisch erstellen. Die Bibliothek verwendet `nutype`, um sicherzustellen, dass alle Daten gültig sind (z.B. müssen Untertitel-Indizes positiv sein, Text darf nicht leer sein).

   ```rust
   use chrono::Duration;
   use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};

   let new_subtitle = Subtitle::new(
       SubtitleIndex::try_new(3)?,
       SubtitleTimestamp::try_new(Duration::milliseconds(7000))?,
       SubtitleTimestamp::try_new(Duration::milliseconds(8000))?,
       SubtitleText::try_new("This is a new subtitle, created in code.".to_string())?,
   )?;
   ```

4. **Untertitel schreiben**:
   Die Methode `write_all` nimmt ein Slice von `Subtitle`s und schreibt sie in eine Datei, wobei sie diese überschreibt, falls sie existiert.

   ```rust
   service.write_all("output.srt", &subtitles)?;
   ```

## Weitere Erkundung

Für fortgeschrittenere Anwendungsfälle können Sie die anderen Methoden des `SubtitleService`-Traits erkunden, wie zum Beispiel:

- `get_by_id`: Einen einzelnen Untertitel anhand seines Index abrufen.
- `set`: Einen vorhandenen Untertitel aktualisieren.
- `add`: Einen neuen Untertitel an eine Datei anhängen.

Detailliertere Informationen finden Sie in der Quellcode-Dokumentation.
