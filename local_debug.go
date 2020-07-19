package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zigbee"
)

const ZigbeeLocalDebugMediaType = "application/vnd.shimmeringbee.zda.localdebug+json"

type ZigbeeLocalDebug struct {
	gateway *ZigbeeGateway
}

type LocalDebugNodeData struct {
	IEEEAddress     string
	NodeDescription zigbee.NodeDescription

	Endpoints            []int
	EndpointDescriptions map[zigbee.Endpoint]zigbee.EndpointDescription

	Devices map[string]LocalDebugDeviceData
}

type LocalDebugDeviceData struct {
	Identifier string

	DeviceId          uint16
	DeviceVersion     uint8
	AssignedEndpoints []int

	ProductName         string
	ProductManufacturer string
}

func (z *ZigbeeLocalDebug) Start(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.LocalDebugFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDev, found := z.gateway.getDevice(device.Identifier)

	if !found {
		return fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}

	iNode := iDev.node

	z.gateway.sendEvent(capabilities.LocalDebugStart{Device: device})

	iNode.mutex.RLock()

	devices := map[string]LocalDebugDeviceData{}

	for id, dev := range iNode.devices {
		dev.mutex.RLock()
		var endpoints []int

		for _, endpoint := range dev.endpoints {
			endpoints = append(endpoints, int(endpoint))
		}

		devices[id.String()] = LocalDebugDeviceData{
			Identifier:          id.String(),
			DeviceId:            dev.deviceID,
			DeviceVersion:       dev.deviceVersion,
			AssignedEndpoints:   endpoints,
			ProductName:         dev.productInformation.Name,
			ProductManufacturer: dev.productInformation.Manufacturer,
		}
		dev.mutex.RUnlock()
	}

	var endpoints []int

	for _, endpoint := range iNode.endpoints {
		endpoints = append(endpoints, int(endpoint))
	}

	debug := LocalDebugNodeData{
		IEEEAddress:          iNode.ieeeAddress.String(),
		NodeDescription:      iNode.nodeDesc,
		Endpoints:            endpoints,
		EndpointDescriptions: iNode.endpointDescriptions,
		Devices:              devices,
	}

	iNode.mutex.RUnlock()

	z.gateway.sendEvent(capabilities.LocalDebugSuccess{Device: device, MediaType: ZigbeeLocalDebugMediaType, Debug: debug})

	return nil
}
