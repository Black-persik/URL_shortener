package httpx
// internal/http/httpx/error.go


import "net/http"

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, code, msg string) {
	var resp ErrorResponse
	resp.Error.Code = code
	resp.Error.Message = msg
	WriteJSON(w, status, resp)
}
