//gobotlcd is a driver for LCD devices, based on the
//Arduino source code here: https://www.arduino.cc/en/Reference/LiquidCrystal

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

//GobotLCD controls a Liquid Crystal LCD with an I2C connection.
type GobotLCD struct {
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

//New connects to an LCD with the given I2C connection, the given row
//and column size, and the given dot size.
func New(a i2c.Connector, cols, rows byte, dotSize DotSize, options ...func(i2c.Config)) *GobotLCD {
	ret := &GobotLCD{
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
func (lcd *GobotLCD) SetName(name string) {
	lcd.name = name
}

// Name returns the label for the Driver
func (lcd *GobotLCD) Name() string {
	return lcd.name
}

// Connection returns the Connection associated with the Driver
func (lcd *GobotLCD) Connection() gobot.Connection {
	return lcd.connection.(gobot.Connection)
}

// Start initiates the Driver
func (lcd *GobotLCD) Start() (err error) {
	return lcd.init()
}

// Halt terminates the Driver
func (lcd *GobotLCD) Halt() (err error) {
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

func (lcd *GobotLCD) write(val byte) error {
	return lcd.connection.WriteByte(val)
}

func (lcd *GobotLCD) expandWrite(val byte) error {
	return lcd.write(val | lcd.backLight)
}

func (lcd *GobotLCD) pulseEnable(val byte) error {
	if err := lcd.expandWrite(val | en); err != nil {
		return err
	}
	time.Sleep(time.Microsecond)

	if err := lcd.expandWrite(val & ^en); err != nil {
		return err
	}
	time.Sleep(50 * time.Microsecond)

	return nil
}

func (lcd *GobotLCD) write4bits(val byte) error {
	if err := lcd.expandWrite(val); err != nil {
		return err
	}

	return lcd.pulseEnable(val)
}

func (lcd *GobotLCD) send(val, mode byte) error {
	high := val & byte(0xF0)
	low := (val << 4) & 0xF0

	if err := lcd.write4bits(high | mode); err != nil {
		return err
	}

	return lcd.write4bits(low | mode)
}

func (lcd *GobotLCD) command(val byte) error {
	return lcd.send(val, 0)
}

func (lcd *GobotLCD) init() (err error) {
	bus := lcd.GetBusOrDefault(1)
	address := lcd.GetAddressOrDefault(0x27)

	lcd.connection, err = lcd.connector.GetConnection(address, bus)
	if err != nil {
		return err
	}

	// LCD requires 40 ms after power-on before receiving commands
	time.Sleep(50 * time.Millisecond)

	if err = lcd.write(lcd.backLight); err != nil {
		return err
	}

	time.Sleep(time.Second)

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

func (lcd *GobotLCD) init4BitMode() error {
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
func (lcd *GobotLCD) Clear() error {
	if err := lcd.command(lcdClearDisplay); err != nil {
		return err
	}

	time.Sleep(2 * time.Millisecond)
	return nil
}

//Home returns the cursor to the top-left
func (lcd *GobotLCD) Home() error {
	if err := lcd.command(lcdReturnHome); err != nil {
		return err
	}

	time.Sleep(2 * time.Millisecond)
	return nil
}

//DisplayOn turns the text display on
func (lcd *GobotLCD) DisplayOn() error {
	lcd.displCntrl |= lcdDisplayOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//DisplayOff turns the text display off
func (lcd *GobotLCD) DisplayOff() error {
	lcd.displCntrl &= ^lcdDisplayOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//BacklightOn turns the lcd light on
func (lcd *GobotLCD) BacklightOn() error {
	lcd.backLight = lcdBacklight
	return lcd.expandWrite(0)
}

//BacklightOff turns the lcd light off
func (lcd *GobotLCD) BacklightOff() error {
	lcd.backLight = lcdNoBacklight
	return lcd.expandWrite(0)
}

//UnderlineOn turns on the underline cursor
func (lcd *GobotLCD) UnderlineOn() error {
	lcd.displCntrl |= lcdCursorOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//UnderlineOff turns off the underline cursor
func (lcd *GobotLCD) UnderlineOff() error {
	lcd.displCntrl &= ^lcdCursorOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//CursorOn turns on the blinking cursor
func (lcd *GobotLCD) CursorOn() error {
	lcd.displCntrl |= lcdBlinkOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//CursorOff turns off the blinking cursor
func (lcd *GobotLCD) CursorOff() error {
	lcd.displCntrl &= ^lcdBlinkOn
	return lcd.command(lcdDisplayControl | lcd.displCntrl)
}

//ShiftDisplayLeft moves the text on the entire display to the left
func (lcd *GobotLCD) ShiftDisplayLeft() error {
	return lcd.command(lcdCursorShift | lcdDisplayMove | lcdMoveLeft)
}

//ShiftDisplayRight moves the text on the entire display to the right
func (lcd *GobotLCD) ShiftDisplayRight() error {
	return lcd.command(lcdCursorShift | lcdDisplayMove | lcdMoveRight)
}

//PrintLeftToRight prints text from left to right. e.g. 'foo' will display as 'foo'
func (lcd *GobotLCD) PrintLeftToRight() error {
	lcd.displMode |= lcdEntryLeft
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//PrintRightToLeft prints text from right to left. e.g. 'foo' will display as 'oof'
func (lcd *GobotLCD) PrintRightToLeft() error {
	lcd.displMode &= ^lcdEntryLeft
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//AutoScrollOn 'left justifies' the text so that the display moves when
//printing characters rather than moving the cursor
func (lcd *GobotLCD) AutoScrollOn() error {
	lcd.displMode |= lcdEntryShiftIncrement
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//AutoScrollOff 'right justifies' the text so that the cursor moves when
//printing characters rather than moving the display
func (lcd *GobotLCD) AutoScrollOff() error {
	lcd.displMode &= ^lcdEntryShiftIncrement
	return lcd.command(lcdEntryModeSet | lcd.displMode)
}

//SetCursor positions the cursor at the specified row/column.
func (lcd *GobotLCD) SetCursor(col, row byte) error {
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
func (lcd *GobotLCD) RegisterCharacter(location byte, charmap *CustomCharacter) error {
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
func (lcd *GobotLCD) Write(str []byte) (int, error) {
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
