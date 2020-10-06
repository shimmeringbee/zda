package zda

import (
	"context"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zda/rules"
	"time"
)

type deviceConfigShim struct {
	ruleList          *rules.Rule
	capabilityFetcher FetchCapability
	composeDADevice   ComposeDADevice
	nodeTable         nodeTable
}

func (s *deviceConfigShim) Get(d Device, ns string) Config {
	rule := rules.Rule{}

	if s.ruleList != nil {
		md := s.constructMatchData(d)
		rule = *s.ruleList.Match(md)
	}

	return ruleBaseConfig{r: rule, ns: ns}
}

func (s *deviceConfigShim) constructMatchData(d Device) rules.MatchData {
	var product capabilities.ProductInformation

	if hasProductInfoCap := s.capabilityFetcher.Get(capabilities.HasProductInformationFlag); hasProductInfoCap != nil {
		if castCap, castOk := hasProductInfoCap.(capabilities.HasProductInformation); castOk {
			daDevice := s.composeDADevice.Compose(d)
			product, _ = castCap.ProductInformation(context.Background(), daDevice)
		}
	}

	iDev := s.nodeTable.getDevice(d.Identifier)
	if iDev == nil {
		return rules.MatchData{}
	}

	iDev.node.mutex.RLock()
	defer iDev.node.mutex.RUnlock()
	iDev.mutex.RLock()
	defer iDev.mutex.RUnlock()

	return rules.MatchData{
		ManufacturerCode: iDev.node.nodeDesc.ManufacturerCode,
		ManufacturerName: product.Manufacturer,
		ProductName:      product.Name,
		DeviceId:         iDev.deviceID,
	}
}

type ruleBaseConfig struct {
	r  rules.Rule
	ns string
}

func (c ruleBaseConfig) String(k string, d string) string {
	return c.r.StringSetting(c.ns, k, d)
}

func (c ruleBaseConfig) Int(k string, d int) int {
	return c.r.IntSetting(c.ns, k, d)
}

func (c ruleBaseConfig) Float(k string, d float64) float64 {
	return c.r.FloatSetting(c.ns, k, d)
}

func (c ruleBaseConfig) Bool(k string, d bool) bool {
	return c.r.BooleanSetting(c.ns, k, d)
}

func (c ruleBaseConfig) Duration(k string, d time.Duration) time.Duration {
	return c.r.DurationSetting(c.ns, k, d)
}
