diskii
======

**Note:** diskii is not stable yet! I don't expect to remove
functionality, but I'm still experimenting with the command syntax and
organization, so don't get too comfy with it.

![Seagull Srs Micro Software](img/seagull-srs.png)

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

# Goals

Eventually, it aims to be a comprehensive disk image manipulation
tool, but for now only some parts work.

The library code aims (a) to support the commandline tool operations,
and (b) to replace the "read and write disk images" code of the
[goapple2 emulator](https://github.com/zellyn/goapple2).

Current disk operations supported:

| Feature          | DOS 3.3  | ProDOS | NakedOS/Super-Mon  |
| ---------------- | -------- | ------ | ------------------ |
| basic structures | ✓        | ✓      | ✓                  |
| ls               | ✓        | ✓      | ✓                  |
| dump             | ✓        | ✗      | ✓                  |
| put              | ✗        | ✗      | ✓                  |
| dumptext         | ✗        | ✗      | ✗                  |
| delete           | ✗        | ✗      | ✗                  |
| rename           | ✗        | ✗      | ✗                  |
| put              | ✗        | ✗      | ✗                  |
| puttext          | ✗        | ✗      | ✗                  |
| extract (all)    | ✗        | ✗      | ✗                  |
| lock/unlock      | ✗        | ✗      | ✗                  |
| init             | ✗        | ✗      | ✗                  |
| defrag           | ✗        | ✗      | ✗                  |

# Installing/updating
Assuming you have Go installed, run `go get -u github.com/zellyn/diskii`

You can also download automatically-built binaries from the
[latest release
page](https://github.com/zellyn/diskii/releases/latest). If you
need binaries for a different architecture, please send a pull
request or open an issue.

# Short-term TODOs/roadmap/easy ways to contribute

My rough TODO list (apart from anything marked (✗) in the disk
operations matrix is listed below. Anything that an actual user needs
will be likely to get priority.

- [x] Build per-platform binaries for Linux, MacOS, Windows.
- [x] Implement `GetFile` for DOS 3.3
- [ ] Add and implement the `-l` flag for `ls`
- [x] Add `Delete` to the `disk.Operator` interface
  - [x] Implement it for Super-Mon
  - [ ] Implement it for DOS 3.3
- [ ] Make 13-sector DOS disks work
- [ ] Read/write nybble formats
- [ ] Read/write gzipped files
- [ ] Add basic ProDOS structures
- [ ] Add ProDOS support

# Related tools

- http://a2ciderpress.com/ - the great grandaddy of them all. Windows only, unless you Wine
  - http://retrocomputingaustralia.com/rca-downloads/ Michael Mulhern's MacOS package of CiderPress
- http://applecommander.sourceforge.net/ - the commandline, cross-platform alternative to CiderPress
- http://brutaldeluxe.fr/products/crossdevtools/cadius/index.html - Brutal Deluxe's commandline tools
- https://github.com/paleotronic/dskalyzer - cross-platform disk analysis tool (also written in Go!) from the folks who brought you [Octalyzer](http://octalyzer.com/).
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
- https://github.com/dmolony/DiskBrowser - graphical (Java) disk browser that knows how to interpret and display many file formats
- https://github.com/slotek/apple2-disk-util - ruby
- https://github.com/slotek/dsk2nib - C
- https://github.com/robmcmullen/atrcopy - dos3.3, python

# Notes

## Disk formats

- `.do`
- `.po`
- `.dsk` - could be DO or PO.

DOS 3.2.1: the 13 sectors are physically skewed on disk.

DOS 3.3+: the 16 physical sectors are stored in ascending order on disk, not physically skewed at all. The 


| Logical Sector  | DOS 3.3 Physical Sector | ProDOS Physical Sector |
| --------------- | -------------- | ------------- |
| 0 | 0 | x |
| 1 | D | x | 
| 2 | B | x | 
| 3 | 9 | x | 
| 4 | 7 | x | 
| 5 | 5 | x | 
| 6 | 3 | x | 
| 7 | 1 | x | 
| 8 | E | x | 
| 9 | C | x | 
| A | A | x | 
| B | 8 | x | 
| C | 6 | x | 
| D | 4 | x | 
| E | 2 | x | 
| F | F | x | 

### RWTS - DOS

Sector mapping:
http://www.textfiles.com/apple/ANATOMY/rwts.s.txt and search for INTRLEAV

Mapping from specified sector to physical sector (the reverse of what the comment says):

`00 0D 0B 09 07 05 03 01 0E 0C 0A 08 06 04 02 0F`

So if you write to "T0S1" with DOS RWTS, it ends up in physical sector 0D.

## Commandline examples for thinking about how it should work

diskii ls dos33.dsk
diskii --order=do ls dos33.dsk
diskii --order=do --system=nakedos ls nakedos.dsk
