package handler
// internal/http/handler/links.go


import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"urlShort/internal/http/httpx"
)

type LinksService interface {
	// Create создаёт короткую ссылку и возвращает code (например "Ab3kL1Z")
	Create(ctx context.Context, originalURL string) (string, error)
}

type LinksHandler struct {
	svc     LinksService
	baseURL string // например "http://localhost:8080"
}

func NewLinksHandler(svc LinksService, baseURL string) *LinksHandler {
	baseURL = strings.TrimRight(baseURL, "/")
	return &LinksHandler{svc: svc, baseURL: baseURL}
}

// --- DTO ---

type CreateLinkRequest struct {
	URL string `json:"url"`
}

type CreateLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

// --- Handlers ---

// CreateLink handles POST /links
func (h *LinksHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	// ограничим тело запроса, чтобы не принять гигабайт JSON
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req CreateLinkRequest
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "Некорректный JSON")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_url", "URL не должен быть пустым")
		return
	}

	code, err := h.svc.Create(r.Context(), req.URL)
	if err != nil {
		// бизнес-ошибки маппим на HTTP статусы
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			httpx.WriteError(w, http.StatusBadRequest, "invalid_url", "Некорректный URL")
			return
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Внутренняя ошибка")
			return
		}
	}

	resp := CreateLinkResponse{
		Code:     code,
		ShortURL: h.baseURL + "/" + code,
	}

	w.Header().Set("Location", "/"+code)
	httpx.WriteJSON(w, http.StatusCreated, resp)
}

// Заглушки на будущее (следующие дни плана)

// Redirect handles GET /{code}
func (h *LinksHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotImplemented, "not_implemented", "Redirect пока не реализован")
}

// Stats handles GET /links/{code}/stats
func (h *LinksHandler) Stats(w http.ResponseWriter, r *http.Request) {
	httpx.WriteError(w, http.StatusNotImplemented, "not_implemented", "Stats пока не реализован")
}
