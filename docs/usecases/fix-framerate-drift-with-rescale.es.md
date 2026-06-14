# Corregir deriva de velocidad de fotogramas (23.976 ↔ 25)

> La sincronía está bien al principio pero se desvía progresivamente hacia el final — un desajuste de velocidad de fotogramas, no un offset constante. Estira la línea de tiempo en lugar de desplazarla.

## Escenario

Los subtítulos creados para una publicación a 25 fps (PAL) se reproducen contra una
codificación a 23.976 fps (NTSC/cine), o viceversa. Las primeras líneas parecen casi
correctas, pero el error crece linealmente hasta que las últimas líneas tienen segundos de
diferencia. Ningún `delay` puede corregir esto — la línea de tiempo necesita ser
*escalada*.

## Comandos utilizados

- `sm detect-offset` — confirmar que es deriva, no un offset constante
- `sm rescale` ★ — estirar/comprimir la línea de tiempo por factor, par de fps o dos puntos de anclaje

## Archivos de ejemplo

`movie.srt` — tres fragmentos distribuidos en 20 minutos:

```srt
1
00:00:10,000 --> 00:00:12,000
Opening line.

2
00:10:00,000 --> 00:10:02,000
Midpoint line.

3
00:20:00,000 --> 00:20:02,000
Closing line.
```

## Paso a paso

### 1. Confirmar que es deriva, no offset

Compara el archivo desincronizado con una referencia correctamente sincronizada:

```bash
sm detect-offset stretched.srt movie.srt
```

```
Median offset (ms): -25620
Stddev (ms):        20745

Note: drift detected (offset varies across the timeline); consider 'sm rescale'.
```

La clave está en un **`Stddev` grande** y la nota explícita de **"drift detected"**: el
offset medido es pequeño al principio y enorme al final. Esa varianza es la firma de un
desajuste de velocidad de fotogramas — usa `rescale`, no `delay`.

### 2. Reescalar por par de fps

Si conoces las velocidades de fotogramas de origen y destino, nómbralas directamente:

```bash
sm rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt --dry-run
```

```
Dry-run: rescale would be applied

Mode: fps
Factor: 1.0427093760427095
Offset: 0 ms
Subtitles affected: 3
Output: ntsc.srt

Sample (first 3 subtitles):
  [1] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,427 --> 00:00:12,513
  [2] 00:10:00,000 --> 00:10:02,000  =>  00:10:25,626 --> 00:10:27,711
  [3] 00:20:00,000 --> 00:20:02,000  =>  00:20:51,251 --> 00:20:53,337
```

La corrección crece con el tiempo — 0,4 s al principio, ~51 s en el minuto 20 — deshaciendo
exactamente la deriva lineal. `sm` calcula el factor (25 / 23.976) por ti.

### 3. O por factor explícito

Cuando ya conoces el multiplicador:

```bash
sm rescale movie.srt --factor 1.0427 --out scaled.srt --dry-run
```

```
Mode: factor
Factor: 1.0427
```

### 4. O por dos puntos de anclaje

Cuando no conoces los fps pero puedes identificar dos fragmentos y los códigos de tiempo
que *deberían* tener, proporciona dos puntos `--anchor IDX=HH:MM:SS,mmm`. `sm` resuelve
la transformación lineal que mapea cada fragmento a su objetivo (calcula tanto un factor
de escala como un pequeño offset):

```bash
sm rescale movie.srt --anchor "1=00:00:09,592" --anchor "3=00:19:10,096" --out anchored.srt --dry-run
```

```
Dry-run: rescale would be applied

Mode: anchor
Factor: 0.9584067226890757
Offset: 8 ms
Subtitles affected: 3
Output: anchored.srt

Sample (first 3 subtitles):
  [1] 00:00:10,000 --> 00:00:12,000  =>  00:00:09,592 --> 00:00:11,509
  [2] 00:10:00,000 --> 00:10:02,000  =>  00:09:35,052 --> 00:09:36,969
  [3] 00:20:00,000 --> 00:20:02,000  =>  00:19:10,096 --> 00:19:12,013
```

Los fragmentos 1 y 3 caen exactamente en sus códigos de tiempo solicitados; todo lo que
está entre ellos se interpola.

### 5. Aplicar

```bash
sm rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt
```

```
✓ Rescale applied successfully

Mode: fps
Factor: 1.0427093760427095
Offset: 0 ms
Subtitles rescaled: 3
Output: ntsc.srt
Backup: N/A (new file)
```

## Salida JSON

```bash
sm --output json rescale movie.srt --from-fps 25 --to-fps 23.976 --out ntsc.srt --dry-run
```

```json
{"ok":true,"data":{"file":"movie.srt","mode":"fps","factor":1.0427093760427095,"offset_ms":0,"total_count":3,"output":"ntsc.srt","backup_path":"N/A (dry-run)","dry_run":true}}
```

- `mode` — `factor`, `fps` o `anchor`.
- `factor` / `offset_ms` — la transformación lineal resuelta `t' = factor·t + offset`.
- `output` — el archivo de resultado.

## Problemas y casos extremos

- **Proporciona exactamente un modo.** `--factor`, el par `--from-fps`/`--to-fps`, o dos
  valores `--anchor` — combinarlos u omitirlos produce un error *"specify one of …"*.
- **`--out`, no `--output`.** `--out` es el archivo de resultado; `--output text|json` es
  el flag de formato global.
- **Las marcas de tiempo del anclaje contienen una coma** (`HH:MM:SS,mmm`). Pon entre
  comillas cada argumento `--anchor` para que la shell lo pase intacto.
- **Si `Stddev` está cerca de cero**, no es deriva — usa
  [delay](detect-and-fix-constant-offset.es.md).

## Véase también

- [Detectar y corregir un offset de sincronía constante](detect-and-fix-constant-offset.es.md)
- [Resincronizar solo una parte del archivo](resync-a-partial-range.es.md)
- `sm rescale --help`, `sm detect-offset --help`, `sm describe`
