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

Discussion/support is on the
[apple2infinitum Slack](https://apple2infinitum.slack.com/)
(invites [here](http://apple2.gs:3000/)).

# Examples

Get a listing of files on a DOS 3.3 disk image:
```
diskii ls dos33master.dsk
```

… or a ProDOS disk image:
```
diskii ls ProDOS_2_4_2.po
```

… or a Super-Mon disk image:
```
diskii ls Super-Mon-2.0.dsk 
```

Reorder the sectors in a disk image:
```
diskii reorder ProDOS_2_4_2.dsk ProDOS_2_4_2.po
```


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
| dump             | ✗        | ✗      | ✗                  |
| put              | ✗        | ✗      | ✗                  |
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

- [ ] Make `put` accept load address for appropriate filetypes.
- [ ] Implement `GetFile` for prodos
- [x] Build per-platform binaries for Linux, MacOS, Windows.
- [x] Implement `GetFile` for DOS 3.3
- [ ] Add and implement the `-l` flag for `ls`
- [x] Add `Delete` to the `disk.Operator` interface
  - [ ] Implement it for Super-Mon
  - [ ] Implement it for DOS 3.3
- [ ] Add ProDOS support for all commands
- [x] Make `filetypes` command use a tabwriter to write as a table

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
- `.dsk` - could be DO or PO. When in doubt, assume DO.

| Physical Sectors | DOS 3.2 Logical | DOS 3.3 Logical | ProDOS/Pascal Logical | CP/M Logical |
|------------------|-----------------|-----------------|-----------------------|------------- |
|        0         |        0        |        0        |          0.0          |      0.0     |
|        1         |        1        |        7        |          4.0          |      2.3     |
|        2         |        2        |        E        |          0.1          |      1.2     |
|        3         |        3        |        6        |          4.1          |      0.1     |
|        4         |        4        |        D        |          1.0          |      3.0     |
|        5         |        5        |        5        |          5.0          |      1.3     |
|        6         |        6        |        C        |          1.1          |      0.2     |
|        7         |        7        |        4        |          5.1          |      3.1     |
|        8         |        8        |        B        |          2.0          |      2.0     |
|        9         |        9        |        3        |          6.0          |      0.3     |
|        A         |        A        |        A        |          2.1          |      3.2     |
|        B         |        B        |        2        |          6.1          |      2.1     |
|        C         |        C        |        9        |          3.0          |      1.0     |
|        D         |                 |        1        |          7.0          |      3.3     |
|        E         |                 |        8        |          3.1          |      2.2     |
|        F         |                 |        F        |          7.1          |      1.1     |

_Note: DOS 3.2 rearranged the physical sectors on disk to achieve interleaving._
### RWTS - DOS

Sector mapping:
http://www.textfiles.com/apple/ANATOMY/rwts.s.txt and search for INTRLEAV

Mapping from specified sector to physical sector:

`00 0D 0B 09 07 05 03 01 0E 0C 0A 08 06 04 02 0F`

So if you write to "T0S1" with DOS RWTS, it ends up in physical sector 0D.

## Commandline examples for thinking about how it should work

diskii ls dos33.dsk
diskii --order=do ls dos33.dsk
diskii --order=do --system=nakedos ls nakedos.dsk
