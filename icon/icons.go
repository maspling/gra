package icon

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

var (
	//go:embed icon16.png
	size16 []byte

	//go:embed icon32.png
	size32 []byte

	//go:embed icon48.png
	size48 []byte

	//go:embed icon64.png
	size64 []byte
)

func LoadIcons() ([]image.Image, error) {
	var icons []image.Image

	for _, b := range [][]byte{size16, size32, size48, size64} {
		icon, err := loadIcon(b)
		if err != nil {
			return nil, err
		}
		icons = append(icons, icon)
	}

	return icons, nil
}

func loadIcon(b []byte) (image.Image, error) {
	icon, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return icon, nil
}
