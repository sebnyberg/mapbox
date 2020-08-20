package mapbox

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"sync"

	"golang.org/x/net/context/ctxhttp"
)

// putMultiPart uploads the file provided by path to a URL using PUT.
func putMultipart(ctx context.Context, client *http.Client, url string, filename string, r io.Reader) (*http.Response, error) {
	return doMultipart(ctx, client, http.MethodPut, url, filename, r)
}

// postMultiPart uploads the file provided by path to a URL using PUT.
func postMultipart(ctx context.Context, client *http.Client, url string, filename string, r io.Reader) (*http.Response, error) {
	return doMultipart(ctx, client, http.MethodPost, url, filename, r)
}

// doMultipart uploads the provided file
func doMultipart(ctx context.Context, client *http.Client, method string, url string, filename string, r io.Reader) (*http.Response, error) {
	// Create a pipe which will allow the request to read
	// while we are writing blocks from the file.
	bodyReader, bodyWriter := io.Pipe()
	formWriter := multipart.NewWriter(bodyWriter)

	// Store the first write error in writeErr.
	var (
		writeErr error
		errOnce  sync.Once
	)
	setErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { writeErr = err })
		}
	}
	go func() {
		partWriter, err := formWriter.CreateFormFile("file", filename)
		setErr(err)
		_, err = io.Copy(partWriter, r)
		setErr(err)
		setErr(formWriter.Close())
		setErr(bodyWriter.Close())
	}()

	req, err := http.NewRequest(http.MethodPut, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", formWriter.FormDataContentType())

	// This operation will block until both the formWriter
	// and bodyWriter have been closed by the goroutine,
	// or in the event of a HTTP error.
	resp, err := ctxhttp.Do(ctx, client, req)

	if writeErr != nil {
		return nil, writeErr
	}

	return resp, nil
}
