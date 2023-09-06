package main

import (
	"encoding/base64"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/NT-community/citizen-gen/erc721"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
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
)

func part(season int, ethClient *ethclient.Client) func(ctx echo.Context) error {
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
				return ctx.String(http.StatusInternalServerError, err.Error())
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

			resp := ctx.Response()

			resp.Header().Set("Content-Type", decoded.ContentType)

			resp.WriteHeader(200)
			resp.Write(decoded.Raw)
			return nil
		}

		return ctx.String(http.StatusBadRequest, "unknown part")
	}
}
