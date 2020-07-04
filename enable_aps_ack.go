package zda

import (
	"context"
	"github.com/shimmeringbee/zigbee"
)

const XiaomiManufacturer = zigbee.ManufacturerCode(0x115f)
const LegrandManufacturer = zigbee.ManufacturerCode(0x1021)

func (z *ZigbeeGateway) enableAPSACK(ctx context.Context, ine internalNodeEnumeration) error {
	iNode := ine.node

	iNode.mutex.Lock()
	defer iNode.mutex.Unlock()

	switch iNode.nodeDesc.ManufacturerCode {
	case XiaomiManufacturer, LegrandManufacturer:
		iNode.supportsAPSAck = false
	default:
		iNode.supportsAPSAck = true
	}

	return nil
}
