package izapple2

import (
	"fmt"
)

type tracePascalPCode struct {
	a           *Apple2
	skipConsole bool
}

const (
	// p-machine pseudo-registers
	pascalBASE   uint16 = 0x0050 // P-Machine BASE Procedure
	pascalMP     uint16 = 0x0052 // P-Machine Markstack Pointer
	pascalJTAB   uint16 = 0x0054
	pascalSEG    uint16 = 0x0056
	pascalIPC    uint16 = 0x0058 // P-Machine IPC
	pascalNP     uint16 = 0x005a // P-Machine New Pointer
	pascalKP     uint16 = 0x005c // P-Machine program stack pointer
	pascalSTRP   uint16 = 0x005e // P-Machine string pointer
	pascalSPTEMP uint16 = 0x0067
)

// TODO: STACK code is wrong - eg retrieved 2nd from tos when was looking for tos
// A structure to hold the signatures for each p-code interpreter
type pascalSignature struct {
	version int    // the Apple Pascal version * 10
	runtime bool   // true if this is a runtime vs developer p-machine
	mem128  bool   // true if this is a 128K p-machine
	pcode   uint16 // the address of the modifiable JMP for p-code handling
	pcode2  uint16 // the address of the alternate JMP for p-code handling
	csp     uint16 // the address of the modifiable JMP for CSP handling
	sldc    uint16 // the address of the start of the SLDC handler
}

type ParamType int

const (
	SB ParamType = iota
	UB
	DB
	BIG
	WORD
)

type opCode struct {
	mnemonic        string
	params          []ParamType
	stackParamCount int
	description     string
}

// All p-code interpreter signatures known so far
var pascalSignatures = []pascalSignature{
	{version: 10, runtime: false, mem128: false, pcode: 0x006e, csp: 0x0071, sldc: 0xd238},
	{version: 11, runtime: false, mem128: false, pcode: 0x006e, csp: 0x0071, sldc: 0xd248},
	{version: 11, runtime: true, mem128: false, pcode: 0x006e, csp: 0x0071, sldc: 0x9888},
	{version: 12, runtime: false, mem128: false, pcode: 0xd26e, csp: 0xe658, sldc: 0xd259},
	{version: 12, runtime: false, mem128: true, pcode: 0xd28f, csp: 0xea8f, sldc: 0xd274},
	{version: 13, runtime: false, mem128: false, pcode: 0xd297, pcode2: 0xd284, csp: 0xe696, sldc: 0xd271},
	{version: 13, runtime: false, mem128: true, pcode: 0xd2c4, csp: 0xeca0, sldc: 0xd295},
}

var initialised = false

