package resources

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
	"time"
)

var globalETag atomic.Uint64

func generateNextETag(currentETag uint64) uint64 {
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	if err != nil {
		return currentETag + uint64(time.Now().UnixNano()%1000)
	}
	return currentETag + uint64(binary.BigEndian.Uint32(buf)%1000)
}

func GetETag() uint64 {
	for {
		now := uint64(time.Now().UnixNano())
		oldEtag := globalETag.Load()
		etag := oldEtag
		if now > etag {
			etag = now
		}
		newEtag := generateNextETag(etag)
		if globalETag.CompareAndSwap(oldEtag, newEtag) {
			return newEtag
		}
	}
}
