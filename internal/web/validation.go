package web

import (
	"context"
	"errors"
	"fmt"
	"time"

	"goscouter/internal/utils"
)

const TIMEOUT time.Duration = 5 * time.Second

func CheckSiteStatus(siteURL string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	resp, err := utils.Get(ctx, siteURL)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("site check timed out after %s", TIMEOUT)
		}

		return 0, err
	}

	defer resp.Body.Close()
	return resp.StatusCode, nil
}
