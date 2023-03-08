#include "textflag.h"

// func And(addr *uint32, v uint32)
TEXT ·And(SB),NOSPLIT,$0
	JMP	runtime∕internal∕atomic·And(SB)

// func Or(addr *uint32, v uint32)
TEXT ·Or(SB),NOSPLIT,$0
	JMP	runtime∕internal∕atomic·Or(SB)

// uint32 ·Load(uint32 volatile* addr)
TEXT ·Load(SB),NOSPLIT,$0-12
    JMP	runtime∕internal∕atomic·Load(SB)
