# Usar Submarine como biblioteca

Submarine está diseñado principalmente como herramienta de línea de comandos, pero su funcionalidad principal también está disponible como biblioteca que puede integrar en sus propios proyectos Rust. Esta guía le explica los conceptos básicos para usar la biblioteca `submarine-rs`.

**Idioma:** [EN](LIBRARY.md) | [DE](LIBRARY.DE.md) | **ES**

## Añadir `submarine-rs` a su proyecto

Para usar `submarine-rs` como biblioteca, primero añádala a su archivo `Cargo.toml`.

```toml
[dependencies]
submarine-rs = { git = "https://github.com/lebe-dev/submarine" }
```

Nota: Como `submarine-rs` todavía no está publicada en crates.io, debe añadirla directamente desde su repositorio Git.

## Conceptos principales

La funcionalidad de la biblioteca se centra en el trait `SubtitleService`, que define las operaciones principales para trabajar con archivos de subtítulos. La implementación principal de este trait es `SubRipService`, que trabaja con archivos SubRip (.srt).

- **`Subtitle`**: Esta struct representa una única entrada de subtítulo, incluyendo su índice, marcas de tiempo de inicio y fin, y texto.
- **`SubRipService`**: Es el punto de entrada para la mayoría de las operaciones basadas en archivos. Permite leer, escribir y modificar archivos .srt.

## Uso básico

A continuación se muestra un ejemplo simple de cómo usar la biblioteca para leer subtítulos de un archivo, añadir un nuevo subtítulo y guardar el resultado en un nuevo archivo.

El código completo también está disponible en `examples/simple.rs`.

```rust
use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use std::fs;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // El servicio de la biblioteca está basado en archivos.
    // Creamos una instancia del servicio que opera en el directorio 'examples'.
    let service = SubRipService::new("examples");
    let sample_filename = "sample.srt";
    let output_filename = "output.srt";
    let sample_filepath = format!("examples/{}", sample_filename);
    let output_filepath = format!("examples/{}", output_filename);

    // Ejemplo 1: Crear un archivo de muestra y cargar subtítulos de él
    println!("--- Cargando subtítulos de un archivo ---");
    let srt_content = "1\n00:00:03,000 --> 00:00:04,000\nThis is a sample subtitle.\n\n2\n00:00:05,000 --> 00:00:06,000\nThis is another one.\n";
    fs::write(&sample_filepath, srt_content)?;

    let mut subtitles = service.get_all(sample_filename)?;

    println!("Cargados {} subtítulos de {}:", subtitles.len(), sample_filename);
    for sub in &subtitles {
        println!("{}", sub.to_string().trim());
        println!("---");
    }

    // Ejemplo 2: Crear un subtítulo programáticamente y añadirlo al vector
    println!("\n--- Creando un nuevo subtítulo y añadiéndolo ---");
    let new_subtitle = Subtitle::new(
        SubtitleIndex::try_new(3)?,
        SubtitleTimestamp::try_new(Duration::milliseconds(7000))?,
        SubtitleTimestamp::try_new(Duration::milliseconds(8000))?,
        SubtitleText::try_new("This is a new subtitle, created in code.".to_string())?,
    )?;
    println!("Nuevo subtítulo creado:\n{}", new_subtitle);
    subtitles.push(new_subtitle);

    // Ejemplo 3: Guardar la lista modificada de subtítulos en un nuevo archivo
    println!("\n--- Guardando subtítulos en un archivo ---");
    service.write_all(output_filename, &subtitles)?;
    println!("Subtítulos guardados en {}", output_filepath);

    // Verificar el contenido del archivo de salida
    let output_content = fs::read_to_string(&output_filepath)?;
    println!("\n--- Contenido de {} ---", output_filename);
    println!("{}", output_content.trim());
    println!("---");

    // Limpiar los archivos creados
    fs::remove_file(&sample_filepath)?;
    fs::remove_file(&output_filepath)?;
    println!("\nArchivos temporales eliminados.");

    Ok(())
}
```

### Explicación paso a paso

1. **Instanciar `SubRipService`**:
   `SubRipService` necesita un directorio base. Todos los nombres de archivo que proporcione serán relativos a esta ruta.

   ```rust
   use lib::subtitle::service::SubRipService;
   let service = SubRipService::new("examples");
   ```

2. **Leer subtítulos**:
   El método `get_all` lee un archivo .srt y devuelve un `Vec<Subtitle>` si tiene éxito.

   ```rust
   let subtitles = service.get_all("sample.srt")?;
   ```

3. **Crear un nuevo subtítulo**:
   Puede crear nuevas instancias de `Subtitle` programáticamente. La biblioteca usa `nutype` para garantizar que todos los datos sean válidos (por ejemplo, los índices deben ser positivos, el texto no puede estar vacío).

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

4. **Escribir subtítulos**:
   El método `write_all` toma un slice de `Subtitle`s y los escribe en un archivo, sobreescribiéndolo si ya existe.

   ```rust
   service.write_all("output.srt", &subtitles)?;
   ```

## Exploración adicional

Para usos más avanzados, puede explorar otros métodos disponibles en el trait `SubtitleService`, como:

- `get_by_id`: Recuperar un único subtítulo por su índice.
- `set`: Actualizar un subtítulo existente.
- `add`: Añadir un nuevo subtítulo a un archivo.

Puede encontrar información más detallada en la documentación del código fuente.
