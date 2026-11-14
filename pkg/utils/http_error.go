package utils

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HTTPError adalah error bisnis yang membawa status HTTP dan pesan aman untuk klien.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

func ErrBadRequest(msg string) error { return &HTTPError{http.StatusBadRequest, msg} }
func ErrForbidden(msg string) error  { return &HTTPError{http.StatusForbidden, msg} }
func ErrNotFound(msg string) error   { return &HTTPError{http.StatusNotFound, msg} }
func ErrConflict(msg string) error   { return &HTTPError{http.StatusConflict, msg} }

// RespondError memetakan error ke status HTTP. Error tak dikenal dicatat di log
// dan dikembalikan sebagai 500 tanpa membocorkan detail internal.
func RespondError(ctx *gin.Context, message string, err error) {
	var he *HTTPError
	switch {
	case errors.As(err, &he):
		ctx.AbortWithStatusJSON(he.Status, BuildResponseFailed(message, he.Message, nil))
	case errors.Is(err, gorm.ErrRecordNotFound):
		ctx.AbortWithStatusJSON(http.StatusNotFound, BuildResponseFailed(message, "data tidak ditemukan", nil))
	default:
		log.Printf("%s: %v", message, err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, BuildResponseFailed(message, "terjadi kesalahan pada server", nil))
	}
}
