package ppu

import "emulator/common"

type Ppu struct {
	ChrRom []uint8
	//from rom
	Palette [32]uint8
	//colors
	Ram [2048]uint8
	//ppu mem
	Oam [256]uint8
	//sprite state monitoring
	Mirror int

	AddrRegister addrReg
	OamAddr      PPU_OAM_ADDRESS
	OamData      PPU_OAM_DATA
	Status       PPU_STATUS_REGISTER
	Scroll       PPU_SCROLL_REGISTER
	//DataRegister    dataReg
	ControlRegister PPU_CONTROL

	buffer uint8
}

type PPU_OAM_ADDRESS uint8

func (reg *PPU_OAM_ADDRESS) WriteAddressOam(addr uint8) {
	*reg = PPU_OAM_ADDRESS(addr)
}

func (reg *PPU_OAM_ADDRESS) Increment() {
	*reg++
}

type PPU_OAM_DATA uint8

func (ppu *Ppu) WriteDataOam(data uint8) {
	ppu.Oam[ppu.OamAddr] = data
	ppu.OamAddr.Increment()
}

func (ppu *Ppu) ReadDataOamRegister() uint8 {
	return ppu.Oam[ppu.OamAddr]
}

type PPU_OAM_DMA uint8

func (ppu *Ppu) WriteOamDMA(addr uint8) {
	charRomaddr := uint16(0x100) * uint16(addr)

	for i := 0; i < 256; i++ {
		ppu.Oam[i] = ppu.ChrRom[charRomaddr]
		charRomaddr++
	}
}

//sus
func (ppu *Ppu) mirriorPPU(addr uint16) uint16 {
	if addr >= 0x3000 && addr <= 0x3eff {
		addr = addr & 0x2fff
	}
	mirror := ppu.Mirror
	switch {
	case mirror == common.HORIZONTAL:
		if addr >= 0x2000 && addr <= 0x2400 {
			return addr - 0x2000
		}
		if addr >= 0x2400 && addr <= 0x2800 {
			return addr - 0x2400
		}

		if addr >= 0x2800 && addr <= 0x2c00 {
			return (addr - 0x2800) + 0x400
		}
		if addr >= 0x2c00 && addr <= 0x3f00 {
			return (addr - 0x2c00) + 0x400
		}
	case mirror == common.VERTICAL:
		if addr >= 0x2000 && addr <= 0x2400 {
			return addr - 0x2000
		}
		if addr >= 0x2400 && addr <= 0x2800 {
			return addr - 0x2400 + 400
		}

		if addr >= 0x2800 && addr <= 0x2c00 {
			return (addr - 0x2800)
		}
		if addr >= 0x2c00 && addr <= 0x3f00 {
			return (addr - 0x2c00) + 0x400
		}
	}
	return 0
}

const (
	BASE_NAME_TABLE_ONE = iota
	BASE_NAME_TABLE_TWO
	VRAM_INCREMENT
	SPRITE_ADDRESS
	BACKGROUND_PATTERN
	SPRITE_SIZE
	PPU_MASTER
	GENERATE_NMI
	CHR_ROM_START = 0
	CHR_ROM_END   = 0x1FFF
	PPU_RAM_START = 0x2000
	PPU_RAM_END   = 0x3EFF
	PALETTE_START = 0x3F00
	PALETTE_END   = 0x3FFF
)

func (ppu *Ppu) ReadData() uint8 {
	addr := ppu.AddrRegister.Get()
	val := ppu.ControlRegister.ValueToIncrementBy()
	ppu.AddrRegister.Increment(val)

	switch {

	case addr >= PALETTE_START && addr <= PALETTE_END:
		return ppu.Palette[(addr - PALETTE_START)]
	case addr >= CHR_ROM_START && addr <= CHR_ROM_END:
		result := ppu.buffer
		ppu.buffer = ppu.ChrRom[(addr)]
		return result
	case addr >= PPU_RAM_START && addr <= PPU_RAM_END:
		result := ppu.buffer
		ppu.buffer = ppu.Ram[(ppu.mirriorPPU(addr))]
		return result
	}
	return 0
}

func (ppu *Ppu) WriteData(data uint8) {
	addr := ppu.AddrRegister.Get()

	switch {
	case addr >= PALETTE_START && addr <= PALETTE_END:
		ppu.Palette[(addr - PALETTE_START)] = data
	case addr >= PPU_RAM_START && addr <= PPU_RAM_END:
		ppu.Ram[(ppu.mirriorPPU(addr))] = data
	case addr >= CHR_ROM_START && addr <= CHR_ROM_END:
		ppu.ChrRom[(addr)] = data
	}

	val := ppu.ControlRegister.ValueToIncrementBy()
	ppu.AddrRegister.Increment(val)

}

type addrReg struct {
	values [2]uint8
	ptr    int
}

type PPU_MASK uint8

