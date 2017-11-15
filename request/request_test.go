package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetGetMetaPropForReq(t *testing.T) {
	expected := "bar"
	req := Request{}

	req.SetMetaProp("foo", expected)

	assert.Equal(t, expected, req.GetMetaProp("foo"))
}

func TestNewReq(t *testing.T) {
	req := New()

	assert.NotNil(t, req.Meta)
	assert.NotNil(t, req.TracerData)
}
