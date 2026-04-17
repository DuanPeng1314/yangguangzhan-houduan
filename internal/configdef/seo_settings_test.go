package configdef

import (
	"testing"

	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
)

func TestAllSettingsIncludesSEOSettings(t *testing.T) {
	wanted := map[string]struct{}{
		constant.KeySeoAutoSubmit.String():       {},
		constant.KeySeoRetryTimes.String():       {},
		constant.KeySeoRetryInterval.String():    {},
		constant.KeySeoBaiduEnable.String():      {},
		constant.KeySeoBaiduSite.String():        {},
		constant.KeySeoBaiduToken.String():       {},
		constant.KeySeoBingEnable.String():       {},
		constant.KeySeoBingApiKey.String():       {},
		constant.KeySeoGoogleEnable.String():     {},
		constant.KeySeoGoogleCredential.String(): {},
	}

	actual := make(map[string]Definition, len(AllSettings))
	for _, def := range AllSettings {
		actual[def.Key.String()] = def
	}

	for key := range wanted {
		if _, ok := actual[key]; !ok {
			t.Fatalf("缺少 SEO 配置定义: %s", key)
		}
	}

	if def, ok := actual[constant.KeySeoBaiduToken.String()]; ok && def.IsPublic {
		t.Fatalf("SEO 敏感配置不应标记为公开: %s", constant.KeySeoBaiduToken)
	}

	if def, ok := actual[constant.KeySeoBingApiKey.String()]; ok && def.IsPublic {
		t.Fatalf("SEO 敏感配置不应标记为公开: %s", constant.KeySeoBingApiKey)
	}

	if def, ok := actual[constant.KeySeoGoogleCredential.String()]; ok && def.IsPublic {
		t.Fatalf("SEO 敏感配置不应标记为公开: %s", constant.KeySeoGoogleCredential)
	}

	if def, ok := actual[constant.KeySeoAutoSubmit.String()]; ok && def.Value != "false" {
		t.Fatalf("SEO 自动推送默认值应为 false，实际为 %q", def.Value)
	}

	if def, ok := actual[constant.KeySeoRetryTimes.String()]; ok && def.Value != "3" {
		t.Fatalf("SEO 重试次数默认值应为 3，实际为 %q", def.Value)
	}

	if def, ok := actual[constant.KeySeoRetryInterval.String()]; ok && def.Value != "1000" {
		t.Fatalf("SEO 重试间隔默认值应为 1000，实际为 %q", def.Value)
	}
}
