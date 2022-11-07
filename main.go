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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/NT-community/citizen-gen/erc721"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	descriptionRegex              = regexp.MustCompile(`(\"description\":\s\")(.+)(\",)`)
	hexColorRegex                 = regexp.MustCompile(`([a-fA-F0-9]{6})`)
	santaHat, emptyFist, snowBall image.Image
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

func season(contract *erc721.Erc721, season int) func(c echo.Context) error {
	return func(c echo.Context) error {
		return generate(c, season, contract)
	}
}

func validateBGColor(bgColor string) (*color.RGBA, error) {
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

func generate(c echo.Context, season int, contract *erc721.Erc721) error {
	// TODO: implement caching of images on

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
	bgColorHex := c.QueryParam("bg-color")

	var backgroundColor *color.RGBA

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

	tokenUri, err := contract.TokenURI(nil, big.NewInt(int64(id)))

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	rawJson, _ := base64.StdEncoding.DecodeString(strings.Split(tokenUri, ",")[1])

	rawJson = []byte(descriptionRegex.ReplaceAllString(string(rawJson), ""))

	var decodedJson map[string]json.RawMessage

	if err := json.Unmarshal(rawJson, &decodedJson); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	xml, _ := base64.StdEncoding.DecodeString(strings.Split(string(decodedJson["image_data"]), ",")[1])

	imgs, err := collectImages(xml)

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

	imgGen := newImageGenerator(width, height, fetchedImages)

	imgGen.NoBackground = noBg
	imgGen.SantaHat = santaHat
	imgGen.Snowball = snowball
	imgGen.SeasonNumber = season
	imgGen.Female = female
	imgGen.BackgroundColor = backgroundColor

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

	e := echo.New() // create our new echo handler

	e.Use(middleware.CORS())

	e.GET("/s1/:dimensions/:id", season(s1contract, 1))
	e.GET("/s2/:dimensions/:id", season(s2contract, 2))
	if os.Getenv("CERT") != "" && os.Getenv("KEY") != "" {
		log.Fatalln(e.StartTLS(os.Getenv("HOST"), os.Getenv("CERT"), os.Getenv("KEY")))
	} else {
		log.Fatalln(e.Start(os.Getenv("HOST")))
	}

}
