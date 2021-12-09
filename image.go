package main

import (
	"image"
	"image/draw"
	"net/http"

	"github.com/disintegration/imaging"

	_ "image/jpeg"
	_ "image/png"
)

type ImageGenerator struct {
	Width, Height int
	NoBackground  bool
	Layers        []image.Image
}

func (i *ImageGenerator) Generate() image.Image {
	base := image.NewRGBA(image.Rect(0, 0, i.Width, i.Height))

	for idx, img := range i.Layers {
		if i.NoBackground && idx == 0 {
			// don't draw background if requested otherwise
			continue
		}

		rw, rh := 0, 0
		if img.Bounds().Dx() != base.Rect.Dx() {
			rw = base.Rect.Dx()
		}

		if img.Bounds().Dy() != base.Rect.Dy() {
			rh = base.Rect.Dy()
		}

		if rw > 0 || rh > 0 {
			img = imaging.Resize(img, rw, rh, imaging.NearestNeighbor)
		}

		draw.Draw(base, img.Bounds(), img, image.Pt(0, 0), draw.Over)
	}
	return base
}

func newImageGenerator(w, h int, layers []image.Image) *ImageGenerator {
	return &ImageGenerator{
		Width:  w,
		Height: h,
		Layers: layers,
	}
}

func fetchImage(url string) (image.Image, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(resp.Body)
	return img, err
}
