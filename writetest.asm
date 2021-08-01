    * = $6000
    START = *

    ;; Free addresses: 6, 7, 8, 9

    IOB = $6
    RWTS = $3D9
    GET_IOB = $3E3
    BUF = $6100

start:
    ; set up IOB address
    jsr GET_IOB
    sta IOB+1
    sty IOB

    lda #$f
outer:
    ldy #0
inner:
    sta BUF,Y
    iny
    bne inner
    pha
    jsr write
    pla
    tax
    dex
    txa
    bpl outer
    rts





write:
    ldy #$5     ; sector number
    sta ($6), Y
    ldy #$2     ; drive number
    lda #$2     ; drive 2
    sta ($6), Y
    iny         ; 3: volume
    lda #0      ; any volume
    sta ($6), Y
    iny         ; 4: track
    sta ($6), Y
    ldy #$8     ; LO of buffer
    lda #<BUF
    sta ($6), Y
    iny         ; $9: HI of buffer
    lda #>BUF
    sta ($6), Y
    iny         ; $A: unused
    iny         ; $B: Byte count
    lda #0
    sta ($6),Y
    iny         ; $C: command
    lda #$02    ; write
    sta ($6),Y
    lda IOB+1
    ldy IOB
    jsr RWTS
    rts
