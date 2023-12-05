package main

import (
	"image"
	"image/color"
	"image/draw"
	"net/http"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"

	_ "image/jpeg"
	_ "image/png"
)

var IPFSRegex = regexp.MustCompile(`(https:\/\/neotokyo\.mypinata\.cloud\/ipfs)\/(Qm[\w]+)\/(.+)`)

type IPFSBucket struct {
	Male, Female string
}

// season => IPFS buckets based on male/female
var IPFSBuckets map[int]IPFSBucket = map[int]IPFSBucket{
	1: {
		Male:   "QmZxhDwLcoK7cipX3Y1qMEpWUHExN7F2jCz7QNfGcexUEu",
		Female: "QmPVfdHHdjyZb6BKHhwaJ1eEdCx9Jz4mvCn4KHiCJQaB8e",
	},
	2: {
		Male:   "QmeqeBpsYTuJL8AZhY9fGBeTj9QuvMVqaZeRWFnjA24QEE",
		Female: "QmeqeBpsYTuJL8AZhY9fGBeTj9QuvMVqaZeRWFnjA24QEE",
	},
}

type FetchedImage struct {
	Img image.Image
	URL string
}

type ImageGenerator struct {
	Width, Height                                                     int
	NoBackground, SantaHat, Snowball, Female, NoClothes, PFP, Preview bool

	BackgroundColor *color.RGBA

	SeasonNumber int
	Layers       []*FetchedImage
}

func (i *ImageGenerator) Generate() image.Image {
	base := image.NewNRGBA(image.Rect(0, 0, 1200, 1200))

	if i.SantaHat {
		i.Layers = append(i.Layers, &FetchedImage{santaHat, ""})
	}

	midPointX := base.Bounds().Dx() / 2
	highestPixelY := 128

	for idx, fetchedImg := range i.Layers {
		img := fetchedImg.Img

		if strings.Contains(fetchedImg.URL, "body") && strings.Contains(fetchedImg.URL, "QmPVfdHHdjyZb6BKHhwaJ1eEdCx9Jz4mvCn4KHiCJQaB8e") {

			if strings.HasSuffix(fetchedImg.URL, "5.png") {
				bounds := fetchedImg.Img.Bounds()

				// 300x300
				if bounds.Dx()*bounds.Dy() == 90000 {
					// Resize to 1200x1200
					img = imaging.Resize(img, 1200, 1200, imaging.NearestNeighbor)
				}
			}
		}

		if strings.Contains(fetchedImg.URL, "helm") || strings.Contains(fetchedImg.URL, "hair") {
			highest := findHighestColoredPixel(img, midPointX)
			if highest < highestPixelY {
				highestPixelY = highest
			}
		}

		if strings.Contains(fetchedImg.URL, "cloth") && i.NoClothes {
			continue
		}

		if idx == 0 && i.BackgroundColor != nil {
			img = image.NewUniform(i.BackgroundColor)

			draw.Draw(base, base.Bounds(), img, image.Pt(0, 0), draw.Over)
			continue
		} else if (i.NoBackground || i.Preview) && idx == 0 {
			// don't draw background if requested otherwise
			continue
		}

		if i.Snowball && strings.Contains(fetchedImg.URL, "weapon") {
			continue // don't render the weapon with a snowball
		}

		if i.Snowball && strings.Contains(fetchedImg.URL, "hand") {
			img = emptyFist
			draw.Draw(img.(*image.NRGBA), emptyFist.Bounds(), snowBall, image.Pt(0, 0), draw.Over)
		}

		draw.Draw(base, img.Bounds(), img, image.Pt(0, 0), draw.Over)
	}

	var finalizedImage image.Image = base

	if i.PFP || i.Preview {
		startX := midPointX
		startY := highestPixelY
		endY := (startY + 640)

		if startY < 0 {
			endY += 640 - (startY + 640)
		}
		finalizedImage = imaging.Crop(finalizedImage, image.Rect(startX-320, startY, startX+320, endY))
	} else {
		rw, rh := 0, 0
		if base.Bounds().Dx() != i.Width {
			rw = i.Width
		}

		if base.Bounds().Dy() != i.Height {
			rh = i.Height
		}

		if rw > 0 || rh > 0 {
			finalizedImage = imaging.Resize(base, rw, rh, imaging.NearestNeighbor)
		}
	}

	return finalizedImage
}

func findHighestColoredPixel(img image.Image, x int) int {
	highestY := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		color := img.At(x, y)
		r, g, b, a := color.RGBA()

		if r >= 0 && g >= 0 && b >= 0 && a > 0 {
			highestY = y
			break
		}
	}

	size := 256

	for mx := max(x-size/2, 0); mx < x+size/2; mx++ {
		for my := max(highestY-(size/2), 0); my < highestY+(size/2); my++ {
			color := img.At(mx, my)
			r, g, b, a := color.RGBA()
			if r >= 0 && g >= 0 && b >= 0 && a > 0 {
				if highestY > my {
					highestY = my
				}
			}
		}
	}

	return highestY - 40
}

func NewImageGenerator(w, h int, layers []*FetchedImage) *ImageGenerator {
	return &ImageGenerator{
		BackgroundColor: nil,
		Width:           w,
		Height:          h,
		Layers:          layers,
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
