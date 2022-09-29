package xiaomi

import (
	"github.com/shimmeringbee/bytecodec"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_AttributeList(t *testing.T) {
	t.Run("unmarshalls proprietary Xiaomi attribute list", func(t *testing.T) {
		inputPayload := []byte{0x01, 0x21, 0x9f, 0x0b, 0x04, 0x21, 0xa8, 0x13, 0x05, 0x21,
			0x2d, 0x00, 0x06, 0x24, 0x02, 0x00, 0x00, 0x00, 0x00, 0x64,
			0x29, 0x2c, 0x07, 0x65, 0x21, 0x64, 0x11, 0x0a, 0x21, 0xe1, 0x76}

		var list AttributeList

		err := bytecodec.Unmarshal(inputPayload, &list)
		assert.NoError(t, err)

		assert.Equal(t, uint8(1), list[0].Id)
		assert.Equal(t, uint64(2975), list[0].Attribute.Value)

		assert.Equal(t, uint8(4), list[1].Id)
		assert.Equal(t, uint64(5032), list[1].Attribute.Value)

		assert.Equal(t, uint8(100), list[4].Id)
		assert.Equal(t, int64(1836), list[4].Attribute.Value)

		assert.Equal(t, uint8(101), list[5].Id)
		assert.Equal(t, uint64(4452), list[5].Attribute.Value)
	})
}

func TestParseAttributeList(t *testing.T) {
	t.Run("unmarshalls and reformats proprietary Xiaomi attribute list", func(t *testing.T) {
		inputPayload := []byte{
			0x01, 0x21, 0x9f, 0x0b, 0x04, 0x21, 0xa8, 0x13,
			0x05, 0x21, 0x2d, 0x00, 0x06, 0x24, 0x02, 0x00,
			0x00, 0x00, 0x00, 0x64, 0x29, 0x2c, 0x07, 0x65,
			0x21, 0x64, 0x11, 0x0a, 0x21, 0xe1, 0x76,
		}

		xal, err := ParseAttributeList(inputPayload)
		assert.NoError(t, err)

		assert.Equal(t, uint64(2975), xal[1].Attribute.Value)
		assert.Equal(t, uint64(5032), xal[4].Attribute.Value)
		assert.Equal(t, int64(1836), xal[100].Attribute.Value)
		assert.Equal(t, uint64(4452), xal[101].Attribute.Value)
	})
}
