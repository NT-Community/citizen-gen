package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
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
		Male:   "QmPLW6u5MRut1b8iyVc47ET5zAj9VaG2GwyjcuKLoetWsT",
		Female: "QmPVfdHHdjyZb6BKHhwaJ1eEdCx9Jz4mvCn4KHiCJQaB8e",
	},
	2: {
		Male:   "QmeqeBpsYTuJL8AZhY9fGBeTj9QuvMVqaZeRWFnjA24QEE",
		Female: "QmeqeBpsYTuJL8AZhY9fGBeTj9QuvMVqaZeRWFnjA24QEE",
	},
}

type FetchedImage struct {
	img image.Image
	url string
}

type ImageGenerator struct {
	Width, Height                                 int
	NoBackground, SantaHat, Snowball, Female, PFP bool

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
	highestPixelY := base.Bounds().Dy() / 12

	for idx, fetchedImg := range i.Layers {
		img := fetchedImg.img

		log.Println(fetchedImg.url)

		if strings.Contains(fetchedImg.url, "helm") || strings.ContainsAny(fetchedImg.url, "hair") {
			highest := findHighestColoredPixel(img, midPointX)

			if highest < highestPixelY {
				highestPixelY = highest
			}
		}

		if idx == 0 && i.BackgroundColor != nil {
			img = image.NewUniform(i.BackgroundColor)

			draw.Draw(base, base.Bounds(), img, image.Pt(0, 0), draw.Over)
			continue

		} else if i.NoBackground && idx == 0 {
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

		draw.Draw(base, img.Bounds(), img, image.Pt(0, 0), draw.Over)
	}

	var finalizedImage image.Image = base

	if i.PFP {
		startX := midPointX
		startY := highestPixelY
		finalizedImage = imaging.Crop(finalizedImage, image.Rect(startX-320, startY, startX+320, startY+640))
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
	for y := 0; y < img.Bounds().Dy(); y++ {
		color := img.At(x, y)
		r, g, b, a := color.RGBA()

		if r >= 0 && g >= 0 && b >= 0 && a > 0 {
			return y - 10
		}
	}

	return 0
}

func newImageGenerator(w, h int, layers []*FetchedImage) *ImageGenerator {
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
