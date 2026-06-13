# Using Submarine as a Library

Submarine is designed primarily as a command-line tool, but its core functionality is available as a library that you can integrate into your own Go projects.

**Language:** **EN** | [DE](LIBRARY.DE.md) | [ES](LIBRARY.ES.md)

This guide will walk you through the basics of using the `submarine` library.

## Adding `submarine` to Your Project

`submarine` is a Go module. Add it to your project with `go get`:

```bash
go get github.com/lebe-dev/submarine
```

It requires Go 1.26 or newer. The reusable library packages live under `pkg/` (the CLI-only code stays in `internal/` and is not importable).

## Core Concepts

The library's functionality is centered around the `Service` interface in the `pkg/subtitle` package, which defines the primary operations for working with subtitle files. The main implementation is `SubRipService`, which works with SubRip (.srt) files.

- **`subtitle.Subtitle`**: represents a single subtitle entry — its index, start and end timestamps, and text.
- **`subtitle.SubRipService`**: the entry point for most file-based operations. It lets you read, write, and modify .srt files.

Validated value types (`SubtitleIndex`, `SubtitleTimestamp`, `SubtitleText`) are constructed through `New…` functions that return an `error` when the input is invalid (e.g. an index must be `>= 1`, text must be non-empty). This mirrors the guarantees the Rust version enforced with `nutype`.

## Basic Usage

Here's a simple example of how to use the library to read subtitles from a file, add a new subtitle, and write the results to a new file. The complete, runnable program is in [`examples/simple/main.go`](../examples/simple/main.go) — run it with `go run ./examples/simple`.

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
	// The service is file-based; every filename is resolved relative to this
	// base directory. We use a temporary directory to keep the example self-contained.
	baseDir, err := os.MkdirTemp("", "submarine-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	service := subtitle.NewSubRipService(baseDir)

	// 1. Create a sample file and load subtitles from it.
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

	// 2. Create a subtitle programmatically.
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

	// 3. Save the modified list to a new file.
	if err := service.WriteAll("output.srt", subtitles); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Subtitles saved to output.srt")
}
```

### Step-by-Step Explanation

1. **Instantiate `SubRipService`**:
   The service needs a base directory to work from. All filenames you provide are resolved relative to this path.

   ```go
   service := subtitle.NewSubRipService(baseDir)
   ```

2. **Reading Subtitles**:
   `GetAll` reads a .srt file and returns a `[]subtitle.Subtitle`.

   ```go
   subtitles, err := service.GetAll("sample.srt")
   ```

3. **Creating a New Subtitle**:
   You build a `Subtitle` from validated value types. Each `New…` constructor returns an `error` when the input is invalid (indices must be positive, text cannot be empty, end must be after start).

   ```go
   index, err := subtitle.NewSubtitleIndex(3)
   start, err := subtitle.NewSubtitleTimestamp(7000 * time.Millisecond)
   end, err := subtitle.NewSubtitleTimestamp(8000 * time.Millisecond)
   text, err := subtitle.NewSubtitleText("This is a new subtitle, created in code.")
   newSubtitle, err := subtitle.NewSubtitle(index, start, end, text)
   ```

4. **Writing Subtitles**:
   `WriteAll` takes a slice of `Subtitle`s and writes them to a file, overwriting it if it exists.

   ```go
   err := service.WriteAll("output.srt", subtitles)
   ```

## Further Exploration

For more advanced usage, explore the other methods on the `subtitle.Service` interface:

- `GetByID(filename, id)` — retrieve a single subtitle by its index (returns `(nil, nil)` when not found).
- `Set(filename, id, update)` — update an existing subtitle.
- `Add(filename, start, end, text)` — append a new subtitle to a file.

Beyond `pkg/subtitle`, the other library packages cover the remaining toolkit features:

| Package | Purpose |
|---|---|
| `pkg/subtitle` | Core domain model, SRT parsing/writing (`SubRipService`) |
| `pkg/backup` | Timestamped file backups (`SubRipBackupService`) |
| `pkg/doctor` | Diagnose and fix malformed SRT files |
| `pkg/importer` | Import subtitles from CSV and anchored formats |
| `pkg/rename` | Template-based mass renaming of subtitle files |
| `pkg/verify` | Compare two files for index/timestamp discrepancies (`CompareSubtitles`) |
| `pkg/translationstatus` | Translation progress against a reference (`CheckTranslationStatus`) |

## A Note on Logging

The library logs through the standard `log/slog` package. By default Go's `slog` writes `Info`-level records to stderr, so you may see log lines when calling library functions. To control or silence them, install your own default handler, e.g.:

```go
import "log/slog"

// Only show warnings and above.
slog.SetLogLoggerLevel(slog.LevelWarn)
```
