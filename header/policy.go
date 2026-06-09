package header

import (
	"net/http"
	"strings"

	"github.com/skys-mission/lmm-adapter-go/adapter"
)

type Action int

const (
	ActionStrip Action = iota
	ActionPassthrough
)

type Policy struct {
	Passthrough []string
	Mappings    map[string]string
	Strip       []string
	Default     Action
}

func NewPolicy() *Policy {
	return &Policy{
		Passthrough: []string{},
		Mappings:    make(map[string]string),
		Strip:       []string{},
		Default:     ActionStrip,
	}
}

func (p *Policy) WithPassthrough(headers ...string) *Policy {
	p.Passthrough = append(p.Passthrough, headers...)
	return p
}

func (p *Policy) WithMapping(from, to string) *Policy {
	p.Mappings[from] = to
	return p
}

func (p *Policy) WithStrip(headers ...string) *Policy {
	p.Strip = append(p.Strip, headers...)
	return p
}

func (p *Policy) WithDefault(action Action) *Policy {
	p.Default = action
	return p
}

func (p *Policy) Apply(src, dst http.Header) {
	stripSet := make(map[string]bool)
	for _, h := range p.Strip {
		stripSet[strings.ToLower(h)] = true
	}

	passthroughSet := make(map[string]bool)
	for _, h := range p.Passthrough {
		passthroughSet[strings.ToLower(h)] = true
	}

	for key, values := range src {
		lowerKey := strings.ToLower(key)

		if stripSet[lowerKey] {
			continue
		}

		targetKey, mapped := p.Mappings[key]
		if !mapped {
			targetKey, mapped = p.Mappings[lowerKey]
		}
		if mapped {
			for _, v := range values {
				dst.Add(targetKey, v)
			}
			continue
		}

		if passthroughSet[lowerKey] {
			for _, v := range values {
				dst.Add(key, v)
			}
			continue
		}

		if p.Default == ActionPassthrough {
			for _, v := range values {
				dst.Add(key, v)
			}
		}
	}
}

func (p *Policy) ApplyAuth(src http.Header, dst http.Header, targetProtocol string) {
	switch targetProtocol {
	case string(adapter.ProtocolClaudeMessages):
		if auth := src.Get("Authorization"); auth != "" {
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				dst.Set("x-api-key", token)
			} else {
				dst.Set("x-api-key", auth)
			}
		}
	case string(adapter.ProtocolOpenAIChat), string(adapter.ProtocolOpenAIResponses):
		if apiKey := src.Get("x-api-key"); apiKey != "" {
			dst.Set("Authorization", "Bearer "+apiKey)
		} else if auth := src.Get("Authorization"); auth != "" {
			dst.Set("Authorization", auth)
		}
	default:
		if auth := src.Get("Authorization"); auth != "" {
			dst.Set("Authorization", auth)
		}
	}
}

// DefaultProxyPolicy returns a best-effort passthrough policy.
// By default all headers are forwarded except:
//   - Cookie, Authorization — security (auth is handled separately by ApplyAuth)
//   - X-Forwarded-For, X-Real-IP, Host — rebuilt by the proxy
//   - Content-Type, Content-Length, Transfer-Encoding — rebuilt after body conversion
func DefaultProxyPolicy() *Policy {
	return NewPolicy().
		WithStrip(
			"Cookie",
			"Authorization",
			"X-Forwarded-For",
			"X-Real-IP",
			"Host",
			"Content-Type",
			"Content-Length",
			"Transfer-Encoding",
			"Connection",
		).
		WithDefault(ActionPassthrough)
}
