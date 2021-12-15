package main

import (
	"image"
	"image/draw"
	"net/http"
	"strings"

	"github.com/disintegration/imaging"

	_ "image/jpeg"
	_ "image/png"
)

type FetchedImage struct {
	img image.Image
	url string
}

type ImageGenerator struct {
	Width, Height                    int
	NoBackground, SantaHat, Snowball bool
	Layers                           []*FetchedImage
}

func (i *ImageGenerator) Generate() image.Image {
	base := image.NewRGBA(image.Rect(0, 0, i.Width, i.Height))

	for idx, fetchedImg := range i.Layers {
		img := fetchedImg.img
		if i.NoBackground && idx == 0 {
			// don't draw background if requested otherwise
			continue
		}

		if i.Snowball && strings.Contains(fetchedImg.url, "weapon") {
			continue // don't render the weapon with a snowball
		}

		if i.Snowball && strings.Contains(fetchedImg.url, "hand") {
			img = emptyFist
			draw.Draw(img.(*image.NRGBA), emptyFist.Bounds(), snowBall, image.Pt(0, 0), draw.Over)
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

	if i.SantaHat {
		draw.Draw(base, santaHat.Bounds(), santaHat, image.Pt(0, 0), draw.Over)
	}
	return base
}

func newImageGenerator(w, h int, layers []*FetchedImage) *ImageGenerator {
	return &ImageGenerator{
		Width:  w,
		Height: h,
		Layers: layers,
	}
}

func fetchImage(url string) (*FetchedImage, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(resp.Body)
	return &FetchedImage{img, url}, err
}
