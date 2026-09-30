package resources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	goerrors "errors"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

func nullAuditObject() types.Object {
	return types.ObjectNull(providerschema.AuditAttrTypes())
}

// executeAllowingStatuses runs ExecuteWithRetry and treats alternate HTTP success
// codes as success when the client only accepts a single SuccessStatus.
func executeAllowingStatuses(
	ctx context.Context,
	client *api.Client,
	cfg api.EndpointCfg,
	payload any,
	token string,
	headers map[string]string,
	alternateOK ...int,
) (*api.Response, error) {
	resp, err := client.ExecuteWithRetry(ctx, cfg, payload, token, headers)
	if err == nil {
		return resp, nil
	}
	for _, code := range alternateOK {
		if body, ok := unexpectedStatusBody(err, code, cfg.SuccessStatus); ok {
			return &api.Response{Body: body}, nil
		}
	}
	return nil, err
}

func unexpectedStatusBody(err error, got, expected int) ([]byte, bool) {
	prefix := fmt.Sprintf("unexpected code: %d, expected: %d, body: ", got, expected)
	msg := err.Error()
	if strings.HasPrefix(msg, prefix) {
		return []byte(strings.TrimPrefix(msg, prefix)), true
	}
	return nil, false
}

func isNotFoundOrConflict(err error) bool {
	notFound, _ := api.CheckResourceNotFoundError(err)
	if notFound {
		return true
	}
	var apiErr *api.Error
	if goerrors.As(err, &apiErr) {
		return apiErr.HttpStatusCode == http.StatusConflict
	}
	return false
}
