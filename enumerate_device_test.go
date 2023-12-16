package zda

import (
	"context"
	"github.com/shimmeringbee/da"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/discard"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/global"
	"github.com/shimmeringbee/zcl/commands/local/basic"
	"github.com/shimmeringbee/zda/implcaps"
	"github.com/shimmeringbee/zda/implcaps/factory"
	"github.com/shimmeringbee/zda/implcaps/generic"
	"github.com/shimmeringbee/zda/rules"
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"
	"io"
	"sync"
	"testing"
)

func Test_enumerateDevice_startEnumeration(t *testing.T) {
	t.Run("returns an error if the node is already being enumerated", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard())}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		n.enumerationSem.TryAcquire(1)
		err := ed.startEnumeration(context.Background(), n)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "enumeration already in progress")
	})

	t.Run("returns nil if node is not being enumerated, and marks the node in progress", func(t *testing.T) {
		mnq := &mockNodeQuerier{}
		defer mnq.AssertExpectations(t)
		mnq.On("QueryNodeDescription", mock.Anything, mock.Anything).Return(zigbee.NodeDescription{}, io.ErrUnexpectedEOF).Maybe()

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		err := ed.startEnumeration(context.Background(), n)
		assert.Nil(t, err)
		assert.False(t, n.enumerationSem.TryAcquire(1))
	})
}

type mockReadAttribute struct {
	mock.Mock
}

func (m *mockReadAttribute) ReadAttribute(ctx context.Context, ieeeAddress zigbee.IEEEAddress, requireAck bool, cluster zigbee.ClusterID, code zigbee.ManufacturerCode, sourceEndpoint zigbee.Endpoint, destEndpoint zigbee.Endpoint, transactionSequence uint8, attributes []zcl.AttributeID) ([]global.ReadAttributeResponseRecord, error) {
	args := m.Called(ctx, ieeeAddress, requireAck, cluster, code, sourceEndpoint, destEndpoint, transactionSequence, attributes)
	return args.Get(0).([]global.ReadAttributeResponseRecord), args.Error(1)
}

func Test_enumerateDevice_interrogateNode(t *testing.T) {
	t.Run("interrogates a node for description, endpoints and subsequent endpoint descriptions", func(t *testing.T) {
		expectedAddr := zigbee.GenerateLocalAdministeredIEEEAddress()

		expectedNodeDescription := zigbee.NodeDescription{
			LogicalType:      zigbee.Router,
			ManufacturerCode: 0x1234,
		}
		expectedEndpoints := []zigbee.Endpoint{0x01, 0x02}

		expectedEndpointDescs := []zigbee.EndpointDescription{
			{
				Endpoint:       0x01,
				ProfileID:      0x02,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  []zigbee.ClusterID{zcl.BasicId},
				OutClusterList: nil,
			},
			{
				Endpoint:       0x02,
				ProfileID:      0x03,
				DeviceID:       0x03,
				DeviceVersion:  1,
				InClusterList:  nil,
				OutClusterList: nil,
			},
		}

		mnq := &mockNodeQuerier{}
		defer mnq.AssertExpectations(t)
		mnq.On("QueryNodeDescription", mock.Anything, expectedAddr).Return(expectedNodeDescription, nil)
		mnq.On("QueryNodeEndpoints", mock.Anything, expectedAddr).Return(expectedEndpoints, nil)
		mnq.On("QueryNodeEndpointDescription", mock.Anything, expectedAddr, zigbee.Endpoint(0x01)).Return(expectedEndpointDescs[0], nil)
		mnq.On("QueryNodeEndpointDescription", mock.Anything, expectedAddr, zigbee.Endpoint(0x02)).Return(expectedEndpointDescs[1], nil)

		mra := &mockReadAttribute{}
		defer mra.AssertExpectations(t)
		mra.On("ReadAttribute", mock.Anything, expectedAddr, false, zcl.BasicId, zigbee.NoManufacturer, DefaultGatewayHomeAutomationEndpoint, zigbee.Endpoint(1), uint8(0), []zcl.AttributeID{basic.ManufacturerName, basic.ModelIdentifier, basic.ManufacturerVersionDetails, basic.SerialNumber}).
			Return([]global.ReadAttributeResponseRecord{
				{
					Identifier: basic.ManufacturerName,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "manufacturer",
					},
				},
				{
					Identifier: basic.ModelIdentifier,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "model",
					},
				},
				{
					Identifier: basic.ManufacturerVersionDetails,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "version",
					},
				},
				{
					Identifier: basic.SerialNumber,
					Status:     0,
					DataTypeValue: &zcl.AttributeDataTypeValue{
						DataType: 0,
						Value:    "serial",
					},
				},
			}, nil)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq, zclReadFn: mra.ReadAttribute}
		n := &node{address: expectedAddr, sequence: makeTransactionSequence()}

		inv, err := ed.interrogateNode(context.Background(), n)
		assert.NoError(t, err)

		assert.Equal(t, expectedNodeDescription, *inv.description)
		assert.Equal(t, expectedEndpointDescs[0], inv.endpoints[0x01].description)
		assert.Equal(t, expectedEndpointDescs[1], inv.endpoints[0x02].description)

		assert.Equal(t, "model", inv.endpoints[0x01].productInformation.product)
		assert.Equal(t, "serial", inv.endpoints[0x01].productInformation.serial)
		assert.Equal(t, "version", inv.endpoints[0x01].productInformation.version)
		assert.Equal(t, "manufacturer", inv.endpoints[0x01].productInformation.manufacturer)
	})
}

