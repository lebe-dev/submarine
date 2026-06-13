package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lebe-dev/submarine/internal/cli"
)

// describeVersion is the port of the Rust `env!("CARGO_PKG_VERSION")` compile-time
// constant (from Cargo.toml: version = "0.14.0").
const describeVersion = "0.14.0"

// describeSuccessEnvelope is the port of the Rust
// `serde_json::json!({"ok": true, "data": dto})` envelope for describe.
type describeSuccessEnvelope struct {
	Ok   bool            `json:"ok"`
	Data cli.DescribeDto `json:"data"`
}

// describeErrorEnvelope is the port of the Rust
// `serde_json::json!({"ok": false, "error": {"code", "message"}})` envelope.
type describeErrorEnvelope struct {
	Ok    bool                 `json:"ok"`
	Error describeErrorPayload `json:"error"`
}

// describeErrorPayload mirrors the inner error object of the describe error envelope.
type describeErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HandleDescribe is the port of the Rust `cmd::describe::handle`. It always
// outputs JSON regardless of the --output flag.
func HandleDescribe(command *string) error {
	version := describeVersion

	allCommands := buildCommandDescriptions()

	var commands []cli.CommandDescDto
	if command != nil {
		cmdName := *command
		found := make([]cli.CommandDescDto, 0)
		for _, c := range allCommands {
			if c.Name == cmdName {
				found = append(found, c)
			}
		}

		if len(found) == 0 {
			envelope := describeErrorEnvelope{
				Ok: false,
				Error: describeErrorPayload{
					Code:    "unknown_command",
					Message: fmt.Sprintf("Unknown command: %s", cmdName),
				},
			}
			s, err := describeMarshal(envelope, false)
			if err != nil {
				return err
			}
			fmt.Println(s)
			return errors.New("")
		}

		commands = found
	} else {
		commands = allCommands
	}

	dto := cli.DescribeDto{
		Tool:     "sm",
		Version:  version,
		Commands: commands,
	}

	envelope := describeSuccessEnvelope{
		Ok:   true,
		Data: dto,
	}
	s, err := describeMarshal(envelope, true)
	if err != nil {
		return err
	}
	fmt.Println(s)

	return nil
}

