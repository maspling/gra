package icon

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
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

	//go:embed progression.svg
	progressionSVG []byte

	//go:embed win.svg
	winSVG []byte

	//go:embed missable.svg
	missableSVG []byte
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

func LoadProgression(size int) (image.Image, error) {
	return rasterizeSVG(progressionSVG, size)
}

func LoadWin(size int) (image.Image, error) {
	return rasterizeSVG(winSVG, size)
}

func LoadMissable(size int) (image.Image, error) {
	return rasterizeSVG(missableSVG, size)
}

func rasterizeSVG(data []byte, size int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())

	scanner.SetColor(color.Black)
	filler := rasterx.NewFiller(size, size, scanner)
	center := float64(size) / 2
	rasterx.AddCircle(center, center, float64(size)*0.4167, filler)
	filler.Draw()

	icon.Draw(rasterx.NewDasher(size, size, scanner), 1)
	return rgba, nil
}
