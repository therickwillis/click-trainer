package utility

import (
	"fmt"
	"math/rand"
)

func RandomColorHex() string {
	r := uint8(rand.Intn(248) + 4)
	g := uint8(rand.Intn(248) + 4)
	b := uint8(rand.Intn(248) + 4)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}
