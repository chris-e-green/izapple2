package izapple2

import "fmt"

type traceMonitor struct {
	a *Apple2
}

func (t *traceMonitor) connect(a *Apple2) {
	t.a = a
}

func (t *traceMonitor) inspect() {
	pc, _ := t.a.cpu.GetPCAndSP()
	ch := t.a.mmu.Peek(0x24)
	cv := t.a.mmu.Peek(0x25)
	basl := t.a.mmu.Peek(0x28)
	a, x, y, _ := t.a.cpu.GetAXYP()
	ac := a & 0x7f
	var as string
	if ac < 0x20 {
		ac += 0x40
		as = "^" + string(ac)
	} else {
		as = string(ac)
	}
	switch pc {
	case 0xc305:
		fmt.Println("BASICIN")
	case 0xc307:
		fmt.Printf("BASICOUT('%s' at (%v, %v)))\n", as, ch, cv)
	case 0xff3a:
		fmt.Println("BELL")
	case 0xfbdd:
		fmt.Println("BELL1")
	case 0xfc9c:
		fmt.Printf("CLREOL(from (%v, %v)))\n", ch, cv)
	case 0xfc9e:
		fmt.Printf("CLEOLZ(from %#02x indexed by %#02x)\n", basl, y)
	case 0xfc42:
		fmt.Printf("CLREOP(from (%v, %v))\n", ch, cv)
	case 0xf832:
		fmt.Println("CLRSCR")
	case 0xf836:
		fmt.Println("CLRTOP")
	case 0xfded:
		fmt.Printf("COUT('%s')\n", as)
	case 0xfdf0:
		fmt.Printf("COUT1('%s' at (%v, %v)))\n", as, ch, cv)
	case 0xfd8e:
		fmt.Println("CROUT")
	case 0xfd8b:
		fmt.Printf("CROUT1(clear from (%v, %v))\n", ch, cv)
	case 0xfd6a:
		fmt.Printf("GETLN(prompt %v)\n", t.a.mmu.Peek(0x33))
	case 0xfd67:
		fmt.Println("GETLNZ")
	case 0xfd6f:
		fmt.Println("GETLN1")
	case 0xf819:
		fmt.Printf("HLINE(%v, %v, %v\n)", y, t.a.mmu.Peek(0x2c), a)
	case 0xfc58:
		fmt.Println("HOME")
	case 0xff3f:
		fmt.Println("IOREST")
	case 0xff4a:
		fmt.Println("IOSAVE")
	case 0xfd1b:
		fmt.Println("KEYIN")
	case 0xfe2c:
		fmt.Printf("MOVE(%#04x-%#04x to %#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8,
			uint16(t.a.mmu.Peek(0x42))+uint16(t.a.mmu.Peek(0x43))<<8,
		)
	case 0xf85f:
		fmt.Println("NEXTCOL")
	case 0xf800:
		fmt.Printf("PLOT(%v, %v)\n", y, a)
	case 0xf948:
		fmt.Println("PRBLNK")
	case 0xf94a:
		fmt.Printf("PRBL2(%v)\n", x)
	case 0xfdda:
		fmt.Printf("PRBYTE(%v)\n", a)
	case 0xfb1e:
		fmt.Printf("PREAD(%v)\n", x)
	case 0xff2d:
		fmt.Println("PRERR")
	case 0xfde3:
		fmt.Printf("PRHEX(%v)\n", a)
	case 0xf941:
		fmt.Printf("PRNTAX(%#02x, %#02x)\n", a, x)
	case 0xfd35:
		fmt.Println("RDCHAR")
	case 0xfd0c:
		fmt.Println("RDKEY")
	case 0xfefd:
		fmt.Printf("READ(%#04x-%#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8)
	case 0xf871:
		fmt.Printf("SCRN(%v, %v)\n", y, a)
	case 0xf864:
		fmt.Printf("SETCOL(%v)\n", a)
	case 0xfe80:
		fmt.Println("SETINV")
	case 0xfe84:
		fmt.Println("SETNORM")
	case 0xfe36:
		fmt.Printf("VERIFY(%#04x-%#04x to %#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8,
			uint16(t.a.mmu.Peek(0x42))+uint16(t.a.mmu.Peek(0x43))<<8,
		)
	case 0xf828:
		fmt.Printf("VLINE(%v, %v, %v)\n", y, a, t.a.mmu.Peek(0x2d))
	case 0xfca8:
		fa := float32(a)
		fmt.Printf("WAIT(%vms)\n", 0.5*(26.0+27.0*fa+5.0*fa*fa))
	case 0xfecd:
		fmt.Printf("WRITE(%#04x-%#04x)\n",
			uint16(t.a.mmu.Peek(0x3c))+uint16(t.a.mmu.Peek(0x3d))<<8,
			uint16(t.a.mmu.Peek(0x3e))+uint16(t.a.mmu.Peek(0x3f))<<8)
	}
}

func newTraceMonitor() *traceMonitor {
	var t traceMonitor
	return &t
}
