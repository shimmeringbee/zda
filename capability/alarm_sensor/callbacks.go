package alarm_sensor

import (
	"context"
	"fmt"
	"github.com/shimmeringbee/da/capabilities"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zcl/commands/local/ias_zone"
	"github.com/shimmeringbee/zda"
	"github.com/shimmeringbee/zigbee"
	"time"
)

func (i *Implementation) AddedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	if _, found := i.data[d.Identifier]; !found {
		i.data[d.Identifier] = Data{}
	}

	return nil
}

func (i *Implementation) RemovedDevice(ctx context.Context, d zda.Device) error {
	i.datalock.Lock()
	defer i.datalock.Unlock()

	delete(i.data, d.Identifier)

	return nil
}

func selectEndpoint(found []zigbee.Endpoint, device map[zigbee.Endpoint]zigbee.EndpointDescription) zigbee.Endpoint {
	if len(found) > 0 {
		return found[0]
	}

	if len(device) > 0 {
		for endpoint := range device {
			return endpoint
		}
	}

	return 0
}

func (i *Implementation) EnumerateDevice(ctx context.Context, d zda.Device) error {
	cfg := i.supervisor.DeviceConfig().Get(d, i.Name())

	endpoints := zda.FindEndpointsWithClusterID(d, zcl.IASZoneId)

	hasCapability := cfg.Bool("HasCapability", len(endpoints) > 0)

	if !hasCapability {
		i.datalock.Lock()

		i.data[d.Identifier] = Data{}
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Remove(d, capabilities.AlarmSensorFlag)
	} else {
		data := Data{Alarms: map[capabilities.SensorType]bool{}}
		data.Endpoint = zigbee.Endpoint(cfg.Int("Endpoint", int(selectEndpoint(endpoints, d.Endpoints))))

		coordinatorAddress := i.supervisor.DeviceLookup().Self().Identifier.IEEEAddress

		results, err := i.supervisor.ZCL().WriteAttributes(ctx, d, data.Endpoint, zcl.IASZoneId, map[zcl.AttributeID]zcl.AttributeDataTypeValue{ias_zone.IASCIEAddress: {
			DataType: zcl.TypeIEEEAddress,
			Value:    coordinatorAddress,
		}})

		if err != nil {
			return err
		}

		if results[ias_zone.IASCIEAddress].Status != 0 {
			return fmt.Errorf("unable to set IAS CIE Address")
		}

		msg, err := i.supervisor.ZCL().WaitForMessage(ctx, d, data.Endpoint, zcl.IASZoneId, ias_zone.ZoneEnrollRequestId)
		if err != nil {
			return err
		}

		enrollReq, ok := msg.Command.(*ias_zone.ZoneEnrollRequest)
		if !ok {
			return fmt.Errorf("retrieved message that was not a ZoneEnrollRequest")
		}

		data.ZoneType = uint16(cfg.Int("ZoneType", int(enrollReq.ZoneType)))

		err = i.supervisor.ZCL().SendCommand(ctx, d, data.Endpoint, zcl.IASZoneId, &ias_zone.ZoneEnrollResponse{})
		if err != nil {
			return err
		}

		enrolled := false

		for j := 0; j < cfg.Int("PostEnrollPolls", 20); j++ {
			time.Sleep(cfg.Duration("PostEnrollPollsDelay", 250*time.Millisecond))

			reads, err := i.supervisor.ZCL().ReadAttributes(ctx, d, data.Endpoint, zcl.IASZoneId, []zcl.AttributeID{ias_zone.ZoneState, ias_zone.ZoneStatus})
			if err != nil {
				return err
			}

			if reads[ias_zone.ZoneStatus].Status == 0 && reads[ias_zone.ZoneStatus].DataTypeValue.DataType == zcl.TypeEnum16 {
				primarySensorType := capabilities.SensorType(cfg.Int("PrimarySensorType", int(mapZoneTypeToSensorType(i.data[d.Identifier].ZoneType, true))))
				secondarySensorType := capabilities.SensorType(cfg.Int("SecondarySensorType", int(mapZoneTypeToSensorType(i.data[d.Identifier].ZoneType, false))))

				status := reads[ias_zone.ZoneStatus].DataTypeValue.Value.(uint16)

				alarms := map[capabilities.SensorType]bool{}

				alarms[primarySensorType] = (status&0x0001)&0x0001 == 0x0001
				alarms[secondarySensorType] = (status>>1)&0x0001 == 0x0001
				alarms[capabilities.DeviceTamper] = (status>>2)&0x0001 == 0x0001
				alarms[capabilities.DeviceBatteryLow] = (status>>3)&0x0001 == 0x0001
				alarms[capabilities.DeviceFailure] = (status>>6)&0x0001 == 0x0001
				alarms[capabilities.DeviceMainsFailure] = (status>>7)&0x0001 == 0x0001
				alarms[capabilities.DeviceTest] = (status>>8)&0x0001 == 0x0001
				alarms[capabilities.DeviceBatteryFailure] = (status>>9)&0x0001 == 0x0001

				data.Alarms = alarms
			}

			if reads[ias_zone.ZoneState].Status == 0 && reads[ias_zone.ZoneState].DataTypeValue.DataType == zcl.TypeEnum8 {
				state := reads[ias_zone.ZoneState].DataTypeValue.Value.(uint8)

				if state == 1 {
					enrolled = true
					break
				}
			}
		}

		if !enrolled {
			return fmt.Errorf("failed to complete enrollment of sensor, never saw enrolled state")
		}

		i.datalock.Lock()
		i.data[d.Identifier] = data
		i.datalock.Unlock()

		i.supervisor.ManageDeviceCapabilities().Add(d, capabilities.AlarmSensorFlag)
	}

	return nil
}

