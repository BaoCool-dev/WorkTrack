package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
)

// errorBody là định dạng lỗi chuẩn: {"error":{"code":"X","message":"..."}}.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// httpError mang sẵn status + mã lỗi + thông điệp tiếng Việt cho người dùng.
type httpError struct {
	status int
	code   string
	msg    string
}

func (e *httpError) Error() string { return e.code + ": " + e.msg }

var errDBUnavailable = &httpError{
	status: http.StatusServiceUnavailable,
	code:   "DB_UNAVAILABLE",
	msg:    "Không kết nối được cơ sở dữ liệu",
}

// writeError là nơi DUY NHẤT ánh xạ lỗi → HTTP status + mã lỗi.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var he *httpError
	switch {
	case errors.As(err, &he):
		writeJSON(w, he.status, errorBody{Error: errorDetail{Code: he.code, Message: he.msg}})
	default:
		slog.ErrorContext(r.Context(), "unhandled error", "err", err)
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: errorDetail{
			Code:    "INTERNAL",
			Message: "Đã có lỗi xảy ra, vui lòng thử lại sau",
		}})
	}
}
