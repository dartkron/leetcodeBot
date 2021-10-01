package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewYdbStorage(t *testing.T) {
	ydbStore := newYdbStorage()
	assert.NotNil(t, ydbStore.ydbExecuter, "ydbExecuter must be set in constructor")
}

func TestNewYDBExecuter(t *testing.T) {
	e := newYDBQueryExecuter()
	assert.NotNil(t, e.txc, "txc must be set in constructor")
	assert.NotNil(t, e.ExecQueryFunc, "ExecQueryFunc must be set in constructor")
	assert.NotNil(t, e.GetConnectionFunc, "GetConnectionFunc must be set in constructor")
}
