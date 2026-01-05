# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Motivation

I prefer watching movies, cartoons, and anime with the original audio. However, subtitles are often unavailable in my language. While we can now use LLMs to translate subtitles, they aren't perfect. They sometimes make mistakes, such as messing up subtitle numbering or timestamps. Even the best models fail often.

Submarine is designed to assist the translation process by providing various editing and validation tools. It helps ensure that translated subtitles are accurate and consistent with the original content.

## Features

- Subtitle formats: [SubRip](https://en.wikipedia.org/wiki/SubRip), .srt
- **Toolset:**
  - Get subtitle by index
  - Add a new subtitle
  - Set subtitle by offset
  - Timestamps management: add delay
  - Mass-rename subtitle files
- **Validation:**
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc). For example, you can compare english subtitles with translated by LLM.
- **Verification:**
  - Verify translated subtitles against the original content
- **Auto-backups:** automatically create backups of your subtitle files before making changes.

## Usage

```bash
# Show information about subtitles file
# sm info [FILE.srt]
$ sm info FILE.srt

# Get subtitle by index or range
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get FILE.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# Also supports range syntax
$ sm get FILE.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Set subtitle for index
# sm set [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]  
$ sm set ResidentAlienS01E01.srt 123 \
       --text "Okay"

# Add subtitle to the end of file
# Automatically increment index and makes backup
# Creates srt file if not exists
# sm add [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add ResidentAlienS01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Import subtitles from csv file
# Creates srt file if not exists
# sm import [--dry-run] [--force] [FILE.srt] [IMPORT.csv]
$ sm import ResidentAlienS01E01.srt import.csv

# Check file integrity
# sm doctor [--fix] [FILE.srt]
sm doctor --fix ResidentAlienS01E01.eng.srt

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
$ sm compare FILE1.srt FILE2.srt

# Verify subtitle files
# sm verity [REFERENCE-FILE] [FILE2]
$ sm verify ResidentAlienS01E01.eng.srt ResidentAlienS01E01.rus.srt

Results
==================
 
Matched: 874/876 (99.8%)
Missing in FILE2: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in FILE2.srt)
  [...] ...

# Adjust timestamps
# Delay in seconds
# sm add-delay [FILE.srt] [DELAY]
```

## Usage in chat with LLM

Save content into srt-file and check:

```bash
sm doctor your-file.srt
```

## Usage with LLM agent

Put `sm` usage description in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` or whatever and tell LLM to use it as a tool.

## RoadMap

- Feature: sync
- Feature: merge
- Feature: adjust timestamps
