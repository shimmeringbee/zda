package rules

import "github.com/shimmeringbee/zigbee"

type Filter struct {
	ManufacturerCode *zigbee.ManufacturerCode
	ManufacturerName *string
	ProductName      *string
	DeviceId         *uint16
}

func (f Filter) matches(m MatchData) bool {
	if f.ManufacturerCode != nil && *f.ManufacturerCode != m.ManufacturerCode {
		return false
	}

	if f.ManufacturerName != nil && *f.ManufacturerName != m.ManufacturerName {
		return false
	}

	if f.ProductName != nil && *f.ProductName != m.ProductName {
		return false
	}

	if f.DeviceId != nil && *f.DeviceId != m.DeviceId {
		return false
	}

	return true
}

type MatchData struct {
	ManufacturerCode zigbee.ManufacturerCode
	ManufacturerName string
	ProductName      string
	DeviceId         uint16
}
