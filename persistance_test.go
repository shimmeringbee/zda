package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestZigbeeGateway_SaveState(t *testing.T) {
	t.Run("populates a state with devices that exist", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()
		node, _ := zgw.nodeTable.createNode(ieee)
		node.endpoints = []zigbee.Endpoint{0x01, 0x02}
		node.endpointDescriptions = map[zigbee.Endpoint]zigbee.EndpointDescription{
			0x01: {
				Endpoint:       0x01,
				ProfileID:      zigbee.ProfileHomeAutomation,
				DeviceID:       0x01,
				DeviceVersion:  0x01,
				InClusterList:  []zigbee.ClusterID{0x0101},
				OutClusterList: []zigbee.ClusterID{0x0102},
			},
			0x02: {
				Endpoint:       0x02,
				ProfileID:      zigbee.ProfileCommercialBuildingAutomation,
				DeviceID:       0x02,
				DeviceVersion:  0x02,
				InClusterList:  []zigbee.ClusterID{0x0201},
				OutClusterList: []zigbee.ClusterID{0x0202},
			},
		}

		devOne := zgw.nodeTable.createNextDevice(ieee)
		devOne.capabilities = append(devOne.capabilities, da.Capability(0xff01))
		devOne.deviceID = 0x01
		devOne.deviceVersion = 0x01
		devOne.endpoints = []zigbee.Endpoint{0x01}

		devTwo := zgw.nodeTable.createNextDevice(ieee)
		devTwo.capabilities = append(devTwo.capabilities, da.Capability(0xff02))
		devTwo.deviceID = 0x02
		devTwo.deviceVersion = 0x02
		devTwo.endpoints = []zigbee.Endpoint{0x02}

		actualState := zgw.SaveState()

		expectedState := State{
			Nodes: map[zigbee.IEEEAddress]StateNode{
				ieee: {
					Devices: map[uint8]StateDevice{
						0x00: {
							DeviceID:      0x01,
							DeviceVersion: 0x01,
							Endpoints:     []zigbee.Endpoint{0x01},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff01)},
						},
						0x01: {
							DeviceID:      0x02,
							DeviceVersion: 0x02,
							Endpoints:     []zigbee.Endpoint{0x02},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff02)},
						},
					},
					Endpoints: []zigbee.EndpointDescription{
						{
							Endpoint:       0x01,
							ProfileID:      zigbee.ProfileHomeAutomation,
							DeviceID:       0x01,
							DeviceVersion:  0x01,
							InClusterList:  []zigbee.ClusterID{0x0101},
							OutClusterList: []zigbee.ClusterID{0x0102},
						},
						{
							Endpoint:       0x02,
							ProfileID:      zigbee.ProfileCommercialBuildingAutomation,
							DeviceID:       0x02,
							DeviceVersion:  0x02,
							InClusterList:  []zigbee.ClusterID{0x0201},
							OutClusterList: []zigbee.ClusterID{0x0202},
						},
					},
				},
			},
		}

		assert.Equal(t, expectedState, actualState)
	})
}

func TestZigbeeGateway_LoadState(t *testing.T) {
	t.Run("correctly loads nodes from a state", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		expectedState := State{
			Nodes: map[zigbee.IEEEAddress]StateNode{
				ieee: {
					Devices: map[uint8]StateDevice{
						0x00: {
							DeviceID:      0x01,
							DeviceVersion: 0x01,
							Endpoints:     []zigbee.Endpoint{0x01},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff01)},
						},
						0x01: {
							DeviceID:      0x02,
							DeviceVersion: 0x02,
							Endpoints:     []zigbee.Endpoint{0x02},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff02)},
						},
					},
					Endpoints: []zigbee.EndpointDescription{
						{
							Endpoint:       0x01,
							ProfileID:      zigbee.ProfileHomeAutomation,
							DeviceID:       0x01,
							DeviceVersion:  0x01,
							InClusterList:  []zigbee.ClusterID{0x0101},
							OutClusterList: []zigbee.ClusterID{0x0102},
						},
						{
							Endpoint:       0x02,
							ProfileID:      zigbee.ProfileCommercialBuildingAutomation,
							DeviceID:       0x02,
							DeviceVersion:  0x02,
							InClusterList:  []zigbee.ClusterID{0x0201},
							OutClusterList: []zigbee.ClusterID{0x0202},
						},
					},
				},
			},
		}

		zgw.LoadState(expectedState)

		actualState := zgw.SaveState()

		assert.Equal(t, expectedState, actualState)
	})

	t.Run("correctly raises events for loaded state", func(t *testing.T) {
		zgw, mockProvider, stop := NewTestZigbeeGateway()
		mockProvider.On("ReadEvent", mock.Anything).Return(nil, nil).Maybe()
		mockProvider.On("RegisterAdapterEndpoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		zgw.Start()
		defer stop(t)

		ieee := zigbee.GenerateLocalAdministeredIEEEAddress()

		expectedState := State{
			Nodes: map[zigbee.IEEEAddress]StateNode{
				ieee: {
					Devices: map[uint8]StateDevice{
						0x00: {
							DeviceID:      0x01,
							DeviceVersion: 0x01,
							Endpoints:     []zigbee.Endpoint{0x01},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff01)},
						},
						0x01: {
							DeviceID:      0x02,
							DeviceVersion: 0x02,
							Endpoints:     []zigbee.Endpoint{0x02},
							Capabilities:  []da.Capability{da.Capability(1), da.Capability(0xff00), da.Capability(0xff02)},
						},
					},
					Endpoints: []zigbee.EndpointDescription{
						{
							Endpoint:       0x01,
							ProfileID:      zigbee.ProfileHomeAutomation,
							DeviceID:       0x01,
							DeviceVersion:  0x01,
							InClusterList:  []zigbee.ClusterID{0x0101},
							OutClusterList: []zigbee.ClusterID{0x0102},
						},
						{
							Endpoint:       0x02,
							ProfileID:      zigbee.ProfileCommercialBuildingAutomation,
							DeviceID:       0x02,
							DeviceVersion:  0x02,
							InClusterList:  []zigbee.ClusterID{0x0201},
							OutClusterList: []zigbee.ClusterID{0x0202},
						},
					},
				},
			},
		}

		zgw.LoadState(expectedState)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		var events []interface{}

		for event, err := zgw.ReadEvent(ctx); err == nil; event, err = zgw.ReadEvent(ctx) {
			events = append(events, event)
		}

		assert.Len(t, events, 4)

		if len(events) >= 4 {
			assert.IsType(t, da.DeviceAdded{}, events[0])
			assert.IsType(t, da.DeviceLoaded{}, events[1])
			assert.IsType(t, da.DeviceAdded{}, events[2])
			assert.IsType(t, da.DeviceLoaded{}, events[3])
		}
	})
}