type mockNodeQuerier struct {
	mock.Mock
}

func (m *mockNodeQuerier) QueryNodeDescription(ctx context.Context, networkAddress zigbee.IEEEAddress) (zigbee.NodeDescription, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).(zigbee.NodeDescription), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpoints(ctx context.Context, networkAddress zigbee.IEEEAddress) ([]zigbee.Endpoint, error) {
	args := m.Called(ctx, networkAddress)
	return args.Get(0).([]zigbee.Endpoint), args.Error(1)
}

func (m *mockNodeQuerier) QueryNodeEndpointDescription(ctx context.Context, networkAddress zigbee.IEEEAddress, endpoint zigbee.Endpoint) (zigbee.EndpointDescription, error) {
	args := m.Called(ctx, networkAddress, endpoint)
	return args.Get(0).(zigbee.EndpointDescription), args.Error(1)
}

type mockRulesEngine struct {
	mock.Mock
}

func (m *mockRulesEngine) Execute(i rules.Input) (rules.Output, error) {
	args := m.Called(i)
	return args.Get(0).(rules.Output), args.Error(1)
}

func Test_enumerateDevice_runRules(t *testing.T) {
	t.Run("executes rules on all endpoints in an inventory and adds capabilities to the returned inventory", func(t *testing.T) {
		inInv := inventory{
			description: &zigbee.NodeDescription{
				LogicalType:      zigbee.Router,
				ManufacturerCode: 0x1234,
			},
			endpoints: map[zigbee.Endpoint]endpointDetails{
				10: {
					description: zigbee.EndpointDescription{
						Endpoint:      zigbee.Endpoint(10),
						ProfileID:     zigbee.ProfileHomeAutomation,
						DeviceID:      0x0400,
						DeviceVersion: 1,
						InClusterList: []zigbee.ClusterID{0x0000, 0x0006},
					},
					productInformation: productData{
						manufacturer: "manufacturer",
						product:      "product",
						version:      "version",
						serial:       "serial",
					},
				},
				20: {
					description: zigbee.EndpointDescription{
						Endpoint:      zigbee.Endpoint(20),
						ProfileID:     zigbee.ProfileHomeAutomation,
						DeviceID:      0x0400,
						DeviceVersion: 1,
						InClusterList: []zigbee.ClusterID{0x0008},
					},
				},
			},
		}

		e := rules.New()

		err := e.LoadFS(rules.Embedded)
		assert.NoError(t, err)

		err = e.CompileRules()
		assert.NoError(t, err)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), runRulesFn: e.Execute}

		outEnv, err := ed.runRules(inInv)
		assert.NoError(t, err)

		assert.Contains(t, outEnv.endpoints[zigbee.Endpoint(10)].rulesOutput.Capabilities, "ZCLOnOff")
		assert.Contains(t, outEnv.endpoints[zigbee.Endpoint(20)].rulesOutput.Capabilities, "ZCLLevel")
	})
}

