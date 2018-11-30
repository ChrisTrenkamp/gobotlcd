package gobotlcd

import (
	"fmt"
	"time"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/i2c"
)

const (
	// command modes
	lcdClearDisplay   byte = 0x01
	lcdReturnHome     byte = 0x02
	lcdEntryModeSet   byte = 0x04
	lcdDisplayControl byte = 0x08
	lcdCursorShift    byte = 0x10
	lcdFunctionSet    byte = 0x20
	lcdSetCGramAddr   byte = 0x40
	lcdSetDDRAMAddr   byte = 0x80

	// flags for display entry mode
	lcdEntryLeft           byte = 0x02
	lcdEntryShiftIncrement byte = 0x01
	lcdEntryShiftDecrement byte = 0x00

	// flags for display on/off control
	lcdDisplayOn byte = 0x04
	lcdCursorOn  byte = 0x02
	lcdCursorOff byte = 0x00
	lcdBlinkOn   byte = 0x01
	lcdBlinkOff  byte = 0x00

	// flags for display/cursor shift
	lcdDisplayMove byte = 0x08
	lcdMoveRight   byte = 0x04
	lcdMoveLeft    byte = 0x00

	// flags for function set
	lcd8BitMode byte = 0x10
	lcd4BitMode byte = 0x00
	lcd2Line    byte = 0x08
	lcd1Line    byte = 0x00
	lcd5x10Dots      = 0x04
	lcd5x8Dots       = 0x00

	// flags for backlight control
	lcdBacklight   byte = 0x08
	lcdNoBacklight byte = 0x00

	en byte = 0x04 // Enable bit
	rs byte = 0x01 // Register select bit
)

//DotSize reprexents either a 5x8 or 5x10 dot mode for the lcd
type DotSize byte

const (
	//DotSize5x10 is used to connect to a 5x10 dot size LCD
	DotSize5x10 DotSize = lcd5x10Dots
	//DotSize5x8 is used to connect to a 5x8 dot size LCD
	DotSize5x8 DotSize = lcd5x8Dots
)

//LiquidCrystalLCD controls a Liquid Crystal LCD with an I2C connection.
type LiquidCrystalLCD struct {
	name       string
	displFn    byte
	displCntrl byte
	displMode  byte
	cols       byte
	rows       byte
	backLight  byte
	connector  i2c.Connector
	connection i2c.Connection
	i2c.Config
}

//NewLiquidCrystalLCD connects to an LCD with the given I2C connection, the given row
//and column size, and the given dot size.
func NewLiquidCrystalLCD(a i2c.Connector, cols, rows byte, dotSize DotSize, options ...func(i2c.Config)) *LiquidCrystalLCD {
	ret := &LiquidCrystalLCD{
		name:      gobot.DefaultName("LiquidCrystalLCD"),
		displFn:   lcd4BitMode | lcd1Line | lcd5x8Dots,
		backLight: lcdNoBacklight,
		cols:      cols,
		rows:      rows,
		connector: a,
		Config:    i2c.NewConfig(),
	}

	if ret.rows > 1 {
		ret.displFn |= lcd2Line
	}

	if dotSize != DotSize5x8 && rows == 1 {
		ret.displFn |= lcd5x10Dots
	}

	for _, option := range options {
		option(ret)
	}

	return ret
}

// Gobot driver interface methods

// SetName sets the label for the Driver
func (lcd *LiquidCrystalLCD) SetName(name string) {
	lcd.name = name
}

// Name returns the label for the Driver
func (lcd *LiquidCrystalLCD) Name() string {
	return lcd.name
}

// Connection returns the Connection associated with the Driver
func (lcd *LiquidCrystalLCD) Connection() gobot.Connection {
	return lcd.connection.(gobot.Connection)
}

// Start initiates the Driver
func (lcd *LiquidCrystalLCD) Start() (err error) {
	return lcd.init()
}

// Halt terminates the Driver
func (lcd *LiquidCrystalLCD) Halt() (err error) {
	gatherErrs := func(e error) {
		if e != nil && err != nil {
			err = fmt.Errorf("%v\n%v", err, e)
		} else if e != nil && err == nil {
			err = fmt.Errorf("%v", e)
		}
	}

	gatherErrs(lcd.Clear())
	gatherErrs(lcd.BacklightOff())
	gatherErrs(lcd.CursorOff())
	gatherErrs(lcd.DisplayOff())
	return
}

