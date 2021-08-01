// Package data is a bunch of go:embed embedded files.
package data

import _ "embed" // Mark this file as using embeds.

// DOS33masterDSK is a DOS 3.3 Master Disk image.
//go:embed disks/dos33master.dsk
var DOS33masterDSK []byte

// DOS33masterWOZ is a DOS 3.3 Master Disk, as a .woz file.
//go:embed disks/dos33master.woz
var DOS33masterWOZ []byte

// ProDOS242PO is John Brooks' update to ProDOS.
// Website: https://prodos8.com
// Announcements: https://www.callapple.org/author/jbrooks/
//go:embed disks/ProDOS_2_4_2.po
var ProDOS242PO []byte

// ProDOSNewBootSector0 is the new ProDOS sector 0, used on and after the IIGS System 4.0. Understands sparse PRODOS.SYSTEM files.
//go:embed boot/prodos-new-boot0.bin
var ProDOSNewBootSector0 []byte

// ProDOSOldBootSector0 is the old ProDOS sector 0, used before the IIGS System 4.0 system disk.
//go:embed boot/prodos-old-boot0.bin
var ProDOSOldBootSector0 []byte
