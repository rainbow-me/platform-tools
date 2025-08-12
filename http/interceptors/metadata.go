package interceptors

import (
	"encoding/json"
	"net/http"

	"github.com/rainbow-me/platform-tools/common/metadata"
)

func InjectRequestInfo(h http.Header, info metadata.RequestInfo) error {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return err
	}
	h.Set(metadata.HeaderXCorrelationID, info.CorrelationID)
	h.Set(metadata.HeaderXRequestID, info.RequestID)
	h.Set(metadata.HeaderXRequestInfo, string(infoJSON))
	return nil
}

func ExtractRequestInfo(h http.Header) (metadata.RequestInfo, bool, error) {
	info := h.Get(metadata.HeaderXRequestInfo)
	if info == "" {
		return metadata.RequestInfo{}, false, nil
	}
	var reqInfo metadata.RequestInfo
	if err := json.Unmarshal([]byte(info), &reqInfo); err != nil {
		return metadata.RequestInfo{}, false, err
	}
	return reqInfo, true, nil
}
