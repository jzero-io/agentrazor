package agent

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrServiceStopped  = errors.New("agent service is stopped")
	ErrThreadArchived  = errors.New("agent thread is archived")
	ErrInvalidThreadID = errors.New("invalid Codex thread id")
)

func newID(prefix string) string {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(fmt.Sprintf("generate id: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(data[:])
}
