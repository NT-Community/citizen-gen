package main

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

type Data struct {
	Raw         []byte
	ContentType string
}

func DecodeData(uri *url.URL) (*Data, error) {
	split := strings.Split(uri.Opaque, ";")
	dataType := split[0]
	dataParts := strings.Split(split[1], ",")
	var rawData []byte
	var err error

	switch dataParts[0] {
	case "base64":
		rawData, err = base64.StdEncoding.DecodeString(dataParts[1])
	case "base32":
		rawData, err = base32.StdEncoding.DecodeString(dataParts[1])
	}

	return &Data{
		ContentType: dataType,
		Raw:         rawData,
	}, err
}

type Resource string

func (r Resource) Decode() (*Data, error) {
	url, err := url.Parse(string(r))

	if err != nil {
		return nil, err
	}

	if url.Scheme == "data" {
		return DecodeData(url)
	}

	return nil, nil
}

type Attribute struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
}
type Metadata struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes"`

	Image        Resource `json:"image"`
	ImageData    Resource `json:"image_data"`
	AnimationURL Resource `json:"animation_url"`
}

func ParseMetadata(b []byte) (*Metadata, error) {
	var metadata Metadata
	err := json.Unmarshal(b, &metadata)
	return &metadata, err
}
