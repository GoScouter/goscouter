package web

import (
	"context"
	"errors"
	"fmt"
	"goscouter/internal/logger"
	"net/http"
	"time"
)

const TIMEOUT time.Duration = 5 * time.Second

func IsValidSite(siteURL string) bool {
    code, err := CheckSiteStatus(siteURL)
    if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
		    fmt.Printf("site check timed out after %s\r\n", TIMEOUT)
		} else {
            logger.Log.Error(err.Error())
        }

        return false
    }

    return code >= 200 && code <= 299
}

func CheckSiteStatus(siteURL string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, siteURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("site check timed out after %s", TIMEOUT)
		}

        return 0, err
	}

    defer resp.Body.Close()
	return resp.StatusCode, nil
}
