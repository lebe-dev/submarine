# Submarine

Submarine is a tiny toolkit for LLM translation assistance.

![Submarine Toolkit Logo](logo.png)

## Features

- Validation:
  - Check integrity
  - Compare with another subtitle files (quantity of subtitles, timestamps, etc)

## Usage

```bash
# Check integrity
# submarine validate [VALID-FILE.srt] [FILE-FOR-VALIDATION.srt]
submarine validate [--llm] [VALID-FILE.srt] [TARGET.srt]
```
