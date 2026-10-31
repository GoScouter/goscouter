package web

import (
	"context"
	"errors"
	"fmt"

	"goscouter/internal/utils"
	"goscouter/pkg/records"
)

func FetchHTTPRecords(siteURL, scheme string) (*records.HTTPRecords, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	resp, err := utils.Get(ctx, siteURL)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("site fetch timed out after %s", TIMEOUT)
		}

		return nil, err
	}
	defer resp.Body.Close()

	return &records.HTTPRecords{
		Scheme:     scheme,
		RequestURL: siteURL,
		FinalURL:   resp.Request.URL.String(),
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		Headers:    resp.Header,
	}, nil
}
