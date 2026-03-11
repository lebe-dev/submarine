# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

**Language:** **EN** | [RU](README.ru.md) | [DE](README.de.md) | [ES](README.es.md)

## Motivation

I prefer watching movies, cartoons, and anime with the original audio. However, subtitles are often unavailable in my language. While we can now use LLMs to translate subtitles, they aren't perfect. They sometimes make mistakes, such as messing up subtitle numbering or timestamps. Even the best models fail often.

Submarine is designed to assist the translation process by providing various editing and validation tools. It helps ensure that translated subtitles are accurate and consistent with the original content.

## Features

- Subtitles format: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Supported flows:
  - Agent: [Eng](docs/AGENT-TRANSLATION-FLOW.md), [Rus](docs/AGENT-TRANSLATION-FLOW.RU.md), [De](docs/AGENT-TRANSLATION-FLOW.DE.md), [Es](docs/AGENT-TRANSLATION-FLOW.ES.md)
  - Chatbot: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md), [De](docs/CHATBOT-TRANSLATION-FLOW.DE.md), [Es](docs/CHATBOT-TRANSLATION-FLOW.ES.md)
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

### MacOS

Homebrew ([lebe-dev tap](https://github.com/lebe-dev/homebrew-tap)):

```bash
brew install lebe-dev/tap/submarine
```

### Linux

```bash
curl -L -o sm-0.14.0-linux-amd64.zip \
  https://github.com/lebe-dev/submarine/releases/download/0.14.0/sm-0.14.0-linux-amd64.zip
unzip sm-0.14.0-linux-amd64.zip
sudo install -m 0755 sm /usr/local/bin/sm
sm --help
```


## Usage

```bash
# Show information about subtitles file
# sm info [FILE.srt]
$ sm info Resident.Alien.S01E01.srt

# Get subtitle by index or range
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get Resident.Alien.S01E01.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# Also supports range syntax
$ sm get Resident.Alien.S01E01.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Set subtitle for index
# sm set [--dry-run] [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Preview changes without modifying the file
$ sm set --dry-run Resident.Alien.S01E01.srt 123 --text "Okay"

# Add subtitle to the end of file
# Automatically increment index and makes backup
# Creates srt file if not exists
# sm add [--dry-run] [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Adjust subtitle timestamps by offset
# Supports positive and negative offsets in milliseconds
# sm delay [--dry-run] [FILE.srt] [OFFSET]
$ sm delay Resident.Alien.S01E01.srt "+1000"  # Add 1 second
$ sm delay Resident.Alien.S01E01.srt "-500"   # Subtract 0.5 seconds

# Import subtitles from csv file
# Creates srt file if not exists
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv Resident.Alien.S01E01.srt import.csv
$ sm import --format=anchored Resident.Alien.S01E01.srt import.txt

# Check file integrity
# sm doctor [--fix] [FILE.srt]
sm doctor --fix Resident.Alien.S01E01.eng.srt

# Mass rename
# - file-mask is case-insensitive
# sm mass-rename [--dry-run] [--force] [--name="Resident Alien"] \
#          [--series-mode] [--season=3] \
#          [--language="rus"] \
#          [--separator="."] \
#          [--file-template="{{ name }}{{ separator }}S{{ season }}{{ separator }}E{{ episode }}.srt"] \
#          [FILE-MASK]
$ sm mass-rename --dry-run \
          --name="Resident Alien" \
          --series-mode --season=3 \
          --separator="." \
          "Resident"

# Compare subtitles in interactive mode
# sm compare [FILE1.srt] [FILE2.srt]
$ sm compare Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

# Verify subtitle files
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt
$ sm verify --range=1-50 Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt 

Results
==================
 
Matched: 874/876 (99.8%)
Missing in Resident.Alien.S01E01.rus.srt: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in Resident.Alien.S01E01.rus.srt)
  [...] ...

# Get translation progress
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Export subtitles in specified format
# sm export [--format=anchored] [FILE.srt] [RANGE]
$ sm export --format=anchored movie.eng.srt 1-50

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.

# JSON output (available on all commands except compare)
$ sm get Resident.Alien.S01E01.srt 1 --output json
{"ok":true,"data":{"index":1,"start_time":"00:00:01,436",...}}

$ sm info Resident.Alien.S01E01.srt --output json

# Discover available commands and their schemas
$ sm describe
$ sm describe get
```

## How to use as library

In addition to its command-line interface, Submarine can be used as a library in your own Rust projects. For detailed information on how to integrate it, please see the [library documentation](docs/LIBRARY.md).

## RoadMap

- Feature: sync
- Feature: merge
