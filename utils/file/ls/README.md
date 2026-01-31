# ls

`ls` - list directory contents

## Synopsis

`ls [OPTION]... [FILE]...`

## Description

List information about the FILEs (the current directory by default). Sort entries alphabetically by default.

## Options

| Short Flag | Long Flag            | Description                                                        |
|:-----------|:---------------------|:-------------------------------------------------------------------|
| `-a`       | `--all`              | do not ignore entries starting with `.`                            |
| `-A`       | `--almost-all`       | do not list implied `.` and `..`                                   |
| `-l`       | `--long`             | use a long listing format                                          |
| `-1`       | `--one-file-per-line`| list one file per line                                             |
| `-C`       |                      | list entries by columns (default when output is a terminal)        |
| `-t`       |                      | sort by time, newest first; see `--time`                           |
| `-h`       | `--human-readable`   | with `-l`, print sizes in human readable format (e.g., 1K 234M 2G) |
|            | `--si`               | with `-l`, print sizes in powers of 1000 not 1024                  |
|            | `--block-size`       | with `-l`, scale sizes by SIZE when printing them                  |
|            | `--time`             | select which timestamp is used to display or sort                  |
|            | `--time-style`       | time/date format with `-l`; see TIME_STYLE below                   |
|            | `--full-time`        | like `-l --time-style=full-iso`                                    |
|            | `--author`           | with `-l`, print the author of each file                           |
| `-G`       | `--no-group`         | in a long listing, don't print group names                         |
| `-b`       | `--escape`           | print C-style escapes for nongraphic characters                    |
| `-B`       | `--ignore-backups`   | do not list implied entries ending with `~`                        |
| `-d`       | `--directory`        | list directories themselves, not their contents                    |
| `-?`       | `--help`             | display help text and exit                                         |

## SIZE format for --block-size

SIZE can be a positive integer (bytes), `human-readable`, or `si`.
You can also use suffixes: binary (powers of 1024) like `K`, `M`, `G`,
`T`, `P`, `E` (or `KiB`, `MiB`, etc.), and decimal (powers of 1000) like
`kB`, `MB`, `GB`, `TB`, `PB`, `EB`. A leading apostrophe (`'`) requests
thousands separators based on `LC_NUMERIC`, and a bare suffix like `kB`
implies `1kB` and appends the suffix to the output.

## WORD values for --time

`--time=WORD` selects which timestamp is used. Supported values:
`atime`, `access`, `use` (access time); `ctime`, `status` (metadata change time);
`mtime`, `modification` (modified time, default); `birth`, `creation` (birth time).

## TIME_STYLE values for --time-style

`--time-style=TIME_STYLE` only applies with `-l`. Supported values:
`full-iso`, `long-iso`, `iso`, `locale`, or `+FORMAT`. `FORMAT` follows
`date(1)`-style tokens; `FORMAT1\nFORMAT2` uses `FORMAT1` for non-recent
files and `FORMAT2` for recent files. A `posix-` prefix only takes effect
outside the POSIX locale. Unknown or unsupported formats are ignored with
a warning.