const (
	RED = iota
	BLUE
	GREEN
)

func (ctrl *PPU_MASK) Update(val uint8) {
	*ctrl = PPU_MASK(val)
}

func (ctrl *PPU_MASK) BackgroundRender() bool {
	return hasBit(uint8(*ctrl), 3)
}
func (ctrl *PPU_MASK) SpriteRender() bool {
	return hasBit(uint8(*ctrl), 4)
}

func (ctrl *PPU_MASK) BackgroundRenderTop() bool {
	return hasBit(uint8(*ctrl), 1)
}
func (ctrl *PPU_MASK) SpriteRenderTop() bool {
	return hasBit(uint8(*ctrl), 2)
}

func (ctrl *PPU_MASK) IsGreyScale() bool {
	return hasBit(uint8(*ctrl), 0)
}

func (ctrl *PPU_MASK) EmphasizeRed() bool {
	return hasBit(uint8(*ctrl), 5)
}
func (ctrl *PPU_MASK) EmphasizeGreen() bool {
	return hasBit(uint8(*ctrl), 6)
}

func (ctrl *PPU_MASK) EmphasizeBlue() bool {
	return hasBit(uint8(*ctrl), 7)
}

func (ctrl *PPU_MASK) EnableRendring() bool {
	if uint8(*ctrl) == 0x1e {
		return true
	}

	if uint8(*ctrl) == 0x00 {
		return false
	}
	return true
}

type PPU_CONTROL uint8

func (ctrl *PPU_CONTROL) ValueToIncrementBy() uint8 {
	if hasBit(uint8(*ctrl), VRAM_INCREMENT) {
		return 32
	} else {
		return 1
	}
}

func (ctrl *PPU_CONTROL) Update(val uint8) {
	*ctrl = PPU_CONTROL(val)
}

func NewPPU(rom []uint8, mirror int) *Ppu {
	ppu := &Ppu{
		Mirror: mirror,
		ChrRom: rom,
	}

	return ppu
}

type PPU_SCROLL_REGISTER struct {
	values [2]uint8
	ptr    int
}

func (reg *PPU_SCROLL_REGISTER) Update(value uint8) {
	reg.values[reg.ptr] = value
	reg.ptr++
	reg.ptr = (reg.ptr) % 2
}

type PPU_STATUS_REGISTER uint8

func (reg *PPU_STATUS_REGISTER) SetVBlank() {
	*reg = PPU_STATUS_REGISTER(setBit(uint8(*reg), 7))
}
func (reg *PPU_STATUS_REGISTER) ClearVBlank() {
	*reg = PPU_STATUS_REGISTER(clearBit(uint8(*reg), 7))
}

func (reg *PPU_STATUS_REGISTER) SetSpriteZero() {
	*reg = PPU_STATUS_REGISTER(setBit(uint8(*reg), 6))
}
func (reg *PPU_STATUS_REGISTER) ClearSpriteZero() {
	*reg = PPU_STATUS_REGISTER(clearBit(uint8(*reg), 6))
}

func (reg *PPU_STATUS_REGISTER) SetSpriteOverflow() {
	*reg = PPU_STATUS_REGISTER(setBit(uint8(*reg), 5))
}
func (reg *PPU_STATUS_REGISTER) ClearSpriteOverflow() {
	*reg = PPU_STATUS_REGISTER(clearBit(uint8(*reg), 5))
}
func (reg *PPU_STATUS_REGISTER) InVBlank() bool {
	return hasBit(uint8(*reg), 7)
}

func (ppu *Ppu) ReadStatus() uint8 {
	temp := uint8(ppu.Status)
	ppu.Status.ClearVBlank()
	ppu.Scroll.ptr = 0
	ppu.AddrRegister.ptr = 0
	return temp
}

func (reg *addrReg) Update(value uint8) {
	reg.values[reg.ptr] = value
	reg.ptr++
	reg.ptr = (reg.ptr) % 2
	if reg.Get() > 0x3fff {
		reg.Set(reg.Get() & 0x3fff)
		//mirror back to ppu registers
	}
}

func (reg *addrReg) Get() uint16 {
	return (uint16(reg.values[0]))<<8 | (uint16(reg.values[1]))
}

func (reg *addrReg) Set(val uint16) {
	hi := uint8(val >> 8)
	low := uint8(val & 0x00FF)
	reg.values[0] = hi
	reg.values[1] = low
}

func (reg *addrReg) Increment(val uint8) {
	reg.Set(reg.Get() + uint16(val))
}

//this is a duplicate remove later
func hasBit(n uint8, pos int) bool {
	val := n & (1 << pos)
	return (val > 0)
}

func setBit(num uint8, pos int) uint8 {

	num |= (uint8(1) << pos)
	return num
}

func clearBit(n uint8, pos int) uint8 {
	var mask uint8 = ^(1 << pos)
	n &= mask
	return n
}
