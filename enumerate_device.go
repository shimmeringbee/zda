package zda

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/retry"
	"github.com/shimmeringbee/zigbee"
	"time"
)

const EnumerateDeviceQueueSize = 50
const EnumerationConcurrency = 4
const MaximumEnumerationTime = 3 * time.Minute
const DefaultNetworkTimeout = 5 * time.Second
const DefaultNetworkRetries = 5

type ZigbeeEnumerateDevice struct {
	gateway *ZigbeeGateway

	queue     chan da.Device
	queueStop chan bool
}

func (z *ZigbeeEnumerateDevice) Enumerate(ctx context.Context, device da.Device) error {
	if da.DeviceDoesNotBelongToGateway(z.gateway, device) {
		return da.DeviceDoesNotBelongToGatewayError
	}

	if !device.HasCapability(capabilities.EnumerateDeviceFlag) {
		return da.DeviceDoesNotHaveCapability
	}

	select {
	case z.queue <- device:
		z.gateway.sendEvent(capabilities.EnumerateDeviceStart{
			Device: device,
		})
	default:
		return fmt.Errorf("unable to queue enumeration request, likely channel full")
	}

	return nil
}

func (z *ZigbeeEnumerateDevice) Start() {
	z.queue = make(chan da.Device, EnumerateDeviceQueueSize)
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
		case device := <-z.queue:
			if err := z.enumerateDevice(device); err != nil {
				fmt.Printf("failed to enumerate device: %s: %s", device.Identifier, err)

				z.gateway.sendEvent(capabilities.EnumerateDeviceFailure{
					Device: device,
					Error:  err,
				})
			} else {
				z.gateway.sendEvent(capabilities.EnumerateDeviceSuccess{
					Device: device,
				})
			}
		}
	}
}

func (z *ZigbeeEnumerateDevice) Stop() {
	for i := 0; i < EnumerationConcurrency; i++ {
		z.queueStop <- true
	}
}

func (z *ZigbeeEnumerateDevice) enumerateDevice(device da.Device) error {
	pCtx, cancel := context.WithTimeout(context.Background(), MaximumEnumerationTime)
	defer cancel()

	zDevice, found := z.gateway.getDevice(device.Identifier)
	if !found {
		return fmt.Errorf("device not found in gateway though gateway is correct, may have left network: %s", device.Identifier)
	}

	if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		nd, err := z.gateway.provider.QueryNodeDescription(ctx, device.Identifier.(zigbee.IEEEAddress))

		if err == nil {
			zDevice.mutex.Lock()
			zDevice.nodeDesc = nd
			zDevice.mutex.Unlock()
		}

		return err
	}); err != nil {
		return err
	}

	if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
		eps, err := z.gateway.provider.QueryNodeEndpoints(ctx, device.Identifier.(zigbee.IEEEAddress))

		if err == nil {
			zDevice.mutex.Lock()
			zDevice.endpoints = eps
			zDevice.mutex.Unlock()
		}

		return err
	}); err != nil {
		return err
	}

	zDevice.mutex.RLock()
	endpoints := zDevice.endpoints
	zDevice.mutex.RUnlock()

	for _, endpoint := range endpoints {
		if err := retry.Retry(pCtx, DefaultNetworkTimeout, DefaultNetworkRetries, func(ctx context.Context) error {
			epd, err := z.gateway.provider.QueryNodeEndpointDescription(ctx, device.Identifier.(zigbee.IEEEAddress), endpoint)

			if err == nil {
				zDevice.mutex.Lock()
				zDevice.endpointDescriptions[endpoint] = epd
				zDevice.mutex.Unlock()
			}

			return err
		}); err != nil {
			return err
		}
	}

	return nil
}
