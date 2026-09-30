package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
)

// GetPaginated handles Capella list endpoints that require page/perPage query params.
// It flattens all pages into a single slice.
func GetPaginated[DataSchema ~[]T, T any](
	ctx context.Context,
	client *Client,
	token string,
	cfg EndpointCfg,
) (DataSchema, error) {
	var (
		responses DataSchema
		page      = 1
		perPage   = 25
		baseURL   = cfg.Url
	)

	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}

	for {
		cfg.Url = baseURL + fmt.Sprintf("%spage=%d&perPage=%d", separator, page, perPage)
		response, err := client.ExecuteWithRetry(ctx, cfg, nil, token, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errors.ErrExecutingRequest, err)
		}

		var decoded struct {
			Data   DataSchema `json:"data"`
			Cursor Cursor     `json:"cursor"`
		}
		if err := json.Unmarshal(response.Body, &decoded); err != nil {
			return nil, fmt.Errorf("%s: %w", errors.ErrUnmarshallingResponse, err)
		}

		responses = append(responses, decoded.Data...)
		if decoded.Cursor.Pages.Next == 0 {
			break
		}
		page = decoded.Cursor.Pages.Next
	}

	return responses, nil
}
