//go:build manual_test
// +build manual_test

package metadata

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/rarimo/evm-saver-svc/pkg/ipfs"
	"github.com/stretchr/testify/assert"
)

func TestLoadMetadata(t *testing.T) {
	c := NewClient(http.DefaultClient, ipfs.NewMockGateway(t), 10*time.Second)

	ctx := context.Background()

	var payload Payload

	err := c.LoadMetadata(ctx, "https://wzrds.xyz/metadata/full-skull-3", &payload)

	if !assert.NoError(t, err, "expected metadata to load successfully") {
		return
	}

	if !assert.NotEmpty(t, payload.URI, "expected uri not to be empty") {
		return
	}

	if !assert.NotEmpty(t, payload.RawMetadata, "expected raw_metadata not to be empty") {
		return
	}

	spew.Dump(payload)
}
