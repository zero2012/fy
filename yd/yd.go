package sg

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xwjdsh/fy"
)

type youdao struct{}

var langConvertMap = map[string]string{
	fy.Chinese: "zh-CHS",
}

func init() {
	fy.Register(new(youdao))
}

func (y *youdao) Desc() (string, string, string) {
	return "yd", "youdao", "http://fanyi.youdao.com/"
}

func (y *youdao) Translate(req fy.Request) (resp *fy.Response) {
	resp = fy.NewResp(y)

	r, err := fy.NotReadResp(http.Get("http://youdao.com"))
	if err != nil {
		resp.Err = fmt.Errorf("fy.NotReadResp error: %v", err)
		return
	}
	cookies := r.Cookies()

	// S = "fanyideskweb"
	// D = "ebSeFb%=XZ%T[KZ)c(sy!"
	// r = "" + ((new Date).getTime() + parseInt(10 * Math.random(), 10))
	// o = u.md5(S + n + r + D);
	salt := fmt.Sprintf("%d", int(time.Now().UnixNano()/1000000)+rand.Intn(10))
	h := md5.New()
	h.Write([]byte("fanyideskweb" + req.Text + salt + `ebSeFb%=XZ%T[KZ)c(sy!`))
	sign := hex.EncodeToString(h.Sum(nil))

	if tl, ok := langConvertMap[req.TargetLang]; ok {
		req.TargetLang = tl
	}
	param := url.Values{
		"from":    {"AUTO"},
		"to":      {req.TargetLang},
		"i":       {req.Text},
		"client":  {"fanyideskweb"},
		"salt":    {salt},
		"sign":    {sign},
		"version": {"3.0"},
		"keyfrom": {"fanyi.web"},
	}
	urlStr := "http://fanyi.youdao.com/translate_o"
	body := strings.NewReader(param.Encode())
	_, data, err := fy.SendRequest("POST", urlStr, body, func(req *http.Request) error {
		req.Header.Set("Referer", "http://fanyi.youdao.com/")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X x.y; rv:42.0) Gecko/20100101 Firefox/42.0")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		fy.AddCookies(req, cookies)
		return nil
	})
	if err != nil {
		resp.Err = fmt.Errorf("fy.SendRequest error: %v", err)
		return
	}

	jr := gjson.Parse(string(data))
	if errorCode := jr.Get("errorCode").Int(); errorCode != 0 {
		resp.Err = fmt.Errorf("json result errorCode is %d", errorCode)
		return
	}

	resp.Result = jr.Get("translateResult.0").String()
	return
}
