package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/header"
	"github.com/skys-mission/lmm-adapter-go/stream"
)

type Handler struct {
	converter    *adapter.Converter
	srcProtocol  adapter.Protocol
	dstProtocol  adapter.Protocol
	targetURL    string
	headerPolicy *header.Policy
	client       *http.Client
}

type HandlerOption func(*Handler)

func WithHeaderPolicy(policy *header.Policy) HandlerOption {
	return func(h *Handler) {
		h.headerPolicy = policy
	}
}

func WithHTTPClient(client *http.Client) HandlerOption {
	return func(h *Handler) {
		h.client = client
	}
}

func NewHandler(
	converter *adapter.Converter,
	srcProtocol, dstProtocol adapter.Protocol,
	targetURL string,
	opts ...HandlerOption,
) *Handler {
	h := &Handler{
		converter:    converter,
		srcProtocol:  srcProtocol,
		dstProtocol:  dstProtocol,
		targetURL:    targetURL,
		headerPolicy: header.DefaultProxyPolicy(),
		client:       http.DefaultClient,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	convertedReq, report, err := h.converter.ConvertRequest(h.srcProtocol, h.dstProtocol, reqBody)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert request: %v", err), http.StatusBadRequest)
		return
	}

	if len(report.LostFields) > 0 {
		w.Header().Set("X-Conversion-Lost-Fields", fmt.Sprintf("%d", len(report.LostFields)))
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, h.targetURL, bytes.NewReader(convertedReq))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Apply header policy from client request to upstream request
	if h.headerPolicy != nil {
		h.headerPolicy.Apply(r.Header, httpReq.Header)
		h.headerPolicy.ApplyAuth(r.Header, httpReq.Header, string(h.dstProtocol))
	}

	// Add X-Forwarded-For (read prior from original request, policy strips it)
	if clientIP := r.RemoteAddr; clientIP != "" {
		prior := r.Header.Get("X-Forwarded-For")
		if prior != "" {
			httpReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			httpReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	resp, err := h.client.Do(httpReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	isStreaming := IsStreamingRequest(r, convertedReq)

	if resp.StatusCode >= 400 {
		h.handleErrorResponse(w, resp)
		return
	}

	if isStreaming {
		h.handleStreamingResponse(w, resp)
		return
	}

	h.handleNormalResponse(w, resp)
}

func (h *Handler) handleErrorResponse(w http.ResponseWriter, resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read error response: %v", err), http.StatusBadGateway)
		return
	}

	// Forward upstream response headers (best-effort)
	copyResponseHeaders(resp.Header, w.Header())

	convertedErr, _, err := h.converter.ConvertError(h.dstProtocol, h.srcProtocol, body, resp.StatusCode)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(convertedErr)
}

func (h *Handler) handleStreamingResponse(w http.ResponseWriter, resp *http.Response) {
	srcAdapter, err := h.converter.Get(h.dstProtocol)
	if err != nil {
		http.Error(w, fmt.Sprintf("Source adapter not found: %v", err), http.StatusInternalServerError)
		return
	}
	dstAdapter, err := h.converter.Get(h.srcProtocol)
	if err != nil {
		http.Error(w, fmt.Sprintf("Target adapter not found: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward upstream response headers (best-effort)
	copyResponseHeaders(resp.Header, w.Header())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	if flusher, ok := w.(http.Flusher); ok {
		defer flusher.Flush()
	}

	pipeline := stream.NewPipeline(srcAdapter, dstAdapter)
	pipeline.Pipe(resp.Request.Context(), resp.Body, w)
}

func (h *Handler) handleNormalResponse(w http.ResponseWriter, resp *http.Response) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusBadGateway)
		return
	}

	convertedResp, _, err := h.converter.ConvertResponse(h.dstProtocol, h.srcProtocol, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert response: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward upstream response headers (best-effort)
	copyResponseHeaders(resp.Header, w.Header())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(convertedResp)
}

// copyResponseHeaders copies upstream response headers to the client,
// skipping headers that the proxy manages itself (Content-Type, Content-Length, etc.).
func copyResponseHeaders(src, dst http.Header) {
	skip := map[string]bool{
		"content-type":     true,
		"content-length":   true,
		"transfer-encoding": true,
		"connection":       true,
	}
	for key, values := range src {
		if skip[strings.ToLower(key)] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

func IsStreamingRequest(r *http.Request, body []byte) bool {
	if r.URL.Query().Get("stream") == "true" {
		return true
	}

	var req map[string]any
	if err := json.Unmarshal(body, &req); err == nil {
		if stream, ok := req["stream"].(bool); ok && stream {
			return true
		}
	}

	return false
}