// describeMarshal serializes v to JSON the way serde_json does: it does NOT
// HTML-escape '<', '>', '&' (Go's default does). When pretty is true it uses a
// 2-space indent like serde_json::to_string_pretty; otherwise compact like
// serde_json::to_string. The result carries no trailing newline (the caller adds
// one via println, matching the Rust `println!`).
func describeMarshal(v any, pretty bool) (string, error) {
	// The Rust describe command builds its output as a serde_json::Value, whose
	// object keys are stored in a BTreeMap and therefore serialized in sorted
	// order. Go's encoding/json preserves struct field order, so we round-trip
	// through a generic value (objects become map[string]any) — the encoder then
	// emits keys alphabetically, recursively, matching serde_json exactly.
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(generic); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// buildCommandDescriptions is the port of the Rust `build_command_descriptions`.
func buildCommandDescriptions() []cli.CommandDescDto {
	return []cli.CommandDescDto{
		{
			Name:           "get",
			Description:    "Get a subtitle by its index from an SRT file",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "index",
					ArgType:     "string",
					Required:    true,
					Description: "Subtitle index (>= 1) or range (e.g., 120-123)",
				},
			},
		},
		{
			Name:           "set",
			Description:    "Set/update subtitle properties by index",
			Mutates:        true,
			SupportsDryRun: true,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "index",
					ArgType:     "u32",
					Required:    true,
					Description: "Subtitle index to update (>= 1)",
				},
				{
					Name:        "--start",
					ArgType:     "timestamp",
					Required:    false,
					Description: "New start timestamp (HH:MM:SS,mmm)",
				},
				{
					Name:        "--end",
					ArgType:     "timestamp",
					Required:    false,
					Description: "New end timestamp (HH:MM:SS,mmm)",
				},
				{
					Name:        "--text",
					ArgType:     "string",
					Required:    false,
					Description: "New subtitle text",
				},
				{
					Name:        "--dry-run",
					ArgType:     "bool",
					Required:    false,
					Description: "Preview changes without modifying the file",
				},
			},
		},
		{
			Name:           "add",
			Description:    "Add a new subtitle to the end of an SRT file",
			Mutates:        true,
			SupportsDryRun: true,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "timestamps",
					ArgType:     "string",
					Required:    true,
					Description: "Timestamp range (HH:MM:SS,mmm-HH:MM:SS,mmm)",
				},
				{
					Name:        "text",
					ArgType:     "string",
					Required:    true,
					Description: "Subtitle text",
				},
				{
					Name:        "--dry-run",
					ArgType:     "bool",
					Required:    false,
					Description: "Preview changes without modifying the file",
				},
			},
		},
		{
			Name:           "info",
			Description:    "Display statistics and information about an SRT file",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
			},
		},
		{
			Name:           "doctor",
			Description:    "Diagnose and fix issues in SRT subtitle files",
			Mutates:        true,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "--fix",
					ArgType:     "bool",
					Required:    false,
					Description: "Fix issues automatically (without --fix, runs in diagnostic mode)",
				},
			},
		},
		{
			Name:           "import",
			Description:    "Import subtitles from a file into an SRT file",
			Mutates:        true,
			SupportsDryRun: true,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "srt_file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "input_file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the input file",
				},
				{
					Name:        "--format",
					ArgType:     "enum(csv,anchored)",
					Required:    true,
					Description: "Import format",
				},
				{
					Name:        "--reference",
					ArgType:     "path",
					Required:    false,
					Description: "Reference SRT file (required for anchored format)",
				},
				{
					Name:        "--delimiter",
					ArgType:     "char",
					Required:    false,
					Description: "Delimiter for CSV format (default: '|')",
				},
				{
					Name:        "--dry-run",
					ArgType:     "bool",
					Required:    false,
					Description: "Preview changes without modifying the file",
				},
				{
					Name:        "--force",
					ArgType:     "bool",
					Required:    false,
					Description: "Skip confirmation prompt",
				},
			},
		},
		{
			Name:           "export",
			Description:    "Export subtitles in specified format",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "range",
					ArgType:     "string",
					Required:    true,
					Description: "Subtitle range (START-END, e.g., 1-50)",
				},
				{
					Name:        "--format",
					ArgType:     "enum(anchored)",
					Required:    true,
					Description: "Export format",
				},
			},
		},
		{
			Name:           "verify",
			Description:    "Verify two subtitle files for index and timestamp discrepancies",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file1",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the reference SRT file",
				},
				{
					Name:        "file2",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the target SRT file",
				},
				{
					Name:        "--range",
					ArgType:     "string",
					Required:    false,
					Description: "Subtitle range to verify (START-END)",
				},
			},
		},
		{
			Name:           "translation-status",
			Description:    "Check translation progress against reference file",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "--reference",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the reference SRT file",
				},
				{
					Name:        "translation",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the translation SRT file",
				},
				{
					Name:        "--chunk-size",
					ArgType:     "usize",
					Required:    false,
					Description: "Chunk size for translation suggestion (default: 50)",
				},
			},
		},
		{
			Name:           "delay",
			Description:    "Adjust subtitle timestamps by specified milliseconds offset",
			Mutates:        true,
			SupportsDryRun: true,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the SRT file",
				},
				{
					Name:        "offset",
					ArgType:     "string",
					Required:    true,
					Description: "Time offset in milliseconds (e.g., '+100', '-500')",
				},
				{
					Name:        "--dry-run",
					ArgType:     "bool",
					Required:    false,
					Description: "Preview changes without modifying the file",
				},
			},
		},
		{
			Name:           "compare",
			Description:    "Compare two SRT files side-by-side in TUI mode",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   false,
			Args: []cli.ArgDescDto{
				{
					Name:        "file1",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the first SRT file",
				},
				{
					Name:        "file2",
					ArgType:     "path",
					Required:    true,
					Description: "Path to the second SRT file",
				},
			},
		},
		{
			Name:           "mass-rename",
			Description:    "Mass rename subtitle files using templates",
			Mutates:        true,
			SupportsDryRun: true,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "file_mask",
					ArgType:     "string",
					Required:    true,
					Description: "File mask for case-insensitive matching",
				},
				{
					Name:        "--dry-run",
					ArgType:     "bool",
					Required:    false,
					Description: "Show preview without renaming",
				},
				{
					Name:        "--force",
					ArgType:     "bool",
					Required:    false,
					Description: "Skip confirmation prompt",
				},
				{
					Name:        "--series-mode",
					ArgType:     "bool",
					Required:    false,
					Description: "Enable series mode with auto-incrementing episodes",
				},
				{
					Name:        "--name",
					ArgType:     "string",
					Required:    false,
					Description: "Series/show name",
				},
				{
					Name:        "--season",
					ArgType:     "u32",
					Required:    false,
					Description: "Season number",
				},
				{
					Name:        "--language",
					ArgType:     "string",
					Required:    false,
					Description: "Language code",
				},
			},
		},
		{
			Name:           "describe",
			Description:    "Describe available commands and their schemas (always JSON)",
			Mutates:        false,
			SupportsDryRun: false,
			SupportsJSON:   true,
			Args: []cli.ArgDescDto{
				{
					Name:        "command",
					ArgType:     "string",
					Required:    false,
					Description: "Optional command name to describe",
				},
			},
		},
	}
}
