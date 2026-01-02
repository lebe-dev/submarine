# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Motivation

I always watch movies, cartoons and anime with original audio. However, quite often subtitles aren't available on my language.
Nowadays we have large language models (LLMs) that can help us translate subtitles. But they are not perfect and sometimes they make mistakes, put wrong subtitle numbers, timestamps, etc.

Submarine is designed to assist in the translation process by providing validation and comparison features. It helps ensure that the translated subtitles are accurate and consistent with the original content.

## Features

- **Validation:**
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc). For example, you can compare english subtitles with translated by LLM.
- **Toolset:**
  - Edit SRT: replace text by index
  - Mass-rename subtitle files

## Usage

```bash
# Check integrity
# submarine validate [VALID-FILE.srt] [FILE-FOR-VALIDATION.srt]
submarine validate [--llm] [VALID-FILE.srt] [TARGET.srt]

# Replace text by index
# submarine replace [FILE.srt] [INDEX] "[NEW-TEXT]"
submarine replace [FILE.srt] [INDEX] "[NEW-TEXT]"
```

## RoadMap

- Feature: edit by offset
- Feature: mass-rename
