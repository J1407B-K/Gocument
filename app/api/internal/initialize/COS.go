package initialize

import (
	"Gocument/app/api/global"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"net/url"
)

func SetUpCos() {
	u, _ := url.Parse("http://" + global.Config.CosConfig.BucketnameAppid + ".cos." + global.Config.CosConfig.CosRegion + ".myqcloud.com")
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  global.Config.CosConfig.SecretId,
			SecretKey: global.Config.CosConfig.SecretKey,
		},
	})
}
