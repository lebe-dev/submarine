# Detectar y corregir un offset de sincronía constante

> Tus subtítulos van consistentemente adelantados o retrasados en una cantidad fija — mide el desplazamiento y luego corrige todo el archivo de una vez.

## Escenario

Los subtítulos coinciden con el diálogo pero cada línea aparece demasiado pronto o
demasiado tarde en la misma cantidad — un desajuste clásico entre tu archivo de subtítulos
y tu publicación de video. Tienes una referencia de confianza (otra pista que está
sincronizada, o el propio tiempo del video) y quieres ajustar tu archivo para que coincida.

## Comandos utilizados

- `sm detect-offset` ★ — medir el desplazamiento contra una referencia
- `sm delay` — aplicar el desplazamiento a todo el archivo
- `sm detect-offset` de nuevo — confirmar que el resultado es ~0

## Archivos de ejemplo

`reference.srt` — correctamente sincronizado:

```srt
1
00:00:10,000 --> 00:00:12,000
Good morning.

2
00:00:14,000 --> 00:00:16,000
How are you?

3
00:00:20,000 --> 00:00:22,000
See you tomorrow.
```

`subs.srt` — el mismo diálogo, con 2,5 segundos de retraso:

```srt
1
00:00:12,500 --> 00:00:14,500
Good morning.

2
00:00:16,500 --> 00:00:18,500
How are you?

3
00:00:22,500 --> 00:00:24,500
See you tomorrow.
```

## Paso a paso

### 1. Medir el offset

```bash
sm detect-offset subs.srt reference.srt
```

```
Offset Detection
================

File A: subs.srt
File B: reference.srt

Anchor matches:     3
Median offset (ms): -2500
Stddev (ms):        0
```

`Median offset (ms): -2500` se lee como "el archivo A va 2500 ms por detrás del archivo
B". `Stddev: 0` confirma un offset constante limpio (sin deriva), por lo que un único
desplazamiento lo corregirá. Para mover `subs.srt` de vuelta a la sincronía, aplica la
negación de ese retraso: **−2500 ms**.

### 2. Previsualizar el desplazamiento con `--dry-run`

```bash
sm delay --dry-run subs.srt -2500
```

```
Dry-run: time offset would be applied

Offset: -2500 ms
Subtitles affected: 3

Sample (first 3 subtitles):
  [1] 00:00:12,500 --> 00:00:14,500  =>  00:00:10,000 --> 00:00:12,000
  [2] 00:00:16,500 --> 00:00:18,500  =>  00:00:14,000 --> 00:00:16,000
  [3] 00:00:22,500 --> 00:00:24,500  =>  00:00:20,000 --> 00:00:22,000
```

La muestra muestra cada línea cayendo exactamente en los tiempos de referencia. No se
escribe nada.

> **El orden de los flags importa con un offset negativo.** Un `-` al inicio hace que el
> offset parezca un flag, así que coloca los flags de opciones (`--dry-run`, `--range`,
> `--from-timestamp`) **antes** del offset: `sm delay --dry-run subs.srt -2500`. Los
> offsets positivos (`+2500`) no se ven afectados.

### 3. Aplicar

```bash
sm delay subs.srt -2500
```

```
✓ Time offset applied successfully

Backup: backups/subs.srt.2026-06-14_10-43-26.541735
Offset: -2500 ms
Subtitles adjusted: 3
```

`sm delay` modifica el archivo en su lugar, escribiendo primero una copia con marca de
tiempo en `backups/`.

### 4. Confirmar

```bash
sm detect-offset subs.srt reference.srt
```

```
Median offset (ms): 0
```

Una mediana de `0` significa que los archivos están ahora sincronizados.

## Salida JSON

```bash
sm --output json delay subs.srt -2500
```

```json
{"ok":true,"data":{"offset_ms":-2500,"subtitles_adjusted":1,"backup_path":"backups/subs.srt.2026-06-14_10-43-26.549470"}}
```

- `offset_ms` — el desplazamiento aplicado.
- `subtitles_adjusted` — cuántos fragmentos se movieron.
- `backup_path` — dónde se escribió la copia previa al cambio (`N/A (dry-run)` con
  `--dry-run`).

Y el envoltorio de detección que impulsa todo este flujo:

```bash
sm --output json detect-offset subs.srt reference.srt
```

```json
{"ok":true,"data":{"anchor_matches":3,"median_offset_ms":-2500,"stddev_ms":0,"drift_detected":false,"same_video":false}}
```

`drift_detected:false` es la señal de que un único `delay` es la herramienta correcta.

## Problemas y casos extremos

- **El offset debe llevar signo:** `+100` o `-500`. Un `100` sin signo es rechazado.
- **Desplazar hacia atrás puede llegar a cero.** `sm delay` rechaza un offset que empuje
  alguna línea antes de `00:00:00,000`. Verifica con `--dry-run` si aparece un error de
  marca de tiempo negativa.
- **Un `Stddev` alto o "drift detected" significa que esta no es la corrección adecuada** —
  el retraso crece con el tiempo. Usa [rescale](fix-framerate-drift-with-rescale.es.md) en
  su lugar.
- **¿Solo una parte del archivo está desajustada?** Consulta
  [Resincronizar solo una parte del archivo](resync-a-partial-range.es.md).

## Véase también

- [Resincronizar solo una parte del archivo](resync-a-partial-range.es.md)
- [Corregir deriva de velocidad de fotogramas (23.976 ↔ 25)](fix-framerate-drift-with-rescale.es.md)
- [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md)
- `sm detect-offset --help`, `sm delay --help`, `sm describe`
