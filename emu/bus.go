package emu

// SMSBus adapts Memory and SMSIO into the go-chip-z80 Bus interface.
type SMSBus struct {
	mem *Memory
	io  *SMSIO
}

// NewSMSBus creates a new SMSBus bridging memory and I/O.
func NewSMSBus(mem *Memory, io *SMSIO) *SMSBus {
	return &SMSBus{mem: mem, io: io}
}

func (b *SMSBus) Fetch(addr uint16) uint8      { return b.mem.Get(addr) }
func (b *SMSBus) Read(addr uint16) uint8       { return b.mem.Get(addr) }
func (b *SMSBus) Write(addr uint16, val uint8) { b.mem.Set(addr, val) }
func (b *SMSBus) In(port uint16) uint8         { return b.io.In(uint8(port)) }
func (b *SMSBus) Out(port uint16, val uint8)   { b.io.Out(uint8(port), val) }
