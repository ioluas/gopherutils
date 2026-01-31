# cp

`cp` - copy files and directories

## Synopsis

`cp [OPTION]... [-T] SOURCE DEST`
`cp [OPTION]... SOURCE... DIRECTORY`
`cp [OPTION]... -t DIRECTORY SOURCE...`

## Description

Copy SOURCE to DEST, or multiple SOURCE(s) to DIRECTORY.

## Options

| Short Flag | Long Flag                  | Description                                                                       | Status          |
|:-----------|:---------------------------|:----------------------------------------------------------------------------------|:----------------|
| `-a`       | `--archive`                | same as `-dR --preserve=all`                                                      | not-implemented |
|            | `--attributes-only`        | don't copy the file data, just the attributes                                     | not-implemented |
|            | `--backup[=CONTROL]`       | make a backup of each existing destination file                                   | not-implemented |
| `-b`       |                            | like `--backup` but does not accept an argument                                   | not-implemented |
|            | `--copy-contents`          | copy contents of special files when recursive                                     | not-implemented |
| `-d`       |                            | same as `--no-dereference --preserve=links`                                       | not-implemented |
| `-f`       | `--force`                  | if an existing destination file cannot be opened, remove it and try again         | not-implemented |
| `-i`       | `--interactive`            | prompt before overwrite                                                           | not-implemented |
| `-H`       |                            | follow command-line symbolic links in SOURCE                                      | not-implemented |
| `-l`       | `--link`                   | hard link files instead of copying                                                | not-implemented |
| `-L`       | `--dereference`            | always follow symbolic links in SOURCE                                            | not-implemented |
| `-n`       | `--no-clobber`             | do not overwrite an existing file                                                 | not-implemented |
| `-P`       | `--no-dereference`         | never follow symbolic links in SOURCE                                             | not-implemented |
| `-p`       |                            | same as `--preserve=mode,ownership,timestamps`                                    | not-implemented |
|            | `--preserve[=ATTR_LIST]`   | preserve the specified attributes (default: mode,ownership,timestamps)            | not-implemented |
|            | `--no-preserve=ATTR_LIST`  | don't preserve the specified attributes                                           | not-implemented |
|            | `--parents`                | use full source file name under DIRECTORY                                         | not-implemented |
| `-R`, `-r` | `--recursive`              | copy directories recursively                                                      | not-implemented |
|            | `--reflink[=WHEN]`         | control clone/CoW copies                                                          | not-implemented |
|            | `--remove-destination`     | remove each existing destination file before attempting to open it                | not-implemented |
|            | `--sparse=WHEN`            | control creation of sparse files                                                  | not-implemented |
|            | `--strip-trailing-slashes` | remove any trailing slashes from each SOURCE argument                             | not-implemented |
| `-s`       | `--symbolic-link`          | make symbolic links instead of copying                                            | not-implemented |
| `-S`       | `--suffix=SUFFIX`          | override the usual backup suffix                                                  | not-implemented |
| `-t`       | `--target-directory=DIR`   | copy all SOURCE arguments into DIRECTORY                                          | not-implemented |
| `-T`       | `--no-target-directory`    | treat DEST as a normal file                                                       | not-implemented |
| `-u`       | `--update`                 | copy only when the SOURCE file is newer than the destination file or missing      | not-implemented |
| `-v`       | `--verbose`                 | explain what is being done                                                        | done            |
| `-v`       | `--debug`                  | explain how a file is copied.  Implies -v                                         | not-implemented |
| `-x`       | `--one-file-system`        | stay on this file system                                                          | not-implemented |
| `-Z`       |                            | set SELinux security context of destination file to default type                  | not-implemented |
|            | `--context[=CTX]`          | like `-Z`, or if CTX is specified then set the SELinux or SMACK security context  | not-implemented |
|            | `--help`                   | display this help and exit                                                        | not-implemented |
|            | `--version`                | output version information and exit                                               | not-implemented |

## Backup control

The backup suffix is `~`, unless set with `--suffix` or SIMPLE_BACKUP_SUFFIX.
The version control method may be selected via the `--backup` option or through
the VERSION_CONTROL environment variable.  Here are the values:

* `none`, `off`: never make backups (even if `--backup` is given)
* `numbered`, `t`: make numbered backups
* `existing`, `nil`: numbered if numbered backups exist, simple otherwise
* `simple`, `never`: always make simple backups