// end Gobot driver interface methods

func (lcd *LiquidCrystalLCD) write(val byte) error {
	return lcd.connection.WriteByte(val)
}

func (lcd *LiquidCrystalLCD) expandWrite(val byte) error {
	return lcd.write(val | lcd.backLight)
}

func (lcd *LiquidCrystalLCD) pulseEnable(val byte) error {
	if err := lcd.expandWrite(val | en); err != nil {
		return err
	}
	time.Sleep(time.Millisecond)

	if err := lcd.expandWrite(val & ^en); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	return nil
}

func (lcd *LiquidCrystalLCD) write4bits(val byte) error {
	if err := lcd.expandWrite(val); err != nil {
		return err
	}

	return lcd.pulseEnable(val)
}

func (lcd *LiquidCrystalLCD) send(val, mode byte) error {
	high := val & byte(0xF0)
	low := (val << 4) & 0xF0

	if err := lcd.write4bits(high | mode); err != nil {
		return err
	}

	return lcd.write4bits(low | mode)
}

func (lcd *LiquidCrystalLCD) command(val byte) error {
	return lcd.send(val, 0)
}

func (lcd *LiquidCrystalLCD) init() (err error) {
	bus := lcd.GetBusOrDefault(1)
	address := lcd.GetAddressOrDefault(0x27)

	lcd.connection, err = lcd.connector.GetConnection(address, bus)
	if err != nil {
		return err
	}

	// LCD requires 40 ms after power-on before receiving commands
	time.Sleep(50 * time.Microsecond)

	if err = lcd.write(lcd.backLight); err != nil {
		return err
	}

	time.Sleep(time.Millisecond)

	if err = lcd.init4BitMode(); err != nil {
		return err
	}

	if err = lcd.command(lcdFunctionSet | lcd.displFn); err != nil {
		return err
	}

	lcd.displCntrl = lcdDisplayOn | lcdCursorOff | lcdBlinkOff

	if err = lcd.DisplayOn(); err != nil {
		return err
	}

	if err = lcd.Clear(); err != nil {
		return err
	}

	lcd.displMode = lcdEntryLeft | lcdEntryShiftDecrement

	if err = lcd.command(lcdEntryModeSet | lcd.displMode); err != nil {
		return err
	}

	return lcd.Home()
}

func (lcd *LiquidCrystalLCD) init4BitMode() error {
	if err := lcd.write4bits(0x03 << 4); err != nil {
		return err
	}
	time.Sleep(45 * time.Millisecond)

	if err := lcd.write4bits(0x03 << 4); err != nil {
		return err
	}
	time.Sleep(45 * time.Millisecond)

	if err := lcd.write4bits(0x03 << 4); err != nil {
		return err
	}
	time.Sleep(150 * time.Microsecond)

	return lcd.write4bits(0x02 << 4)
}

//Clear wipes all text from the screen and positions the cursor at the top-left
func (lcd *LiquidCrystalLCD) Clear() error {
	if err := lcd.command(lcdClearDisplay); err != nil {
		return err
	}

	time.Sleep(2 * time.Millisecond)
	return nil
}

//Home returns the cursor to the top-left
func (lcd *LiquidCrystalLCD) Home() error {
	if err := lcd.command(lcdReturnHome); err != nil {
		return err
	}

	time.Sleep(2 * time.Millisecond)
	return nil
}

