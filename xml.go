package main

import (
	"encoding/xml"
)

type XMLImage struct {
	Href string `xml:"href,attr"`
}

func CollectImages(data []byte) ([]XMLImage, error) {
	type Imgs struct {
		XMLName xml.Name   `xml:"svg"`
		Images  []XMLImage `xml:"image"`
	}

	var imgs Imgs
	err := xml.Unmarshal(data, &imgs)

	if err != nil {
		return nil, err
	}

	return imgs.Images, nil
}
