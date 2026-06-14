# Fusionar una traducción incompleta con un archivo donante

> Rellena los huecos en una traducción parcial con líneas de otro archivo de subtítulos, sin tocar las líneas en las que ya confías.

## Escenario

Tienes una traducción hecha a mano a la que le faltan algunas líneas — se omitió una
escena o el traductor se detuvo a medias. Otro archivo (un "donante") cubre esas líneas
faltantes, pero está desplazado una cantidad constante (una publicación diferente) y
contiene una línea extra que tu archivo no tiene. Quieres conservar tu propia redacción
en todos los lugares donde existe y solo tomar las líneas del donante para los huecos.

Este es el flujo de trabajo que antes se hacía con scripts de Python ad-hoc. Con `sm` es
un solo comando, precedido de dos diagnósticos rápidos.

## Comandos utilizados

- `sm detect-offset` — medir el desplazamiento de tiempo constante entre los dos archivos
- `sm diff` — ver exactamente qué líneas aportaría el donante
- `sm gaps` — localizar los huecos en el archivo base
- `sm merge` ★ — realizar la fusión, corrigiendo automáticamente el offset
- `sm info` / `sm verify` — confirmar el resultado

## Archivos de ejemplo

Crea estos dos archivos para seguir el ejemplo.

`base.srt` — tu traducción, a la que le falta la línea alrededor de 00:00:10:

```srt
1
00:00:01,000 --> 00:00:03,000
Good morning.

2
00:00:04,000 --> 00:00:06,000
How are you?

3
00:00:20,000 --> 00:00:22,000
See you tomorrow.
```

`donor.srt` — el mismo diálogo, desplazado +200 ms, con una línea extra en el hueco:

```srt
1
00:00:01,200 --> 00:00:03,200
Good morning.

2
00:00:04,200 --> 00:00:06,200
How are you?

3
00:00:10,200 --> 00:00:12,200
I missed you.

4
00:00:20,200 --> 00:00:22,200
See you tomorrow.
```

## Paso a paso

### 1. Medir el offset entre los archivos

```bash
sm detect-offset base.srt donor.srt
```

```
Offset Detection
================

File A: base.srt
File B: donor.srt

Anchor matches:     3
Median offset (ms): 200
Stddev (ms):        0
```

`Median offset (ms): 200` significa que el donante va un constante de 200 ms por detrás
del base. Un `Stddev (ms)` de `0` confirma que el desplazamiento es uniforme — un único
offset, no deriva progresiva. (Con solo tres líneas también puede aparecer una nota de
"files share little dialogue"; esa heurística está pensada para archivos de longitud
completa con cientos de fragmentos y puede ignorarse aquí.)

### 2. Ver qué añadiría el donante

```bash
sm diff base.srt donor.srt --by time
```

```
Diff between subtitle files
===========================

--- base.srt
+++ donor.srt

Common:    3
Only in A: 0
Only in B: 1

+ [3] 00:00:10,200 --> 00:00:12,200  I missed you.
```

Tres líneas se solapan en el tiempo (`Common: 3`) y exactamente una línea existe solo en
el donante (`Only in B: 1`) — el fragmento `I missed you.` que rellena tu hueco.
`diff --by time` compara por solapamiento de línea de tiempo, por lo que el desplazamiento
de +200 ms no crea entradas falsas en "only in B".

### 3. (Opcional) Localizar el hueco en el base

```bash
sm gaps base.srt --min-gap 5000
```

```
after index 2: 00:00:06,000 --> 00:00:20,000, duration 14.000s
```

Un silencio de 14 segundos después de la línea 2 es donde pertenece el diálogo faltante.

### 4. Previsualizar la fusión con `--dry-run`

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```
Dry-run: merge would be applied

Strategy: fill-gaps
Applied offset: 200 ms

Base subtitles:  3
Donor subtitles: 4

Would add:     1
Would skip (overlapping): 3
Would replace: 0
Total after merge: 4

Output (not written): merged.srt
```

`--auto-offset` reutiliza la detección del paso 1 (`Applied offset: 200 ms`). La estrategia
predeterminada `fill-gaps` añade solo la línea del donante que cae en un hueco
(`Would add: 1`) y omite las tres que se solapan con tus líneas existentes
(`Would skip (overlapping): 3`). Aún no se escribe nada.

