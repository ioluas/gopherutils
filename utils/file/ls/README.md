# ls

`ls` - list directory contents

## Synopsis

`ls [OPTION]... [FILE]...`

## Description

List information about the FILEs (the current directory by default). Sort entries alphabetically by default.

## Options

| Short Flag | Long Flag                                   | Description                                                                                                                                                                                                   | Status          |
|:-----------|:--------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:----------------|
| `-a`       | `--all`                                     | do not ignore entries starting with `.`                                                                                                                                                                       | done            |
| `-A`       | `--almost-all`                              | do not list implied `.` and `..`                                                                                                                                                                              | done            |
|            | `--author`                                  | with `-l`, print the author of each file                                                                                                                                                                      | done            |
| `-b`       | `--escape`                                  | print C-style escapes for nongraphic characters                                                                                                                                                               | done            |
|            | `--block-size=SIZE`                         | with `-l`, scale sizes by SIZE when printing them; e.g., `--block-size=M`                                                                                                                                     | done            |
| `-B`       | `--ignore-backups`                          | do not list implied entries ending with `~`                                                                                                                                                                   | done            |
| `-c`       |                                             | with `-lt`: sort by, and show, ctime (time of last change of file status information); with `-l`: show ctime and sort by name; otherwise: sort by ctime, newest first                                         | not-implemented |
| `-C`       |                                             | list entries by columns                                                                                                                                                                                       | done            |
|            | `--color[=WHEN]`                            | color the output WHEN; more info below                                                                                                                                                                        | not-implemented |
| `-d`       | `--directory`                               | list directories themselves, not their contents                                                                                                                                                               | done            |
| `-D`       | `--dired`                                   | generate output designed for Emacs' dired mode                                                                                                                                                                | done            |
| `-f`       |                                             | same as `-a -U`                                                                                                                                                                                               | not-implemented |
| `-F`       | `--classify[=WHEN]`                         | append indicator (one of `*/=>@\|`) to entries WHEN                                                                                                                                                           | not-implemented |
|            | `--file-type`                               | likewise, except do not append `*`                                                                                                                                                                            | not-implemented |
|            | `--format=WORD`                             | across, horizontal (`-x`), commas (`-m`), long (`-l`), single-column (`-1`), verbose (`-l`), vertical (`-C`)                                                                                                  | not-implemented |
|            | `--full-time`                               | like `-l --time-style=full-iso`                                                                                                                                                                               | done            |
| `-g`       |                                             | like `-l`, but do not list owner                                                                                                                                                                              | not-implemented |
|            | `--group-directories-first`                 | group directories before files                                                                                                                                                                                | not-implemented |
| `-G`       | `--no-group`                                | in a long listing, don't print group names                                                                                                                                                                    | done            |
| `-h`       | `--human-readable`                          | with `-l` and `-s`, print sizes like 1K 234M 2G etc.                                                                                                                                                          | done            |
|            | `--si`                                      | likewise, but use powers of 1000 not 1024                                                                                                                                                                     | done            |
| `-H`       | `--dereference-command-line`                | follow symbolic links listed on the command line                                                                                                                                                              | not-implemented |
|            | `--dereference-command-line-symlink-to-dir` | follow each command line symbolic link that points to a directory                                                                                                                                             | not-implemented |
|            | `--hide=PATTERN`                            | do not list implied entries matching shell PATTERN (overridden by `-a` or `-A`)                                                                                                                               | not-implemented |
|            | `--hyperlink[=WHEN]`                        | hyperlink file names WHEN                                                                                                                                                                                     | not-implemented |
|            | `--indicator-style=WORD`                    | append indicator with style WORD to entry names: none (default), slash (`-p`), file-type (`--file-type`), classify (`-F`)                                                                                     | not-implemented |
| `-i`       | `--inode`                                   | print the index number of each file                                                                                                                                                                           | not-implemented |
| `-I`       | `--ignore=PATTERN`                          | do not list implied entries matching shell PATTERN                                                                                                                                                            | not-implemented |
| `-k`       | `--kibibytes`                               | default to 1024-byte blocks for file system usage; used only with `-s` and per directory totals                                                                                                               | not-implemented |
| `-l`       | `--long`                                    | use a long listing format                                                                                                                                                                                     | done            |
| `-L`       | `--dereference`                             | when showing file information for a symbolic link, show information for the file the link references rather than for the link itself                                                                          | not-implemented |
| `-m`       |                                             | fill width with a comma separated list of entries                                                                                                                                                             | not-implemented |
| `-n`       | `--numeric-uid-gid`                         | like `-l`, but list numeric user and group IDs                                                                                                                                                                | not-implemented |
| `-N`       | `--literal`                                 | print entry names without quoting                                                                                                                                                                             | not-implemented |
| `-o`       |                                             | like `-l`, but do not list group information                                                                                                                                                                  | not-implemented |
| `-p`       | `--indicator-style=slash`                   | append / indicator to directories                                                                                                                                                                             | not-implemented |
| `-q`       | `--hide-control-chars`                      | print `?` instead of nongraphic characters                                                                                                                                                                    | not-implemented |
|            | `--show-control-chars`                      | show nongraphic characters as-is (the default, unless program is `ls` and output is a terminal)                                                                                                               | not-implemented |
| `-Q`       | `--quote-name`                              | enclose entry names in double quotes                                                                                                                                                                          | not-implemented |
|            | `--quoting-style=WORD`                      | use quoting style WORD for entry names: literal, locale, shell, shell-always, shell-escape, shell-escape-always, c, escape                                                                                    | done            |
| `-r`       | `--reverse`                                 | reverse order while sorting                                                                                                                                                                                   | not-implemented |
| `-R`       | `--recursive`                               | list subdirectories recursively                                                                                                                                                                               | not-implemented |
| `-s`       | `--size`                                    | print the allocated size of each file, in blocks                                                                                                                                                              | not-implemented |
| `-S`       |                                             | sort by file size, largest first                                                                                                                                                                              | not-implemented |
|            | `--sort=WORD`                               | change default `name` sort to WORD: none (`-U`), size (`-S`), time (`-t`), version (`-v`), extension (`-X`), name, width                                                                                      | not-implemented |
|            | `--time=WORD`                               | select which timestamp used to display or sort; access time (`-u`): atime, access, use; metadata change time (`-c`): ctime, status; modified time (default): mtime, modification; birth time: birth, creation | done            |
|            | `--time-style=TIME_STYLE`                   | time/date format with `-l`; see TIME_STYLE below                                                                                                                                                              | done            |
| `-t`       |                                             | sort by time, newest first; see `--time`                                                                                                                                                                      | done            |
| `-T`       | `--tabsize=COLS`                            | assume tab stops at each COLS instead of 8                                                                                                                                                                    | not-implemented |
| `-u`       |                                             | with `-lt`: sort by, and show, access time; with `-l`: show access time and sort by name; otherwise: sort by access time, newest first                                                                        | not-implemented |
| `-U`       |                                             | do not sort directory entries                                                                                                                                                                                 | not-implemented |
| `-v`       |                                             | natural sort of (version) numbers within text                                                                                                                                                                 | not-implemented |
| `-w`       | `--width=COLS`                              | set output width to COLS. 0 means no limit                                                                                                                                                                    | not-implemented |
| `-x`       |                                             | list entries by lines instead of by columns                                                                                                                                                                   | not-implemented |
| `-X`       |                                             | sort alphabetically by entry extension                                                                                                                                                                        | not-implemented |
| `-Z`       | `--context`                                 | print any security context of each file                                                                                                                                                                       | not-implemented |
|            | `--zero`                                    | end each output line with NUL, not newline                                                                                                                                                                    | not-implemented |
| `-1`       | `--one-file-per-line`                       | list one file per line                                                                                                                                                                                        | done            |
| `-?`       | `--help`                                    | display help and exit                                                                                                                                                                                         | done            |
|            | `--version`                                 | output version information and exit                                                                                                                                                                           | not-implemented |

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

## WORD values for --quoting-style

`--quoting-style=WORD` uses quoting style `WORD` for entry names.
Supported values: `literal` (default), `locale`, `shell`, `shell-always`, `shell-escape`, `shell-escape-always`, `c`, `escape`.
The `QUOTING_STYLE` environment variable can also be used to set the default style.
