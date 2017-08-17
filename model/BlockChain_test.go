package model

import (
	"testing"
)

func TestBlkFileName(t *testing.T) {

	t.Log("assemble FilePath : ", blkFileName("yyx/", 8))
}
