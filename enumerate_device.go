package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/callbacks"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zigbee"
	"sort"
	"time"
)

const EnumerateDeviceQueueSize = 50
const EnumerationConcurrency = 4
const MaximumEnumerationTime = 1 * time.Minute
const DefaultNetworkTimeout = 1500 * time.Millisecond
const DefaultNetworkRetries = 5

type ZigbeeEnumerateDevice struct {
	gateway           da.Gateway
	nodeTable         nodeTable
	eventSender       eventSender
	nodeQuerier       zigbee.NodeQuerier
	internalCallbacks callbacks.AdderCaller

	queue     chan *internalNode
	queueStop chan bool
}

func (z *ZigbeeEnumerateDevice) NodeJoinCallback(ctx context.Context, join internalNodeJoin) error {
	return z.queueEnumeration(ctx, join.node)
}

func (z *ZigbeeEnumerateDevice) Enumerate(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.EnumerateDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	iDev := z.nodeTable.getDevice(device.Identifier().(IEEEAddressWithSubIdentifier))

	if iDev != nil {
		return z.queueEnumeration(ctx, iDev.node)
	} else {
		return fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}
}

func (z *ZigbeeEnumerateDevice) queueEnumeration(ctx context.Context, node *internalNode) error {
	select {
	case z.queue <- node:
		node.mutex.RLock()
		for _, device := range node.getDevices() {
			z.eventSender.sendEvent(capabilities.EnumerateDeviceStart{
				Device: device.toDevice(z.gateway),
			})
		}
		node.mutex.RUnlock()

		return nil
	default:
		return fmt.Errorf("unable to queue enumeration request, likely channel full")
	}
}

func (z *ZigbeeEnumerateDevice) Init() {
	z.internalCallbacks.Add(z.NodeJoinCallback)
}

func (z *ZigbeeEnumerateDevice) Start() {
	z.queue = make(chan *internalNode, EnumerateDeviceQueueSize)
	z.queueStop = make(chan bool, EnumerationConcurrency)

	for i := 0; i < EnumerationConcurrency; i++ {
		go z.enumerateLoop()
	}
}

func (z *ZigbeeEnumerateDevice) enumerateLoop() {
	for {
		select {
		case <-z.queueStop:
			return
		case node := <-z.queue:
			if err := z.enumerateNode(node); err != nil {
				fmt.Printf("failed to enumerate node: %s: %s", node.ieeeAddress, err)

				node.mutex.RLock()
				for _, device := range node.getDevices() {
					z.eventSender.sendEvent(capabilities.EnumerateDeviceFailure{
						Device: device.toDevice(z.gateway),
						Error:  err,
					})
				}
				node.mutex.RUnlock()
			} else {
				node.mutex.RLock()
				for _, device := range node.getDevices() {
					z.eventSender.sendEvent(capabilities.EnumerateDeviceSuccess{
						Device: device.toDevice(z.gateway),
					})
				}
				node.mutex.RUnlock()
			}
		}
	}
}

func (z *ZigbeeEnumerateDevice) Stop() {
	for i := 0; i < EnumerationConcurrency; i++ {
		z.queueStop <- true
	}
}

func (z *ZigbeeEnumerateDevice) enumerateNode(iNode *internalNode) error {
	ctx, cancel := context.WithTimeout(context.Background(), MaximumEnumerationTime)
	defer cancel()

	if err := z.enumerateNodeDescription(ctx, iNode); err != nil {
		return err
	}

	if err := z.enumerateNodeEndpoints(ctx, iNode); err != nil {
		return err
	}

	iNode.mutex.RLock()
	endpoints := iNode.endpoints
	iNode.mutex.RUnlock()

	for _, endpoint := range endpoints {
		if err := z.enumerateNodeEndpointDescription(ctx, iNode, endpoint); err != nil {
			return err
		}
	}

	z.removeMissingEndpointDescriptions(iNode)
	z.allocateEndpointsToDevices(iNode)
	z.deallocateDevicesFromMissingEndpoints(iNode)

	if err := z.internalCallbacks.Call(ctx, internalNodeEnumeration{node: iNode}); err != nil {
		return err
	}

	return nil
}

func (z *ZigbeeEnumerateDevice) enumerateNodeDescription(pCtx context.Context, iNode *internalNode) error {
	return retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		nd, err := z.nodeQuerier.QueryNodeDescription(ctx, iNode.ieeeAddress)

		if err == nil {
			iNode.mutex.Lock()
			iNode.nodeDesc = nd
			iNode.mutex.Unlock()
		}

		return err
	})
}

