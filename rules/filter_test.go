package rules

import (
	"github.com/shimmeringbee/zigbee"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilter_matches(t *testing.T) {
	t.Run("returns true if there are no negative matches on present fields", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			DeviceId:         deviceId,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			DeviceId:         &deviceId,
		}

		assert.True(t, f.matches(m))
	})

	t.Run("returns true if filter is empty", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			DeviceId:         deviceId,
		}

		f := Filter{}

		assert.True(t, f.matches(m))
	})

	t.Run("returns false if filter ManufacturerCode does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: 0,
			ManufacturerName: manuName,
			ProductName:      prodName,
			DeviceId:         deviceId,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			DeviceId:         &deviceId,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter ManufacturerName does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: "",
			ProductName:      prodName,
			DeviceId:         deviceId,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			DeviceId:         &deviceId,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter ProductName does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      "",
			DeviceId:         deviceId,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			DeviceId:         &deviceId,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter DeviceID does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		deviceId := uint16(0x55aa)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			DeviceId:         0,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			DeviceId:         &deviceId,
		}

		assert.False(t, f.matches(m))
	})
}
