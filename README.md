# Liquid Crystal LCD Gobot driver

A [GoBot](https://gobot.io/) driver for [Liquid Crystal LCDs](https://www.arduino.cc/en/Reference/LiquidCrystalConstructor).  Requires an I2C connection.

### Getting started

```
go get -u github.com/ChrisTrenkamp/liquidcrystallcd
```

### Hello World

```go
package main

import (
	"fmt"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/raspi"

	lclcd "github.com/ChrisTrenkamp/liquidcrystallcd"
)

var smiley = lclcd.NewCharacter([8]byte{
	0x00,
	0x00,
	0x0A,
	0x00,
	0x00,
	0x11,
	0x0E,
	0x00,
})

func main() {
	rpi := raspi.NewAdaptor()
	lcd := lclcd.NewLiquidCrystalLCD(rpi, 16, 2, lclcd.DotSize5x8)

	work := func() {
		lcd.BacklightOn()
		lcd.RegisterCharacter(0, smiley)
		lcd.Home()
		fmt.Fprintf(lcd, "Hello World! %v", smiley)
	}

	robot := gobot.NewRobot("LiquidCrystalLCD",
		[]gobot.Connection{rpi},
		[]gobot.Device{lcd},
		work,
	)

	robot.Start()
}

```