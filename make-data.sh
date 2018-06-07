go-bindata -pkg data -o data/data.go \
    data/disks/ProDOS_2_4_1.dsk \
    data/disks/dos33master.woz \
    data/boot/prodos-new-boot0.bin \
    data/boot/prodos-old-boot0.bin
goimports -w data/data.go
