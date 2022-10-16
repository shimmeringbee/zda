package zda

import (
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"golang.org/x/sync/semaphore"
	"math"
	"sync"
)

type productData struct {
	manufacturer string
	product      string
	version      string
	serial       string
}

type endpointDetails struct {
	description        zigbee.EndpointDescription
	productInformation productData
	rulesOutput        rules.Output
}

type inventory struct {
	description *zigbee.NodeDescription
	endpoints   map[zigbee.Endpoint]endpointDetails
}

func (i inventory) toRulesInput() rules.Input {
	ri := rules.Input{
		Node: rules.InputNode{
			ManufacturerCode: uint16(i.description.ManufacturerCode),
			Type:             i.description.LogicalType.String(),
		},
		Product:  make(map[uint8]rules.InputProductData),
		Endpoint: make(map[uint8]rules.InputEndpoint),
	}

	for id, details := range i.endpoints {
		ri.Product[uint8(id)] = rules.InputProductData{
			Name:         details.productInformation.product,
			Manufacturer: details.productInformation.manufacturer,
			Version:      details.productInformation.version,
			Serial:       details.productInformation.serial,
		}

		var inClusters []uint16
		var outClusters []uint16

		for _, cid := range details.description.InClusterList {
			inClusters = append(inClusters, uint16(cid))
		}

		for _, cid := range details.description.OutClusterList {
			outClusters = append(outClusters, uint16(cid))
		}

		ri.Endpoint[uint8(id)] = rules.InputEndpoint{
			ID:          uint8(id),
			ProfileID:   uint16(details.description.ProfileID),
			DeviceID:    details.description.DeviceID,
			InClusters:  inClusters,
			OutClusters: outClusters,
		}
	}

	return ri
}

type node struct {
	// Immutable data.
	address zigbee.IEEEAddress
	m       *sync.RWMutex

	// Thread safe data.
	sequence chan uint8

	// Mutable data, obtain lock first.
	device map[uint8]*device

	useAPSAck bool

	// Enumeration data.
	enumerationSem *semaphore.Weighted
	inventory      inventory
}

func makeTransactionSequence() chan uint8 {
	ch := make(chan uint8, math.MaxUint8)

	for i := uint8(0); i < math.MaxUint8; i++ {
		ch <- i
	}

	return ch
}

func (n *node) nextTransactionSequence() uint8 {
	nextSeq := <-n.sequence
	n.sequence <- nextSeq

	return nextSeq
}

func (n *node) _nextDeviceSubIdentifier() uint8 {
	for i := uint8(0); i < math.MaxUint8; i++ {
		if _, found := n.device[i]; !found {
			return i
		}
	}

	return math.MaxUint8
}
