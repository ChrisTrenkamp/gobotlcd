package liquidcrystallcd

import (
	"fmt"
	"io"
	"testing"

	"gobot.io/x/gobot"
)

func TestInterfaces(t *testing.T) {
	var _ gobot.Driver = (*LiquidCrystalLCD)(nil)
	var _ io.Writer = (*LiquidCrystalLCD)(nil)
	var _ fmt.Stringer = (*CustomCharacter)(nil)
}
