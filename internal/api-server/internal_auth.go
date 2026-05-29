package apiserver

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vamosdalian/kinetic/internal/clusterauth"
)

// InternalAuthMiddleware authenticates worker -> controller traffic on the
// /api/internal/* endpoints using the shared cluster secret. The verification
// always runs; an empty secret simply means the signature is computable by
// anyone (quick-start / single-node default).
func InternalAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body []byte
		if c.Request.Body != nil {
			data, err := io.ReadAll(c.Request.Body)
			if err != nil {
				ResponseError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "failed to read request body")
				c.Abort()
				return
			}
			body = data
			c.Request.Body = io.NopCloser(bytes.NewReader(data))
		}

		err := clusterauth.Verify(
			secret,
			c.GetHeader(clusterauth.HeaderTimestamp),
			c.GetHeader(clusterauth.HeaderSignature),
			c.Request.Method,
			c.Request.URL.Path,
			body,
			time.Now(),
			clusterauth.DefaultTolerance,
		)
		if err != nil {
			ResponseError(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		c.Next()
	}
}
