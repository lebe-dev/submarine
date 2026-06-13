// Command simple demonstrates using submarine as a Go library: loading
// subtitles from an .srt file, creating one programmatically, and writing the
// result back to a new file. It is the Go counterpart of the original
// examples/simple.rs.
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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// The library's service is file-based: every filename you pass is resolved
	// relative to this base directory. We use a temporary directory so the
	// example is self-contained.
	baseDir, err := os.MkdirTemp("", "submarine-example-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(baseDir)

	service := subtitle.NewSubRipService(baseDir)

	const sampleFilename = "sample.srt"
	const outputFilename = "output.srt"

	// Example 1: create a sample file and load subtitles from it.
	fmt.Println("--- Loading subtitles from a file ---")
	srtContent := "1\n00:00:03,000 --> 00:00:04,000\nThis is a sample subtitle.\n\n" +
		"2\n00:00:05,000 --> 00:00:06,000\nThis is another one.\n"
	if err := os.WriteFile(filepath.Join(baseDir, sampleFilename), []byte(srtContent), 0o644); err != nil {
		return err
	}

	subtitles, err := service.GetAll(sampleFilename)
	if err != nil {
		return err
	}

	fmt.Printf("Loaded %d subtitles from %s:\n", len(subtitles), sampleFilename)
	for _, sub := range subtitles {
		fmt.Println(strings.TrimSpace(sub.String()))
		fmt.Println("---")
	}

	// Example 2: create a subtitle programmatically and add it to the slice.
	fmt.Println("\n--- Creating a new subtitle and adding it ---")
	index, err := subtitle.NewSubtitleIndex(3)
	if err != nil {
		return err
	}
	start, err := subtitle.NewSubtitleTimestamp(7000 * time.Millisecond)
	if err != nil {
		return err
	}
	end, err := subtitle.NewSubtitleTimestamp(8000 * time.Millisecond)
	if err != nil {
		return err
	}
	text, err := subtitle.NewSubtitleText("This is a new subtitle, created in code.")
	if err != nil {
		return err
	}
	newSubtitle, err := subtitle.NewSubtitle(index, start, end, text)
	if err != nil {
		return err
	}
	fmt.Printf("Created new subtitle:\n%s\n", newSubtitle)
	subtitles = append(subtitles, newSubtitle)

	// Example 3: save the modified list of subtitles to a new file.
	fmt.Println("\n--- Saving subtitles to a file ---")
	if err := service.WriteAll(outputFilename, subtitles); err != nil {
		return err
	}
	fmt.Printf("Subtitles saved to %s\n", filepath.Join(baseDir, outputFilename))

	outputContent, err := os.ReadFile(filepath.Join(baseDir, outputFilename))
	if err != nil {
		return err
	}
	fmt.Printf("\n--- Content of %s ---\n%s\n---\n", outputFilename, strings.TrimSpace(string(outputContent)))

	return nil
}
