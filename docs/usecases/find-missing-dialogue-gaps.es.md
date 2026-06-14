# Encontrar diálogos faltantes con análisis de huecos

> Detecta los silencios en un archivo de subtítulos donde puede haberse omitido diálogo.

## Escenario

Una traducción parece incompleta — una escena no tiene subtítulos. En lugar de revisar el
video manualmente, puedes listar los huecos de tiempo entre fragmentos consecutivos e
inspeccionar los sospechosos, y luego comparar con un archivo donante para comprobar si
realmente falta diálogo ahí.

## Comandos utilizados

- `sm gaps` ★ — listar silencios más largos que un umbral
- `sm info` — obtener el hueco promedio del archivo para elegir un umbral
- `sm diff` — verificar si un donante tiene diálogo en el hueco

## Paso a paso

Este ejemplo usa un archivo de ejemplo incluido en el proyecto,
`test-data/valid/complex.srt`.

### 1. Listar los huecos

```bash
sm gaps test-data/valid/complex.srt --min-gap 1500
```

```
after index 2: 00:00:05,135 --> 00:00:07,790, duration 2.655s
```

Cada línea reporta un silencio: `after index 2` es el fragmento que le precede, luego la
ventana (`fin del fragmento 2 → inicio del siguiente`) y su duración. Un hueco de 2,655
segundos después de la línea 2 es el lugar a investigar.

### 2. Elegir `--min-gap`

`--min-gap` es el piso (en milisegundos, predeterminado `1000`) por debajo del cual se
ignoran los huecos. Ponlo justo por encima de la pausa típica del archivo para que solo
aparezcan los silencios inusuales. `sm info` te da esa línea base:

```bash
sm info test-data/valid/complex.srt
```

Mira **Average gap between subtitles**. Si el promedio de pausas de un archivo es ~9 s,
una escena omitida aparece como un hueco mucho mayor — empieza con `--min-gap 15000` para
destacar solo los agujeros reales, luego bájalo si sospechas omisiones más pequeñas.

### 3. Confirmar con un donante

Un hueco por sí solo no demuestra que falte diálogo — puede que simplemente no haya
habla. Compara con una pista más completa para estar seguro:

```bash
sm diff test-data/valid/complex.srt donor.srt --by time
```

Las líneas reportadas como **Only in B** que caigan dentro del hueco son diálogos que
le faltan a tu archivo — exactamente la entrada para
[una fusión con donante](merge-incomplete-translation-with-donor.es.md).

## Salida JSON

```bash
sm --output json gaps test-data/valid/complex.srt --min-gap 1500
```

```json
{"ok":true,"data":{"file":"test-data/valid/complex.srt","min_gap_ms":1500,"count":1,"gaps":[{"after_index":2,"start":"00:00:05,135","end":"00:00:07,790","duration_ms":2655}]}}
```

- `count` — número de huecos iguales o superiores a `min_gap_ms`.
- `gaps[].after_index` — el índice del fragmento que precede al hueco.
- `gaps[].start` / `end` — la ventana de silencio.
- `gaps[].duration_ms` — su duración en milisegundos.

## Problemas y casos extremos

- **Un hueco no es prueba de una línea faltante.** Las pausas, cambios de escena y música
  son normales — verifica con `sm diff` antes de concluir que se omitió diálogo.
- **El umbral está en milisegundos.** `--min-gap 1500` son 1,5 s, no 1500 s.
- **Los fragmentos solapados no generan hueco** (la ventana sería negativa), por lo que
  nunca aparecen en el reporte; corrígelos con [normalize](normalize-structure.es.md).

## Véase también

- [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md)
- [Comparar dos pistas de subtítulos](compare-two-subtitle-tracks.es.md)
- `sm gaps --help`, `sm info --help`, `sm describe`