func Test_enumerateDevice_groupInventoryDevices(t *testing.T) {
	t.Run("aggregates into devices and sorts endpoints and device ids", func(t *testing.T) {
		inv := inventory{
			endpoints: map[zigbee.Endpoint]endpointDetails{
				1: {
					description: zigbee.EndpointDescription{
						Endpoint: 1,
						DeviceID: 1,
					},
				},
				2: {
					description: zigbee.EndpointDescription{
						Endpoint: 2,
						DeviceID: 2,
					},
				},
				3: {
					description: zigbee.EndpointDescription{
						Endpoint: 3,
						DeviceID: 1,
					},
				},
			},
		}

		expected := []inventoryDevice{
			{
				deviceId: 1,
				endpoints: []endpointDetails{
					{
						description: zigbee.EndpointDescription{
							Endpoint: 1,
							DeviceID: 1,
						},
					},
					{
						description: zigbee.EndpointDescription{
							Endpoint: 3,
							DeviceID: 1,
						},
					},
				},
			},
			{
				deviceId: 2,
				endpoints: []endpointDetails{
					{
						description: zigbee.EndpointDescription{
							Endpoint: 2,
							DeviceID: 2,
						},
					},
				},
			},
		}

		ed := enumerateDevice{logger: logwrap.New(discard.Discard())}
		actual := ed.groupInventoryDevices(inv)

		assert.Equal(t, expected, actual)
	})
}

type mockDeviceManager struct {
	mock.Mock
}

func (m *mockDeviceManager) createNextDevice(n *node) *device {
	args := m.Called(n)
	return args.Get(0).(*device)
}

func (m *mockDeviceManager) removeDevice(i IEEEAddressWithSubIdentifier) bool {
	args := m.Called(i)
	return args.Bool(0)
}

func Test_enumerateDevice_updateNodeTable(t *testing.T) {
	t.Run("creates new device if missing from node", func(t *testing.T) {
		mdm := &mockDeviceManager{}
		defer mdm.AssertExpectations(t)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), dm: mdm}
		n := &node{m: &sync.RWMutex{}}
		d := &device{m: &sync.RWMutex{}, capabilities: map[da.Capability]implcaps.ZDACapability{}}

		mdm.On("createNextDevice", n).Return(d)

		expectedDeviceId := uint16(0x2000)

		id := []inventoryDevice{
			{
				deviceId: expectedDeviceId,
			},
		}

		mapping := ed.updateNodeTable(n, id)

		assert.Equal(t, d, mapping[expectedDeviceId])
		assert.Equal(t, expectedDeviceId, d.deviceId)

		c := d.Capability(capabilities.EnumerateDeviceFlag)
		assert.NotNil(t, c)

		cc, ok := c.(*enumeratedDeviceAttachment)
		assert.True(t, ok)

		assert.Equal(t, n, cc.node)
		assert.Equal(t, d, cc.device)
	})

	t.Run("returns an existing on in mapping if present", func(t *testing.T) {
		mdm := &mockDeviceManager{}
		defer mdm.AssertExpectations(t)

		existingDeviceId := uint16(0x2000)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), dm: mdm}
		d := &device{m: &sync.RWMutex{}, deviceId: existingDeviceId}
		n := &node{m: &sync.RWMutex{}, device: map[uint8]*device{0: d}}

		id := []inventoryDevice{
			{
				deviceId: existingDeviceId,
			},
		}

		mapping := ed.updateNodeTable(n, id)

		assert.Equal(t, d, mapping[existingDeviceId])
		assert.Equal(t, existingDeviceId, d.deviceId)
	})

	t.Run("removes an device that should not be present", func(t *testing.T) {
		mdm := &mockDeviceManager{}
		defer mdm.AssertExpectations(t)

		unwantedDeviceId := uint16(0x2000)
		address := IEEEAddressWithSubIdentifier{IEEEAddress: zigbee.GenerateLocalAdministeredIEEEAddress(), SubIdentifier: 0}

		mdm.On("removeDevice", address).Return(true)

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), dm: mdm}
		d := &device{m: &sync.RWMutex{}, deviceId: unwantedDeviceId, address: address}
		n := &node{m: &sync.RWMutex{}, device: map[uint8]*device{0: d}}

		mapping := ed.updateNodeTable(n, nil)

		assert.Nil(t, mapping[unwantedDeviceId])
	})
}

func Test_enumerateDevice_onNodeJoin(t *testing.T) {
	t.Run("node join callback invokes enumeration", func(t *testing.T) {
		mnq := &mockNodeQuerier{}
		defer mnq.AssertExpectations(t)
		mnq.On("QueryNodeDescription", mock.Anything, mock.Anything).Return(zigbee.NodeDescription{}, io.ErrUnexpectedEOF).Maybe()

		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), nq: mnq}
		n := &node{m: &sync.RWMutex{}, enumerationSem: semaphore.NewWeighted(1)}

		err := ed.onNodeJoin(context.Background(), nodeJoin{n: n})
		assert.Nil(t, err)
		assert.False(t, n.enumerationSem.TryAcquire(1))
	})
}

