diskii
======

**Note:** diskii is not stable yet! I don't expect to remove
functionality, but I'm still experimenting with the command syntax and
organization, so don't get too comfy with it.

diskii-the-tool is a commandline tool for working with Apple II disk
images. Given that
[AppleCommander](http://applecommander.sourceforge.net/) already does
everything, it's not terribly necessary. It is, however, mine. Minor
benefits (right now) are binaries you can copy around (no Java
needed), support for Super-Mon symbol tables on NakedOS disks, and
creation of
"[Standard Delivery](https://github.com/peterferrie/standard-delivery)"
disk images.

diskii-the-library is probably more useful: a library of
disk-image-manipulation code that can be used by other Go programs.

diskii's major disadvantage is that it mostly doesn't exist yet.

[![Build Status](https://travis-ci.org/zellyn/diskii.svg?branch=master)](https://travis-ci.org/zellyn/diskii)
[![Report Card](https://goreportcard.com/badge/github.com/zellyn/diskii)](https://goreportcard.com/report/github.com/zellyn/diskii)
[![GoDoc](https://godoc.org/github.com/zellyn/diskii/lib?status.svg)](https://godoc.org/github.com/zellyn/diskii/lib)

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

| Feature          | DOS 3.3            | ProDOS | NakedOS/Super-Mon  |
| ---------------- | ------------------ | ------ | ------------------ |
| basic structures | :white_check_mark: | :x:    | :white_check_mark: |
| ls               | :white_check_mark: | :x:    | :white_check_mark: |
| dump             | :white_check_mark: | :x:    | :white_check_mark: |
| put              | :x:                | :x:    | :white_check_mark: |
| dumptext         | :x:                | :x:    | :x:                |
| delete           | :x:                | :x:    | :x:                |
| rename           | :x:                | :x:    | :x:                |
| put              | :x:                | :x:    | :x:                |
| puttext          | :x:                | :x:    | :x:                |
| extract (all)    | :x:                | :x:    | :x:                |
| lock/unlock      | :x:                | :x:    | :x:                |
| init             | :x:                | :x:    | :x:                |
| defrag           | :x:                | :x:    | :x:                |

### Installing/updating
Assuming you have Go installed, run `go get -u github.com/zellyn/diskii`

You can also download automatically-built binaries from the
[latest release
page](https://github.com/zellyn/diskii/releases/latest). If you
need binaries for a different architecture, please send a pull
request or open an issue.

### Short-term TODOs/roadmap/easy ways to contribute

My rough TODO list (apart from anything marked (:x:) in the disk
operations matrix is listed below. Anything that an actual user needs
will be likely to get priority.

- [x] Build per-platform binaries for Linux, MacOS, Windows.
- [ ] Implement `GetFile` for DOS 3.3
- [ ] Add and implement the `-l` flag for `ls`
- [x] Add `Delete` to the `disk.Operator` interface
  - [x] Implement it for Super-Mon
  - [ ] Implement it for DOS 3.3
- [ ] Make 13-sector DOS disks work
- [ ] Read/write nybble formats
- [ ] Read/write gzipped files
- [ ] Add ProDOS support (add `lib/prodos/prodos.go` and register a ProDOS operator factory)

### Related tools

- http://a2ciderpress.com/ - the great grandaddy of them all. Windows only, unless you Wine
- http://applecommander.sourceforge.net/ - the commandline, cross-platform alternative to CiderPress
- http://brutaldeluxe.fr/products/crossdevtools/cadius/index.html - Brutal Deluxe's commandline tools
- https://github.com/cybernesto/dsktool.rb
- https://github.com/cmosher01/Apple-II-Disk-Tools
- https://github.com/madsen/perl-libA2
- https://github.com/markdavidlong/AppleSAWS
- https://github.com/dmolony/DiskBrowser
- https://github.com/deater/dos33fsprogs
- https://github.com/jtauber/a2disk
- https://github.com/datajerk/c2d
- https://github.com/thecompu/Driv3rs - A Python Script to work with Apple III SOS DSK files
- http://www.callapple.org/software/an-a-p-p-l-e-review-shink-fit-x-for-mac-os-x
