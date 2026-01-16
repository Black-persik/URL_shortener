package handler

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"urlShort/internal/http/httpx"
	"urlShort/internal/service"
)

type LinksHandler struct {
	svc            service.LinksService
	baseURL        string
	redirectStatus int
	maxBodyBytes   int64
}

func NewLinksHandler(svc service.LinksService, baseURL string) *LinksHandler {
	return &LinksHandler{
		svc:            svc,
		baseURL:        strings.TrimRight(baseURL, "/"),
		redirectStatus: http.StatusFound, // 302
		maxBodyBytes:   1 << 20,          // 1MB
	}
}

type createLinkRequest struct {
	URL string `json:"url"`
}

type createLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type statsResponse struct {
	Code        string `json:"code"`
	TotalClicks int64  `json:"total_clicks"`
}

func (h *LinksHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}

	var req createLinkRequest
	if err := httpx.ReadJSON(w, r, h.maxBodyBytes, &req); err != nil {
		return
	}

	code, err := h.svc.CreateShortLink(r.Context(), strings.TrimSpace(req.URL))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, createLinkResponse{
		Code:     code,
		ShortURL: h.baseURL + "/" + code,
	})
}

func (h *LinksHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	meta := service.ClickMeta{
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	}

	originalURL, err := h.svc.Resolve(r.Context(), code, meta)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	http.Redirect(w, r, originalURL, h.redirectStatus)
}

func (h *LinksHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	total, err := h.svc.TotalClicks(r.Context(), code)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, statsResponse{
		Code:        code,
		TotalClicks: total,
	})
}

func (h *LinksHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "code not found")
	case errors.Is(err, service.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "conflict")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
