package http

import (
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/rainbow-me/platform-tools/common/logger"
	interceptors "github.com/rainbow-me/platform-tools/http/interceptors/resty"
)

func NewRestyWithClient(client *http.Client, log *logger.Logger, opt ...interceptors.InterceptorOpt) *resty.Client {
	restyClient := resty.NewWithClient(client)
	interceptors.InjectInterceptors(restyClient, opt...)

	if log != nil {
		restyClient.SetLogger(log)
	}
	return restyClient
}
