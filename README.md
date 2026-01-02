# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Motivation

I always watch movies, cartoons and anime with original audio. However, quite often subtitles aren't available on my language. Nowadays we have large language models (LLMs) that can help us translate subtitles. But they are not perfect and sometimes they make mistakes, put wrong subtitle numbers, timestamps, etc. Even best ones make mistakes.

Submarine is designed to assist in the translation process by providing various edit and validation tools. It helps ensure that the translated subtitles are accurate and consistent with the original content.

## Features

- **Toolset:**
  - Get subtitle by index
  - Add a new subtitle
  - Set subtitle by offset
  - Mass-rename subtitle files
- **Validation:**
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc). For example, you can compare english subtitles with translated by LLM.

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
# sm rename [FILE.srt] "[NEW-NAME]"
sm rename [--separator="."] [--name="Resident Alien"] [--series-mode] [--season=3] [--episodes=8] [--language="ru"] [FILE.srt]

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