### 5. Ejecutar de verdad

```bash
sm merge base.srt donor.srt --out merged.srt --auto-offset
```

```
✓ Merge completed successfully

Backup: N/A (new file)
Applied offset: 200 ms
Added: 1
Skipped (overlapping): 3
Replaced: 0
Total subtitles: 4
Output: merged.srt
```

`merged.srt` contiene ahora tus tres líneas originales más la línea del donante, desplazada
por el offset detectado para que encaje naturalmente en el hueco:

```srt
3
00:00:10,400 --> 00:00:12,400
I missed you.
```

Cuando `--out` apunta a un archivo existente, `sm` escribe una copia con marca de tiempo
en `backups/` antes de sobreescribir (aquí es un archivo nuevo, por lo que
`Backup: N/A`).

#### Elegir una estrategia

- `fill-gaps` (predeterminada) — conserva todas las líneas del base; añade líneas del
  donante solo donde el base está en silencio. Úsala para completar una traducción sin
  alterarla.
- `keep-base` — nunca reemplaza; idéntica a fill-gaps para adiciones pero rechaza
  cualquier solapamiento. La más segura cuando el base es autoritativo.
- `prefer-donor` — cuando una línea del donante se solapa con una del base, reemplaza el
  texto del base con el del donante. Úsala cuando el donante es la mejor traducción.

`--overlap-tolerance` (predeterminado `250` ms) controla cuánta holgura de tiempo aún
cuenta como "la misma línea", absorbiendo pequeñas diferencias de publicación que quedan
tras el offset.

### 6. Verificar

```bash
sm info merged.srt
```

```
Subtitle File Information
========================

File: merged.srt

Basic Information:
  Total subtitles: 4
  Total duration:  00:00:21,000 (21.000s)
```

## Salida JSON

```bash
sm --output json merge base.srt donor.srt --out merged.srt --auto-offset --dry-run
```

```json
{"ok":true,"data":{"base_count":3,"donor_count":4,"added":1,"skipped_overlapping":3,"replaced":0,"applied_offset_ms":200,"total_count":4,"output":"merged.srt","dry_run":true}}
```

- `added` — líneas del donante insertadas en huecos.
- `skipped_overlapping` — líneas del donante descartadas por solaparse con una línea del base.
- `replaced` — líneas del base sobreescritas (distinto de cero solo con `--prefer-donor`).
- `applied_offset_ms` — el desplazamiento aplicado al donante (de `--auto-offset` o `--offset`).
- `total_count` — número de subtítulos tras la fusión.

## Problemas y casos extremos

- **`--out`, no `--output`.** `--out` es el archivo de resultado; `--output text|json` es
  el flag de formato global. Son distintos.
- **Las líneas del donante fuera del rango temporal del base también se añaden**, no solo
  las que están en huecos interiores. Inspecciona primero con `sm diff --by time`.
- **La comparación es por marca de tiempo, no por texto**, así que líneas cortas y
  repetidas (p. ej. "Thanks") en el donante no se confundirán con líneas en otro lugar
  del base.
- **Pasa un offset manual con `--offset`** si `--auto-offset` falla (p. ej. con muy poco
  diálogo compartido): `sm merge base.srt donor.srt --out merged.srt --offset=-212`. Usa
  la forma con `=` para valores negativos.
- **¿Deriva en lugar de offset?** Si `sm detect-offset` reporta un `Stddev` grande o
  "drift detected", un único offset no alineará los archivos — consulta
  [Corregir deriva de velocidad de fotogramas](fix-framerate-drift-with-rescale.es.md).

## Véase también

- [Detectar y corregir un offset de sincronía constante](detect-and-fix-constant-offset.es.md)
- [Comparar dos pistas de subtítulos](compare-two-subtitle-tracks.es.md)
- [Encontrar diálogos faltantes con análisis de huecos](find-missing-dialogue-gaps.es.md)
- [Eliminar duplicados y solapamientos tras una fusión](deduplicate-and-clean-after-merge.es.md)
- `sm merge --help`, `sm describe`
