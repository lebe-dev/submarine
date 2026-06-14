# Unir partes divididas (CD1/CD2) en un solo archivo

> Una publicación dividida en dos archivos que se reproducen uno tras otro — únelos en uno, con la segunda parte desplazada para continuar después de la primera.

## Escenario

Una publicación antigua viene como `CD1` y `CD2`, con sus códigos de tiempo empezando
desde cero. Al reproducirlos como un solo video, los fragmentos del segundo archivo
necesitan comenzar donde termina el primero. `sm concat` añade las partes y desplaza cada
parte posterior por el total acumulado, luego renumera el resultado.

Esto es diferente de [merge](merge-incomplete-translation-with-donor.es.md): las partes
**no** se solapan en el tiempo — se ejecutan de forma consecutiva.

## Comandos utilizados

- `sm concat` ★ — añadir partes y desplazar cada una para que siga a la anterior
- `sm info` — confirmar la duración y el número total de fragmentos del resultado unido

## Archivos de ejemplo

`part1.srt`:

```srt
1
00:00:01,000 --> 00:00:03,000
CD1 first line.

2
00:00:04,000 --> 00:00:06,000
CD1 last line.
```

`part2.srt` — también comenzando cerca de cero:

```srt
1
00:00:01,000 --> 00:00:03,000
CD2 first line.

2
00:00:05,000 --> 00:00:07,000
CD2 last line.
```

## Paso a paso

### 1. Previsualizar con `--dry-run`

`--gap` establece el silencio (en ms) insertado entre partes:

```bash
sm concat part1.srt part2.srt --out full.srt --gap 2000 --dry-run
```

```
Dry-run: subtitles would be concatenated

Parts: part1.srt, part2.srt
Gap: 2000 ms
Total subtitles: 4
Output: full.srt
```

### 2. Aplicar

```bash
sm concat part1.srt part2.srt --out full.srt --gap 2000
```

```
✓ Subtitles concatenated successfully

Parts: part1.srt, part2.srt
Gap: 2000 ms
Total subtitles: 4
Output: full.srt
Backup: N/A (new file)
```

`full.srt` conserva la parte 1 sin cambios, y luego la parte 2 desplazada para comenzar
después del fin de la parte 1 más el hueco, con los índices renumerados de forma continua:

```srt
1
00:00:01,000 --> 00:00:03,000
CD1 first line.

2
00:00:04,000 --> 00:00:06,000
CD1 last line.

3
00:00:09,000 --> 00:00:11,000
CD2 first line.

4
00:00:13,000 --> 00:00:15,000
CD2 last line.
```

### 3. Confirmar

```bash
sm info full.srt
```

Verifica que **Total subtitles** y **Total duration** coincidan con las dos partes más
el hueco.

## Salida JSON

```bash
sm --output json concat part1.srt part2.srt --out full.srt --gap 2000 --dry-run
```

```json
{"ok":true,"data":{"parts":2,"gap_ms":2000,"total_count":4,"output":"full.srt","dry_run":true}}
```

- `parts` — número de archivos de entrada unidos.
- `gap_ms` — silencio insertado entre partes.
- `total_count` — fragmentos en el resultado.

## Problemas y casos extremos

- **Cada parte conserva su propio offset inicial.** La parte 2 comenzaba en
  `00:00:01,000`, por lo que tras la parte 1 (termina en `00:00:06,000`) más el hueco de
  2 s cae en `00:00:09,000` — el silencio inicial propio de 1 s de la parte se preserva
  por encima del hueco. Recorta el silencio inicial de una parte de antemano si quieres
  que el hueco sea exacto.
- **El orden importa.** Las partes se unen de izquierda a derecha en el orden en que las
  listas.
- **`--out`, no `--output`.** `--out` es el archivo de resultado; `--output text|json` es
  el flag de formato global.
- **¿Las partes se solapan en lugar de ser consecutivas?** Si comparten diálogo en el
  tiempo, necesitas [merge](merge-incomplete-translation-with-donor.es.md) en su lugar.

## Véase también

- [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md)
- [Normalizar la estructura: ordenar, renumerar, corregir solapamientos](normalize-structure.es.md)
- `sm concat --help`, `sm info --help`, `sm describe`
