package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/NT-community/citizen-gen/erc721"
	"github.com/disintegration/imaging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "image/jpeg"
)

var (
	descriptionRegex              = regexp.MustCompile(`(\"description\":\s\")(.+)(\",)`)
	hexColorRegex                 = regexp.MustCompile(`([a-fA-F0-9]{6})`)
	santaHat, emptyFist, snowBall image.Image

	ColorCodedRarity = map[string]string{
		"elite":   "faac27", // gold color for elite
		"default": "849ef3",
		"outer":   "b0d774",
	}
)

// loads an image without any error checks
func mustLoadImage(path string) image.Image {
	file, err := os.Open(path)
	// ignore if not present
	if err == nil {
		img, _ := png.Decode(file)
		return img
	}
	return nil
}

func init() {
	// load the .env file
	err := godotenv.Load()

	if err != nil {
		log.Fatalln(err)
	}

	os.Mkdir("images", os.ModePerm)

	santaHat = mustLoadImage("assets/santa_hat.png")
	emptyFist = mustLoadImage("assets/empty_fist.png")
	snowBall = mustLoadImage("assets/emptyhand_snowball.png")
}

func teardown(oldContract, newContract *erc721.Erc721, season int) func(c echo.Context) error {
	return func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		tokenUri, err := newContract.TokenURI(nil, big.NewInt(int64(id)))

		if err != nil {
			tokenUri, err = oldContract.TokenURI(nil, big.NewInt(int64(id)))

			if err != nil {
				return c.String(http.StatusNotFound, err.Error())
			}
		}

		rawJson, _ := base64.StdEncoding.DecodeString(strings.Split(tokenUri, ",")[1])

		rawJson = []byte(descriptionRegex.ReplaceAllString(string(rawJson), ""))

		var decodedJson map[string]json.RawMessage

		if err := json.Unmarshal(rawJson, &decodedJson); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		xml, _ := base64.StdEncoding.DecodeString(strings.Split(string(decodedJson["image_data"]), ",")[1])

		imgs, err := CollectImages(xml)

		partToUrl := map[string]string{}

		for _, img := range imgs {
			imgUrl, err := url.Parse(img.Href)

			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}

			partName := strings.Split(imgUrl.Path, "/")[3] // strip the IPFS part
			partToUrl[partName] = img.Href
		}

		return c.JSON(http.StatusOK, partToUrl)
	}
}

func season(oldContract, newContract *erc721.Erc721, season int) func(c echo.Context) error {
	return func(c echo.Context) error {
		return generate(c, season, oldContract, newContract)
	}
}

func upscale(c echo.Context) error {
	size := c.QueryParam("size")

	whArray := strings.Split(size, "x")

	if len(whArray) != 2 {
		return c.String(http.StatusBadRequest, "invalid length")
	}

	width, err := strconv.Atoi(whArray[0])

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	height, err := strconv.Atoi(whArray[1])

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	img, _, err := image.Decode(c.Request().Body)

	if err != nil {
		return nil
	}

	return png.Encode(c.Response().Writer, imaging.Resize(img, width, height, imaging.NearestNeighbor))
}

func validateBGColor(bgColor string) (*color.RGBA, error) {

	// this essentially adds a shortcut for
	if v, ok := ColorCodedRarity[strings.ToLower(bgColor)]; ok {
		bgColor = v
	}

	// test if the passed in string is in hexadecimal
	if hexColorRegex.MatchString(bgColor) {
		// parse the integer from hexadecimal, base-16, 32-bit integer.
		parsed, err := strconv.ParseInt(bgColor, 16, 32)

		if err != nil {
			return nil, errors.New("failed to parse integer")
		}

		return &color.RGBA{
			R: uint8(parsed >> 16),
			G: uint8((parsed >> 8) & 0xFF),
			B: uint8(parsed & 0xFF),
			A: 0xFF,
		}, nil
	}
	return nil, errors.New("background color string is invalid hex")
}