func (z *ZigbeeEnumerateDevice) enumerateNodeEndpoints(pCtx context.Context, iNode *internalNode) error {
	return retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		eps, err := z.nodeQuerier.QueryNodeEndpoints(ctx, iNode.ieeeAddress)

		if err == nil {
			iNode.mutex.Lock()
			iNode.endpoints = eps
			iNode.mutex.Unlock()
		}

		return err
	})
}

func (z *ZigbeeEnumerateDevice) enumerateNodeEndpointDescription(pCtx context.Context, iNode *internalNode, endpoint zigbee.Endpoint) error {
	return retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		epd, err := z.nodeQuerier.QueryNodeEndpointDescription(ctx, iNode.ieeeAddress, endpoint)

		if err == nil {
			iNode.mutex.Lock()
			iNode.endpointDescriptions[endpoint] = epd
			iNode.mutex.Unlock()
		}

		return err
	})
}

func (z *ZigbeeEnumerateDevice) allocateEndpointsToDevices(iNode *internalNode) {
	iNode.mutex.Lock()
	endpointDescriptions := iNode.endpointDescriptions
	iNode.mutex.Unlock()

	var endpoints []zigbee.Endpoint

	for endpoint := range endpointDescriptions {
		endpoints = append(endpoints, endpoint)
	}

	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i] < endpoints[j]
	})

	for _, endpoint := range endpoints {
		desc := endpointDescriptions[endpoint]
		iDev := z.findDeviceWithDeviceId(iNode, desc.DeviceID, desc.DeviceVersion)

		iDev.mutex.Lock()
		if !isEndpointInSlice(iDev.endpoints, endpoint) {
			iDev.endpoints = append(iDev.endpoints, endpoint)
		}
		iDev.mutex.Unlock()
	}
}

func (z *ZigbeeEnumerateDevice) removeMissingEndpointDescriptions(iNode *internalNode) {
	iNode.mutex.Lock()

	for endpoint := range iNode.endpointDescriptions {
		if !isEndpointInSlice(iNode.endpoints, endpoint) {
			delete(iNode.endpointDescriptions, endpoint)
		}
	}

	iNode.mutex.Unlock()
}

func (z *ZigbeeEnumerateDevice) deallocateDevicesFromMissingEndpoints(iNode *internalNode) {
	iNode.mutex.Lock()
	devices := iNode.devices
	deviceCount := len(devices)
	iNode.mutex.Unlock()

	for _, iDev := range devices {
		iDev.mutex.Lock()

		existingEndpoints := iDev.endpoints
		iDev.endpoints = []zigbee.Endpoint{}

		for _, endpoint := range existingEndpoints {
			endpointDesc, found := iNode.endpointDescriptions[endpoint]

			if found && iDev.deviceID == endpointDesc.DeviceID {
				iDev.endpoints = append(iDev.endpoints, endpoint)
			}
		}

		toDelete := len(iDev.endpoints) == 0
		iDev.mutex.Unlock()

		if toDelete && deviceCount > 1 {
			z.nodeTable.removeDevice(iDev.generateIdentifier())
			deviceCount--
		}
	}
}

func (z *ZigbeeEnumerateDevice) findDeviceWithDeviceId(iNode *internalNode, deviceId uint16, deviceVersion uint8) *internalDevice {
	iNode.mutex.Lock()
	nodeDevices := iNode.devices
	iNode.mutex.Unlock()

	for _, iDev := range nodeDevices {
		iDev.mutex.RLock()

		if iDev.deviceID == deviceId {
			iDev.mutex.RUnlock()
			return iDev
		}

		iDev.mutex.RUnlock()
	}

	for _, iDev := range nodeDevices {
		iDev.mutex.Lock()

		if iDev.deviceID == 0 {
			iDev.deviceID = deviceId
			iDev.deviceVersion = deviceVersion
			iDev.mutex.Unlock()
			return iDev
		}

		iDev.mutex.Unlock()
	}

	iDev := z.nodeTable.createNextDevice(iNode.ieeeAddress)

	iDev.mutex.Lock()
	iDev.deviceID = deviceId
	iDev.deviceVersion = deviceVersion
	iDev.mutex.Unlock()
	return iDev
}
