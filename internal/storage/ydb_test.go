package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewYdbStorage(t *testing.T) {
	ydbStore := newYdbStorage()
	assert.NotNil(t, ydbStore.txc, "txc must be set in constructor")
}
