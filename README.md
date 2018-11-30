# Gobot LCD Gobot driver

[![GoDoc](https://godoc.org/gopkg.in/src-d/go-git.v2?status.svg)](https://godoc.org/github.com/ChrisTrenkamp/gobotlcd) [![Go Report Card](https://goreportcard.com/badge/github.com/ChrisTrenkamp/gobotlcd)](https://goreportcard.com/report/github.com/ChrisTrenkamp/gobotlcd)

A [GoBot](https://gobot.io/) driver for [LCDs](https://www.arduino.cc/en/Reference/LiquidCrystal).  Requires an I2C connection.  Has only been tested with a 20x4 display.

### Getting started

```
go get -u github.com/ChrisTrenkamp/gobotlcd
```

### Hello World

```go
package main

import (
	"fmt"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/raspi"

	"github.com/ChrisTrenkamp/gobotlcd"
)

var smiley = gobotlcd.NewCharacter([8]byte{
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
	lcd := gobotlcd.NewLiquidCrystalLCD(rpi, 16, 2, gobotlcd.DotSize5x8)

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
