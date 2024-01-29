package izapple2

import (
	"fmt"
	"os"
	"path/filepath"
)

/*
Apple II DAN ][ CONTROLLER CARD.]

See:
	https://github.com/profdc9/Apple2Card
	https://www.applefritter.com/content/dan-sd-card-disk-controller

*/

// CardDan2Controller represents a Dan ][ controller card
type CardDan2Controller struct {
	cardBase

	commandBuffer  []uint8
	responseBuffer []uint8

	receivingWiteBuffer bool
	writeBuffer         []uint8
	commitWrite         func([]uint8) error

	portB uint8
	portC uint8

	slotA *cardDan2ControllerSlot
	slotB *cardDan2ControllerSlot
}

type cardDan2ControllerSlot struct {
	path     string
	fileNo   uint8
	fileName string
}

func newCardDan2ControllerBuilder() *cardBuilder {
	return &cardBuilder{
		name:        "Dan ][ Controller card",
		description: "Apple II Peripheral Card that Interfaces to a ATMEGA328P for SD card storage.",
		defaultParams: &[]paramSpec{
			{"slot1", "Image in slot 1. File for raw device, folder for fs mode using files as BLKDEV0x.PO", ""},
			{"slot1file", "Device selected in slot 1: 0 for raw device, 1 to 9 for file number", "0"},
			{"slot2", "Image in slot 2. File for raw device, folder for fs mode using files as BLKDEV0x.PO", ""},
			{"slot2file", "Device selected in slot 2: 0 for raw device, 1 to 9 for file number", "0"},
		},
		buildFunc: func(params map[string]string) (Card, error) {
			var c CardDan2Controller
			c.responseBuffer = make([]uint8, 0, 1000)

			c.slotA = &cardDan2ControllerSlot{}
			c.slotA.path = params["slot1"]
			num, _ := paramsGetInt(params, "slot1file")
			c.slotA.fileNo = uint8(num)
			c.slotA.initializeDrive()

			c.slotB = &cardDan2ControllerSlot{}
			c.slotB.path = params["slot2"]
			num, _ = paramsGetInt(params, "slot2file")
			c.slotB.fileNo = uint8(num)
			c.slotB.initializeDrive()

			err := c.loadRomFromResource("<internal>/Apple2CardFirmware.bin")
			if err != nil {
				return nil, err
			}

			return &c, nil
		},
	}
}

func (c *CardDan2Controller) assign(a *Apple2, slot int) {
	c.addCardSoftSwitches(func(address uint8, data uint8, write bool) uint8 {
		address &= 0x03 // only A0 and A1 are connected
		if write {
			c.write(address, data)
			return 0
		} else {
			return c.read(address)
		}
	}, "DAN2CONTROLLER")

	c.cardBase.assign(a, slot)
}

func (c *CardDan2Controller) write(address uint8, data uint8) {
	switch address {
	case 0: // Port A
		if c.receivingWiteBuffer {
			c.writeBuffer = append(c.writeBuffer, data)
			if len(c.writeBuffer) == 512 {
				c.commitWrite(c.writeBuffer)
			}
		} else if c.commandBuffer == nil {
			if data == 0xac {
				c.commandBuffer = make([]uint8, 0)
			}
		} else {
			c.commandBuffer = append(c.commandBuffer, data)
			c.processCommand()
		}
	case 3: // Control
		if data&0x80 == 0 {
			bit := (data >> 1) & 0x08
			if data&1 == 0 {
				// Reset bit
				c.portC &^= uint8(1) << bit
			} else {
				// Set bit
				c.portC |= uint8(1) << bit
			}
			c.romCsxx.setPage((c.portC & 0x07) | ((c.portB << 4) & 0xf0))
		} else {
			if data != 0xfa {
				c.tracef("Not supported status %v, it must be 0xfa\n", data)
			}
			/* Sets the 8255 with status 0xfa, 1111_1010:
			1:  set mode
			11: port A mode 2
			1:  port A input
			1:  port C(upper) input
			0:  port B mode 0
			1:  port B input
			0:  port C(lower) output
			*/

		}
	}
}

