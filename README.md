diskii
======

**Note:** diskii is not stable yet! I don't expect to remove
functionality, but I'm still experimenting with the command syntax and
organization, so don't get too comfy with it yet.

diskii is a commandline tool for working with Apple II disk images.

It is also a library of code that can be used by other Go programs.

Its major advantage is that it's written in Go, hence
cross-platform.

Its major disadvantage is that it mostly doesn't exist yet.

[![Build Status](https://travis-ci.org/zellyn/diskii.svg?branch=master)](https://travis-ci.org/zellyn/diskii)

It rhymes with “whiskey”.

Discussion/support is in
[#apple2 on the retrocomputing Slack](https://retrocomputing.slack.com/messages/apple2/)
(invites [here](https://retrocomputing.herokuapp.com)).

### Goals

Eventually, it aims to be a comprehensive disk image manipulation
tool, but for now only the `applesoft decode` command works.

The library code aims (a) to support the commandline tool operations,
and (b) to replace the "read and write disk images" code of the
[goapple2 emulator](https://github.com/zellyn/goapple2).

Current disk operations supported:

| Feature       | DOS 3.3            | NakedOS/Super-Mon  |
| ------------- | ------------------ | ------------------ |
| ls            | :white_check_mark: | :white_check_mark: |
| dump          | :x:                | :white_check_mark: |

### Installing/updating
Assuming you have Go installed, run `go get -u github.com/zellyn/diskii`

You can also download automatically-built binaries from the
[latest release
page](https://github.com/zellyn/diskii/releases/latest). If you
need binaries for a different architecture, please send a pull
request or open an issue.

### Short-term TODOs/roadmap/easy ways to contribute

- [x] Build per-platform binaries for Linux, MacOS, Windows.
- [ ] Implement `GetFile` for DOS 3.3
- [ ] Add and implement the `-l` flag for `ls`
- [ ] Add `Delete` to the `disk.Operator` interface
  - [ ] Implement it for supermon
  - [ ] Implement it for DOS 3.3
- [ ] Add ProDOS support (add `lib/prodos/prodos.go` and register a ProDOS operator factory)

### Related tools

- http://a2ciderpress.com/
- http://applecommander.sourceforge.net/
- https://github.com/cybernesto/dsktool.rb
- https://github.com/cmosher01/Apple-II-Disk-Tools
- https://github.com/madsen/perl-libA2
- https://github.com/markdavidlong/AppleSAWS
- https://github.com/dmolony/DiskBrowser
- https://github.com/deater/dos33fsprogs
- https://github.com/jtauber/a2disk
