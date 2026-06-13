# Submarine

Submarine es un pequeño toolkit de asistencia para la traducción de subtítulos con LLM.

![Logo de Submarine](logo.png)

**Idioma:** [EN](README.md) | [RU](README.ru.md) | [DE](README.de.md) | **ES**

## Motivación

Prefiero ver películas, dibujos animados y anime en audio original. Sin embargo, los subtítulos en mi idioma suelen no estar disponibles. Aunque los LLM ya permiten traducirlos, no son perfectos: a veces mezclan numeraciones o marcas de tiempo. Incluso los mejores modelos cometen errores con frecuencia.

Submarine está diseñado para ayudar en el proceso de traducción con herramientas de edición y validación, garantizando que los subtítulos traducidos sean precisos y coherentes con el contenido original.

## Características

- Formato de subtítulos: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Flujos de trabajo compatibles:
  - Agente: [Eng](docs/AGENT-TRANSLATION-FLOW.md), [Rus](docs/AGENT-TRANSLATION-FLOW.RU.md), [De](docs/AGENT-TRANSLATION-FLOW.DE.md), [Es](docs/AGENT-TRANSLATION-FLOW.ES.md)
  - Chatbot: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md), [De](docs/CHATBOT-TRANSLATION-FLOW.DE.md), [Es](docs/CHATBOT-TRANSLATION-FLOW.ES.md)
- **Herramientas:**
  - Obtener subtítulo por índice o rango
  - Añadir un nuevo subtítulo
  - Modificar propiedades de un subtítulo
  - Importar subtítulos desde archivo (CSV, formato anclado)
  - Ajustar marcas de tiempo con desplazamiento
  - Renombrado masivo de archivos de subtítulos
  - Exportar subtítulos en formato anclado
  - Diagnosticar y reparar problemas del archivo (`doctor`)
- **Verificación:**
  - Verificar subtítulos traducidos contra el original
  - Seguimiento del progreso de traducción
- **Adaptado para agentes:**
  - Salida JSON estructurada (`--output json`) en todos los comandos
  - Vista previa de cambios (`--dry-run`) en todos los comandos que modifican archivos
  - Introspección del esquema (`sm describe`)
  - Códigos de error y sugerencias legibles por máquinas
- **Copias de seguridad automáticas:** crea backups de los archivos de subtítulos antes de realizar cambios.

## Instalación

### macOS

Homebrew ([tap lebe-dev](https://github.com/lebe-dev/homebrew-tap)):

```bash
brew install lebe-dev/tap/submarine
```

### Linux

```bash
curl -L -o sm-0.15.0-linux-amd64.zip \
  https://github.com/lebe-dev/submarine/releases/download/0.15.0/sm-0.15.0-linux-amd64.zip
unzip sm-0.15.0-linux-amd64.zip
sudo install -m 0755 sm /usr/local/bin/sm
sm --help
```

### Docker

La imagen se publica como [`tinyops/submarine`](https://hub.docker.com/r/tinyops/submarine). Monte el directorio con sus subtítulos dentro del contenedor para que `sm` pueda leer y escribir los archivos:

```bash
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 --help

# Ejemplo: mostrar información de un archivo del directorio actual
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 \
  info Resident.Alien.S01E01.srt
```

Para que los archivos creados (por ejemplo, las copias de seguridad) pertenezcan a su usuario y no al usuario del contenedor, pase `-u`:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data \
  tinyops/submarine:0.15.0 set Resident.Alien.S01E01.srt 123 --text "Okay"
```

Un alias de shell práctico:

```bash
alias sm='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data tinyops/submarine:0.15.0'
sm get Resident.Alien.S01E01.srt 123
```

## Uso

```bash
# Mostrar información sobre el archivo de subtítulos
# sm info [FILE.srt]
$ sm info Resident.Alien.S01E01.srt

# Obtener subtítulo por índice o rango
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get Resident.Alien.S01E01.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# También admite sintaxis de rango
$ sm get Resident.Alien.S01E01.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Establecer subtítulo para un índice
# sm set [--dry-run] [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Vista previa sin modificar el archivo
$ sm set --dry-run Resident.Alien.S01E01.srt 123 --text "Okay"

# Añadir subtítulo al final del archivo
# Incrementa el índice automáticamente y crea un backup
# Crea el archivo srt si no existe
# sm add [--dry-run] [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Ajustar marcas de tiempo con un desplazamiento
# Admite valores positivos y negativos en milisegundos
# sm delay [--dry-run] [FILE.srt] [OFFSET]
$ sm delay Resident.Alien.S01E01.srt "+1000"  # Añadir 1 segundo
$ sm delay Resident.Alien.S01E01.srt "-500"   # Restar 0.5 segundos

# Importar subtítulos desde un archivo csv
# Crea el archivo srt si no existe
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv Resident.Alien.S01E01.srt import.csv
$ sm import --format=anchored Resident.Alien.S01E01.srt import.txt

# Verificar integridad del archivo
# sm doctor [--fix] [FILE.srt]
sm doctor --fix Resident.Alien.S01E01.eng.srt

# Renombrado masivo
# - file-mask no distingue mayúsculas de minúsculas
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

# Comparar subtítulos en modo interactivo
# sm compare [FILE1.srt] [FILE2.srt]
$ sm compare Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

# Verificar archivos de subtítulos
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt
$ sm verify --range=1-50 Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

Resultados
==================

Matched: 874/876 (99.8%)
Missing in Resident.Alien.S01E01.rus.srt: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in Resident.Alien.S01E01.rus.srt)
  [...] ...

# Progreso de traducción
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Exportar subtítulos en el formato especificado
# sm export [--format=anchored] [FILE.srt] [RANGE]
$ sm export --format=anchored movie.eng.srt 1-50

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.

# Salida JSON (disponible en todos los comandos excepto compare)
$ sm get Resident.Alien.S01E01.srt 1 --output json
{"ok":true,"data":{"index":1,"start_time":"00:00:01,436",...}}

$ sm info Resident.Alien.S01E01.srt --output json

# Descubrir los comandos disponibles y sus esquemas
$ sm describe
$ sm describe get
```

## Uso como biblioteca

Además de la interfaz de línea de comandos, Submarine puede usarse como biblioteca en proyectos Rust propios. Para más detalles sobre la integración, consulte la [documentación de la biblioteca](docs/LIBRARY.md).

## Hoja de ruta

- Feature: sincronización
- Feature: fusión
