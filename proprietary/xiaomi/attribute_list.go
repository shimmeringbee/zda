package xiaomi

import (
	"fmt"
	"github.com/shimmeringbee/bytecodec"
	"github.com/shimmeringbee/zcl"
)

type Attribute struct {
	Id        uint8
	Attribute *zcl.AttributeDataTypeValue
}

type AttributeList []Attribute

func ParseAttributeList(xiaomiBytes []byte) (map[uint8]Attribute, error) {
	var xal AttributeList

	if err := bytecodec.Unmarshal(xiaomiBytes, &xal); err != nil {
		return nil, fmt.Errorf("failed to parse Xiaomi attribute list: %w", err)
	}

	ret := make(map[uint8]Attribute)

	for _, attribute := range xal {
		ret[attribute.Id] = attribute
	}

	return ret, nil
}
