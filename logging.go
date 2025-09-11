// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 1. 读取 Body 内容
		// io.ReadAll 会从 r.Body 中读取所有数据直到 EOF
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			// 如果读取失败，记录错误并继续，此时 body 为空
			log.Printf("Failed to read request body: %v", err)
		}

		// 2. 将 Body 内容重新放回 r.Body
		// 因为 r.Body 是一个流，读取后就空了，必须重新创建一个
		// io.NopCloser 包装一个 io.Reader (bytes.Buffer) 使其成为一个 io.ReadCloser
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 3. 将 bodyBytes 转换为字符串用于打印
		// 注意：如果 Body 是二进制文件（如图片上传），这里会打印出乱码，这是正常的
		bodyString := string(bodyBytes)

		// 4. 打印优化后的日志
		// 使用 %s 来打印 body 字符串
		log.Printf("[REQUEST] %s | %s | %s %s | body: %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			bodyString)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration)
	})
}
