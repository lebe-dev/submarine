use crate::dto::{ArgDescDto, CommandDescDto, DescribeDto};

/// Describe command always outputs JSON regardless of --output flag.
pub fn handle(command: Option<&str>) -> anyhow::Result<()> {
    let version = env!("CARGO_PKG_VERSION").to_string();

    let all_commands = build_command_descriptions();

    let commands = if let Some(cmd_name) = command {
        let found: Vec<CommandDescDto> = all_commands
            .into_iter()
            .filter(|c| c.name == cmd_name)
            .collect();

        if found.is_empty() {
            let envelope = serde_json::json!({
                "ok": false,
                "error": {
                    "code": "unknown_command",
                    "message": format!("Unknown command: {}", cmd_name)
                }
            });
            println!("{}", serde_json::to_string(&envelope)?);
            return Err(anyhow::anyhow!(""));
        }

        found
    } else {
        all_commands
    };

    let dto = DescribeDto {
        tool: "sm".into(),
        version,
        commands,
    };

    let envelope = serde_json::json!({
        "ok": true,
        "data": dto
    });
    println!("{}", serde_json::to_string_pretty(&envelope)?);

    Ok(())
}

fn build_command_descriptions() -> Vec<CommandDescDto> {
    vec![
        CommandDescDto {
            name: "get".into(),
            description: "Get a subtitle by its index from an SRT file".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "index".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "Subtitle index (>= 1) or range (e.g., 120-123)".into(),
                },
            ],
        },
        CommandDescDto {
            name: "set".into(),
            description: "Set/update subtitle properties by index".into(),
            mutates: true,
            supports_dry_run: true,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "index".into(),
                    arg_type: "u32".into(),
                    required: true,
                    description: "Subtitle index to update (>= 1)".into(),
                },
                ArgDescDto {
                    name: "--start".into(),
                    arg_type: "timestamp".into(),
                    required: false,
                    description: "New start timestamp (HH:MM:SS,mmm)".into(),
                },
                ArgDescDto {
                    name: "--end".into(),
                    arg_type: "timestamp".into(),
                    required: false,
                    description: "New end timestamp (HH:MM:SS,mmm)".into(),
                },
                ArgDescDto {
                    name: "--text".into(),
                    arg_type: "string".into(),
                    required: false,
                    description: "New subtitle text".into(),
                },
                ArgDescDto {
                    name: "--dry-run".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Preview changes without modifying the file".into(),
                },
            ],
        },
        CommandDescDto {
            name: "add".into(),
            description: "Add a new subtitle to the end of an SRT file".into(),
            mutates: true,
            supports_dry_run: true,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "timestamps".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "Timestamp range (HH:MM:SS,mmm-HH:MM:SS,mmm)".into(),
                },
                ArgDescDto {
                    name: "text".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "Subtitle text".into(),
                },
                ArgDescDto {
                    name: "--dry-run".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Preview changes without modifying the file".into(),
                },
            ],
        },
        CommandDescDto {
            name: "info".into(),
            description: "Display statistics and information about an SRT file".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![ArgDescDto {
                name: "file".into(),
                arg_type: "path".into(),
                required: true,
                description: "Path to the SRT file".into(),
            }],
        },
        CommandDescDto {
            name: "doctor".into(),
            description: "Diagnose and fix issues in SRT subtitle files".into(),
            mutates: true,
            supports_dry_run: false,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "--fix".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description:
                        "Fix issues automatically (without --fix, runs in diagnostic mode)".into(),
                },
            ],
        },
        CommandDescDto {
            name: "import".into(),
            description: "Import subtitles from a file into an SRT file".into(),
            mutates: true,
            supports_dry_run: true,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "srt_file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "input_file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the input file".into(),
                },
                ArgDescDto {
                    name: "--format".into(),
                    arg_type: "enum(csv,anchored)".into(),
                    required: true,
                    description: "Import format".into(),
                },
                ArgDescDto {
                    name: "--reference".into(),
                    arg_type: "path".into(),
                    required: false,
                    description: "Reference SRT file (required for anchored format)".into(),
                },
                ArgDescDto {
                    name: "--delimiter".into(),
                    arg_type: "char".into(),
                    required: false,
                    description: "Delimiter for CSV format (default: '|')".into(),
                },
                ArgDescDto {
                    name: "--dry-run".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Preview changes without modifying the file".into(),
                },
                ArgDescDto {
                    name: "--force".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Skip confirmation prompt".into(),
                },
            ],
        },
        CommandDescDto {
            name: "export".into(),
            description: "Export subtitles in specified format".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "range".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "Subtitle range (START-END, e.g., 1-50)".into(),
                },
                ArgDescDto {
                    name: "--format".into(),
                    arg_type: "enum(anchored)".into(),
                    required: true,
                    description: "Export format".into(),
                },
            ],
        },
        CommandDescDto {
            name: "verify".into(),
            description: "Verify two subtitle files for index and timestamp discrepancies".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file1".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the reference SRT file".into(),
                },
                ArgDescDto {
                    name: "file2".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the target SRT file".into(),
                },
                ArgDescDto {
                    name: "--range".into(),
                    arg_type: "string".into(),
                    required: false,
                    description: "Subtitle range to verify (START-END)".into(),
                },
            ],
        },
        CommandDescDto {
            name: "translation-status".into(),
            description: "Check translation progress against reference file".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "--reference".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the reference SRT file".into(),
                },
                ArgDescDto {
                    name: "translation".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the translation SRT file".into(),
                },
                ArgDescDto {
                    name: "--chunk-size".into(),
                    arg_type: "usize".into(),
                    required: false,
                    description: "Chunk size for translation suggestion (default: 50)".into(),
                },
            ],
        },
        CommandDescDto {
            name: "delay".into(),
            description: "Adjust subtitle timestamps by specified milliseconds offset".into(),
            mutates: true,
            supports_dry_run: true,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the SRT file".into(),
                },
                ArgDescDto {
                    name: "offset".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "Time offset in milliseconds (e.g., '+100', '-500')".into(),
                },
                ArgDescDto {
                    name: "--dry-run".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Preview changes without modifying the file".into(),
                },
            ],
        },
        CommandDescDto {
            name: "compare".into(),
            description: "Compare two SRT files side-by-side in TUI mode".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: false,
            args: vec![
                ArgDescDto {
                    name: "file1".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the first SRT file".into(),
                },
                ArgDescDto {
                    name: "file2".into(),
                    arg_type: "path".into(),
                    required: true,
                    description: "Path to the second SRT file".into(),
                },
            ],
        },
        CommandDescDto {
            name: "mass-rename".into(),
            description: "Mass rename subtitle files using templates".into(),
            mutates: true,
            supports_dry_run: true,
            supports_json: true,
            args: vec![
                ArgDescDto {
                    name: "file_mask".into(),
                    arg_type: "string".into(),
                    required: true,
                    description: "File mask for case-insensitive matching".into(),
                },
                ArgDescDto {
                    name: "--dry-run".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Show preview without renaming".into(),
                },
                ArgDescDto {
                    name: "--force".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Skip confirmation prompt".into(),
                },
                ArgDescDto {
                    name: "--series-mode".into(),
                    arg_type: "bool".into(),
                    required: false,
                    description: "Enable series mode with auto-incrementing episodes".into(),
                },
                ArgDescDto {
                    name: "--name".into(),
                    arg_type: "string".into(),
                    required: false,
                    description: "Series/show name".into(),
                },
                ArgDescDto {
                    name: "--season".into(),
                    arg_type: "u32".into(),
                    required: false,
                    description: "Season number".into(),
                },
                ArgDescDto {
                    name: "--language".into(),
                    arg_type: "string".into(),
                    required: false,
                    description: "Language code".into(),
                },
            ],
        },
        CommandDescDto {
            name: "describe".into(),
            description: "Describe available commands and their schemas (always JSON)".into(),
            mutates: false,
            supports_dry_run: false,
            supports_json: true,
            args: vec![ArgDescDto {
                name: "command".into(),
                arg_type: "string".into(),
                required: false,
                description: "Optional command name to describe".into(),
            }],
        },
    ]
}
