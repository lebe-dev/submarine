# Usar Submarine como biblioteca

Submarine está diseñado principalmente como herramienta de línea de comandos, pero su funcionalidad principal también está disponible como biblioteca que puede integrar en sus propios proyectos Go.

**Idioma:** [EN](LIBRARY.md) | [DE](LIBRARY.DE.md) | **ES**

Esta guía le explica los conceptos básicos para usar la biblioteca `submarine`.

## Añadir `submarine` a su proyecto

`submarine` es un módulo de Go. Añádalo a su proyecto con `go get`:

```bash
go get github.com/lebe-dev/submarine
```

Requiere Go 1.26 o superior. Los paquetes reutilizables de la biblioteca se encuentran en `pkg/` (el código exclusivo de la CLI permanece en `internal/` y no es importable).

## Conceptos principales

La funcionalidad de la biblioteca se centra en la interfaz `Service` del paquete `pkg/subtitle`, que define las operaciones principales para trabajar con archivos de subtítulos. La implementación principal es `SubRipService`, que trabaja con archivos SubRip (.srt).

- **`subtitle.Subtitle`**: representa una entrada de subtítulo individual: su índice, las marcas de tiempo de inicio y fin, y el texto.
- **`subtitle.SubRipService`**: el punto de entrada para la mayoría de las operaciones basadas en archivos. Permite leer, escribir y modificar archivos .srt.

Los tipos de valor validados (`SubtitleIndex`, `SubtitleTimestamp`, `SubtitleText`) se construyen mediante funciones `New…` que devuelven un `error` cuando la entrada no es válida (por ejemplo, un índice debe ser `>= 1`, el texto no puede estar vacío). Esto refleja las garantías que la versión en Rust imponía con `nutype`.

## Uso básico

Aquí tiene un ejemplo sencillo de cómo usar la biblioteca para leer subtítulos de un archivo, añadir uno nuevo y escribir el resultado en un archivo nuevo. El programa completo y ejecutable está en [`examples/simple/main.go`](../examples/simple/main.go); ejecútelo con `go run ./examples/simple`.

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
	// El servicio se basa en archivos; cada nombre de archivo se resuelve en
	// relación con este directorio base. Usamos un directorio temporal para que
	// el ejemplo sea autónomo.
	baseDir, err := os.MkdirTemp("", "submarine-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	service := subtitle.NewSubRipService(baseDir)

	// 1. Crear un archivo de ejemplo y cargar los subtítulos desde él.
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

	// 2. Crear un subtítulo mediante código.
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

	// 3. Guardar la lista modificada en un archivo nuevo.
	if err := service.WriteAll("output.srt", subtitles); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Subtitles saved to output.srt")
}
```

### Explicación paso a paso

1. **Instanciar `SubRipService`**:
   El servicio necesita un directorio base. Todos los nombres de archivo que proporcione se resuelven en relación con esta ruta.

   ```go
   service := subtitle.NewSubRipService(baseDir)
   ```

2. **Leer subtítulos**:
   `GetAll` lee un archivo .srt y devuelve un `[]subtitle.Subtitle`.

   ```go
   subtitles, err := service.GetAll("sample.srt")
   ```

3. **Crear un subtítulo nuevo**:
   Construya un `Subtitle` a partir de tipos de valor validados. Cada constructor `New…` devuelve un `error` cuando la entrada no es válida (los índices deben ser positivos, el texto no puede estar vacío, el fin debe ser posterior al inicio).

   ```go
   index, err := subtitle.NewSubtitleIndex(3)
   start, err := subtitle.NewSubtitleTimestamp(7000 * time.Millisecond)
   end, err := subtitle.NewSubtitleTimestamp(8000 * time.Millisecond)
   text, err := subtitle.NewSubtitleText("This is a new subtitle, created in code.")
   newSubtitle, err := subtitle.NewSubtitle(index, start, end, text)
   ```

4. **Escribir subtítulos**:
   `WriteAll` toma un slice de `Subtitle` y los escribe en un archivo, sobrescribiéndolo si ya existe.

   ```go
   err := service.WriteAll("output.srt", subtitles)
   ```

## Exploración adicional

Para un uso más avanzado, explore los demás métodos de la interfaz `subtitle.Service`:

- `GetByID(filename, id)` — obtener un único subtítulo por su índice (devuelve `(nil, nil)` cuando no se encuentra).
- `Set(filename, id, update)` — actualizar un subtítulo existente.
- `Add(filename, start, end, text)` — añadir un nuevo subtítulo a un archivo.

Además de `pkg/subtitle`, los demás paquetes de la biblioteca cubren el resto de las funciones del conjunto de herramientas:

| Paquete | Propósito |
|---|---|
| `pkg/subtitle` | Modelo de dominio principal, lectura/escritura de SRT (`SubRipService`) |
| `pkg/backup` | Copias de seguridad de archivos con marca de tiempo (`SubRipBackupService`) |
| `pkg/doctor` | Diagnosticar y reparar archivos SRT con errores |
| `pkg/importer` | Importar subtítulos desde formatos CSV y anclado |
| `pkg/rename` | Renombrado masivo de archivos de subtítulos basado en plantillas |
| `pkg/verify` | Comparar dos archivos en busca de discrepancias de índice/marca de tiempo (`CompareSubtitles`) |
| `pkg/translationstatus` | Progreso de traducción respecto a una referencia (`CheckTranslationStatus`) |

## Nota sobre el registro (logging)

La biblioteca registra mensajes a través del paquete estándar `log/slog`. De forma predeterminada, `slog` de Go escribe registros de nivel `Info` en stderr, por lo que puede ver líneas de registro al llamar a las funciones de la biblioteca. Para controlarlas o silenciarlas, instale su propio handler predeterminado, por ejemplo:

```go
import "log/slog"

// Mostrar solo advertencias y niveles superiores.
slog.SetLogLoggerLevel(slog.LevelWarn)
```
