package filters

import "net/http"

type Auth struct {
	nextFilter Filter
}

func (au *Auth) Process(req *http.Request, res *http.Response) error {
	return nil
}

func (au *Auth) SetNextFilter(filter Filter) error {
	au.nextFilter = filter
	return nil
}
