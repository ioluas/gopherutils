# ls

`ls` - list directory contents

## Synopsis

`ls [OPTION]... [FILE]...`

## Description

List information about the FILEs (the current directory by default). Sort entries alphabetically by default.

## Options

| Short Flag | Long Flag          | Description                                                        |
|:-----------|:-------------------|:-------------------------------------------------------------------|
| `-a`       | `--all`            | do not ignore entries starting with `.`                            |
| `-A`       | `--almost-all`     | do not list implied `.` and `..`                                   |
| `-l`       | `--long`           | use a long listing format                                          |
| `-h`       | `--human-readable` | with `-l`, print sizes in human readable format (e.g., 1K 234M 2G) |
|            | `--si`             | with `-l`, print sizes in powers of 1000 not 1024                  |
|            | `--block-size`     | with `-l`, scale sizes by SIZE when printing them                  |
|            | `--author`         | with `-l`, print the author of each file                           |
| `-G`       | `--no-group`       | in a long listing, don't print group names                         |
| `-b`       | `--escape`         | print C-style escapes for nongraphic characters                    |
| `-B`       | `--ignore-backups` | do not list implied entries ending with `~`                        |
| `-d`       | `--directory`      | list directories themselves, not their contents                    |
| `-?`       | `--help`           | display help text and exit                                         |

## SIZE format for --block-size

SIZE can be a positive integer (bytes), `human-readable`, or `si`.
You can also use suffixes: binary (powers of 1024) like `K`, `M`, `G`,
`T`, `P`, `E` (or `KiB`, `MiB`, etc.), and decimal (powers of 1000) like
`kB`, `MB`, `GB`, `TB`, `PB`, `EB`. A leading apostrophe (`'`) requests
thousands separators based on `LC_NUMERIC`, and a bare suffix like `kB`
implies `1kB` and appends the suffix to the output.
