# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Motivation

I prefer watching movies, cartoons, and anime with the original audio. However, subtitles are often unavailable in my language. While we can now use LLMs to translate subtitles, they aren't perfect. They sometimes make mistakes, such as messing up subtitle numbering or timestamps. Even the best models fail often.

Submarine is designed to assist the translation process by providing various editing and validation tools. It helps ensure that translated subtitles are accurate and consistent with the original content.

## Features

- Subtitle formats: [SubRip](https://en.wikipedia.org/wiki/SubRip), .srt
- Support flows: 
  - Chatbot: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md)
- **Toolset:**
  - Get subtitle by index
  - Add a new subtitle
  - Set subtitle by offset
  - Mass-rename subtitle files
  - Export subtitles in various formats (anchored)
- **Validation:**
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc). For example, you can compare english subtitles with translated by LLM.
- **Verification:**
  - Verify translated subtitles against the original content
  - Track translation progress
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
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv ResidentAlienS01E01.srt import.csv
$ sm import --format=anchored ResidentAlienS01E01.srt import.txt

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
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify ResidentAlienS01E01.eng.srt ResidentAlienS01E01.rus.srt
$ sm verify --range=1-50 ResidentAlienS01E01.eng.srt ResidentAlienS01E01.rus.srt 

Results
==================
 
Matched: 874/876 (99.8%)
Missing in FILE2: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in FILE2.srt)
  [...] ...

# Get translation progress
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.1080p.eng.srt Resident.Alien.S03E08.1080p.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Export subtitles in specified format
# sm export [FILE.srt] [RANGE] [--format=anchored]
$ sm export movie.eng.srt 1-50 --format=anchored

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.
```

## Usage in chat with LLM

- [Chatbot translation flow](docs/CHATBOT-TRANSLATION-FLOW.md)

## Usage with LLM agent

Put `sm` usage description in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` or whatever and tell LLM to use it as a tool.

## RoadMap

- Feature: sync
- Feature: merge
- Feature: adjust timestamps
