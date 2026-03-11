# Chatbot Translation Flow

How to translate subtitles from one language to another using the submarine tool and LLM chatbot (ChatGPT, Claude, DeepSeek, etc.).

**Language:** **EN** | [RU](CHATBOT-TRANSLATION-FLOW.RU.md) | [DE](CHATBOT-TRANSLATION-FLOW.DE.md) | [ES](CHATBOT-TRANSLATION-FLOW.ES.md)

For example, you'd like to translate Episode 8 from Resident Alien Season 3 to Russian.

**Requirements:**
- Submarine CLI installed
- English subtitle file for Resident Alien Season 3

First of all we need to export subtitles in a special format (anchored). It's format is suitable for any popular LLM's cause original SubRip format is complicated for all modern LLM's.

Export subtitles chunk and copy to clipboard:

```bash
# Linux (X11)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | xclip

# Linux (Wayland)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | wl-copy

# MacOS
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | pbcopy
```

Prepare initial prompt for chatbot in such way:

```prompt
Let's translate subtitles from English to Russian for Resident Alien Season 3 Episode 8.

I'll give you the subtitles in a special format, your task is to save the format and translate them to Russian.

Rules:
- Save the format [INDEX] text
- Translate ALL lines, don't skip any
- Don't add explanations, only translation

Format example:
[1] Hello, how are you?
[2] I'm fine, thanks.
[3] What brings you here today?
...
[50] See you tomorrow.

### CONTENT ###

[1] <i>Previously on "Resident Alien"...</i>
[2] Get in!
[3] Maybe he's not dead.
- What?
[4] We have to go back and see.
[5] <i>Joseph is an alien.</i>
[6] They're real.
[7] <i>They abduct people,</i>
[8] <i>and they don't remember.</i>
[9] No!
[10] At least let me remember
this time.
```

Then create `chatbot-import.md` file and append translated content:

```bash
touch chatbot-import.md

# Linux (X11)
xclip -selection clipboard -o >> chatbot-import.md

# Linux (Wayland)
wl-paste >> chatbot-import.md

# MacOS
pbpaste >> chatbot-import.md
```

Then copy next chunks `101-200`, `201-300` until the end. Always append to
`chatbot-import.md` file.

Finally, create subtitle file:

```bash
touch Resident.Alien.S03E08.rus.srt

sm import --format=anchored --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt chatbot-import.md
```

And verify the result:

```bash
sm verify --format=anchored Resident.Alien.S03E08.rus.srt Resident.Alien.S03E08.eng.srt

Verifying subtitle files
========================

Reference file: Resident.Alien.S03E08.eng.srt (876 subtitles)
Target file:    Resident.Alien.S03E08.rus.srt (876 subtitles)

Results
=======

Matched: 876/876 (100.0%)

Verification: SUCCESS
```
