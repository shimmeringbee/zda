package rules

import "github.com/shimmeringbee/zigbee"

type Filter struct {
	ManufacturerCode *zigbee.ManufacturerCode
	ManufacturerName *string
	ProductName      *string
	Endpoint         *zigbee.Endpoint
	ClusterID        *zigbee.ClusterID
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

	if f.Endpoint != nil && *f.Endpoint != m.Endpoint {
		return false
	}

	if f.ClusterID != nil && *f.ClusterID != m.ClusterID {
		return false
	}

	return true
}

type MatchData struct {
	ManufacturerCode zigbee.ManufacturerCode
	ManufacturerName string
	ProductName      string
	Endpoint         zigbee.Endpoint
	ClusterID        zigbee.ClusterID
}
