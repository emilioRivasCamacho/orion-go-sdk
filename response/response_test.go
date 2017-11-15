package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewReq(t *testing.T) {
	res := New()

	assert.NotNil(t, res)
}
