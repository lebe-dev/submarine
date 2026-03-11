# Flujo de traducción con chatbot

Cómo traducir subtítulos de un idioma a otro usando la herramienta submarine y un chatbot LLM (ChatGPT, Claude, DeepSeek, etc.).

Por ejemplo, quiere traducir el Episodio 8 de la Temporada 3 de Resident Alien al ruso.

**Idioma:** [EN](CHATBOT-TRANSLATION-FLOW.md) | [RU](CHATBOT-TRANSLATION-FLOW.RU.md) | [DE](CHATBOT-TRANSLATION-FLOW.DE.md) | **ES**

**Requisitos:**
- Submarine CLI instalado
- Archivo de subtítulos en inglés para Resident Alien Season 3

Primero es necesario exportar los subtítulos en un formato especial (anchored). Este formato es adecuado para cualquier LLM popular, ya que el formato SubRip original es demasiado complejo para los modelos modernos.

Exporte el fragmento de subtítulos y cópielo al portapapeles:

```bash
# Linux (X11)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | xclip

# Linux (Wayland)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | wl-copy

# MacOS
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | pbcopy
```

Prepare el prompt inicial para el chatbot:

```prompt
Vamos a traducir subtítulos del inglés al ruso para Resident Alien Season 3 Episode 8.

Te daré los subtítulos en un formato especial. Tu tarea es conservar el formato y traducirlos al ruso.

Reglas:
- Conservar el formato [ÍNDICE] texto
- Traducir TODAS las líneas, no omitir ninguna
- No añadir explicaciones, solo la traducción

Ejemplo de formato:
[1] Hello, how are you?
[2] I'm fine, thanks.
[3] What brings you here today?
...
[50] See you tomorrow.

### CONTENIDO ###

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

Luego cree el archivo `chatbot-import.md` y añada el contenido traducido:

```bash
touch chatbot-import.md

# Linux (X11)
xclip -selection clipboard -o >> chatbot-import.md

# Linux (Wayland)
wl-paste >> chatbot-import.md

# MacOS
pbpaste >> chatbot-import.md
```

Copie los siguientes fragmentos `101-200`, `201-300` hasta el final. Siempre añada al archivo `chatbot-import.md`.

Finalmente, cree el archivo de subtítulos:

```bash
touch Resident.Alien.S03E08.rus.srt

sm import --format=anchored --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt chatbot-import.md
```

Y verifique el resultado:

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
