package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/retry"
	"time"
)

const EnumerateDeviceQueueSize = 50
const EnumerationConcurrency = 4
const MaximumEnumerationTime = 3 * time.Minute
const DefaultNetworkTimeout = 5 * time.Second
const DefaultNetworkRetries = 5

type ZigbeeEnumerateDevice struct {
	gateway *ZigbeeGateway

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

	iDev, found := z.gateway.getDevice(device.Identifier)

	if found {
		return z.queueEnumeration(ctx, iDev.node)
	} else {
		return fmt.Errorf("unable to find zigbee device in zda, likely old device")
	}
}

func (z *ZigbeeEnumerateDevice) queueEnumeration(ctx context.Context, node *internalNode) error {
	select {
	case z.queue <- node:
		for _, device := range node.getDevices() {
			z.gateway.sendEvent(capabilities.EnumerateDeviceStart{
				Device: device.device,
			})
		}

		return nil
	default:
		return fmt.Errorf("unable to queue enumeration request, likely channel full")
	}
}

func (z *ZigbeeEnumerateDevice) Init() {
	z.gateway.callbacks.Add(z.NodeJoinCallback)
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

				for _, device := range node.getDevices() {
					z.gateway.sendEvent(capabilities.EnumerateDeviceFailure{
						Device: device.device,
						Error:  err,
					})
				}
			} else {
				for _, device := range node.getDevices() {
					z.gateway.sendEvent(capabilities.EnumerateDeviceSuccess{
						Device: device.device,
					})
				}
			}
		}
	}
}

func (z *ZigbeeEnumerateDevice) Stop() {
	for i := 0; i < EnumerationConcurrency; i++ {
		z.queueStop <- true
	}
}

func (z *ZigbeeEnumerateDevice) enumerateNode(node *internalNode) error {
	pCtx, cancel := context.WithTimeout(context.Background(), MaximumEnumerationTime)
	defer cancel()

	if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		nd, err := z.gateway.provider.QueryNodeDescription(ctx, node.ieeeAddress)

		if err == nil {
			node.mutex.Lock()
			node.nodeDesc = nd
			node.mutex.Unlock()
		}

		return err
	}); err != nil {
		return err
	}

	if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		eps, err := z.gateway.provider.QueryNodeEndpoints(ctx, node.ieeeAddress)

		if err == nil {
			node.mutex.Lock()
			node.endpoints = eps
			node.mutex.Unlock()
		}

		return err
	}); err != nil {
		return err
	}

	node.mutex.RLock()
	endpoints := node.endpoints
	node.mutex.RUnlock()

	for _, endpoint := range endpoints {
		if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
			epd, err := z.gateway.provider.QueryNodeEndpointDescription(ctx, node.ieeeAddress, endpoint)

			if err == nil {
				node.mutex.Lock()
				node.endpointDescriptions[endpoint] = epd
				node.mutex.Unlock()
			}

			return err
		}); err != nil {
			return err
		}
	}

	if err := z.gateway.callbacks.Call(pCtx, internalNodeEnumeration{node: node}); err != nil {
		return err
	}

	return nil
}
