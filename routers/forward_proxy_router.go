package routers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

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

	// isEmpty, err = util.IsStructEmpty(hsRoute)
	// if err != nil || isEmpty {
	// 	return nil, errors.New("invalid arg: HttpsRoute")
	// }

	return &ForwardProxyRouter{httpsRoute: hsRoute, httpRoute: hRoute}, nil
}

func (f *ForwardProxyRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	fmt.Println("ProxyHandler.ServeHTTP() sourse Request:------------------")
	rowSourceReq, err := httputil.DumpRequest(r, false)
	if err != nil {
		log.Fatal("ProxyHandler.ServeHTTP() DumpRequest(): ", err)
	}
	fmt.Println(string(rowSourceReq))

	switch r.Method {
	case "CONNECT":
		f.httpsRoute.HandleF(ctx, w, r)
	default:
		f.httpRoute.HandleF(ctx, w, r)
	}
}
