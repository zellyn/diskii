diskii
======

diskii is a commandline tool for working with Apple II disk images.

[![Build Status](https://travis-ci.org/zellyn/diskii.svg?branch=master)](https://travis-ci.org/zellyn/diskii)

Eventually, it aims to be a comprehensive disk image manipulation
tool, but for now only the `applesoft decode` command works.

It is pronounced so as to rhyme with "whiskey".

Discussion/support is in
[#apple2 on the retrocomputing Slack](https://retrocomputing.slack.com/messages/apple2/)
(invites [here](https://retrocomputing.herokuapp.com)).

### Installing/updating
Assuming you have Go installed, run `go get -u github.com/zellyn/diskii`

### Short-term TODOs/roadmap

- [ ] Build per-platform binaries for Linux, MacOS, Windows
- [ ] Implement CATALOG, deletion, and creation of files in DOS 3.3 images
