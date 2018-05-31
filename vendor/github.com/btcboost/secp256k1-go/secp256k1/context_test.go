package secp256k1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ContextCreate_Clone_Destroy(t *testing.T) {
	vFlags := make([]uint, 3)
	vFlags[0] = uint(ContextSign)
	vFlags[1] = uint(ContextVerify)
	vFlags[2] = uint(ContextSign | ContextVerify)

	for i := 0; i < len(vFlags); i++ {

		ctx, err := ContextCreate(vFlags[i])
		assert.NoError(t, err)
		assert.NotNil(t, ctx)
		assert.IsType(t, Context{}, *ctx)

		clone, err := ContextClone(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, ctx)
		assert.IsType(t, Context{}, *ctx)

		ContextDestroy(clone)
		ContextDestroy(ctx)
	}
}
