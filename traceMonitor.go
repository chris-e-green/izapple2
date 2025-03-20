package izapple2

import (
	"fmt"
)

/*

	Trace the inpit and output of using the wozmon calls.

*/

type traceMonitor struct {
	a             *Apple2
	closingBuffer bool
	buffer        string
}

const (
	//wozmonPrompt uint16 = 0x0033
	wozmonBASICIN uint16 = 0xc305
	wozmonBASICOUT uint16 = 0xc307

	wozmonPLOT uint16 = 0xf800
	wozmonHLINE uint16 = 0xf819
	wozmonVLINE uint16 = 0xf828
	wozmonCLRSCR uint16 = 0xf832
	wozmonCLRTOP uint16 = 0xf836
	wozmonNEXTCOL uint16 = 0xf85f
	wozmonSETCOL uint16 = 0xf864
	wozmonSCRN uint16 = 0xf871

	wozmonPRNTAX uint16 = 0xf941
	wozmonPRBLNK uint16 = 0xf948
	wozmonPRBL2 uint16 = 0xf94a

	wozmonPREAD uint16 = 0xfb1e
	wozmonBELL1 uint16 = 0xfbdd

	wozmonCLREOP uint16 = 0xfc42
	wozmonHOME uint16 = 0xfc58
	wozmonCLREOL uint16 = 0xfc9c
	wozmonCLEOLZ uint16 = 0xfc9e
	wozmonWAIT uint16 = 0xfca8

	wozmonRDKEY       uint16 = 0xfd0c
	wozmonKEYIN       uint16 = 0xfd1b
	wozmonRDCHAR      uint16 = 0xfd35
	wozmonGETLNZ uint16 = 0xfd67
	wozmonGETLN1 uint16 = 0xfd6f
	wozmonGETLN uint16 = 0xfd6a
	wozmonCROUT1 uint16 = 0xfd8b
	wozmonCROUT uint16 = 0xfd8e
	wozmonGETLNReturn uint16 = 0xfd90
	wozmonPRBYTE uint16 = 0xfdda
	wozmonPRHEX uint16 = 0xfde3
	wozmonCOUT uint16 = 0xfded
	wozmonCOUT1 uint16 = 0xfdf0
	wozmonCOUTZ uint16 = 0xfdf6

	wozmonMOVE uint16 = 0xfe2c
	wozmonVERIFY uint16 = 0xfe36
	wozmonSETINV uint16 = 0xfe80
	wozmonSETNORM uint16 = 0xfe84
	wozmonWRITE uint16 = 0xfecd
	wozmonREAD uint16 = 0xfefd

	wozmonPRERR uint16 = 0xff2d
	wozmonBELL uint16 = 0xff3a
	wozmonIOREST uint16 = 0xff3f
	wozmonIOSAVE uint16 = 0xff4a
)

func newTraceMonitor() *traceMonitor {
	var t traceMonitor
	return &t
}

func (t *traceMonitor) connect(a *Apple2) {
	t.a = a
}

