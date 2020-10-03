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
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			Endpoint:         endpoint,
			ClusterID:        cluster,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.True(t, f.matches(m))
	})

	t.Run("returns true if filter is empty", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			Endpoint:         endpoint,
			ClusterID:        cluster,
		}

		f := Filter{}

		assert.True(t, f.matches(m))
	})

	t.Run("returns false if filter ManufacturerCode does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: 0,
			ManufacturerName: manuName,
			ProductName:      prodName,
			Endpoint:         endpoint,
			ClusterID:        cluster,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter ManufacturerName does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: "",
			ProductName:      prodName,
			Endpoint:         endpoint,
			ClusterID:        cluster,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter ProductName does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      "",
			Endpoint:         endpoint,
			ClusterID:        cluster,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter Endpoint does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			Endpoint:         0,
			ClusterID:        cluster,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.False(t, f.matches(m))
	})

	t.Run("returns false if filter ClusterID does not match", func(t *testing.T) {
		manuCode := zigbee.ManufacturerCode(0x1234)
		manuName := "manu"
		prodName := "prod"
		endpoint := zigbee.Endpoint(0x11)
		cluster := zigbee.ClusterID(0x4321)

		m := MatchData{
			ManufacturerCode: manuCode,
			ManufacturerName: manuName,
			ProductName:      prodName,
			Endpoint:         endpoint,
			ClusterID:        0,
		}

		f := Filter{
			ManufacturerCode: &manuCode,
			ManufacturerName: &manuName,
			ProductName:      &prodName,
			Endpoint:         &endpoint,
			ClusterID:        &cluster,
		}

		assert.False(t, f.matches(m))
	})
}
