package wal

import (
	"testing"
)

func TestWAL_RollBackWhenPanic(t *testing.T) {
	var log WAL
	defer func() {
		log.RollBackWhenPanic(recover())
	}()

	log.AddRollBack(func() {
		t.Logf("rollback")
	})

	panic(nil)
}
