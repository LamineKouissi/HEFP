package filters

import (
	"context"
	"net/http"
)

type Auth struct {
	nextFilter Filter
}
type contextKey int

const (
	authKey contextKey = iota
)

func AuthFromCtx(ctx context.Context) (string, bool) {
	auth, ok := ctx.Value(authKey).(string)
	return auth, ok
}
func (au *Auth) Process(req *http.Request, res *http.Response) error {
	return nil
}

func (au *Auth) SetNextFilter(filter Filter) error {
	au.nextFilter = filter
	return nil
}
