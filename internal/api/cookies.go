package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) cookieSecure(c *gin.Context) bool {
	switch strings.ToLower(h.config.SecureCookiesMode) {
	case "true", "force", "secure":
		return true
	case "false", "off", "insecure":
		return false
	default:
		return requestIsSecure(c, h.config.TrustProxyHeaders)
	}
}

func requestIsSecure(c *gin.Context, trustProxy bool) bool {
	if c.Request.TLS != nil {
		return true
	}
	if !trustProxy {
		return false
	}
	if proto := firstForwardedProto(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		return strings.EqualFold(proto, "https")
	}
	if forwarded := c.GetHeader("Forwarded"); forwarded != "" {
		if proto := forwardedProto(forwarded); proto != "" {
			return strings.EqualFold(proto, "https")
		}
	}
	return false
}

func firstForwardedProto(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func forwardedProto(header string) string {
	// Example: Forwarded: for=192.0.2.60;proto=https;by=203.0.113.43
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv[0]), "proto") {
			return strings.Trim(strings.TrimSpace(kv[1]), "\"")
		}
	}
	return ""
}
