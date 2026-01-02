# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Motivation

I always watch content in its original language, but subtitles aren't always available in my native tongue. LLMs can help with translation, but they often introduce errors in formatting, timestamps, or numbering.

Submarine is a tool designed to refine this process. It provides editing and validation features to ensure that LLM-generated subtitles are accurate, consistent, and correctly formatted.

## Features

- **Toolset:**
  - Get subtitle by index
  - Add a new subtitle
  - Set subtitle by offset
  - Timestamps management: add delay
  - Mass-rename subtitle files
- **Validation:**
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc). For example, you can compare english subtitles with translated by LLM.
- **Auto-backups:** automatically create backups of your subtitle files before making changes.

## Usage

```bash
# Get subtitle by index
# sm get [FILE.srt] [INDEX]
sm get [FILE.srt] [INDEX]

# Set subtitle for index
# sm set [FILE.srt] [INDEX] "[SUBTITLE]"
sm set [FILE.srt] [INDEX] "[SUBTITLE]"

# Add subtitle
# sm add [FILE.srt] "[NEW-SUBTITLE]"
sm add [FILE.srt] "[NEW-SUBTITLE]"

# Check integrity
# sm validate [VALID-FILE.srt] [FILE-FOR-VALIDATION.srt]
sm validate [--llm] [VALID-FILE.srt] [TARGET.srt]

# Mass rename
sm mass-rename [--name="Resident Alien"] \
          [--series-mode] [--season=3] [--episodes=8] \
          [--language="ru"] \
          [--separator="."] \
          [FILE-MASK]

# Adjust timestamps
# Delay in seconds
sm add-delay [FILE.srt] [DELAY]
```

## Usage with LLM agent

Put `sm` usage description in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md` or whatever and tell LLM to use it as a tool.

## RoadMap

- Feature: add a new subtitle
- Feature: set subtitle by offset
- Feature: mass-rename
- Feature: adjust timestamps
- Feature: auto-backups
