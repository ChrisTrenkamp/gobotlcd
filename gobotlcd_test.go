package gobotlcd

import (
	"fmt"
	"io"
	"testing"

	"gobot.io/x/gobot"
)

func TestInterfaces(t *testing.T) {
	var _ gobot.Driver = (*GobotLCD)(nil)
	var _ io.Writer = (*GobotLCD)(nil)
	var _ fmt.Stringer = (*CustomCharacter)(nil)
}
