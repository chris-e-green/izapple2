package izapple2

import (
	"errors"
	"math/rand"

	"github.com/ivanizag/izapple2/component"
	"github.com/ivanizag/izapple2/storage"
)

type cardDisk2SequencerDrive struct {
	data                *storage.FileWoz
	enabled             bool
	writeProtected      bool
	currentQuarterTrack int

	position    uint32 // Current position on the track
	positionMax uint32 // As tracks may have different lengths position is related of positionMax of the las track

	mc3470Buffer uint8 // Four bit buffer to detect weak bits and to add latency
}

func (d *cardDisk2SequencerDrive) insertDiskette(filename string) error {
	data, writeable, err := storage.LoadResource(filename)
	if err != nil {
		return err
	}
	f, err := storage.NewFileWoz(data)
	if err != nil {
		return err
	}

	// Discard not supported features
	if f.Info.DiskType != 1 {
		return errors.New("Only 5.25 disks are supported")
	}
	if f.Info.BootSectorFormat == 2 { // Info not available in WOZ 1.0
		return errors.New("Woz 13 sector disks are not supported")
	}

	d.data = f
	d.writeProtected = !writeable

	d.mc3470Buffer = 0xf // Test with the buffer full REMOVE

	return nil
}

func (d *cardDisk2SequencerDrive) enable(enabled bool) {
	d.enabled = enabled
}

func (d *cardDisk2SequencerDrive) moveHead(q0, q1, q2, q3 bool) {
	if !d.enabled {
		return
	}

	phases := component.PinsToByte([8]bool{
		q0, q1, q2, q3,
		false, false, false, false,
	})
	d.currentQuarterTrack = moveDriveStepper(phases, d.currentQuarterTrack)
}

func (d *cardDisk2SequencerDrive) readPulse() bool {
	if !d.enabled || d.data == nil {
		return false
	}

	// Get next bit taking into account the MC3470 latency and weak bits
	var fluxBit uint8
	fluxBit, d.position, d.positionMax = d.data.GetNextBitAndPosition(
		d.position,
		d.positionMax,
		d.currentQuarterTrack)
	d.mc3470Buffer = (d.mc3470Buffer<<1 + fluxBit) & 0x0f
	bit := ((d.mc3470Buffer >> 1) & 0x1) != 0 // Use the previous to last bit to add latency
	if d.mc3470Buffer == 0 && rand.Intn(100) < 3 {
		// Four consecutive zeros. It'a a fake bit.
		// Output a random value. 70% zero, 30% one
		bit = true
	}

	return bit
}

func (d *cardDisk2SequencerDrive) writePulse(value uint8) {
	panic("Write not implemented on woz disk implementation")
}
