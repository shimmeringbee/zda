package zda

import (
	"context"
)

func (z *ZigbeeGateway) enableAPSACK(ctx context.Context, ine internalNodeEnumeration) error {
	iNode := ine.node

	iNode.mutex.Lock()
	defer iNode.mutex.Unlock()

	iNode.supportsAPSAck = false

	return nil
}
