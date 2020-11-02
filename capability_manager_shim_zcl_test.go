package zda

import (
	"context"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/communicator"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestCapabilityManager_initSupervisor_ZCL_RegisterCommandLibrary(t *testing.T) {
	t.Run("provides the ZCL command registry to the library provided", func(t *testing.T) {
		expectedCr := zcl.NewCommandRegistry()

		m := CapabilityManager{commandRegistry: expectedCr}
		s := m.initSupervisor()

		called := false

		register := func(cr *zcl.CommandRegistry) {
			called = true
			assert.Equal(t, expectedCr, cr)
		}

		s.ZCL().RegisterCommandLibrary(register)

		assert.True(t, called)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_ReadAttributes(t *testing.T) {
	t.Run("reads attributes from the device via ZCL, filling in missing values", func(t *testing.T) {
		mzcl := &mockZclGlobalCommunicator{}
		defer mzcl.AssertExpectations(t)

		nt, iNode, iDev := generateNodeTableWithData(1)

		m := CapabilityManager{zclGlobalCommunicator: mzcl, nodeTable: nt}
		s := m.initSupervisor()

		clusterId := zigbee.ClusterID(0x0001)
		attributes := []zcl.AttributeID{0x0001, 0x0002}
		endpoint := iDev[0].endpoints[0]

		device := internalDeviceToZDADevice(iDev[0])

		mzcl.On("ReadAttributes", mock.Anything, iNode.ieeeAddress, iNode.supportsAPSAck, clusterId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, endpoint, uint8(0), attributes).Return([]global.ReadAttributeResponseRecord{
			{
				Identifier:    0x0001,
				Status:        0,
				DataTypeValue: nil,
			},
			{
				Identifier:    0x0002,
				Status:        0,
				DataTypeValue: nil,
			},
		}, nil)

		records, err := s.ZCL().ReadAttributes(context.TODO(), device, endpoint, clusterId, attributes)
		assert.NoError(t, err)

		assert.Equal(t, zcl.AttributeID(0x0001), records[0x0001].Identifier)
		assert.Equal(t, zcl.AttributeID(0x0002), records[0x0002].Identifier)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_WriteAttributes(t *testing.T) {
	t.Run("write attributes to the device via ZCL, filling in missing values", func(t *testing.T) {
		mzcl := &mockZclGlobalCommunicator{}
		defer mzcl.AssertExpectations(t)

		nt, iNode, iDev := generateNodeTableWithData(1)

		m := CapabilityManager{zclGlobalCommunicator: mzcl, nodeTable: nt}
		s := m.initSupervisor()

		clusterId := zigbee.ClusterID(0x0001)
		attributes := map[zcl.AttributeID]zcl.AttributeDataTypeValue{
			0x0001: {},
			0x0002: {},
		}
		endpoint := iDev[0].endpoints[0]

		device := internalDeviceToZDADevice(iDev[0])

		mzcl.On("WriteAttributes", mock.Anything, iNode.ieeeAddress, iNode.supportsAPSAck, clusterId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, endpoint, uint8(0), attributes).Return([]global.WriteAttributesResponseRecord{
			{
				Identifier: 0x0002,
				Status:     1,
			},
		}, nil)

		records, err := s.ZCL().WriteAttributes(context.TODO(), device, endpoint, clusterId, attributes)
		assert.NoError(t, err)

		assert.Equal(t, zcl.AttributeID(0x0001), records[0x0001].Identifier)
		assert.Equal(t, zcl.AttributeID(0x0002), records[0x0002].Identifier)

		assert.Equal(t, uint8(0), records[0x0001].Status)
		assert.Equal(t, uint8(1), records[0x0002].Status)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_ConfigureReporting(t *testing.T) {
	t.Run("configure reporting for an attribute on a cluster from the device via ZCL, filling in missing values", func(t *testing.T) {
		mzcl := &mockZclGlobalCommunicator{}
		defer mzcl.AssertExpectations(t)

		nt, iNode, iDev := generateNodeTableWithData(1)

		m := CapabilityManager{zclGlobalCommunicator: mzcl, nodeTable: nt}
		s := m.initSupervisor()

		clusterId := zigbee.ClusterID(0x0001)
		attribute := zcl.AttributeID(1)
		endpoint := iDev[0].endpoints[0]

		device := internalDeviceToZDADevice(iDev[0])

		mzcl.On("ConfigureReporting", mock.Anything, iNode.ieeeAddress, iNode.supportsAPSAck, clusterId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, endpoint, uint8(0), attribute, zcl.TypeBoolean, uint16(0), uint16(60), nil).Return(nil)

		err := s.ZCL().ConfigureReporting(context.TODO(), device, endpoint, clusterId, attribute, zcl.TypeBoolean, 0, 60, nil)
		assert.NoError(t, err)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_Bind(t *testing.T) {
	t.Run("binds a cluster on a device, filling in missing values", func(t *testing.T) {
		mnb := &mockNodeBinder{}
		defer mnb.AssertExpectations(t)

		nt, iNode, iDev := generateNodeTableWithData(1)

		m := CapabilityManager{zigbeeNodeBinder: mnb, nodeTable: nt}
		s := m.initSupervisor()

		clusterId := zigbee.ClusterID(0x0001)
		endpoint := iDev[0].endpoints[0]

		device := internalDeviceToZDADevice(iDev[0])

		mnb.On("BindNodeToController", mock.Anything, iNode.ieeeAddress, DefaultGatewayHomeAutomationEndpoint, endpoint, clusterId).Return(nil)

		err := s.ZCL().Bind(context.TODO(), device, endpoint, clusterId)
		assert.NoError(t, err)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_SendCommand(t *testing.T) {
	t.Run("binds a cluster on a device, filling in missing values", func(t *testing.T) {
		mzcr := &mockZclCommunicatorRequests{}
		defer mzcr.AssertExpectations(t)

		nt, iNode, iDev := generateNodeTableWithData(1)

		m := CapabilityManager{zclCommunicatorRequests: mzcr, nodeTable: nt}
		s := m.initSupervisor()

		clusterId := zigbee.ClusterID(0x0001)
		endpoint := iDev[0].endpoints[0]
		cmd := struct{}{}

		device := internalDeviceToZDADevice(iDev[0])

		mzcr.On("Request", mock.Anything, iNode.ieeeAddress, iNode.supportsAPSAck, zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ClientToServer,
			TransactionSequence: 0,
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           clusterId,
			SourceEndpoint:      DefaultGatewayHomeAutomationEndpoint,
			DestinationEndpoint: endpoint,
			Command:             cmd,
		}).Return(nil)

		err := s.ZCL().SendCommand(context.TODO(), device, endpoint, clusterId, cmd)
		assert.NoError(t, err)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_Listen(t *testing.T) {
	t.Run("creates a new match and adds the callback on listen", func(t *testing.T) {
		mczz := &mockZclCommunicatorCallbacks{}
		defer mczz.AssertExpectations(t)

		m := CapabilityManager{zclCommunicatorCallbacks: mczz}
		s := m.initSupervisor()

		filter := func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
			return false
		}

		cb := func(device Device, message zcl.Message) {}

		match := communicator.Match{}

		mczz.On("NewMatch", mock.AnythingOfType("communicator.Matcher"), mock.Anything).Return(match)
		mczz.On("AddCallback", match)

		s.ZCL().Listen(filter, cb)
	})

	t.Run("the callback function correctly maps from ZCL to ZDA device", func(t *testing.T) {
		mczz := &mockZclCommunicatorCallbacks{}
		defer mczz.AssertExpectations(t)

		nt, iNode, iDevs := generateNodeTableWithData(1)

		m := CapabilityManager{zclCommunicatorCallbacks: mczz, nodeTable: nt}
		s := m.initSupervisor()

		filter := func(address zigbee.IEEEAddress, appMsg zigbee.ApplicationMessage, zclMessage zcl.Message) bool {
			return false
		}

		called := false

		expectedMsg := zcl.Message{
			FrameType:           zcl.FrameLocal,
			Direction:           zcl.ServerToClient,
			TransactionSequence: 0,
			Manufacturer:        zigbee.NoManufacturer,
			ClusterID:           zigbee.ClusterID(0),
			SourceEndpoint:      zigbee.Endpoint(0),
			DestinationEndpoint: 0,
			Command:             nil,
		}

		expectedDevice := internalDeviceToZDADevice(iDevs[0])

		cb := func(device Device, message zcl.Message) {
			called = true
			assert.Equal(t, expectedDevice, device)
			assert.Equal(t, expectedMsg, message)
		}

		match := communicator.Match{}

		mczz.On("NewMatch", mock.AnythingOfType("communicator.Matcher"), mock.Anything).Return(match)
		mczz.On("AddCallback", match)

		s.ZCL().Listen(filter, cb)

		wrappedCb, ok := mczz.Calls[0].Arguments[1].(func(source communicator.MessageWithSource))
		assert.True(t, ok)

		wrappedCb(communicator.MessageWithSource{
			SourceAddress: iNode.ieeeAddress,
			Message:       expectedMsg,
		})

		assert.True(t, called)
	})
}

func TestCapabilityManager_initSupervisor_ZCL_WaitForMessage(t *testing.T) {
	t.Run("creates a new match and adds the callback on listen", func(t *testing.T) {
		mczz := &mockZclCommunicatorCallbacks{}
		defer mczz.AssertExpectations(t)

		m := CapabilityManager{zclCommunicatorCallbacks: mczz}
		s := m.initSupervisor()

		match := communicator.Match{}

		mczz.On("NewMatch", mock.AnythingOfType("communicator.Matcher"), mock.Anything).Return(match)
		mczz.On("AddCallback", match)
		mczz.On("RemoveCallback", match)

		device := Device{
			Identifier: IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress()},
		}

		endpoint := zigbee.Endpoint(1)
		cluster := zigbee.ClusterID(0x0010)
		commandId := zcl.CommandIdentifier(0x20)

		expectedMessage := zcl.Message{}

		go func() {
			time.Sleep(5 * time.Millisecond)

			matcher := mczz.Calls[0].Arguments.Get(0).(communicator.Matcher)

			assert.True(t, matcher(device.Identifier.IEEEAddress, zigbee.ApplicationMessage{}, zcl.Message{
				ClusterID:         cluster,
				SourceEndpoint:    endpoint,
				CommandIdentifier: commandId,
			}))

			assert.False(t, matcher(device.Identifier.IEEEAddress, zigbee.ApplicationMessage{}, zcl.Message{
				ClusterID:         cluster,
				SourceEndpoint:    0,
				CommandIdentifier: commandId,
			}))

			callback := mczz.Calls[0].Arguments.Get(1).(func(source communicator.MessageWithSource))
			callback(communicator.MessageWithSource{
				SourceAddress: 0,
				Message:       expectedMessage,
			})
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		actualMessage, err := s.ZCL().WaitForMessage(ctx, device, endpoint, cluster, commandId)
		assert.NoError(t, err)
		assert.Equal(t, expectedMessage, actualMessage)
	})

}
