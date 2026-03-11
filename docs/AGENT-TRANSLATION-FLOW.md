# Agent translation flow

How to translate subtitles from one language to another using the submarine tool and LLM agent (Codex, Claude Code, opencode, etc.).

**Language:** **EN** | [RU](AGENT-TRANSLATION-FLOW.RU.md) | [DE](AGENT-TRANSLATION-FLOW.DE.md) | [ES](AGENT-TRANSLATION-FLOW.ES.md)

For example, you'd like to translate Episode 8 from Resident Alien Season 3 to Russian.

**Requirements:**
- Submarine CLI installed
- Agent installed
- English subtitle file for Resident Alien Season 3

Put in `CLAUDE.md` / `AGENTS.md` / `GEMINI.md`:

###[AGENTS.md]#######################################################

```markdown
## Submarine Translation Workflow

You are translating subtitles from English to Russian using the `sm` CLI tool.

### Step 1: Check status

touch Resident.Alien.S03E08.rus.srt

sm translation-status --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

This shows progress and suggests next chunk range.

### Step 2: Extract chunk for translation

sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100

Output format:

[1] English text here
[2] More english text
...

### Step 3: Translate and save to temp file

Create file `chunk.txt` with translations in same format:

[1] Русский перевод
[2] Ещё русский текст
...

IMPORTANT: Keep ALL indices. Do not skip any line.

### Step 4: Import translations

sm import --force --reference=Resident.Alien.S03E08.eng.srt --format=anchored Resident.Alien.S03E08.rus.srt chunk.txt 

### Step 5: Verify chunk

sm verify --range=1-50 Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

If missing indices reported, extract only those:

sm export --format=anchored Resident.Alien.S03E08.eng.srt 40-41

Translate missing, append to `chunk.txt`, import again.

### Step 6: Repeat for next chunk

Remove `chunk.txt`.

sm translation-status --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Continue until progress is 100%.

### Rules

- Chunk size: 100 subtitles
- Always verify after import
- Fix all missing before moving to next chunk
- Preserve original meaning, adapt idioms naturally
```

###[/AGENTS.md]#######################################################

Start agent and give it the following prompt:

```prompt
Let's translate subtitles for Resident Alien Season 3 Episode 8.
Take english subtitles from `Resident.Alien.S03E08.eng.srt` then use `sm` tool.

I expect `Resident.Alien.S03E08.rus.srt` as result.
```