func (c *CardDan2Controller) read(address uint8) uint8 {
	switch address {
	case 0: // Port A
		if len(c.responseBuffer) > 0 {
			value := c.responseBuffer[0]
			c.responseBuffer = c.responseBuffer[1:]
			return value
		}
		return 0
	case 2: // Port C
		portC := uint8(0x80) // bit 7-nOBF is always 1, the output buffer is never full
		if len(c.responseBuffer) > 0 {
			portC |= 0x20 // bit 5-niBF is 1 if the input buffer has data
		}
		return portC
	}

	return 0
}

func (s *cardDan2ControllerSlot) blockPosition(unit uint8, block uint16) int64 {
	if s.fileNo == 0 {
		// Raw device
		return 512 * (int64(block) + (int64(unit&0x0f) << 12))
	} else {
		// File device
		return 512 * int64(block)
	}
}

func (s *cardDan2ControllerSlot) status(unit uint8) error {
	file, err := os.OpenFile(s.fileName, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func (s *cardDan2ControllerSlot) readBlock(unit uint8, block uint16) ([]uint8, error) {
	file, err := os.OpenFile(s.fileName, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	position := s.blockPosition(unit, block)
	buffer := make([]uint8, 512)
	_, err = file.ReadAt(buffer, position)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (s *cardDan2ControllerSlot) writeBlock(unit uint8, block uint16, data []uint8) error {
	file, err := os.OpenFile(s.fileName, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	position := s.blockPosition(unit, block)
	_, err = file.WriteAt(data, position)
	if err != nil {
		return err
	}
	return nil
}

func (s *cardDan2ControllerSlot) initializeDrive() {
	if s.fileNo == 255 {
		s.fileNo = 0 // Wide raw not supported, changed to raw
	}
	if s.fileNo == 0 {
		// Raw device
		s.fileName = s.path
	} else {
		s.fileName = filepath.Join(s.path, fmt.Sprintf("BLKDEV%02X.PO", s.fileNo))
	}
}

func (c *CardDan2Controller) selectSlot(unit uint8) *cardDan2ControllerSlot {
	if unit&0x80 == 0 {
		return c.slotA
	} else {
		return c.slotB
	}
}

func (c *CardDan2Controller) processCommand() {
	// See : Apple2Arduino.ino::do_command()
	command := c.commandBuffer[0]
	switch command {
	case 0, 3: // Status and format
		if len(c.commandBuffer) == 6 {
			unit, _, _ := c.getUnitBufBlk()
			slot := c.selectSlot(unit)
			err := slot.status(unit)
			if err != nil {
				c.tracef("Error status : %v\n", err)
				c.sendResponseCode(0x28)
			} else {
				c.sendResponseCode(0x00)
			}

			if command == 0 {
				c.tracef("0-Status unit $%02x\n", unit)
			} else {
				c.tracef("3-Format unit $%02x\n", unit)
			}
			c.commandBuffer = nil
		}

	case 1: // Read block
		if len(c.commandBuffer) == 6 {
			unit, buffer, block := c.getUnitBufBlk()
			c.tracef("1-Read unit $%02x, buffer $%x, block %v\n", unit, buffer, block)

			slot := c.selectSlot(unit)
			data, err := slot.readBlock(unit, block)
			if err != nil {
				c.tracef("Error reading block : %v\n", err)
				c.sendResponseCode(0x28)
			} else {
				c.sendResponse(data...)
			}
			c.commandBuffer = nil
		}

	case 2: // Write block
		if len(c.commandBuffer) == 6 {
			unit, buffer, block := c.getUnitBufBlk()
			c.tracef("2-Write unit $%02x, buffer $%x, block %v\n", unit, buffer, block)

			c.receivingWiteBuffer = true
			c.writeBuffer = make([]uint8, 0, 512)
			c.commitWrite = func(data []uint8) error {
				slot := c.selectSlot(unit)
				err := slot.writeBlock(unit, block, c.writeBuffer)
				if err != nil {
					c.tracef("Error writing block : %v\n", err)
				}
				c.receivingWiteBuffer = false
				c.writeBuffer = nil
				c.commitWrite = nil
				return nil
			}
			c.sendResponseCode(0x00)

			c.commandBuffer = nil
		}

	case 5: // Get volume
		if len(c.commandBuffer) == 6 {
			c.tracef("5-Get Volume\n")

			c.sendResponse(c.slotA.fileNo, c.slotB.fileNo)
			c.commandBuffer = nil
		}

	case 4, 6, 7: // Set volume
		if len(c.commandBuffer) == 6 {
			_, _, block := c.getUnitBufBlk()
			c.slotA.fileNo = uint8(block & 0xff)
			c.slotA.initializeDrive()
			c.slotA.fileNo = uint8((block >> 8) & 0xff)
			c.slotA.initializeDrive()

			c.tracef("%v-Set Volume %v and %v\n",
				command, c.slotA.fileNo, c.slotA.fileNo)

			if command == 4 {
				c.responseBuffer = append(c.responseBuffer, 0x00) // Success code
			} else {
				c.sendResponse()
				// command 6 writes eeprom, not emulated
			}
			c.commandBuffer = nil
		}

	case 13 + 128, 32 + 128: // Read bootblock
		if len(c.commandBuffer) == 6 {
			c.tracef("ac-Read bootblock\n")
			c.sendResponse(PROGMEM[:]...)
			c.commandBuffer = nil
		}

	default: // Unknown command
		c.tracef("Unknown command %v\n", command)
		c.sendResponseCode(0x27)
	}
}

func (c *CardDan2Controller) sendResponseCode(code uint8) {
	c.responseBuffer = append(c.responseBuffer, code)
}

func (c *CardDan2Controller) sendResponse(response ...uint8) {
	c.responseBuffer = append(c.responseBuffer, 0x00) // Success code
	c.responseBuffer = append(c.responseBuffer, response...)
	rest := 512 - len(response)
	if rest > 0 {
		c.responseBuffer = append(c.responseBuffer, make([]uint8, rest)...)
	}
}

func (c *CardDan2Controller) getUnitBufBlk() (uint8, uint16, uint16) {
	return c.commandBuffer[1],
		uint16(c.commandBuffer[2]) + uint16(c.commandBuffer[3])<<8,
		uint16(c.commandBuffer[4]) + uint16(c.commandBuffer[5])<<8
}

var PROGMEM = [512]uint8{
	0xea, 0xa9, 0x20, 0x85, 0xf0, 0xa9, 0x60, 0x85, 0xf3, 0xa5, 0x43, 0x4a,
	0x4a, 0x4a, 0x4a, 0x29, 0x07, 0x09, 0xc0, 0x85, 0xf2, 0xa0, 0x00, 0x84,
	0xf1, 0x88, 0xb1, 0xf1, 0x85, 0xf1, 0x20, 0x93, 0xfe, 0x20, 0x89, 0xfe,
	0x20, 0x58, 0xfc, 0x20, 0xa2, 0x09, 0xa9, 0x00, 0x85, 0x25, 0x20, 0x22,
	0xfc, 0xa5, 0x25, 0x85, 0xf5, 0x85, 0xf6, 0x20, 0x90, 0x09, 0xa9, 0x00,
	0x85, 0x24, 0xa5, 0x25, 0x20, 0xe3, 0xfd, 0xe6, 0x24, 0x20, 0x7a, 0x09,
	0x20, 0x04, 0x09, 0xa9, 0x14, 0x85, 0x24, 0xa5, 0x25, 0x20, 0xe3, 0xfd,
	0xe6, 0x24, 0xa5, 0x43, 0x09, 0x80, 0x85, 0x43, 0x20, 0x7a, 0x09, 0x20,
	0x04, 0x09, 0xa5, 0x43, 0x29, 0x7f, 0x85, 0x43, 0xe6, 0x25, 0xa5, 0x25,
	0xc9, 0x10, 0x90, 0xbe, 0xa9, 0x00, 0x85, 0x24, 0xa9, 0x12, 0x85, 0x25,
	0x20, 0x22, 0xfc, 0xa2, 0x14, 0x20, 0x66, 0x09, 0x20, 0x61, 0x09, 0xa9,
	0x0a, 0x85, 0x24, 0xa5, 0xf7, 0x20, 0xf8, 0x08, 0xa9, 0x14, 0x85, 0x24,
	0x20, 0x5c, 0x09, 0xa9, 0x1e, 0x85, 0x24, 0xa5, 0xf8, 0x20, 0xf8, 0x08,
	0xa9, 0x0a, 0x85, 0x24, 0x20, 0xca, 0x08, 0x85, 0xf5, 0x20, 0xf8, 0x08,
	0xa9, 0x1e, 0x85, 0x24, 0x20, 0xca, 0x08, 0x85, 0xf6, 0x20, 0xf8, 0x08,
	0x20, 0x8c, 0x09, 0x4c, 0xb7, 0x09, 0xa5, 0xf7, 0x85, 0xf5, 0xa5, 0xf8,
	0x85, 0xf6, 0x20, 0x90, 0x09, 0x68, 0x68, 0x4c, 0xb7, 0x09, 0x20, 0x0c,
	0xfd, 0xc9, 0x9b, 0xf0, 0xe9, 0xc9, 0xa1, 0xf0, 0x20, 0xc9, 0xe1, 0x90,
	0x03, 0x38, 0xe9, 0x20, 0xc9, 0xc1, 0x90, 0x04, 0xc9, 0xc7, 0x90, 0x0b,
	0xc9, 0xb0, 0x90, 0xe2, 0xc9, 0xba, 0xb0, 0xde, 0x29, 0x0f, 0x60, 0x38,
	0xe9, 0x07, 0x29, 0x0f, 0x60, 0xa9, 0xff, 0x60, 0xc9, 0xff, 0xf0, 0x03,
	0x4c, 0xe3, 0xfd, 0xa9, 0xa1, 0x4c, 0xed, 0xfd, 0xa2, 0x00, 0xb0, 0x25,
	0xad, 0x05, 0x10, 0x30, 0x20, 0xad, 0x04, 0x10, 0x29, 0xf0, 0xc9, 0xf0,
	0xd0, 0x17, 0xad, 0x04, 0x10, 0x29, 0x0f, 0xf0, 0x10, 0x85, 0xf9, 0xbd,
	0x05, 0x10, 0x09, 0x80, 0x20, 0xed, 0xfd, 0xe8, 0xe4, 0xf9, 0xd0, 0xf3,
	0x60, 0x4c, 0x66, 0x09, 0xbc, 0xce, 0xcf, 0xa0, 0xd6, 0xcf, 0xcc, 0xd5,
	0xcd, 0xc5, 0xbe, 0x00, 0xc3, 0xc1, 0xd2, 0xc4, 0xa0, 0xb1, 0xba, 0x00,
	0xc4, 0xc1, 0xce, 0xa0, 0xdd, 0xdb, 0xa0, 0xd6, 0xcf, 0xcc, 0xd5, 0xcd,
	0xc5, 0xa0, 0xd3, 0xc5, 0xcc, 0xc5, 0xc3, 0xd4, 0xcf, 0xd2, 0x8d, 0x00,
	0xa9, 0xb2, 0x8d, 0x41, 0x09, 0xa2, 0x0c, 0x4c, 0x66, 0x09, 0xbd, 0x30,
	0x09, 0xf0, 0x0e, 0x20, 0xed, 0xfd, 0xe8, 0xd0, 0xf5, 0xa9, 0x00, 0x85,
	0x44, 0xa9, 0x10, 0x85, 0x45, 0x60, 0xa9, 0x01, 0x85, 0x42, 0x20, 0x71,
	0x09, 0xa9, 0x02, 0x85, 0x46, 0xa9, 0x00, 0x85, 0x47, 0x4c, 0xf0, 0x00,
	0xa9, 0x07, 0xd0, 0x02, 0xa9, 0x06, 0x85, 0x42, 0x20, 0x71, 0x09, 0xa5,
	0xf5, 0x85, 0x46, 0xa5, 0xf6, 0x85, 0x47, 0x4c, 0xf0, 0x00, 0xa9, 0x05,
	0x85, 0x42, 0x20, 0x71, 0x09, 0x20, 0xf0, 0x00, 0xad, 0x00, 0x10, 0x85,
	0xf7, 0xad, 0x01, 0x10, 0x85, 0xf8, 0x60, 0xa9, 0x00, 0x85, 0xf1, 0x6c,
	0xf1, 0x00,
}
