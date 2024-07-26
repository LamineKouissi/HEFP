package routers

import (
	"context"
	"errors"
	"net/http"

	"github.com/LamineKouissi/LHP/routers/routes"
	"github.com/LamineKouissi/LHP/util"
)

type ForwardProxyRouter struct {
	httpsRoute routes.HttpsRoute
	httpRoute  routes.HttpRoute
}

func NewForwardProxyRouter(hsRoute routes.HttpsRoute, hRoute routes.HttpRoute) (*ForwardProxyRouter, error) {

	isEmpty, err := util.IsStructEmpty(hRoute)
	if err != nil || isEmpty {
		return nil, errors.New("invalid arg : HttpRoute")
	}

	return &ForwardProxyRouter{httpsRoute: hsRoute, httpRoute: hRoute}, nil
}

func (f *ForwardProxyRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	switch r.Method {
	case "CONNECT":
		f.httpsRoute.HandleF(ctx, w, r)
	default:
		f.httpRoute.HandleF(ctx, w, r)
	}
}
