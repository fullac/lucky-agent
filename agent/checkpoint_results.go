package agent

import (
	"crypto/sha256"
	"encoding/hex"
)

func durableResultDigest(result string) string {
	digest := sha256.Sum256([]byte(result))
	return hex.EncodeToString(digest[:])
}

// The no-progress detector needs equality, not another copy of every tool
// output. Full outputs remain in the operation ledger and message history.
func (cp *durableCheckpoint) migrateResultDigests() {
	if cp.LastResultDigests == nil {
		cp.LastResultDigests = make(map[string]string, len(cp.LastResults))
	}
	for signature, result := range cp.LastResults {
		if _, exists := cp.LastResultDigests[signature]; !exists {
			cp.LastResultDigests[signature] = durableResultDigest(result)
		}
	}
	cp.LastResults = nil
}
