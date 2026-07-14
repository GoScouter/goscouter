package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"

    "goscouter/pkg/records"
)

func FetchHTTPRecords(siteURL string) (*records.HTTPRecords, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, siteURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("site fetch timed out after %s", TIMEOUT)
		}

		return nil, err
	}
	defer resp.Body.Close()

	return &records.HTTPRecords{
		RequestURL: siteURL,
		FinalURL:   resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		Headers:    resp.Header,
	}, nil
}
