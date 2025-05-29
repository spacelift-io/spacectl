package authenticated

import (
	"context"

	"github.com/pkg/errors"
)

type Viewer struct {
	ID   string `graphql:"id" json:"id"`
	Name string `graphql:"name" json:"name"`
}

var ErrViewerUnknown = errors.New("failed to query user information: unauthorized")

func CurrentViewer(ctx context.Context) (*Viewer, error) {
	var query struct {
		Viewer *Viewer
	}
	if err := Client.Query(ctx, &query, map[string]interface{}{}); err != nil {
		return nil, errors.Wrap(err, "failed to query user information")
	}
	if query.Viewer == nil {
		return nil, ErrViewerUnknown
	}

	return query.Viewer, nil
}
