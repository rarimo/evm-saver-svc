//go:build integration_ex
// +build integration_ex

package ipfs

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

func TestIntegrationClient(t *testing.T) {
	client := NewIPFSer(kv.MustFromEnv()).IPFS()

	t.Run("got json object", func(t *testing.T) {
		result, err := client.Get(context.Background(), "QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/3650")
		assert.Nil(t, err)
		assert.Equal(t, `{"image":"ipfs://QmWA4NVxvNCMqp452tLgCyK3DsCrGSu6uecQzVJbD2N8FA","attributes"`+
			`:[{"trait_type":"Mouth","value":"Bored Unshaven Cigarette"},{"trait_type":"Hat","value":"Horns"},`+
			`{"trait_type":"Fur","value":"Blue"},{"trait_type":"Background","value":"New Punk Blue"},`+
			`{"trait_type":"Earring","value":"Silver Hoop"},{"trait_type":"Eyes","value":"Closed"}]}`, strings.TrimSuffix(string(result), "\n"))
	})
	t.Run("object does not exist", func(t *testing.T) {
		result, err := client.Get(context.Background(), "QmNn8xe96w5gx8TyQix1M2S5oMtJZ5rnqwBpnEcjF2xYu8/334.json")
		assert.Equal(t, errors.Cause(err), ErrNotFound)
		assert.Nil(t, result)
	})
}
