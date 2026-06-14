# Submarine

Submarine is a toolkit for subtitles translation with LLM assistance.

![Submarine Toolkit Logo](logo.png)

**Language:** **EN** | [RU](README.ru.md) | [DE](README.de.md) | [ES](README.es.md)

## Motivation

I prefer watching movies, cartoons, and anime with the original audio. However, subtitles are often unavailable in my language. While we can now use LLMs to translate subtitles, they aren't perfect. They sometimes make mistakes, such as messing up subtitle numbering or timestamps. Even the best models fail often.

Beyond missing subtitles, they can be incomplete or have compatibility issues with different releases.
Submarine is designed to assist the translation process by providing various editing and validation tools. It helps ensure that translated subtitles are accurate and consistent with the original content.

## Features

- Supported format: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Supported flows:
  - [Agent](docs/AGENT-TRANSLATION-FLOW.md) (Recommended)
  - [Chatbot](docs/CHATBOT-TRANSLATION-FLOW.md)
- **Toolset:**
  - Get subtitle by index or range
  - Add a new subtitle
  - Set/update subtitle properties
  - Import subtitles from file (CSV, anchored format)
  - Adjust timestamps by offset
  - Mass-rename subtitle files
  - Export subtitles in anchored format
  - Diagnose and fix file issues (`doctor`)
- **Verification:**
  - Verify translated subtitles against the original content
  - Track translation progress
- **Agent-friendly:**
  - Structured JSON output (`--output json`) on all commands
  - Dry-run preview (`--dry-run`) on all mutating commands
  - Schema introspection (`sm describe`)
  - Machine-readable error codes and hints
- **Auto-backups:** automatically create backups of your subtitle files before making changes.

## Installation

macOS (Homebrew), Linux, Docker — see [docs/install/](docs/install/README.md).

## Use cases

Task-oriented recipes for real subtitle problems — syncing, merging, comparing, and
cleaning up — with copy-paste commands and expected output. Full index —
[docs/usecases/](docs/usecases/README.md), including:

- [Merge an incomplete translation with a donor file](docs/usecases/merge-incomplete-translation-with-donor.md)
- [Detect and fix a constant sync offset](docs/usecases/detect-and-fix-constant-offset.md)
- [Fix frame-rate drift (23.976 ↔ 25)](docs/usecases/fix-framerate-drift-with-rescale.md)
- [Re-sync only part of a file](docs/usecases/resync-a-partial-range.md)
- [Join split parts (CD1/CD2) into one file](docs/usecases/join-split-parts-cd1-cd2.md)
- [Compare two subtitle tracks](docs/usecases/compare-two-subtitle-tracks.md)
- [Normalize structure: sort, renumber, fix overlaps](docs/usecases/normalize-structure.md)
- [Remove duplicates and overlaps after merging](docs/usecases/deduplicate-and-clean-after-merge.md)
- [Find missing dialogue with gap analysis](docs/usecases/find-missing-dialogue-gaps.md)

## How to use as library

In addition to its command-line interface, Submarine can be used as a library in your own Rust projects. For detailed information on how to integrate it, please see the [library documentation](docs/LIBRARY.md).
