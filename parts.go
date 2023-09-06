package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/NT-community/citizen-gen/erc721"
	"github.com/chromedp/chromedp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"github.com/tdewolff/canvas"
)

var (
	PartsContracts = map[int]map[string]string{
		1: {
			"identity": "0x059174c2Fef43F06178D23572FE5556F078F2F99",
			"id":       "0x059174c2Fef43F06178D23572FE5556F078F2F99",
			"item":     "0xE7489EA1847395d7EeAd33E9c85fe327D513D249",
			"vault":    "0x17B2f2b8927A8f11edfd7a27E153Be17d68E69C7",
			"land":     "0xCFc6a15b2952B6014A993a0C16c9D580d862e21A",
		},
		2: {
			"identity": "0x8E9F3C6883993A7A69c37213F2eb9A17450ad6D3",
			"id":       "0x8E9F3C6883993A7A69c37213F2eb9A17450ad6D3",
			"land":     "0xB58aE9e93b8bee7d890AD87A2a70c135a3Bf4B4e",
			"item":     "0x0B8F04F2cA4f15d33274a27439412ab7639EFAd9",
		},
	}

	LegacyPartsContracts = map[int]map[string]string{
		1: {
			"item":  "0x0938E3F7AC6D7f674FeD551c93f363109bda3AF9",
			"vault": "0xab0b0dD7e4EaB0F9e31a539074a03f1C1Be80879",
			"land":  "0x3C54b798b3aAD4F6089533aF3bdbD6ce233019bB",
		},
		2: {
			"identity": "0x698FbAACA64944376e2CDC4CAD86eaa91362cF54",
			"id":       "0x698FbAACA64944376e2CDC4CAD86eaa91362cF54",
			"land":     "0xf90980AE7A44E2d18B9615396FF5E9252F1DF639",
			"item":     "0x7AC66d40d80D2d8D1E45D6b5B10a1C9D1fd69354",
		},
	}
)

func part(season int, render bool, ethClient *ethclient.Client) func(ctx echo.Context) error {
	return func(ctx echo.Context) error {
		partType := strings.ToLower(ctx.Param("part"))
		partId, err := strconv.Atoi(ctx.Param("id"))

		if err != nil {
			return ctx.String(http.StatusBadRequest, err.Error())
		}

		if contract, ok := PartsContracts[season][partType]; ok {
			contract, err := erc721.NewErc721(common.HexToAddress(contract), ethClient)

			if err != nil {
				return ctx.String(http.StatusInternalServerError, err.Error())
			}

			tokenUri, err := contract.TokenURI(nil, big.NewInt(int64(partId)))

			if err != nil {
				if legacyContractAddr, ok := LegacyPartsContracts[season][partType]; !ok {
					return ctx.String(http.StatusInternalServerError, err.Error())
				} else {
					legacyContract, err := erc721.NewErc721(common.HexToAddress(legacyContractAddr), ethClient)

					if err != nil {
						return ctx.String(http.StatusInternalServerError, err.Error())
					}

					tokenUri, err = legacyContract.TokenURI(nil, big.NewInt(int64(partId)))

					if err != nil {
						return ctx.String(http.StatusInternalServerError, err.Error())
					}
				}
			}

			rawMetadata, err := base64.StdEncoding.DecodeString(strings.Split(tokenUri, ",")[1])

			if err != nil {
				return ctx.String(http.StatusInternalServerError, err.Error())
			}

			metadata, err := ParseMetadata(rawMetadata)

			if err != nil {
				return ctx.String(http.StatusInternalServerError, err.Error())
			}

			decoded, err := metadata.Image.Decode()

			if err != nil {
				return ctx.String(http.StatusInternalServerError, err.Error())
			}

			if decoded.ContentType == "image/svg+xml" && render {

				svgCanvas, err := canvas.ParseSVG(bytes.NewReader(decoded.Raw))

				if err != nil {
					return ctx.String(http.StatusInternalServerError, err.Error())
				}

				c, cancel := chromedp.NewContext(
					context.Background(),
					// chromedp.WithDebugf(log.Printf),
				)
				defer cancel()

				// capture screenshot of an element
				var buf []byte
				// cnvs, err := canvas.ParseSVG(bytes.NewReader(decoded.Raw))\

				w, h := int(svgCanvas.W*canvas.DefaultResolution.DPMM())*2, int(svgCanvas.H*canvas.DefaultResolution.DPMM())*2

				// capture entire browser viewport, returning png with quality=90
				if err := chromedp.Run(c, fullScreenshot(string(metadata.Image), 100, w, h, &buf)); err != nil {
					return ctx.String(http.StatusInternalServerError, err.Error())
				}

				img, err := png.Decode(bytes.NewReader(buf))

				if err != nil {
					return ctx.String(http.StatusInternalServerError, err.Error())
				}

				// croppedImg := imaging.Crop(img, image.Rect(0, 0, )

				return png.Encode(ctx.Response().Writer, img)

			}

			resp := ctx.Response()

			resp.Header().Set("Content-Type", decoded.ContentType)

			resp.WriteHeader(200)
			resp.Write(decoded.Raw)
			return nil
		}

		return ctx.String(http.StatusBadRequest, "unknown part")
	}
}

func fullScreenshot(urlstr string, quality int, w, h int, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.EmulateViewport(int64(w), int64(h)),
		chromedp.Screenshot("svg", res, chromedp.NodeVisible),
	}
}
