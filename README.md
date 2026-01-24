# <img src="gopherutils.png" width="80" height="80" valign="middle" alt="gopherutils logo"> gopherutils

[![Build](https://github.com/ioluas/gopherutils/actions/workflows/ci.yaml/badge.svg)](https://github.com/ioluas/gopherutils/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/github/ioluas/gopherutils/graph/badge.svg?token=EZPA9HO9SB)](https://codecov.io/github/ioluas/gopherutils)

Personal project to implement Linux coreutils in Go.


## File Utilities

| Command    | Description                                    | Documentation                     | Status   |
|------------|------------------------------------------------|-----------------------------------|:---------|
| `ls`       | List directory contents                        | [README](utils/file/ls/README.md) | &#9083;  |
| `cp`       | Copy files and directories                     |                                   | &#10007; |
| `mv`       | Move (rename) files                            |                                   | &#10007; |
| `rm`       | Remove files or directories                    |                                   | &#10007; |
| `unlink`   | Remove a file                                  |                                   | &#10007; |
| `install`  | Copy files and set ownership/permissions       |                                   | &#10007; |
| `touch`    | Change file timestamps (or create empty files) |                                   | &#10007; |
| `stat`     | Display file or filesystem metadata            |                                   | &#10007; |
| `realpath` | Resolve symbolic links                         |                                   | &#10007; |
| `readlink` | Output symbolic link target                    |                                   | &#10007; |
| `basename` | Strip directory path                           |                                   | &#10007; |
| `dirname`  | Extract directory path                         |                                   | &#10007; |
| `pathchk`  | Check path validity                            |                                   | &#10007; |
| `chown`    | Change file owner                              |                                   | &#10007; |
| `chgrp`    | Change group ownership                         |                                   | &#10007; |
| `chmod`    | Change permissions                             |                                   | &#10007; |


## Directory Utilities

| Command     | Description                           | Documentation | Status   |
|-------------|---------------------------------------|---------------|:---------|
| `mkdir`     | Create directories                    |               | &#10007; |
| `rmdir`     | Remove empty directories              |               | &#10007; |
| `dir`       | List directory contents (alt to `ls`) |               | &#10007; |
| `dircolors` | Configure `ls` color output           |               | &#10007; |
| `vdir`      | Verbose directory listing             |               | &#10007; |


## Text Utilities

| Command     | Description                     | Documentation | Status   |
|-------------|---------------------------------|---------------|:---------|
| `cat`       | Concatenate and print files     |               | &#10007; |
| `tac`       | Print files in reverse          |               | &#10007; |
| `nl`        | Number lines                    |               | &#10007; |
| `wc`        | Count lines, words, bytes       |               | &#10007; |
| `sort`      | Sort lines                      |               | &#10007; |
| `uniq`      | Filter adjacent duplicate lines |               | &#10007; |
| `cut`       | Extract columns                 |               | &#10007; |
| `paste`     | Merge lines                     |               | &#10007; |
| `join`      | Join lines on a common field    |               | &#10007; |
| `fold`      | Line wrap                       |               | &#10007; |
| `fmt`       | Simple text formatter           |               | &#10007; |
| `pr`        | Pretty-print text               |               | &#10007; |
| `head`      | Output first lines              |               | &#10007; |
| `tail`      | Output last lines               |               | &#10007; |
| `split`     | Split files                     |               | &#10007; |
| `csplit`    | Context split                   |               | &#10007; |
| `expand`    | Tabs → spaces                   |               | &#10007; |
| `unexpand`  | Spaces → tabs                   |               | &#10007; |
| `tr`        | Translate characters            |               | &#10007; |
| `od`        | Octal/hex dump                  |               | &#10007; |
| `base32`    | Encode/decode base32            |               | &#10007; |
| `base64`    | Encode/decode base64            |               | &#10007; |
| `shred`     | Overwrite file to hide contents |               | &#10007; |
| `sum`       | Checksum (legacy)               |               | &#10007; |
| `cksum`     | CRC checksum                    |               | &#10007; |
| `md5sum`    | MD5 checksum                    |               | &#10007; |
| `sha1sum`   | SHA-1 checksum                  |               | &#10007; |
| `sha224sum` | SHA-224 checksum                |               | &#10007; |
| `sha256sum` | SHA-256 checksum                |               | &#10007; |
| `sha384sum` | SHA-384 checksum                |               | &#10007; |
| `sha512sum` | SHA-512 checksum                |               | &#10007; |


## Shell Utilities

| Command   | Description                 | Documentation | Status   |
|-----------|-----------------------------|---------------|:---------|
| `echo`    | Print arguments             |               | &#10007; |
| `printf`  | Formatted output            |               | &#10007; |
| `yes`     | Repeated output             |               | &#10007; |
| `true`    | Exit success                |               | &#10007; |
| `false`   | Exit failure                |               | &#10007; |
| `test`    | Evaluate expressions        |               | &#10007; |
| `[`       | Alias for `test`            |               | &#10007; |
| `timeout` | Run command with time limit |               | &#10007; |


## System Context Utilities

| Command   | Description              | Documentation | Status   |
|-----------|--------------------------|---------------|:---------|
| `pwd`     | Print working directory  |               | &#10007; |
| `whoami`  | Print effective username |               | &#10007; |
| `id`      | User identity            |               | &#10007; |
| `groups`  | Show group memberships   |               | &#10007; |
| `logname` | Login name               |               | &#10007; |
| `users`   | Logged-in users          |               | &#10007; |
| `who`     | Logged-in users info     |               | &#10007; |
| `tty`     | Print terminal name      |               | &#10007; |


## File / Device Utilities

| Command     | Description                | Documentation | Status   |
|-------------|----------------------------|---------------|:---------|
| `dd`        | Convert and copy data      |               | &#10007; |
| `sync`      | Flush write buffers        |               | &#10007; |
| `statx`     | **Linux-specific (newer)** |               | &#10007; |
| `du`        | Disk usage                 |               | &#10007; |
| `df`        | Disk free                  |               | &#10007; |
| `stat`      | File statistics            |               | &#10007; |
| `truncate`  | Resize file                |               | &#10007; |
| `fallocate` | Preallocate space          |               | &#10007; |


## Date and Time Utilities

| Command | Description              | Documentation | Status   |
|---------|--------------------------|---------------|:---------|
| `date`  | Display or set date/time |               | &#10007; |
| `sleep` | Delay execution          |               | &#10007; |


## Number Utilities

| Command  | Description          | Documentation | Status   |
|----------|----------------------|---------------|:---------|
| `expr`   | Evaluate expressions |               | &#10007; |
| `factor` | Factor integers      |               | &#10007; |
| `numfmt` | Format numbers       |               | &#10007; |
| `seq`    | Generate sequences   |               | &#10007; |
| `shuf`   | Shuffle lines        |               | &#10007; |
| `comm`   | Compare sorted files |               | &#10007; |


## Miscellaneous Utilities

| Command    | Description                      | Documentation | Status   |
|------------|----------------------------------|---------------|:---------|
| `arch`     | Print machine architecture       |               | &#10007; |
| `hostname` | **Not coreutils on all distros** |               | &#10007; |
| `nproc`    | Number of CPUs                   |               | &#10007; |
| `stdbuf`   | Control buffering                |               | &#10007; |
| `chcon`    | SELinux context                  |               | &#10007; |
| `runcon`   | Run with SELinux context         |               | &#10007; |
| `printenv` | Print environment                |               | &#10007; |
| `env`      | Run command with environment     |               | &#10007; |


# Meta / Information Utilities

| Command   | Description         | Documentation | Status   |
|-----------|---------------------|---------------|:---------|
| `version` | Output version info |               | &#10007; |
| `help`    | Built-in help       |               | &#10007; |
| `nice`    | Run with priority   |               | &#10007; |
| `nohup`   | **Not coreutils**   |               | &#10007; |


## Development

This project uses a `Makefile` to manage builds, testing, and quality control.

| Target           | Description                                                                       |
|------------------|-----------------------------------------------------------------------------------|
| `make all`       | Build all utilities (default)                                                     |
| `make deps`      | Download and install Go dependencies                                              |
| `make build`     | Alias for `all`                                                                   |
| `make clean`     | Remove built binaries and test cache                                              |
| `make test`      | Run tests for all utilities                                                       |
| `make coverage`  | Run tests and generate coverage report                                            |
| `make fmt`       | Format all Go code                                                                |
| `make lint`      | Lint all Go code                                                                  |
| `make vet`       | Vet all Go code                                                                   |
| `make staticcheck`| Run staticcheck                                                                  |
| `make CQ`        | Run Code Quality (lint, vet, staticcheck, fmt, coverage)                         |
| `make list`      | List all discovered utilities                                                     |
| `make install`   | Install binaries to user-local bin directory (configurable with `INSTALL_PREFIX`) |
| `make uninstall` | Remove binaries from installation directory                                       |
| `make help`      | Show help message with all targets                                                |


