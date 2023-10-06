package tfstate

import (
	"context"
	"io"
	"net/http"
)

func readHTTP(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return readHTTPWithRequest(ctx, req)
}

func readHTTPWithRequest(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	if c := req.Context(); c != ctx {
		req = req.WithContext(ctx)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
