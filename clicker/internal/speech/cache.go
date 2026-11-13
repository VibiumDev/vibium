package speech

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/vibium/clicker/internal/paths"
)

// cacheVersion changes when the meaning of a key or the stored audio does.
const cacheVersion = 1

// CacheKey is everything that decides how a phrase sounds.
type CacheKey struct {
	Version  int               `json:"version"`
	Text     string            `json:"text"`
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Voice    string            `json:"voice"`
	Language string            `json:"language"`
	Speed    float64           `json:"speed"`
	Settings map[string]string `json:"settings,omitempty"`
}

// Hash is the SHA-256 of the key's canonical JSON. encoding/json writes
// struct fields in order and map keys sorted, so the bytes are stable.
func (k CacheKey) Hash() string {
	raw, _ := json.Marshal(k)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Cache keeps rendered speech so repeat runs replay identical audio.
type Cache struct{ Dir string }

// DefaultCacheDir is speech/ under Vibium's cache directory.
func DefaultCacheDir() (string, error) {
	dir, err := paths.GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "speech"), nil
}

func (c Cache) path(k CacheKey) string {
	h := k.Hash()
	return filepath.Join(c.Dir, h[:2], h)
}

// Get returns the cached audio for a key.
func (c Cache) Get(k CacheKey) (Audio, bool) {
	raw, err := os.ReadFile(c.path(k) + ".wav")
	if err != nil {
		return Audio{}, false
	}
	a, err := ParseWAV(raw)
	return a, err == nil
}

// Put stores audio for a key. Each file lands by rename, so a reader in
// another process never sees part of one.
func (c Cache) Put(k CacheKey, a Audio) error {
	base := c.path(k)
	if err := os.MkdirAll(filepath.Dir(base), 0o755); err != nil {
		return err
	}
	sidecar, _ := json.MarshalIndent(struct {
		CacheKey
		Created    string  `json:"created"`
		DurationMs float64 `json:"durationMs"`
		SHA256     string  `json:"sha256"`
	}{k, time.Now().UTC().Format(time.RFC3339), a.DurationMs(), a.SHA256()}, "", "  ")
	if err := writeAtomic(base+".wav", a.WAV()); err != nil {
		return err
	}
	return writeAtomic(base+".json", append(sidecar, '\n'))
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), path)
	}
	if werr != nil {
		os.Remove(tmp.Name())
	}
	return werr
}