func Test_enumeratedDeviceAttachment(t *testing.T) {
	t.Run("has basic capability functions", func(t *testing.T) {
		eda := enumeratedDeviceAttachment{}

		assert.Equal(t, capabilities.EnumerateDeviceFlag, eda.Capability())
		assert.Equal(t, capabilities.StandardNames[capabilities.EnumerateDeviceFlag], eda.Name())
	})

	t.Run("returns enumeration results and state", func(t *testing.T) {
		n := &node{enumerationState: true}
		eda := enumeratedDeviceAttachment{
			node: n,
			results: map[da.Capability]*capabilities.EnumerationCapability{
				capabilities.ProductInformationFlag: {
					Attached: true,
				},
			},
			m: &sync.RWMutex{},
		}

		r, err := eda.Status(nil)
		assert.NoError(t, err)
		assert.True(t, r.Enumerating)
		assert.True(t, r.CapabilityStatus[capabilities.ProductInformationFlag].Attached)

	})
}

func Test_enumerateDevice_updateCapabilitiesOnDevice(t *testing.T) {
	t.Run("adds a new capability from rules output", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), capabilityFactory: factory.Create}
		d := &device{m: &sync.RWMutex{}, deviceId: 1, capabilities: map[da.Capability]implcaps.ZDACapability{}}

		id := inventoryDevice{
			deviceId: 1,
			endpoints: []endpointDetails{
				{
					rulesOutput: rules.Output{
						Capabilities: map[string]map[string]interface{}{
							"GenericProductInformation": {
								"Name":         "NEXUS-7",
								"Manufacturer": "Tyrell Corporation",
								"Serial":       "N7FAA52318",
							},
						},
					},
				},
			},
		}

		errs := ed.updateCapabilitiesOnDevice(context.Background(), d, id)

		assert.Len(t, errs, 2)

		assert.True(t, errs[capabilities.EnumerateDeviceFlag].Attached)
		assert.True(t, errs[capabilities.ProductInformationFlag].Attached)

		c := d.Capability(capabilities.ProductInformationFlag)
		assert.NotNil(t, c)

		pic, ok := c.(capabilities.ProductInformation)
		assert.True(t, ok)

		pi, _ := pic.Get(context.Background())
		assert.Equal(t, "NEXUS-7", pi.Name)
	})

	t.Run("calls an existing capability for reenumeration", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), capabilityFactory: factory.Create}
		opi := generic.NewProductInformation()
		d := &device{m: &sync.RWMutex{}, deviceId: 1, capabilities: map[da.Capability]implcaps.ZDACapability{capabilities.ProductInformationFlag: opi}}
		_, _ = opi.Attach(context.Background(), d, implcaps.Enumeration, map[string]interface{}{
			"Name": "NEXUS-6",
		})

		id := inventoryDevice{
			deviceId: 1,
			endpoints: []endpointDetails{
				{
					rulesOutput: rules.Output{
						Capabilities: map[string]map[string]interface{}{
							"GenericProductInformation": {
								"Name": "NEXUS-7",
							},
						},
					},
				},
			},
		}

		errs := ed.updateCapabilitiesOnDevice(context.Background(), d, id)

		assert.Len(t, errs, 2)

		assert.True(t, errs[capabilities.EnumerateDeviceFlag].Attached)
		assert.True(t, errs[capabilities.ProductInformationFlag].Attached)

		c := d.Capability(capabilities.ProductInformationFlag)
		assert.NotNil(t, c)

		pic, ok := c.(capabilities.ProductInformation)
		assert.True(t, ok)

		pi, _ := pic.Get(context.Background())
		assert.Equal(t, "NEXUS-7", pi.Name)
	})

	t.Run("removes an existing capability that's not longer required", func(t *testing.T) {
		ed := enumerateDevice{logger: logwrap.New(discard.Discard()), capabilityFactory: factory.Create}
		opi := generic.NewProductInformation()
		d := &device{m: &sync.RWMutex{}, deviceId: 1, capabilities: map[da.Capability]implcaps.ZDACapability{capabilities.ProductInformationFlag: opi}}
		_, _ = opi.Attach(context.Background(), d, implcaps.Enumeration, map[string]interface{}{
			"Name": "NEXUS-6",
		})

		id := inventoryDevice{
			deviceId:  1,
			endpoints: []endpointDetails{},
		}

		errs := ed.updateCapabilitiesOnDevice(context.Background(), d, id)

		assert.Len(t, errs, 2)

		assert.True(t, errs[capabilities.EnumerateDeviceFlag].Attached)
		assert.False(t, errs[capabilities.ProductInformationFlag].Attached)

		c := d.Capability(capabilities.ProductInformationFlag)
		assert.Nil(t, c)
	})
}