func (t *traceMonitor) inspect() {
	if t.a.dmaActive {
		return
	}

	if t.a.mmu.altMainRAMActiveRead {
		// We want to trace only the activity on the ROM
		return
	}

	pc, _ := t.a.cpu.GetPCAndSP()
	a, x, y, _ := t.a.cpu.GetAXYP()
	ch := t.a.mmu.Peek(0x24)
	cv := t.a.mmu.Peek(0x25)
	basl := t.a.mmu.Peek(0x28)

	desc := ""
	switch pc {
	case wozmonBASICIN:
		desc = "BASICIN"
	case wozmonBASICOUT:
		desc = fmt.Sprintf("'%s' at (%v, %v)))\n", toAscii(a), ch, cv)
	case wozmonBELL:
		desc = "BELL"
	case wozmonBELL1:
		desc = "BELL1"
	case wozmonCLREOL:
		desc = fmt.Sprintf("CLREOL(from (%v, %v)))\n", ch, cv)
	case wozmonCLEOLZ:
		desc = fmt.Sprintf("CLEOLZ(from %#02x indexed by %#02x)\n", basl, y)
	case wozmonCLREOP:
		desc = fmt.Sprintf("CLREOP(from (%v, %v))\n", ch, cv)
	case wozmonCLRSCR:
		desc = "CLRSCR"
	case wozmonCLRTOP:
		desc = "CLRTOP"
	case wozmonCROUT:
		desc = "CROUT"
		case wozmonGETLNZ:
		desc = "GETLNZ"
	case wozmonCROUT1:
		desc = fmt.Sprintf("CROUT1(clear from (%v, %v))\n", ch, cv)

	case wozmonGETLN:
		fmt.Printf("Wozmon output: %s\n", t.buffer)
		t.buffer = ""
		desc = "GETLN"
	case wozmonGETLN1:
		desc = "GETLN1"
	case wozmonGETLNReturn:
		t.closingBuffer = true
		//desc = "GETLN return"
		case wozmonHLINE:
			desc = fmt.Sprintf("HLINE(%v, %v, %v\n)", y, t.a.mmu.Peek(0x2c), a)

	case wozmonHOME:
		desc = "HOME"
	case wozmonIOREST:
		desc = "IOREST"
	case wozmonIOSAVE:
		desc = "IOSAVE"
	case wozmonMOVE:
		desc = fmt.Sprintf("MOVE(%#04x-%#04x to %#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8,
			uint16(t.a.mmu.Peek(0x42))+uint16(t.a.mmu.Peek(0x43))<<8)
	case wozmonNEXTCOL:
		desc = "NEXTCOL"
	case wozmonPLOT:
		desc = fmt.Sprintf("PLOT(%v, %v)\n", y, a)
	case wozmonPRBLNK:
		desc = "PRBLNK"
	case wozmonPRBL2:
		desc = fmt.Sprintf("PRBL2(%v)\n", x)
	case wozmonPRBYTE:
		desc = fmt.Sprintf("PRBYTE(%v)\n", a)
	case wozmonPREAD:
		desc = fmt.Sprintf("PREAD(%v)\n", x)
	case wozmonPRERR:
		desc = "PRERR"
	case wozmonPRHEX:
		desc = fmt.Sprintf("PRHEX(%v)\n", a)
	case wozmonPRNTAX:
		desc = fmt.Sprintf("PRNTAX(%#02x, %#02x)\n", a, x)
	case wozmonRDCHAR:
		desc = "RDCHAR"
	case wozmonREAD:
		desc = fmt.Sprintf("READ(%#04x-%#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8)
	case wozmonSCRN:
		desc = fmt.Sprintf("SCRN(%v, %v)\n", y, a)
	case wozmonSETCOL:
		desc = fmt.Sprintf("SETCOL(%v)\n", a)
	case wozmonSETINV:
		desc = "SETINV"
	case wozmonSETNORM:
		desc = "SETNORM"
	case wozmonVERIFY:
		desc = fmt.Sprintf("VERIFY(%#04x-%#04x to %#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8,
			uint16(t.a.mmu.Peek(0x42))+uint16(t.a.mmu.Peek(0x43))<<8)
	case wozmonVLINE:
		desc = fmt.Sprintf("VLINE(%v, %v, %v)\n", y, a, t.a.mmu.Peek(0x2d))
	case wozmonWAIT:
		fa := float32(a)
		desc = fmt.Sprintf("WAIT(%vms)\n", 0.5*(26.0+27.0*fa+5.0*fa*fa))
	case wozmonWRITE:
		desc = fmt.Sprintf("WRITE(%#04x-%#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8)
		case wozmonRDKEY:
		//desc = "RDKEY"
	case wozmonKEYIN:
		//desc = "KEYIN"
	case wozmonCOUT:
		//desc = fmt.Sprintf("COUT  0x%02x %c", a, toAscii(a))
		if t.closingBuffer {
			fmt.Printf("Wozmon input: %s\n", t.buffer)
			t.buffer = ""
			t.closingBuffer = false
			desc = fmt.Sprintf("GETLN returns <<%s>>", t.getInputBuffer())
		}
	case wozmonCOUT1:
		t.buffer += string(toAscii(a))
		//desc = fmt.Sprintf("COUT1 0x%02x %c", a, toAscii(a))
	case wozmonCOUTZ:
		desc = "COUTZ"
	}

	if desc != "" {
		fmt.Printf("Wozmon call to $%04x %s\n", pc, desc)
	}
}

func toAscii(b uint8) rune {
	b = b & 0x7f
	if b < 0x20 {
		return rune(uint16(b) + 0x2400)
	}
	return rune(b)
}

func (t *traceMonitor) getInputBuffer() string {
	buffer := ""
	for address := uint16(0x200); address < 0x300; address++ {
		b := t.a.mmu.Peek(address)
		buffer += string(toAscii(b))
		if b == 0x8d {
			break
		}
	}

	return buffer
}
