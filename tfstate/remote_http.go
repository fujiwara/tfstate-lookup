package tfstate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

func readHTTP(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	token := os.Getenv("TFE_TOKEN")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
