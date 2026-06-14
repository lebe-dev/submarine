# Comparar dos pistas de subtítulos

> Observa la diferencia real de contenido entre dos archivos de subtítulos — qué líneas son únicas en cada uno y cuáles solo difieren en la redacción.

## Escenario

Tienes dos versiones de una pista — dos traducciones, una editada y la original, o una
publicación y una resincronización — y quieres saber en qué difieren realmente: qué
fragmentos son comunes, cuáles son únicos de un lado y dónde el mismo momento tiene texto
diferente.

## Comandos utilizados

- `sm diff` ★ — comparar por solapamiento de línea de tiempo o por texto
- `sm verify` — para la verificación relacionada pero distinta "¿son estas estructuralmente paralelas?"

## Archivos de ejemplo

`a.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
We need to leave now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

`b.srt` — mismos tiempos, una línea reescrita:

```srt
1
00:00:01,000 --> 00:00:03,000
We have to go now.

2
00:00:04,000 --> 00:00:06,000
Where are the keys?
```

## Paso a paso

### Comparar por texto

```bash
sm diff a.srt b.srt --by text
```

```
Diff between subtitle files
===========================

--- a.srt
+++ b.srt

Common:    1
Only in A: 1
Only in B: 1

- [1] 00:00:01,000 --> 00:00:03,000  We need to leave now.
+ [1] 00:00:01,000 --> 00:00:03,000  We have to go now.
```

`--by text` compara usando el texto normalizado. La línea "Where are the keys?" idéntica
aparece como `Common`; la primera línea reescrita aparece una vez en cada lado (`Only in A`
/ `Only in B`), representada como un par `-`/`+` en estilo diff unificado.

### Comparar por tiempo

```bash
sm diff a.srt b.srt --by time
```

```
Diff between subtitle files
===========================

--- a.srt
+++ b.srt

Common:    2
Only in A: 0
Only in B: 0
```

`--by time` (modo predeterminado) compara por solapamiento de línea de tiempo dentro de
`--tolerance` (predeterminado `250` ms). Ambos fragmentos ocupan las mismas ventanas, por
lo que todo es `Common` — aunque la redacción de la línea 1 cambió. Usa este modo para
encontrar diferencias **estructurales** (líneas añadidas o eliminadas, desplazamientos);
usa `--by text` para encontrar diferencias de **redacción**.

## Salida JSON

```bash
sm --output json diff a.srt b.srt --by text
```

```json
{"ok":true,"data":{"a_file":"a.srt","b_file":"b.srt","by":"text","tolerance_ms":250,"only_in_a":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We need to leave now.","has_html":false}],"only_in_b":[{"index":1,"start_time":"00:00:01,000","start_time_ms":1000,"end_time":"00:00:03,000","end_time_ms":3000,"duration_ms":2000,"text":"We have to go now.","has_html":false}],"common_count":1}}
```

- `by` — el modo utilizado (`time` o `text`).
- `only_in_a` / `only_in_b` — registros completos de fragmentos presentes en solo un lado.
- `common_count` — fragmentos que coinciden en ambos lados.

## `diff` vs `verify`

`sm diff` responde "¿en qué difiere el contenido?" — empareja fragmentos por solapamiento
o por texto y reporta los que no coinciden. `sm verify` responde una pregunta más estricta:
"¿son estos dos archivos paralelos por índice y marca de tiempo?", reportando un porcentaje
de coincidencia y si pasa o falla. Para los archivos de ejemplo de arriba, `verify` reporta
**SUCCESS** (ambos fragmentos coinciden por índice y tiempo) aunque su redacción difiera —
que es exactamente por qué se usa `diff` cuando importa el contenido:

```bash
sm verify a.srt b.srt
```

```
Matched: 2/2 (100.0%)

Verification: SUCCESS
```

## Problemas y casos extremos

- **Elige el modo según la pregunta.** `--by time` ignora la redacción; `--by text` ignora
  el tiempo. Una línea reescrita pero resincronizada es "común" bajo `time` y "única" bajo
  `text`.
- **`--tolerance` solo afecta a `--by time`.** Amplíalo para publicaciones con pequeñas
  derivas.
- **La comparación de texto está normalizada** (espacios/mayúsculas), por lo que las
  diferencias cosméticas no separan líneas por lo demás idénticas.

## Véase también

- [Encontrar diálogos faltantes con análisis de huecos](find-missing-dialogue-gaps.es.md)
- [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md)
- `sm diff --help`, `sm verify --help`, `sm describe`
