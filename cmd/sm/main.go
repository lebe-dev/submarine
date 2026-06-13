// Command sm is the SUBMARINE CLI binary. This file is the 1-to-1 Go port of
// the Rust `src/bin/cli/cli.rs` (the clap command/flag/alias tree) and
// `src/bin/cli/main.rs` (parse, init logging, dispatch to handlers, error
// handling, exit codes). It builds an equivalent urfave/cli/v3 command tree
// whose Actions dispatch to the cmd.HandleX handlers.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lebe-dev/submarine/internal/cli"
	"github.com/lebe-dev/submarine/internal/cli/cmd"
	ucli "github.com/urfave/cli/v3"
)

// version is the application version. It defaults to "dev" for local `go run`
// builds and is overridden at build time via
// `-ldflags "-X main.version=$(cat VERSION)"`, where the canonical value lives
// in the repo-root VERSION file. Replaces the Rust binary's
// env!("CARGO_PKG_VERSION").
var version = "dev"

func init() {
	// clap prints `--version` as "<bin> <version>" (e.g. "sm 0.14.0"); urfave/cli's
	// default prints "<name> version <version>". Override to match clap 1-to-1.
	ucli.VersionPrinter = func(c *ucli.Command) {
		fmt.Fprintf(c.Root().Writer, "%s %s\n", c.Root().Name, c.Root().Version)
	}
}

// optStr returns a pointer to the flag value when it was set, or nil otherwise.
// It is the equivalent of clap's `Option<String>` for optional flags.
func optStr(c *ucli.Command, name string) *string {
	if !c.IsSet(name) {
		return nil
	}
	v := c.String(name)
	return &v
}

// optUint32 returns a pointer to the uint32 flag value when set, or nil.
// Equivalent of clap's `Option<u32>`.
func optUint32(c *ucli.Command, name string) *uint32 {
	if !c.IsSet(name) {
		return nil
	}
	v := c.Uint32(name)
	return &v
}

// resolveOutputFormat reads the global --output flag and parses it into an
// OutputFormat. Mirrors clap's value-enum for `--output text|json`.
func resolveOutputFormat(c *ucli.Command) cli.OutputFormat {
	format, err := cli.ParseOutputFormat(c.String("output"))
	if err != nil {
		return cli.OutputFormatText
	}
	return format
}

func main() {
	// Global flags: port of the top-level clap args in `struct Cli`.
	// --output is global (applies to subcommands too), --log-level and
	// --log-target are root-only like in the Rust Cli struct.
	outputFlag := &ucli.StringFlag{
		Name:     "output",
		Usage:    "Output format (text, json)",
		Value:    "text",
		Local:    false, // global = true in clap
		OnlyOnce: true,
	}

	logLevelFlag := &ucli.StringFlag{
		Name:  "log-level",
		Usage: "Logging level (off, trace, debug, info, warn, error)",
		Value: "off",
		Local: true,
	}

	logTargetFlag := &ucli.StringFlag{
		Name:  "log-target",
		Usage: "Logging target (console, file)",
		Value: "console",
		Local: true,
	}

	root := &ucli.Command{
		Name:            "sm",
		Version:         version,
		Usage:           "SUBMARINE - tiny toolkit for LLM translation assistance",
		HideHelpCommand: true,
		Flags: []ucli.Flag{
			logLevelFlag,
			logTargetFlag,
			outputFlag,
		},
		Commands: buildCommands(),
		// No subcommand given: emit the same no_command error and exit(1).
		// Port of the `else` branch in Rust main.rs.
		Action: func(_ context.Context, c *ucli.Command) error {
			// If a subcommand actually matched, urfave routes to its Action and
			// this root Action is not called; reaching here means no command.
			cli.OutputError(
				cli.OutputFormatText,
				"no_command",
				"No command specified. Use --help for usage information.",
				nil,
			)
			os.Exit(1)
			return nil
		},
	}

	// init logging from flags before dispatch. Done in Before so it runs after
	// flag parsing but before any command Action. Port of main.rs lines that
	// build the logging config and init it.
	root.Before = func(_ context.Context, c *ucli.Command) (context.Context, error) {
		cli.InitLogging(c.String("log-level"), c.String("log-target"))
		return nil, nil
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		// Errors returned by handlers are surfaced here. The handlers that
		// already emitted output via cli.OutputError return an empty-message
		// error; only non-empty messages are formatted (mirrors main.rs).
		dispatchError(root, err)
	}
}

// dispatchError reproduces the error tail of Rust main.rs: format the handler
// error and, if its message is non-empty, emit it via output_error, then
// exit(1).
func dispatchError(root *ucli.Command, err error) {
	format := resolveOutputFormat(root)
	msg := err.Error()
	if msg != "" {
		cli.OutputError(format, "error", msg, nil)
	}
	os.Exit(1)
}

