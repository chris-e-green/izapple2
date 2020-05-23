package apple2

import (
	"encoding/binary"
	"io"
)

// See https://fabiensanglard.net/fd_proxy/prince_of_persia/Inside%20the%20Apple%20IIe.pdf
// See https://i.stack.imgur.com/yn21s.gif

type memoryManager struct {
	apple2 *Apple2

	// Main RAM area: 0x0000 to 0xbfff
	physicalMainRAM    *memoryRange // 0x0000 to 0xbfff, Up to 48 Kb
	physicalMainRAMAlt *memoryRange // 0x0000 to 0xbfff, Up to 48 Kb. Additional

	// Slots area: 0xc000 to 0xcfff
	cardsROM      [8]memoryHandler //0xcs00 to 0xcsff. 256 bytes for each card
	cardsROMExtra [8]memoryHandler // 0xc800 to 0xcfff. 2048 bytes for each card
	physicalROMe  memoryHandler    // 0xc100 to 0xcfff, Zero or 4kb in the Apple2e

	// Upper area: 0xd000 to 0xffff
	physicalROM     [4]memoryHandler // 0xd000 to 0xffff, 12 Kb. Up to four banks
	physicalDRAM    []memoryHandler  // 0xd000 to 0xdfff, 4KB. Up to 8 banks.
	physicalDAltRAM []memoryHandler  // 0xd000 to 0xdfff, 4KB. Up to 8 banks.
	physicalEFRAM   []memoryHandler  // 0xe000 to 0xffff, 8KB. Up to 8 banks.

	// Configuration switches, Language cards
	lcSelectedBlock uint8 // Language card block selected. Usually, allways 0. But Saturn has 8
	lcActiveRead    bool  // Upper RAM active for read
	lcActiveWrite   bool  // Upper RAM active for read
	lcAltBank       bool  // Alternate

	// Configuration switches, Apple //e
	altZeroPage           bool  // Use extra RAM from 0x0000 to 0x01ff. And additional language card block
	altMainRAMActiveRead  bool  // Use extra RAM from 0x0200 to 0xbfff for read
	altMainRAMActiveWrite bool  // Use extra RAM from 0x0200 to 0xbfff for write
	store80Active         bool  // Special pagination for text and graphics areas
	slotC3ROMActive       bool  // Apple2e slot 3  ROM shadow
	intCxROMActive        bool  // Apple2e slots internal ROM shadow
	activeSlot            uint8 // Active slot owner of 0xc800 to 0xcfff

	// Configuration switches, Base64A
	romPage uint8 // Active ROM page
}

const (
	ioC8Off                uint16 = 0xcfff
	addressLimitZero       uint16 = 0x01ff
	addressStartText       uint16 = 0x0400
	addressLimitText       uint16 = 0x07ff
	addressStartHgr        uint16 = 0x2000
	addressLimitHgr        uint16 = 0x3fff
	addressLimitMainRAM    uint16 = 0xbfff
	addressLimitIO         uint16 = 0xc0ff
	addressLimitSlots      uint16 = 0xc7ff
	addressLimitSlotsExtra uint16 = 0xcfff
	addressLimitDArea      uint16 = 0xdfff
)

type memoryHandler interface {
	peek(uint16) uint8
	poke(uint16, uint8)
}

func newMemoryManager(a *Apple2) *memoryManager {
	var mmu memoryManager
	mmu.apple2 = a
	mmu.physicalMainRAM = newMemoryRange(0, make([]uint8, 0xc000))

	mmu.intCxROMActive = true

	return &mmu
}

func (mmu *memoryManager) accessCArea(address uint16) memoryHandler {
	if mmu.intCxROMActive {
		return mmu.physicalROMe
	}
	// First slot area
	if address <= addressLimitSlots {
		slot := uint8((address >> 8) & 0x07)
		mmu.activeSlot = slot
		if !mmu.slotC3ROMActive && (slot == 3) {
			return mmu.physicalROMe
		}
		return mmu.cardsROM[slot]
	}
	// Extra slot area
	if address == ioC8Off {
		// Reset extra slot area owner
		mmu.activeSlot = 0
	}

	if !mmu.slotC3ROMActive && (mmu.activeSlot == 3) {
		return mmu.physicalROMe
	}
	return mmu.cardsROMExtra[mmu.activeSlot]
}

func (mmu *memoryManager) accessLCArea(address uint16) memoryHandler {
	block := mmu.lcSelectedBlock
	if mmu.altZeroPage {
		block = 1
	}
	if address <= addressLimitDArea {
		if mmu.lcAltBank {
			return mmu.physicalDAltRAM[block]
		}
		return mmu.physicalDRAM[block]
	}
	return mmu.physicalEFRAM[block]
}

func (mmu *memoryManager) getPhysicalMainRAM(alt bool) *memoryRange {
	if alt {
		return mmu.physicalMainRAMAlt
	}
	return mmu.physicalMainRAM
}

