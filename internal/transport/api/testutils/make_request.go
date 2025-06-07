package testutils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

type RequestOptions struct {
	headers map[string]string
	gziped  bool
	cookies []*http.Cookie
}

type RequestArgs struct {
	Router http.Handler
	Method string
	URL    string
	Body   io.Reader
}

func MakeRequest(args RequestArgs, opts ...func(*RequestOptions)) (*http.Response, error) {
	options := RequestOptions{
		headers: make(map[string]string),
		gziped:  false,
		cookies: nil,
	}
	for _, opt := range opts {
		opt(&options)
	}

	var body io.Reader
	if args.Body != nil {
		body = args.Body
	}

	// Добавляем gzip сжатие тела запроса, если надо.
	if options.gziped && args.Body != nil {
		var gzipBuffer bytes.Buffer
		gzipW, gzErr := gzip.NewWriterLevel(&gzipBuffer, gzip.BestSpeed)
		if gzErr != nil {
			return nil, fmt.Errorf("failed to create gzip writer: %s", gzErr.Error())
		}

		// копируем тело в gzip.Writer.
		_, copyErr := io.Copy(gzipW, args.Body)
		if copyErr != nil {
			return nil, fmt.Errorf("failed to copy request body to gzip writer: %s", copyErr.Error())
		}

		if err := gzipW.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %s", err.Error())
		}
		body = &gzipBuffer
	}

	request := httptest.NewRequest(args.Method, args.URL, body)
	if len(options.headers) > 0 {
		for k, v := range options.headers {
			request.Header.Set(k, v)
		}
	}

	if options.gziped {
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")
	}

	if options.cookies != nil {
		for _, cookie := range options.cookies {
			request.AddCookie(cookie)
		}
	}

	recorder := httptest.NewRecorder()

	args.Router.ServeHTTP(recorder, request)

	return recorder.Result(), nil
}

func WithHeader(name, value string) func(*RequestOptions) {
	return func(fn *RequestOptions) {
		fn.headers[name] = value
	}
}

func WithGzip(b bool) func(*RequestOptions) {
	return func(fn *RequestOptions) {
		fn.gziped = b
	}
}

func WithCookies(c []*http.Cookie) func(*RequestOptions) {
	return func(fn *RequestOptions) {
		fn.cookies = c
	}
}
