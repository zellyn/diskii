diskii
======

**Note:** diskii is not stable yet! I don't expect to remove
functionality, but I'm still experimenting with the command syntax and
organization, so don't get too comfy with it yet.

diskii is a commandline tool for working with Apple II disk images.

It is also a library of code that can be used by other Go programs.

Its major advantage is that it's written in Go, hence cross-platform.

Its major disadvantage is that it mostly doesn't exist yet.

[![Build Status](https://travis-ci.org/zellyn/diskii.svg?branch=master)](https://travis-ci.org/zellyn/diskii)

It rhymes with “whiskey”.

Discussion/support is in
[#apple2 on the retrocomputing Slack](https://retrocomputing.slack.com/messages/apple2/)
(invites [here](https://retrocomputing.herokuapp.com)).

### Goals

Eventually, it aims to be a comprehensive disk image manipulation
tool, but for now only the `applesoft decode` command works.

The library code aims (a) to support the commandline tool operations, and (b) to replace the "read and write disk images" code of the [goapple2 emulator](https://github.com/zellyn/goapple2).


### Installing/updating
Assuming you have Go installed, run `go get -u github.com/zellyn/diskii`

### Short-term TODOs/roadmap

- [ ] Build per-platform binaries for Linux, MacOS, Windows
- [ ] Implement CATALOG, deletion, and creation of files in DOS 3.3 images

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
