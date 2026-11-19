package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// maxBodyBytes giới hạn kích thước body JSON để chống request quá lớn.
const maxBodyBytes = 1 << 20

func newValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}

// decodeAndValidate đọc body JSON (không cho trường lạ, một object duy nhất) rồi validate.
func decodeAndValidate(w http.ResponseWriter, r *http.Request, v *validator.Validate, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errBadJSON
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errBadJSON
	}
	if err := v.Struct(dst); err != nil {
		return errValidation
	}
	return nil
}