var sldc uint16     // the address of the SLDC handler
var csp uint16      // the address where the CSP handler is set per call
var cspJmp uint16   // the JMP address for the CSP handler
var pcode uint16    // the address where the p-code handler is set per call
var pcodeJmp uint16 //  the JMP address for the p-code handler
var pcode2 uint16
var pcode2Jmp uint16
var pcodeOpcodes = []opCode{
	{mnemonic: "ABI", params: nil, stackParamCount: 1, description: "Absolute value of integer"},
	{mnemonic: "ABR", params: nil, stackParamCount: 1, description: "Absolute value of real"},
	{mnemonic: "ADI", params: nil, stackParamCount: 2, description: "Add integers"},
	{mnemonic: "ADR", params: nil, stackParamCount: 2, description: "Add reals"},
	{mnemonic: "LAND", params: nil, stackParamCount: 2, description: "Logical AND"},
	{mnemonic: "DIF", params: nil, stackParamCount: 2, description: "Set difference"},
	{mnemonic: "DVI", params: nil, stackParamCount: 2, description: "Divide integers"},
	{mnemonic: "DVR", params: nil, stackParamCount: 2, description: "Divide reals"},
	{mnemonic: "CHK", params: nil, stackParamCount: 3, description: "Range check"},
	{mnemonic: "FLO", params: nil, stackParamCount: 2, description: "Float TOS-1"},
	{mnemonic: "FLT", params: nil, stackParamCount: 1, description: "Float TOS"},
	{mnemonic: "INN", params: nil, stackParamCount: 2, description: "Set membership"},
	{mnemonic: "INT", params: nil, stackParamCount: 2, description: "Set intersection"},
	{mnemonic: "LOR", params: nil, stackParamCount: 2, description: "Logical OR"},
	{mnemonic: "MODI", params: nil, stackParamCount: 2, description: "Modulo integers"},
	{mnemonic: "MPI", params: nil, stackParamCount: 2, description: "Multiply integers"},
	{mnemonic: "MPR", params: nil, stackParamCount: 2, description: "Multiply reals"},
	{mnemonic: "NGI", params: nil, stackParamCount: 1, description: "Negate integer"},
	{mnemonic: "NGR", params: nil, stackParamCount: 1, description: "Negate real"},
	{mnemonic: "LNOT", params: nil, stackParamCount: 1, description: "Logical NOT"},
	{mnemonic: "SRS", params: nil, stackParamCount: 2, description: "Build a subrange set"},
	{mnemonic: "SBI", params: nil, stackParamCount: 2, description: "Subtract integers"},
	{mnemonic: "SBR", params: nil, stackParamCount: 2, description: "Subtract reals"},
	{mnemonic: "SGS", params: nil, stackParamCount: 1, description: "Build a one-member set"},
	{mnemonic: "SQI", params: nil, stackParamCount: 1, description: "Square integer"},
	{mnemonic: "SQR", params: nil, stackParamCount: 1, description: "Square real"},
	{mnemonic: "STO", params: nil, stackParamCount: 2, description: "Store indirect word"},
	{mnemonic: "IXS", params: nil, stackParamCount: 2, description: "Index string array"},
	{mnemonic: "UNI", params: nil, stackParamCount: 2, description: "Set union"},
	{mnemonic: "LDE", params: []ParamType{UB, BIG}, stackParamCount: 0, description: "Load extended word"},
	{mnemonic: "CSP", params: []ParamType{UB}, stackParamCount: 0, description: "Call standard procedure"},
	{mnemonic: "LDCN", params: nil, stackParamCount: 0, description: "Load constant NIL"},
	{mnemonic: "ADJ", params: []ParamType{UB}, stackParamCount: 2, description: "Adjust set"},
	{mnemonic: "FJP", params: []ParamType{SB}, stackParamCount: 1, description: "False jump"},
	{mnemonic: "INC", params: []ParamType{BIG}, stackParamCount: 1, description: "Increment field pointer"},
	{mnemonic: "IND", params: []ParamType{BIG}, stackParamCount: 1, description: "Static index and load word"},
	{mnemonic: "IXA", params: []ParamType{BIG}, stackParamCount: 2, description: "Index array"},
	{mnemonic: "LAO", params: []ParamType{BIG}, stackParamCount: 0, description: "Load global address"},
	{mnemonic: "LSA", params: []ParamType{UB}, stackParamCount: 0, description: "Load constant string address"},
	{mnemonic: "LAE", params: []ParamType{UB, BIG}, stackParamCount: 0, description: "Load extended address"},
	{mnemonic: "MOV", params: []ParamType{BIG}, stackParamCount: 2, description: "Move words"},
	{mnemonic: "LDO", params: []ParamType{BIG}, stackParamCount: 0, description: "Load global word"},
	{mnemonic: "SAS", params: []ParamType{UB}, stackParamCount: 2, description: "String assign"},
	{mnemonic: "SRO", params: []ParamType{BIG}, stackParamCount: 1, description: "Store global word"},
	{mnemonic: "XJP", params: []ParamType{WORD, WORD, WORD}, stackParamCount: 1, description: "Case jump"},
	{mnemonic: "RNP", params: []ParamType{DB}, stackParamCount: 0, description: "Return from nonbase procedure"},
	{mnemonic: "CIP", params: []ParamType{UB}, stackParamCount: 0, description: "Call intermediate procedure"},
	{mnemonic: "EQU", params: []ParamType{UB}, stackParamCount: 2, description: "Equal"},
	{mnemonic: "GEQ", params: []ParamType{UB}, stackParamCount: 2, description: "Greater than or equal"},
	{mnemonic: "GRT", params: []ParamType{UB}, stackParamCount: 2, description: "Greater than"},
	{mnemonic: "LDA", params: []ParamType{DB, BIG}, stackParamCount: 0, description: "Load intermediate address"},
	{mnemonic: "LDC", params: []ParamType{UB}, stackParamCount: 0, description: "Load multiple-word constant"},
	{mnemonic: "LEQ", params: []ParamType{UB}, stackParamCount: 2, description: "Less than or equal"},
	{mnemonic: "LES", params: []ParamType{UB}, stackParamCount: 2, description: "Less than"},
	{mnemonic: "LOD", params: []ParamType{DB, BIG}, stackParamCount: 0, description: "Load intermediate word"},
	{mnemonic: "NEQ", params: []ParamType{UB}, stackParamCount: 2, description: "Not equal"},
	{mnemonic: "STR", params: []ParamType{DB, BIG}, stackParamCount: 1, description: "Store intermediate word"},
	{mnemonic: "UJP", params: []ParamType{SB}, stackParamCount: 0, description: "Unconditional jump"},
	{mnemonic: "LDP", params: nil, stackParamCount: 1, description: "Load a packed field"},
	{mnemonic: "STP", params: nil, stackParamCount: 2, description: "Store into a packed field"},
	{mnemonic: "LDM", params: []ParamType{UB}, stackParamCount: 1, description: "Load multiple words"},
	{mnemonic: "STM", params: []ParamType{UB}, stackParamCount: 2, description: "Store multiple words"},
	{mnemonic: "LDB", params: nil, stackParamCount: 2, description: "Load byte"},
	{mnemonic: "STB", params: nil, stackParamCount: 3, description: "Store byte"},
	{mnemonic: "IXP", params: []ParamType{UB, UB}, stackParamCount: 2, description: "Index packed array"},
	{mnemonic: "RBP", params: []ParamType{DB}, stackParamCount: 0, description: "Return from base procedure"},
	{mnemonic: "CBP", params: []ParamType{UB}, stackParamCount: 0, description: "Call base procedure"},
	{mnemonic: "EQUI", params: nil, stackParamCount: 2, description: "Equals integer"},
	{mnemonic: "GEQI", params: nil, stackParamCount: 2, description: "Greater than or equal integer"},
	{mnemonic: "GRTI", params: nil, stackParamCount: 2, description: "Greater than integer"},
	{mnemonic: "LLA", params: []ParamType{BIG}, stackParamCount: 0, description: "Load local address"},
	{mnemonic: "LDCI", params: []ParamType{WORD}, stackParamCount: 0, description: "Load one-word constant"},
	{mnemonic: "LEQI", params: nil, stackParamCount: 2, description: "Less than or equal integer"},
	{mnemonic: "LESI", params: nil, stackParamCount: 2, description: "Less than integer"},
	{mnemonic: "LDL", params: []ParamType{BIG}, stackParamCount: 0, description: "Load local word"},
	{mnemonic: "NEQI", params: nil, stackParamCount: 2, description: "Not equal integer"},
	{mnemonic: "STL", params: []ParamType{BIG}, stackParamCount: 1, description: "Store local word"},
	{mnemonic: "CXP", params: []ParamType{UB, UB}, stackParamCount: 0, description: "Call external procedure"},
	{mnemonic: "CLP", params: []ParamType{UB}, stackParamCount: 0, description: "Call local procedure"},
	{mnemonic: "CGP", params: []ParamType{UB}, stackParamCount: 0, description: "Call global procedure"},
	{mnemonic: "LPA", params: []ParamType{UB}, stackParamCount: 0, description: "Load a packed array"},
	{mnemonic: "STE", params: []ParamType{UB, BIG}, stackParamCount: 1, description: "Store extended word"},
	{mnemonic: "NOP", params: nil, stackParamCount: 0, description: "No operation"},
	{mnemonic: "-", params: nil, stackParamCount: 0},
	{mnemonic: "-", params: nil, stackParamCount: 0},
	{mnemonic: "BPT", params: []ParamType{BIG}, stackParamCount: 0, description: "Breakpoint"},
	{mnemonic: "XIT", params: nil, stackParamCount: 0, description: "Exit the operating system"},
	{mnemonic: "NOP", params: nil, stackParamCount: 0, description: "No operation"},
	{mnemonic: "SLDL 1", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 2", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 3", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 4", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 5", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 6", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 7", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 8", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 9", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 10", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 11", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 12", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 13", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 14", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 15", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDL 16", params: nil, stackParamCount: 0, description: "Short Load local word"},
	{mnemonic: "SLDO 1", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 2", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 3", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 4", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 5", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 6", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 7", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 8", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 9", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 10", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 11", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 12", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 13", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 14", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 15", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SLDO 16", params: nil, stackParamCount: 0, description: "Short Load global word"},
	{mnemonic: "SIND 0", params: nil, stackParamCount: 1, description: "Load indirect word"},
	{mnemonic: "SIND 1", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 2", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 3", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 4", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 5", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 6", params: nil, stackParamCount: 1, description: "Short index and load word"},
	{mnemonic: "SIND 7", params: nil, stackParamCount: 1, description: "Short index and load word"},
}
var pcodeMnemonics = []string{
	"ABI", "ABR", "ADI", "ADR", "LAND", "DIFF", "DVI", "DVR",
	"CHK", "FLO", "FLT", "INN", "INT", "LOR", "MODI", "MPI",
	"MPR", "NGI", "NGR", "LNOT", "SRS", "SBI", "SBR", "SGS",
	"SQI", "SQR", "STO", "IXS", "UNI", "LDE", "CSP", "LDCN",
	"ADJ", "FJP", "INCP", "IND", "IXA", "LAO", "LSA", "LAE",
	"MOV", "LDO", "SAS", "SRO", "XJP", "RNP", "CIP", "CEQL",
	"CGEQ", "CGTR", "LDAP", "LDC", "CLEQ", "CLSS", "LOD", "CNEQ",
	"STR", "UJP", "LDP", "STP", "LDM", "STM", "LDB", "STB",
	"IXP", "RBP", "CBP", "EQUI", "GEQI", "GTRI", "LLA", "LDCI",
	"LEQI", "LESI", "LDL", "NEQI", "STL", "CXP", "CLP", "CGP",
	"LPA", "STE", "NOP", "-", "-", "BPT", "XIT", "NOP",
	"SLDL   0", "SLDL   1", "SLDL   2", "SLDL   3", "SLDL   4", "SLDL   5", "SLDL   6", "SLDL   7",
	"SLDL   8", "SLDL   9", "SLDL  10", "SLDL  11", "SLDL  12", "SLDL  13", "SLDL  14", "SLDL  15",
	"SLDO   0", "SLDO   1", "SLDO   2", "SLDO   3", "SLDO   4", "SLDO   5", "SLDO   6", "SLDO   7",
	"SLDO   8", "SLDO   9", "SLDO  10", "SLDO   8", "SLDO  12", "SLDO  13", "SLDO  14", "SLDO  15",
	"SIND   0", "SIND   1", "SIND   2", "SIND   3", "SIND   4", "SIND   5", "SIND   6", "SIND   7",
}
var cspMnemonics = []string{
	"IOC", "NEW", "MOVL", "MOVR", "EXIT", "UREAD", "UWRT", "IDS",
	"TRS", "TIME", "FLCH", "SCAN", "USTAT", "RSRVD", "RSRVD", "RSRVD",
	"RSRVD", "RSRVD", "RSRVD", "RSRVD", "RSRVD", "LDS", "ULS", "TNC",
	"RND", "SIN", "COS", "LOG", "ATAN", "LN", "EXP", "SQRT",
	"MRK", "RLS", "IOR", "UBUSY", "POT", "UWAIT", "UCLR", "HLT",
	"MEMAV",
}

func newTracePascalPCode() *tracePascalPCode {
	var t tracePascalPCode
	t.skipConsole = true
	return &t
}

func (t *tracePascalPCode) connect(a *Apple2) {
	t.a = a
}

/*
See:

	https://archive.org/details/Hyde_P-Source-A_Guide_to_the_APPLE_Pascal_System_1983/page/n415/mode/1up?view=theater
	https://archive.org/details/Apple_II_Pascal_1.2_Device_and_Interrupt_Support_Tools_Manual
*/
func (t *tracePascalPCode) inspect() {
	if t.a.dmaActive {
		return
	}

	if !initialised {
		for _, version := range pascalSignatures {
			// looking for signatures - JMPs at the p-code and csp handlers, and
			// TAX, TYA at the SLDC entry point
			if t.byteAtAddr(version.pcode) == 0x6c && // indirect JMP
				t.byteAtAddr(version.csp) == 0x6c &&
				t.byteAtAddr(version.sldc) == 0xaa &&
				t.byteAtAddr(version.sldc+1) == 0x98 {
				pcodeJmp = version.pcode // save the address of the JMP instruction
				pcode2Jmp = version.pcode2
				pcode = version.pcode + 1 // save the address of the JMP destination
				pcode2 = version.pcode2 + 1
				cspJmp = version.csp  // save the address of the JMP instruction
				csp = version.csp + 1 // save the address of the JMP destination
				sldc = version.sldc
				initialised = true
			}
		}
	}
	if !initialised { // if no p-machine found (yet) nothing to trace
		return
	}
	pc, _ := t.a.cpu.GetPCAndSP()

	var pcodeOpcode uint8
	// the low byte is an offset into a table of 16-bit instruction handlers
	if pc == pcodeJmp {
		pcodeOpcode = t.byteAtAddr(pcode) >> 1
	} else if pc == pcode2Jmp {
		pcodeOpcode = t.byteAtAddr(pcode2) >> 1
	}
	cspOpcode := t.byteAtAddr(csp) >> 1

	// check if we are at the JMP instruction for p-code handler
	//var mnemonic string
	if pc == pcodeJmp || pc == pcode2Jmp {

		instDetails := pcodeOpcodes[pcodeOpcode]
		msg := ""
		msg = msg + instDetails.mnemonic + " "
		//fmt.Printf("Pascal p-code %-8s", instDetails.mnemonic)
		offset := uint16(0)
		for i, param := range instDetails.params {
			if i > 0 {
				msg = msg + ", "
			}
			switch param {
			case DB:
				val, skip := t.getUB(offset)
				offset += skip
				msg = msg + fmt.Sprintf("%d", val)
				//fmt.Printf("%d ", t.getUB(offset))
			case UB:
				val, skip := t.getUB(offset)
				offset += skip
				msg = msg + fmt.Sprintf("%d", val)
				//fmt.Printf("%d ", t.getUB(offset))
			case SB:
				val, skip := t.getSB(offset)
				offset += skip
				msg = msg + fmt.Sprintf("%d", val)
				//fmt.Printf("%d ", t.getSB(offset))
			case BIG:
				val, skip := t.getBig(offset)
				offset += skip
				msg = msg + fmt.Sprintf("%d", val)
				//fmt.Printf("%d ", t.getBig(offset))
			case WORD:
				val, skip := t.getWord(offset)
				offset += skip
				msg = msg + fmt.Sprintf("%d", val)
				//fmt.Printf("%d ", t.getWord(offset))
			}
		}
		if instDetails.stackParamCount > 0 {
			msg += " Stack: "
			for i := 1; i <= instDetails.stackParamCount; i++ {
				msg += fmt.Sprintf("%d ", t.param(uint8(i*2+1)))
			}
		}

		fmt.Printf("%-12s", msg+" ")
	}

	// check if we are at the JMP instruction for csp handler
	if pc == cspJmp {
		fmt.Printf("Pascal p-code CSP %-6s ", cspMnemonics[cspOpcode])
	}

	if pc == sldc {
		a, _, _, _ := t.a.cpu.GetAXYP()
		fmt.Printf("Pascal p-code SLDC %3d   ", a)
	}

	// don't print the end of the line if we're doing a CSP instruction
	if (pc == pcodeJmp && pcodeOpcode != 0x1e) || pc == cspJmp || pc == sldc {
		fmt.Printf("IPC=%04x BASE=%04x MP=%04x JTAB=%04x SEG=%04x NP=%04x KP=%04x\n",
			t.uint16AtAddr(pascalIPC), t.uint16AtAddr(pascalBASE), t.uint16AtAddr(pascalMP),
			t.uint16AtAddr(pascalJTAB), t.uint16AtAddr(pascalSEG), t.uint16AtAddr(pascalNP),
			t.uint16AtAddr(pascalKP))
	}
}

func (t *tracePascalPCode) param(index uint8) uint16 {
	_, sp := t.a.cpu.GetPCAndSP()
	return uint16(t.a.mmu.Peek(0x100+uint16(sp+index))) +
		uint16(t.a.mmu.Peek(0x100+uint16(sp+index+1)))<<8
}

func (t *tracePascalPCode) uint16AtAddr(addr uint16) uint16 {
	return uint16(t.a.mmu.accessRead(addr).peek(addr)) +
		uint16(t.a.mmu.accessRead(addr+1).peek(addr+1))<<8
}

func (t *tracePascalPCode) byteAtAddr(addr uint16) uint8 {
	return t.a.mmu.accessRead(addr).peek(addr)
}

/* instruction stream format decodes */
func (t *tracePascalPCode) getWord(offset uint16) (int16, uint16) {
	ipc := t.uint16AtAddr(pascalIPC)
	lo := uint16(t.a.mmu.accessRead(ipc + 1 + offset).peek(ipc + 1 + offset))
	hi := uint16(t.a.mmu.accessRead(ipc + 2 + offset).peek(ipc + 2 + offset))
	val := hi<<8 + lo
	if val > 0x7fff {
		return int16(^(val & 0x7fff)) * -1, 2
	} else {
		return int16(val), 2
	}
}

func (t *tracePascalPCode) getBig(offset uint16) (int16, uint16) {
	ipc := t.uint16AtAddr(pascalIPC)
	first := t.a.mmu.accessRead(ipc + 1 + offset).peek(ipc + 1 + offset)
	if first <= 127 { // single-byte value
		return int16(first), 1
	}
	first = first & 0x7f // clear high bit, first becomes high byte, second low
	second := t.a.mmu.accessRead(ipc + 2 + offset).peek(ipc + 2 + offset)
	return int16(first)<<8 + int16(second), 2
}

func (t *tracePascalPCode) getUB(offset uint16) (int16, uint16) {
	ipc := t.uint16AtAddr(pascalIPC)
	return int16(t.a.mmu.accessRead(ipc + 1 + offset).peek(ipc + 1 + offset)), 1
}

func (t *tracePascalPCode) getSB(offset uint16) (int16, uint16) {
	ipc := t.uint16AtAddr(pascalIPC)
	val := uint16(t.a.mmu.accessRead(ipc + 1 + offset).peek(ipc + 1 + offset))
	if val > 0x7f {
		return int16(^(val & 0x7f)) * -1, 1
	} else {
		return int16(val), 1
	}
}

/* stack variable format decodes */
func (t *tracePascalPCode) getBoolean(offset uint16) (bool, uint16) {
	_, sp := t.a.cpu.GetPCAndSP()
	val := t.a.mmu.Peek(0x100 + uint16(sp) + offset)
	return val&0x01 == 0x01, 2
}

func (t *tracePascalPCode) getInteger(offset uint16) (int16, uint16) {
	_, sp := t.a.cpu.GetPCAndSP()
	lo := uint16(t.a.mmu.Peek(0x100 + uint16(sp) + offset))
	hi := uint16(t.a.mmu.Peek(0x100 + uint16(sp) + 1 + offset))
	val := hi<<8 + lo
	if val > 0x7fff {
		return int16(^(val & 0x7fff)) * -1, 2
	} else {
		return int16(val), 2
	}
}

func (t *tracePascalPCode) getLongInt(offset uint16) (int32, uint16) {
	_, sp := t.a.cpu.GetPCAndSP()
	wlen := uint16(t.a.mmu.Peek(0x100+uint16(sp)+offset)) + uint16(t.a.mmu.Peek(0x100+uint16(sp)+1+offset))<<8
	sign := int32(1)
	if t.a.mmu.Peek(0x100+uint16(sp)+2+offset) == 0 {
		sign = -1
	}
	num := int32(0)
	for i := uint16(2); i <= wlen; i++ {
		val := uint16(t.a.mmu.Peek(0x100+uint16(sp)+2+i*2+offset)) + uint16(t.a.mmu.Peek(0x100+uint16(sp)+3+i*2+offset))<<8
		b1 := val & 0xf000 >> 12
		b2 := val & 0x0f00 >> 8
		b3 := val & 0x00f0 >> 4
		b4 := val & 0x000f
		b := int32(b1*1000 + b2*100 + b3*10 + b4)
		num = num*10000 + b
	}
	num = num * sign
	return num, wlen
}
func (t *tracePascalPCode) getScalar(offset uint16) (uint16, uint16) {
	//_, sp := t.a.cpu.GetPCAndSP()
	return 0, 0
}
func (t *tracePascalPCode) getChar(offset uint16) (string, uint16) {
	//_, sp := t.a.cpu.GetPCAndSP()
	return "", 0
}
func (t *tracePascalPCode) getReal(offset uint16) (float32, uint16) {
	//_, sp := t.a.cpu.GetPCAndSP()
	return 0, 0
}
func (t *tracePascalPCode) getString(offset uint16) (string, uint16) {
	//_, sp := t.a.cpu.GetPCAndSP()
	return "", 0
}
