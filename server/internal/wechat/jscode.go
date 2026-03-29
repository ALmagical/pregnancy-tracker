package wechat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type SessionResult struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func Code2Session(appID, secret, code string) (*SessionResult, error) {
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(appID),
		url.QueryEscape(secret),
		url.QueryEscape(code),
	)
	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var out SessionResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("wechat: %d %s", out.ErrCode, out.ErrMsg)
	}
	if out.OpenID == "" {
		return nil, fmt.Errorf("wechat: empty openid")
	}
	return &out, nil
}
