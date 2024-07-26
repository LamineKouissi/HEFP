package connectors

import "net/http"

type UpstreamProxyConnector struct {
}

func (usc *UpstreamProxyConnector) Process(req *http.Request, res *http.Response) error {
	return nil
}