func (mmu *memoryManager) accessRead(address uint16) memoryHandler {
	if address <= addressLimitZero {
		return mmu.getPhysicalMainRAM(mmu.altZeroPage)
	}
	if mmu.store80Active && address <= addressLimitHgr {
		altPage := mmu.apple2.io.isSoftSwitchActive(ioFlagSecondPage)
		if address >= addressStartText && address <= addressLimitText {
			return mmu.getPhysicalMainRAM(altPage)
		}
		hires := mmu.apple2.io.isSoftSwitchActive(ioFlagHiRes)
		if hires && address >= addressStartHgr && address <= addressLimitHgr {
			return mmu.getPhysicalMainRAM(altPage)
		}
	}
	if address <= addressLimitMainRAM {
		return mmu.getPhysicalMainRAM(mmu.altMainRAMActiveRead)
	}
	if address <= addressLimitIO {
		return mmu.apple2.io
	}
	if address <= addressLimitSlotsExtra {
		return mmu.accessCArea(address)
	}
	if mmu.lcActiveRead {
		return mmu.accessLCArea(address)
	}
	return mmu.physicalROM[mmu.romPage]
}

func (mmu *memoryManager) accessWrite(address uint16) memoryHandler {
	if address <= addressLimitZero {
		return mmu.getPhysicalMainRAM(mmu.altZeroPage)
	}
	if address <= addressLimitHgr && mmu.store80Active {
		altPage := mmu.apple2.io.isSoftSwitchActive(ioFlagSecondPage)
		if address >= addressStartText && address <= addressLimitText {
			return mmu.getPhysicalMainRAM(altPage)
		}
		hires := mmu.apple2.io.isSoftSwitchActive(ioFlagHiRes)
		if hires && address >= addressStartHgr && address <= addressLimitHgr {
			return mmu.getPhysicalMainRAM(altPage)
		}
	}
	if address <= addressLimitMainRAM {
		return mmu.getPhysicalMainRAM(mmu.altMainRAMActiveWrite)
	}
	if address <= addressLimitIO {
		return mmu.apple2.io
	}
	if address <= addressLimitSlotsExtra {
		return mmu.accessCArea(address)
	}
	if mmu.lcActiveWrite {
		return mmu.accessLCArea(address)
	}
	return mmu.physicalROM[mmu.romPage]
}

// Peek returns the data on the given address
func (mmu *memoryManager) Peek(address uint16) uint8 {
	mh := mmu.accessRead(address)
	if mh == nil {
		//fmt.Printf("Reading void addressing 0x%x\n", address)
		return 0xf4 // Or some random number
	}
	return mh.peek(address)
}

// Poke sets the data at the given address
func (mmu *memoryManager) Poke(address uint16, value uint8) {
	mh := mmu.accessWrite(address)
	if mh == nil {
		//fmt.Printf("Writing to void addressing 0x%x\n", address)
		return
	}
	mh.poke(address, value)
}

// Memory initialization
func (mmu *memoryManager) setCardROM(slot int, mh memoryHandler) {
	mmu.cardsROM[slot] = mh
}

func (mmu *memoryManager) setCardROMExtra(slot int, mh memoryHandler) {
	mmu.cardsROMExtra[slot] = mh
}

func (mmu *memoryManager) initLanguageRAM(groups int) {
	mmu.physicalDRAM = make([]memoryHandler, groups)
	mmu.physicalDAltRAM = make([]memoryHandler, groups)
	mmu.physicalEFRAM = make([]memoryHandler, groups)
	for i := 0; i < groups; i++ {
		mmu.physicalDRAM[i] = newMemoryRange(0xd000, make([]uint8, 0x1000))
		mmu.physicalDAltRAM[i] = newMemoryRange(0xd000, make([]uint8, 0x1000))
		mmu.physicalEFRAM[i] = newMemoryRange(0xe000, make([]uint8, 0x2000))
	}
}

func (mmu *memoryManager) InitRAMalt() {
	mmu.physicalMainRAMAlt = newMemoryRange(0, make([]uint8, 0xc000))
}

// Memory configuration
func (mmu *memoryManager) setActiveROMPage(page uint8) {
	mmu.romPage = page
}

func (mmu *memoryManager) getActiveROMPage() uint8 {
	return mmu.romPage
}

func (mmu *memoryManager) setLanguageRAM(readActive bool, writeActive bool, altBank bool) {
	mmu.lcActiveRead = readActive
	mmu.lcActiveWrite = writeActive
	mmu.lcAltBank = altBank
}

func (mmu *memoryManager) setLanguageRAMBlock(block uint8) {
	block = block % uint8(len(mmu.physicalDRAM))
	mmu.lcSelectedBlock = block
}

// TODO: complete save and load
func (mmu *memoryManager) save(w io.Writer) error {
	err := mmu.physicalMainRAM.save(w)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, mmu.activeSlot)
	if err != nil {
		return err
	}
	return nil
}

func (mmu *memoryManager) load(r io.Reader) error {
	err := mmu.physicalMainRAM.load(r)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &mmu.activeSlot)
	if err != nil {
		return err
	}
	//	mmu.activateCardRomExtra(mmu.activeSlot)

	return nil
}
