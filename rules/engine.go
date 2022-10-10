package rules

import (
	"io"
	"io/fs"
)

type Engine struct{}

type Capabilities struct {
	Add    map[string]interface{}
	Remove map[string]interface{}
}

type Actions struct {
	Capabilities Capabilities
}

type Rule struct {
	Description string
	Filter      string
	Actions     Actions
	Children    []Rule
}

type InputProductData struct {
	Name         string
	Manufacturer string
	Version      string
	Serial       string
}

type InputNode struct {
	ManufacturerCode uint16
	Type             string
}

type InputEndpoint struct {
	ID          uint8
	ProfileID   uint16
	DeviceID    uint16
	InClusters  []uint16
	OutClusters []uint16
}

type Input struct {
	Product  InputProductData
	Node     InputNode
	Endpoint InputEndpoint
}

type Output struct {
	Capabilities map[string]interface{}
}

func (e Engine) LoadString(_ string) error {
	panic("not yet implemented")
}

func (e Engine) LoadReader(_ io.Reader) error {
	panic("not yet implemented")
}

func (e Engine) LoadFS(_ fs.FS) error {
	panic("not yet implemented")
}

func (e Engine) Execute(_ Input) (Output, error) {
	panic("not yet implemented")
}