// buildCommands constructs the subcommand tree, a 1-to-1 port of the clap
// `enum Commands` in cli.rs (names, flags, value-enums, aliases).
func buildCommands() []*ucli.Command {
	return []*ucli.Command{
		// Get a subtitle by its index from an SRT file
		{
			Name:      "get",
			Usage:     "Get a subtitle by its index from an SRT file",
			ArgsUsage: "<FILE> <INDEX>",
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				index := c.Args().Get(1)
				return cmd.HandleGet(file, index, format)
			},
		},
		// Set/update subtitle properties by index
		{
			Name:      "set",
			Usage:     "Set/update subtitle properties by index",
			ArgsUsage: "<FILE> <INDEX>",
			Flags: []ucli.Flag{
				&ucli.StringFlag{Name: "start", Usage: "New start timestamp in format HH:MM:SS,mmm"},
				&ucli.StringFlag{Name: "end", Usage: "New end timestamp in format HH:MM:SS,mmm"},
				&ucli.StringFlag{Name: "text", Usage: "New subtitle text (can be multi-line with \\n)"},
				&ucli.BoolFlag{Name: "dry-run", Usage: "Preview changes without modifying the file"},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				index := parseIndexArg(c.Args().Get(1))
				start := optStr(c, "start")
				end := optStr(c, "end")
				text := optStr(c, "text")
				dryRun := c.Bool("dry-run")
				return cmd.HandleSet(file, index, start, end, text, dryRun, format)
			},
		},
		// Diagnose and fix issues in SRT subtitle files
		{
			Name:      "doctor",
			Usage:     "Diagnose and fix issues in SRT subtitle files",
			ArgsUsage: "<FILE>",
			Flags: []ucli.Flag{
				&ucli.BoolFlag{Name: "fix", Usage: "Fix issues automatically"},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				fix := c.Bool("fix")
				return cmd.HandleDoctor(file, fix, format)
			},
		},
		// Add a new subtitle to the end of an SRT file
		{
			Name:      "add",
			Usage:     "Add a new subtitle to the end of an SRT file",
			ArgsUsage: "<FILE> <TIMESTAMPS> <TEXT>",
			Flags: []ucli.Flag{
				&ucli.BoolFlag{Name: "dry-run", Usage: "Preview changes without modifying the file"},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				timestamps := c.Args().Get(1)
				text := c.Args().Get(2)
				dryRun := c.Bool("dry-run")
				return cmd.HandleAdd(file, timestamps, text, dryRun, format)
			},
		},
		// Display statistics and information about an SRT file
		{
			Name:      "info",
			Usage:     "Display statistics and information about an SRT file",
			ArgsUsage: "<FILE>",
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				return cmd.HandleInfo(file, format)
			},
		},
		// Import subtitles from a file into an SRT file
		{
			Name:      "import",
			Usage:     "Import subtitles from a file into an SRT file",
			ArgsUsage: "<SRT_FILE> <INPUT_FILE>",
			Flags: []ucli.Flag{
				&ucli.StringFlag{Name: "format", Usage: "Import format", Required: true},
				&ucli.StringFlag{Name: "reference", Usage: "Path to reference SRT file (required for anchored format)"},
				&ucli.StringFlag{Name: "delimiter", Usage: "Delimiter character for CSV format (default: pipe '|')", Value: "|"},
				&ucli.BoolFlag{Name: "dry-run", Usage: "Preview changes without modifying the file"},
				&ucli.BoolFlag{Name: "force", Usage: "Skip confirmation prompt"},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				srtFile := c.Args().Get(0)
				inputFile := c.Args().Get(1)
				importFormat, err := cli.ParseImportFormat(c.String("format"))
				if err != nil {
					return err
				}
				reference := optStr(c, "reference")
				delimiter := c.String("delimiter")
				dryRun := c.Bool("dry-run")
				force := c.Bool("force")
				return cmd.HandleImport(srtFile, inputFile, importFormat, reference, delimiter, dryRun, force, format)
			},
		},
		// Mass rename subtitle files using templates
		{
			Name:      "mass-rename",
			Usage:     "Mass rename subtitle files using templates",
			ArgsUsage: "<FILE-MASK>",
			Flags: []ucli.Flag{
				&ucli.BoolFlag{Name: "dry-run", Usage: "Show preview without renaming"},
				&ucli.BoolFlag{Name: "force", Usage: "Skip confirmation prompt"},
				&ucli.BoolFlag{Name: "series-mode", Usage: "Enable series mode with auto-incrementing episodes"},
				&ucli.StringFlag{Name: "name", Usage: "Series/show name"},
				&ucli.UintFlag{Name: "season", Usage: "Season number"},
				&ucli.StringFlag{Name: "language", Usage: "Language code"},
				&ucli.StringFlag{Name: "separator", Usage: "Separator character", Value: "."},
				&ucli.StringFlag{
					Name:  "file-template",
					Usage: "File name template (Tera syntax)",
					Value: "{{ name }}{{ separator }}S{{ season }}{{ separator }}E{{ episode }}.srt",
				},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				fileMask := c.Args().Get(0)
				dryRun := c.Bool("dry-run")
				force := c.Bool("force")
				seriesMode := c.Bool("series-mode")
				name := optStr(c, "name")
				season := optUint32(c, "season")
				language := optStr(c, "language")
				separator := c.String("separator")
				fileTemplate := c.String("file-template")
				return cmd.HandleMassRename(fileMask, dryRun, force, seriesMode, name, season, language, separator, fileTemplate, format)
			},
		},
		// Compare two SRT files side-by-side in TUI mode
		{
			Name:      "compare",
			Usage:     "Compare two SRT files side-by-side in TUI mode",
			ArgsUsage: "<FILE1> <FILE2>",
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file1 := c.Args().Get(0)
				file2 := c.Args().Get(1)
				return cmd.HandleCompare(file1, file2, format)
			},
		},
		// Verify two subtitle files for index and timestamp discrepancies
		{
			Name:      "verify",
			Usage:     "Verify two subtitle files for index and timestamp discrepancies",
			ArgsUsage: "<FILE1> <FILE2>",
			Flags: []ucli.Flag{
				&ucli.StringFlag{Name: "range", Usage: `Optional subtitle range in format START-END (e.g., "1-50")`},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file1 := c.Args().Get(0)
				file2 := c.Args().Get(1)
				rng := optStr(c, "range")
				return cmd.HandleVerify(file1, file2, rng, format)
			},
		},
		// Check translation progress against reference file
		{
			Name:      "translation-status",
			Aliases:   []string{"ts"},
			Usage:     "Check translation progress against reference file",
			ArgsUsage: "<TRANSLATION_FILE>",
			Flags: []ucli.Flag{
				&ucli.StringFlag{Name: "reference", Aliases: []string{"r"}, Usage: "Path to the reference SRT file", Required: true},
				&ucli.IntFlag{Name: "chunk-size", Usage: "Chunk size for next translation suggestion", Value: 50},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				reference := c.String("reference")
				translation := c.Args().Get(0)
				chunkSize := int(c.Int("chunk-size"))
				return cmd.HandleTranslationStatus(reference, translation, chunkSize, format)
			},
		},
		// Export subtitles in specified format
		{
			Name:      "export",
			Usage:     "Export subtitles in specified format",
			ArgsUsage: "<FILE> <RANGE>",
			Flags: []ucli.Flag{
				&ucli.StringFlag{Name: "format", Usage: "Export format", Required: true},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				rng := c.Args().Get(1)
				exportFormat, err := cli.ParseExportFormat(c.String("format"))
				if err != nil {
					return err
				}
				return cmd.HandleExport(file, rng, exportFormat, format)
			},
		},
		// Adjust subtitle timestamps by specified milliseconds offset
		{
			Name:      "delay",
			Usage:     "Adjust subtitle timestamps by specified milliseconds offset",
			ArgsUsage: "<FILE> <OFFSET>",
			Flags: []ucli.Flag{
				&ucli.BoolFlag{Name: "dry-run", Usage: "Preview changes without modifying the file"},
			},
			Action: func(_ context.Context, c *ucli.Command) error {
				format := resolveOutputFormat(c)
				file := c.Args().Get(0)
				// OFFSET is allow_hyphen_values in clap: "-500" / "+100" are
				// positional values. urfave stops flag parsing at a "-<digit>"
				// token, so it lands in Args().
				offset := c.Args().Get(1)
				dryRun := c.Bool("dry-run")
				return cmd.HandleDelay(file, offset, dryRun, format)
			},
		},
		// Describe available commands and their schemas (always JSON)
		{
			Name:      "describe",
			Usage:     "Describe available commands and their schemas (always JSON)",
			ArgsUsage: "[COMMAND]",
			Action: func(_ context.Context, c *ucli.Command) error {
				var command *string
				if c.Args().Present() {
					v := c.Args().Get(0)
					command = &v
				}
				return cmd.HandleDescribe(command)
			},
		},
	}
}

// parseIndexArg parses the positional INDEX argument of `set` into a uint32,
// mirroring clap's `index: u32` typed positional. Invalid input yields 0 (the
// downstream handler validates the subtitle index against the file).
func parseIndexArg(s string) uint32 {
	var n uint64
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + uint64(ch-'0')
	}
	if s == "" {
		return 0
	}
	return uint32(n)
}
