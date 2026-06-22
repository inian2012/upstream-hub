package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/worryzyy/upstream-hub/internal/notify"
	"github.com/worryzyy/upstream-hub/internal/storage"
)

func registerNotifications(g *gin.RouterGroup, d *Deps) {
	g.GET("/notifications/email-defaults", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": emailDefaults()})
	})

	gpc := g.Group("/notifications/channels")
	gpc.GET("", func(c *gin.Context) {
		list, err := d.Notifies.ListChannels()
		if err != nil {
			fail(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})
	gpc.POST("", func(c *gin.Context) { createNotifyChannel(c, d) })
	gpc.PUT("/:id", func(c *gin.Context) { updateNotifyChannel(c, d) })
	gpc.DELETE("/:id", func(c *gin.Context) {
		id, err := uintParam(c, "id")
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		if err := d.Notifies.DeleteChannel(id); err != nil {
			fail(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	gpc.POST("/:id/test", func(c *gin.Context) { testNotify(c, d) })

	g.GET("/notifications/logs", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := d.Notifies.ListLogs(limit)
		if err != nil {
			fail(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})
}

type notifyChannelInput struct {
	Name          string                          `json:"name" binding:"required"`
	Type          storage.NotificationChannelType `json:"type" binding:"required"`
	Config        string                          `json:"config"` // JSON string；编辑时可留空保留原值
	Subscriptions string                          `json:"subscriptions"`
	Enabled       bool                            `json:"enabled"`
}

// normalizeSubscriptions 把输入的订阅 JSON 字符串规整为 "[]" 或合法 JSON 数组。
// 解析失败返回错误以便 API 返回 400。
func normalizeSubscriptions(raw string) (string, error) {
	if raw == "" || raw == "null" {
		return "[]", nil
	}
	var list []notify.Subscription
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return "", err
	}
	out, err := json.Marshal(list)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func createNotifyChannel(c *gin.Context, d *Deps) {
	var in notifyChannelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if in.Config == "" {
		fail(c, http.StatusBadRequest, errors.New("config is required"))
		return
	}
	subs, err := normalizeSubscriptions(in.Subscriptions)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	configJSON, err := prepareNotifyConfig(in.Type, in.Config, "")
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	cipherCfg, err := d.Cipher.Encrypt(configJSON)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	ch := &storage.NotificationChannel{
		Name:          in.Name,
		Type:          in.Type,
		ConfigCipher:  cipherCfg,
		Subscriptions: subs,
		Enabled:       in.Enabled,
	}
	if err := d.Notifies.CreateChannel(ch); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ch})
}

func updateNotifyChannel(c *gin.Context, d *Deps) {
	id, err := uintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	ch, err := d.Notifies.FindChannel(id)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	var in notifyChannelInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	subs, err := normalizeSubscriptions(in.Subscriptions)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	ch.Name = in.Name
	ch.Type = in.Type
	ch.Enabled = in.Enabled
	ch.Subscriptions = subs
	if in.Config != "" {
		configJSON, err := mergeNotifyConfig(d, ch, in.Config)
		if err != nil {
			fail(c, http.StatusBadRequest, err)
			return
		}
		cipherCfg, err := d.Cipher.Encrypt(configJSON)
		if err != nil {
			fail(c, http.StatusInternalServerError, err)
			return
		}
		ch.ConfigCipher = cipherCfg
	}
	if err := d.Notifies.UpdateChannel(ch); err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ch})
}

func mergeNotifyConfig(d *Deps, ch *storage.NotificationChannel, next string) (string, error) {
	oldRaw := ""
	if ch.Type != storage.NotifyEmail {
		return next, nil
	}
	var err error
	oldRaw, err = d.Cipher.Decrypt(ch.ConfigCipher)
	if err != nil {
		return "", err
	}
	return prepareNotifyConfig(ch.Type, next, oldRaw)
}

func prepareNotifyConfig(t storage.NotificationChannelType, next string, oldRaw string) (string, error) {
	if t != storage.NotifyEmail {
		return next, nil
	}
	var oldCfg map[string]any
	if strings.TrimSpace(oldRaw) != "" {
		if err := json.Unmarshal([]byte(oldRaw), &oldCfg); err != nil {
			return "", err
		}
	}
	var nextCfg map[string]any
	if err := json.Unmarshal([]byte(next), &nextCfg); err != nil {
		return "", err
	}
	merged := make(map[string]any, len(oldCfg)+len(nextCfg))
	for k, v := range oldCfg {
		merged[k] = v
	}
	for k, v := range nextCfg {
		if isEmptyEmailConfigValue(k, v) {
			continue
		}
		merged[k] = v
	}
	applyEmailDefaults(merged)
	if strings.TrimSpace(stringValue(merged["password"])) == "" {
		if password := stringValue(oldCfg["password"]); password != "" {
			merged["password"] = password
		} else if password := emailDefaultPassword(); password != "" {
			merged["password"] = password
		} else {
			delete(merged, "password")
		}
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type emailDefaultsResponse struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	From               string `json:"from"`
	FromName           string `json:"from_name"`
	To                 string `json:"to"`
	UseTLS             bool   `json:"use_tls"`
	PasswordConfigured bool   `json:"password_configured"`
}

func emailDefaults() emailDefaultsResponse {
	username := envFirst("UPSTREAM_HUB_SMTP_USERNAME", "SMTP_USERNAME")
	from := envFirst("UPSTREAM_HUB_SMTP_FROM", "SMTP_FROM")
	if from == "" {
		from = username
	}
	to := envFirst("UPSTREAM_HUB_SMTP_TO", "SMTP_TO")
	if to == "" {
		to = from
	}
	return emailDefaultsResponse{
		Host:               envDefault("smtp.gmail.com", "UPSTREAM_HUB_SMTP_HOST", "SMTP_HOST"),
		Port:               envIntDefault(465, "UPSTREAM_HUB_SMTP_PORT", "SMTP_PORT"),
		Username:           username,
		From:               from,
		FromName:           envDefault("Sub2API", "UPSTREAM_HUB_SMTP_FROM_NAME", "SMTP_FROM_NAME"),
		To:                 to,
		UseTLS:             envBoolDefault(true, "UPSTREAM_HUB_SMTP_USE_TLS", "SMTP_USE_TLS"),
		PasswordConfigured: emailDefaultPassword() != "",
	}
}

func applyEmailDefaults(cfg map[string]any) {
	defaults := emailDefaults()
	if strings.TrimSpace(stringValue(cfg["host"])) == "" {
		cfg["host"] = defaults.Host
	}
	if isMissingNumber(cfg["port"]) {
		cfg["port"] = defaults.Port
	}
	if strings.TrimSpace(stringValue(cfg["username"])) == "" && defaults.Username != "" {
		cfg["username"] = defaults.Username
	}
	if strings.TrimSpace(stringValue(cfg["from"])) == "" && defaults.From != "" {
		cfg["from"] = defaults.From
	}
	if strings.TrimSpace(stringValue(cfg["from_name"])) == "" {
		cfg["from_name"] = defaults.FromName
	}
	if isMissingList(cfg["to"]) && defaults.To != "" {
		cfg["to"] = splitEmailList(defaults.To)
	}
	if _, ok := cfg["use_tls"]; !ok {
		cfg["use_tls"] = defaults.UseTLS
	}
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func isEmptyEmailConfigValue(key string, v any) bool {
	if key == "use_tls" {
		return false
	}
	if strings.TrimSpace(stringValue(v)) == "" {
		switch v.(type) {
		case string:
			return true
		}
	}
	return isMissingNumber(v) || isMissingList(v)
}

func isMissingNumber(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case float64:
		return x == 0
	case int:
		return x == 0
	default:
		return false
	}
}

func isMissingList(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case []any:
		return len(x) == 0
	case []string:
		return len(x) == 0
	default:
		return false
	}
}

func splitEmailList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func emailDefaultPassword() string {
	return envFirst("UPSTREAM_HUB_SMTP_PASSWORD", "SMTP_PASSWORD")
}

func envDefault(def string, keys ...string) string {
	if v := envFirst(keys...); v != "" {
		return v
	}
	return def
}

func envFirst(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func envIntDefault(def int, keys ...string) int {
	raw := envFirst(keys...)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func envBoolDefault(def bool, keys ...string) bool {
	raw := envFirst(keys...)
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func testNotify(c *gin.Context, d *Deps) {
	id, err := uintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	ch, err := d.Notifies.FindChannel(id)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	msg := notify.Message{
		Subject: "[upstream-hub] 测试邮件",
		Body:    "这是一条来自 upstream-hub 的测试邮件。收到这封邮件说明 SMTP 配置可用，余额阈值提醒也会通过此邮箱渠道发送。",
	}
	if err := d.Dispatcher.Send(c.Request.Context(), ch, msg); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
