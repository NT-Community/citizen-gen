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
	descriptionRegex = regexp.MustCompile(`(\"description\":\s\")(.+)(\",)`)
	santaHat         image.Image
)

func init() {
	// load the .env file
	err := godotenv.Load()

	if err != nil {
		log.Fatalln(err)
	}

	os.Mkdir("images", os.ModePerm)

	santaHatFile, err := os.Open("assets/santa_hat.png")

	fmt.Println(err)
	// ignore if not present
	if err == nil {

		img, err := png.Decode(santaHatFile)

		if err != nil {
			log.Println(err)
		} else {
			santaHat = img
		}
	}
}

func main() {
	client, err := ethclient.Dial(os.Getenv("RPC"))

	if err != nil {
		log.Fatalln(err)
	}

	contract, err := erc721.NewErc721(common.HexToAddress(os.Getenv("CONTRACT")), client)

	if err != nil {
		log.Fatalln(err)
	}

	e := echo.New() // create our new echo handler

	e.GET("/:dimensions/:id", func(c echo.Context) error {
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

		if noBg {
			path += "_no_bg" // build on to the paths based on the passed in parameters
		}

		if santaHat {
			path += "_santa"
		}

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

		var fetchedImages []image.Image

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

		finalImage := imgGen.Generate()

		os.Mkdir(filepath.Join("images", dimensions), os.ModePerm)

		f, _ := os.Create(fmt.Sprintf("images/%s.png", path))

		png.Encode(f, finalImage)

		return png.Encode(c.Response().Writer, finalImage)
	})

	if os.Getenv("CERT") != "" && os.Getenv("KEY") != "" {
		log.Fatalln(e.StartTLS(os.Getenv("HOST"), os.Getenv("CERT"), os.Getenv("KEY")))
	} else {
		log.Fatalln(e.Start(os.Getenv("HOST")))
	}

}
