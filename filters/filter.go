package filters

import (
	"context"
	"net/http"
)

func ConstructFilterChain(cnx context.Context, filters []HasNextFilter, connector Filter) (Filter, error) {
	headFilter := connector

	for i := len(filters) - 1; i >= 0; i-- {
		filters[i].SetNextFilter(headFilter)
		headFilter = filters[i].(Filter)

	}
	return headFilter, nil
}

type Filter interface {
	Process(ctx context.Context, req *http.Request, res *http.Response) error
}

type HasNextFilter interface {
	SetNextFilter(f Filter) error
}