func generate(c echo.Context, season int, oldContract, newContract *erc721.Erc721) error {
	var pfp bool
	var whArray []string

	path := c.Request().URL.Path
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	dimensions := c.Param("dimensions")

	if strings.ToLower(dimensions) == "pfp" {
		pfp = true
		whArray = []string{"1200", "1200"}
	} else {
		whArray = strings.Split(dimensions, "x")

		if len(whArray) != 2 {
			return c.String(http.StatusBadRequest, "invalid length")
		}
	}
	noBg := c.QueryParam("no-bg") != ""
	santaHat := c.QueryParam("santa-hat") != ""
	snowball := c.QueryParam("snowball") != ""
	female := c.QueryParam("female") != ""
	noClothes := c.QueryParam("no-clothes") != ""
	bgColorHex := c.QueryParam("bg-color")
	preview := c.QueryParam("crop_preview") != ""
	var backgroundColor *color.RGBA

	if preview {
		// Crop preview is a special flag that will generate 640x640 PFP cropped image
		path += "_crop_preview"
	} else {
		if bgColorHex != "" {

			parsedColor, err := validateBGColor(bgColorHex)

			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}

			backgroundColor = parsedColor

			path += "_bg_color_" + bgColorHex
		}

		if pfp {
			path += "_pfp_crop"
		}

		if noBg {
			path += "_no_bg" // build on to the paths based on the passed in parameters
		}
	}

	if santaHat {
		path += "_santa"
	}

	if snowball {
		path += "_snowball"
	}

	// a citizen that was forced to be rendered using female traits
	if female {
		path += "_female"
	}

	if noClothes {
		path += "_nc"
	}

	seasonString := fmt.Sprintf("s%d", season)

	if _, err := os.Stat(fmt.Sprintf("./images/%s.png", path)); err == nil {

		file, _ := os.Open(fmt.Sprintf("./images/%s.png", path))
		b, _ := ioutil.ReadAll(file)
		_, err := c.Response().Writer.Write(b)
		return err
	}

	width, err := strconv.Atoi(whArray[0])

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	height, err := strconv.Atoi(whArray[1])

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	tokenUri, err := newContract.TokenURI(nil, big.NewInt(int64(id)))

	if err != nil {
		tokenUri, err = oldContract.TokenURI(nil, big.NewInt(int64(id)))

		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
	}

	rawJson, _ := base64.StdEncoding.DecodeString(strings.Split(tokenUri, ",")[1])

	rawJson = []byte(descriptionRegex.ReplaceAllString(string(rawJson), ""))

	var decodedJson map[string]json.RawMessage

	if err := json.Unmarshal(rawJson, &decodedJson); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	xml, _ := base64.StdEncoding.DecodeString(strings.Split(string(decodedJson["image_data"]), ",")[1])

	imgs, err := CollectImages(xml)

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	var fetchedImages []*FetchedImage

	for _, imgUrl := range imgs {
		fetchUrl := imgUrl.Href

		if female {
			// force a replace to the female IPFS bucket

			// highly experimental
			fetchUrl = IPFSRegex.ReplaceAllString(fetchUrl, fmt.Sprintf("$1/%s/$3", IPFSBuckets[season].Female))

			if strings.Contains(fetchUrl, "body") || strings.Contains(fetchUrl, "hand") || strings.Contains(fetchUrl, "head") {
				groups := IPFSRegex.FindAllStringSubmatch(fetchUrl, -1)
				if strings.Contains(groups[0][3], "0.png") {
					replaceString := "0-0.png"

					if strings.Contains(fetchUrl, "hand") {
						replaceString = "fist/" + replaceString
					}

					fetchUrl = strings.ReplaceAll(fetchUrl, groups[0][3], fmt.Sprintf("%s/%s", strings.Split(groups[0][3], "/")[0], replaceString))
				}
			}
		}

		img, err := fetchImage(fetchUrl)
		if err != nil {
			continue
		}
		fetchedImages = append(fetchedImages, img)
	}

	imgGen := NewImageGenerator(width, height, fetchedImages)

	imgGen.Preview = preview
	imgGen.NoBackground = noBg
	imgGen.SantaHat = santaHat
	imgGen.Snowball = snowball
	imgGen.SeasonNumber = season
	imgGen.Female = female
	imgGen.BackgroundColor = backgroundColor
	imgGen.NoClothes = noClothes

	imgGen.PFP = pfp

	finalImage := imgGen.Generate()

	os.Mkdir(filepath.Join("images", seasonString), os.ModePerm)
	os.Mkdir(filepath.Join("images", seasonString, dimensions), os.ModePerm)

	f, _ := os.Create(fmt.Sprintf("images/%s.png", path[1:]))

	png.Encode(f, finalImage)

	return png.Encode(c.Response().Writer, finalImage)
}

func main() {
	client, err := ethclient.Dial(os.Getenv("RPC"))

	if err != nil {
		log.Fatalln(err)
	}

	s1contract, err := erc721.NewErc721(common.HexToAddress(os.Getenv("S1_CONTRACT")), client)

	if err != nil {
		log.Fatalln(err)
	}

	s2contract, err := erc721.NewErc721(common.HexToAddress(os.Getenv("S2_CONTRACT")), client)

	if err != nil {
		log.Fatalln(err)
	}

	s1v2contract, err := erc721.NewErc721(common.HexToAddress(os.Getenv("S1V2_CONTRACT")), client)

	if err != nil {
		log.Fatalln(err)
	}

	s2v2contract, err := erc721.NewErc721(common.HexToAddress(os.Getenv("S2V2_CONTRACT")), client)

	if err != nil {
		log.Fatalln(err)
	}

	e := echo.New() // create our new echo handler

	e.Use(middleware.CORS())

	e.GET("/healthcheck", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.GET("/s1/:dimensions/:id", season(s1contract, s1v2contract, 1))
	e.GET("/s2/:dimensions/:id", season(s2contract, s2v2contract, 2))

	e.GET("/s1/:id/teardown", teardown(s1contract, s1v2contract, 1))
	e.GET("/s2/:id/teardown", teardown(s2contract, s2v2contract, 2))

	e.GET("/s1/parts/:part/:id", part(1, false, client))
	e.GET("/s1/parts/:part/:id/render", part(1, true, client))

	e.GET("/s2/parts/:part/:id", part(2, false, client))
	e.GET("/s2/parts/:part/:id/render", part(2, true, client))

	e.POST("/upscale", upscale)
	if os.Getenv("CERT") != "" && os.Getenv("KEY") != "" {
		log.Fatalln(e.StartTLS(os.Getenv("HOST"), os.Getenv("CERT"), os.Getenv("KEY")))
	} else {
		log.Fatalln(e.Start(os.Getenv("HOST")))
	}

}
