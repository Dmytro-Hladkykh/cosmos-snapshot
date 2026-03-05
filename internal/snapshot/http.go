package snapshot

import (
	"io"
	"log"
	"net/http"
	"strconv"
)

func mustGet(cfg Config, path string) []byte {
	req, err := http.NewRequest(http.MethodGet, cfg.RPC+path, nil)
	if err != nil {
		log.Fatalf("failed to build request: %v", err)
	}
	if cfg.Height > 0 {
		req.Header.Set("x-cosmos-block-height", strconv.FormatInt(cfg.Height, 10))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("request failed (path=%s): %v", path, err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("failed to read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("unexpected status %d for %s: %s", resp.StatusCode, path, body)
	}
	return body
}