//DisplayOn turns the text display on
func (lcd *LiquidCrystalLCD) DisplayOn() error {
	lcd.displCntrl |= lcdDisplayOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//DisplayOff turns the text display off
func (lcd *LiquidCrystalLCD) DisplayOff() error {
	lcd.displCntrl &= ^lcdDisplayOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//BacklightOn turns the lcd light on
func (lcd *LiquidCrystalLCD) BacklightOn() error {
	lcd.backLight = lcdBacklight
	return lcd.expandWrite(0)
}

//BacklightOff turns the lcd light off
func (lcd *LiquidCrystalLCD) BacklightOff() error {
	lcd.backLight = lcdNoBacklight
	return lcd.expandWrite(0)
}

//UnderlineOn turns on the underline cursor
func (lcd *LiquidCrystalLCD) UnderlineOn() error {
	lcd.displCntrl |= lcdCursorOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//UnderlineOff turns off the underline cursor
func (lcd *LiquidCrystalLCD) UnderlineOff() error {
	lcd.displCntrl &= ^lcdCursorOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//CursorOn turns on the blinking cursor
func (lcd *LiquidCrystalLCD) CursorOn() error {
	lcd.displCntrl |= lcdBlinkOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//CursorOff turns off the blinking cursor
func (lcd *LiquidCrystalLCD) CursorOff() error {
	lcd.displCntrl &= ^lcdBlinkOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//ShiftDisplayLeft moves the text on the entire display to the left
func (lcd *LiquidCrystalLCD) ShiftDisplayLeft() error {
	return lcd.command(lcdCursorShift | lcdDisplayMove | lcdMoveLeft)
}

//ShiftDisplayRight moves the text on the entire display to the right
func (lcd *LiquidCrystalLCD) ShiftDisplayRight() error {
	return lcd.command(lcdCursorShift | lcdDisplayMove | lcdMoveRight)
}

//PrintLeftToRight prints text from left to right. e.g. 'foo' will display as 'foo'
func (lcd *LiquidCrystalLCD) PrintLeftToRight() error {
	lcd.displMode |= lcdEntryLeft
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//PrintRightToLeft prints text from right to left. e.g. 'foo' will display as 'oof'
func (lcd *LiquidCrystalLCD) PrintRightToLeft() error {
	lcd.displMode &= ^lcdEntryLeft
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//AutoScrollOn 'left justifies' the text so that the display moves when
//printing characters rather than moving the cursor
func (lcd *LiquidCrystalLCD) AutoScrollOn() error {
	lcd.displMode |= lcdEntryShiftIncrement
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//AutoScrollOff 'right justifies' the text so that the cursor moves when
//printing characters rather than moving the display
func (lcd *LiquidCrystalLCD) AutoScrollOff() error {
	lcd.displMode &= ^lcdEntryShiftIncrement
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//SetCursor positions the cursor at the specified row/column.
func (lcd *LiquidCrystalLCD) SetCursor(col, row byte) error {
	var rowOffset = []byte{0, 0x40, 0x14, 0x54}

	if row > lcd.rows-1 {
		row = lcd.rows - 1
	}

	if col > lcd.cols-1 {
		col = lcd.cols - 1
	}

	return lcd.command(lcdSetDDRAMAddr | (col + rowOffset[row]))
}

//RegisterCharacter registers a custom character to display on the lcd.
//Any custom characters currently on the lcd will be immediately replaced.
//location may be a number from 0 - 7.
func (lcd *LiquidCrystalLCD) RegisterCharacter(location byte, charmap *CustomCharacter) error {
	location &= 0x7

	if err := lcd.command(lcdSetCGramAddr | (location << 3)); err != nil {
		return err
	}

	for _, i := range charmap.CharMap {
		if err := lcd.send(i, rs); err != nil {
			return err
		}
	}

	charmap.Register = location

	return nil
}

//Write satisfies the io.Writer interface so it can be used with fmt or the I/O of your choice.
func (lcd *LiquidCrystalLCD) Write(str []byte) (int, error) {
	i := 0

	for i < len(str) {
		if err := lcd.send(str[i], rs); err != nil {
			return i, err
		}

		i++
	}

	return i, nil
}

//CustomCharacter holds the bytes and register for custom characters
type CustomCharacter struct {
	CharMap  [8]byte
	Register byte
}

//NewCharacter creates a new custom character map to display
func NewCharacter(charmap [8]byte) *CustomCharacter {
	return &CustomCharacter{CharMap: charmap}
}

//String will return an ASCII value of the register.  Use this with Fprintf and pass in
//your custom character, or call LiquidCrystalLCD.Write and pass in the register value.
//  cchar := liquidcrystallcd.NewCharacter([8]byte{...})
//  lcd.RegisterCharacter(0, cchar)
//  lcd.Home()
//  fmt.Fprintf(lcd, "This is a custom character: %v", cchar) // Use Fprintf...
//  lcd.Write([]byte{0}) // ... or call the Write method
func (c *CustomCharacter) String() string {
	return string([]byte{c.Register})
}
