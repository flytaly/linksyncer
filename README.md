# Markdown Link Synchronizer

## linksyncer

This is a command-line utility that helps you organize your Markdown files. It tracks and updates the links in your .md notes.

<img src="https://github.com/user-attachments/assets/fc8bfa59-8022-491f-a3b6-59fe6e344365" width="400">

## Installation (with Go)

[Go](https://go.dev/dl/) should be installed in the system.

```bash
go install github.com/flytaly/linksyncer@latest
```

## Usage

### Caution

**Before using, back up your notes or use version control systems such as Git in case something goes wrong.**

### Running in manual mode

1. In the root directory of your notes, run `linksyncer`. It will search for all files and nested directories with notes.
2. Rename or move files and images in the nested directories.
3. Press `Enter` to check for changes and then `y` (or `Enter` again) to automatically modify paths in the `.md` files.

```bash
linksyncer
```

### Running in watch mode

You can run `linksyncer` in "watch mode" so it will automatically check for changes and update links.
However, **try not to use the watch mode in folders with a huge number of files**, such as your home directory.

```bash
linksyncer watch
```

## Supported link formats

-   `[note](./note1.md)`
-   `![img](/path/to/image)`
-   `<img src="path/to/image" >`

## Flags and Commands

```
  -l, --log string    path to the log file
  -p, --path string   path to the watched directory (default is the working directory)
      --size int      maximum file size in KB (default 1024)
  -v, --version       version for linksyncer
```

## Example

<img src="https://github.com/user-attachments/assets/3133d5b1-61b6-460d-b2c5-6c0f2d055ca0" width="500">
