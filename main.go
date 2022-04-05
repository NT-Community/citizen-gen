package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
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
)

var (
	descriptionRegex              = regexp.MustCompile(`(\"description\":\s\")(.+)(\",)`)
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

func season(contract *erc721.Erc721) func(c echo.Context) error {
	return func(c echo.Context) error {
		return generate(c, contract)
	}
}

func generate(c echo.Context, contract *erc721.Erc721) error {
	// TODO: implement caching of images on

	path := c.Request().URL.Path

	dimensions := c.Param("dimensions")
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	whArray := strings.Split(dimensions, "x")

	if len(whArray) != 2 {
		return c.String(http.StatusBadRequest, "invalid length")
	}

	noBg := c.QueryParam("no-bg") != ""
	santaHat := c.QueryParam("santa-hat") != ""
	snowball := c.QueryParam("snowball") != ""

	if noBg {
		path += "_no_bg" // build on to the paths based on the passed in parameters
	}

	if santaHat {
		path += "_santa"
	}

	if snowball {
		path += "_snowball"
	}

	season := strings.Split(c.Path()[1:], "/")[0]

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

		img, err := fetchImage(imgUrl.Href)
		if err != nil {
			continue
		}
		fetchedImages = append(fetchedImages, img)
	}

	imgGen := newImageGenerator(width, height, fetchedImages)

	imgGen.NoBackground = noBg
	imgGen.SantaHat = santaHat
	imgGen.Snowball = snowball

	finalImage := imgGen.Generate()

	os.Mkdir(filepath.Join("images", season), os.ModePerm)
	os.Mkdir(filepath.Join("images", season, dimensions), os.ModePerm)

	f, err := os.Create(fmt.Sprintf("images/%s.png", path[1:]))

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

	e.GET("/s1/:dimensions/:id", season(s1contract))
	e.GET("/s2/:dimensions/:id", season(s2contract))

	if os.Getenv("CERT") != "" && os.Getenv("KEY") != "" {
		log.Fatalln(e.StartTLS(os.Getenv("HOST"), os.Getenv("CERT"), os.Getenv("KEY")))
	} else {
		log.Fatalln(e.Start(os.Getenv("HOST")))
	}

}