func (i *Implementation) zoneStatusChangeNotification(d zda.Device, message zcl.Message) {
	if !d.HasCapability(capabilities.AlarmSensorFlag) {
		return
	}

	i.datalock.Lock()
	defer i.datalock.Unlock()

	zoneChangeMsg, ok := message.Command.(*ias_zone.ZoneStatusChangeNotification)
	if !ok {
		return
	}

	cfg := i.supervisor.DeviceConfig().Get(d, i.Name())

	primarySensorType := capabilities.SensorType(cfg.Int("PrimarySensorType", int(mapZoneTypeToSensorType(i.data[d.Identifier].ZoneType, true))))
	secondarySensorType := capabilities.SensorType(cfg.Int("SecondarySensorType", int(mapZoneTypeToSensorType(i.data[d.Identifier].ZoneType, false))))

	data := i.data[d.Identifier]

	alarms := map[capabilities.SensorType]bool{}

	alarms[primarySensorType] = zoneChangeMsg.Alarm1
	alarms[secondarySensorType] = zoneChangeMsg.Alarm2
	alarms[capabilities.DeviceTamper] = zoneChangeMsg.Tamper
	alarms[capabilities.DeviceBatteryLow] = zoneChangeMsg.BatteryLow
	alarms[capabilities.DeviceFailure] = zoneChangeMsg.Trouble
	alarms[capabilities.DeviceMainsFailure] = zoneChangeMsg.ACMainsFault
	alarms[capabilities.DeviceTest] = zoneChangeMsg.TestMode
	alarms[capabilities.DeviceBatteryFailure] = zoneChangeMsg.BatteryDefect

	data.Alarms = alarms
	i.data[d.Identifier] = data

	i.supervisor.DAEventSender().Send(capabilities.AlarmSensorUpdate{
		Device: i.supervisor.ComposeDADevice().Compose(d),
		States: alarms,
	})
}

func mapZoneTypeToSensorType(zoneType uint16, primary bool) capabilities.SensorType {
	if primary {
		switch zoneType {
		case 0x0000:
			return capabilities.SecurityInfrastructure
		case 0x000d:
			return capabilities.SecurityMotion
		case 0x0015:
			return capabilities.SecurityContact
		case 0x0028:
			return capabilities.FireOther
		case 0x002a:
			return capabilities.General
		case 0x002b:
			return capabilities.GasCarbonMonoxide
		case 0x002c:
			return capabilities.HealthFall
		case 0x002d:
			return capabilities.SecurityVibration
		case 0x010f, 0x0115, 0x021d:
			return capabilities.SecurityPanic
		case 0x0225:
			return capabilities.GeneralWarningDevice
		case 0x0226:
			return capabilities.SecurityGlassBreak
		case 0x0229:
			return capabilities.SecurityInfrastructure
		default:
			return 0xffff
		}
	} else {
		switch zoneType {
		case 0x0000:
			return 0xffff
		case 0x000d:
			return capabilities.SecurityOther
		case 0x0015:
			return capabilities.SecurityOther
		case 0x0028:
			return 0xffff
		case 0x002a:
			return 0xffff
		case 0x002b:
			return capabilities.General
		case 0x002c:
			return capabilities.GeneralEmergency
		case 0x002d:
			return capabilities.SecurityVibration
		case 0x010f, 0x0115, 0x021d:
			return capabilities.GeneralEmergency
		case 0x0225:
			return 0xffff
		case 0x0226:
			return 0xffff
		case 0x0229:
			return 0xffff
		default:
			return 0xffff
		}
	}
}
