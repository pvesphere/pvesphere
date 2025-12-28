package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"pvesphere/pkg/log"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
)

func RequestLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// The configuration is initialized once per request
		uuid, err := random.UUIdV4()
		if err != nil {
			return
		}
		trace := cryptor.Md5String(uuid)
		logger.WithValue(ctx, zap.String("trace", trace))
		logger.WithValue(ctx, zap.String("request_method", ctx.Request.Method))
		logger.WithValue(ctx, zap.Any("request_headers", ctx.Request.Header))
		logger.WithValue(ctx, zap.String("request_url", ctx.Request.URL.String()))

		// 对于非 multipart/form-data 请求，记录 body（截断避免过大）
		if ctx.Request.Body != nil {
			ct := ctx.ContentType()
			if strings.HasPrefix(ct, "multipart/form-data") {
				// 大文件上传，避免读取整个 body 进内存
				logger.WithValue(ctx, zap.String("request_params", "[multipart/form-data body omitted]"))
			} else {
				bodyBytes, _ := ctx.GetRawData()
				// 还原 Body，后续 handler 依然可以读取
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				const maxLogBody = 4096
				logBody := bodyBytes
				if len(logBody) > maxLogBody {
					logBody = logBody[:maxLogBody]
				}
				logger.WithValue(ctx, zap.String("request_params", string(logBody)))
			}
		}
		logger.WithContext(ctx).Info("Request")
		ctx.Next()
	}
}
func ResponseLogMiddleware(logger *log.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// WebSocket 请求跳过 ResponseLogMiddleware，避免干扰 WebSocket 握手
		if ctx.GetHeader("Upgrade") == "websocket" {
			startTime := time.Now()
			ctx.Next()
			duration := time.Since(startTime).String()
			logger.WithContext(ctx).Info("Response (WebSocket)", zap.Any("time", duration))
			return
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = blw
		startTime := time.Now()
		ctx.Next()
		duration := time.Since(startTime).String()
		logger.WithContext(ctx).Info("Response", zap.Any("response_body", blw.body.String()), zap.Any("time", duration))
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
