package httpkit

import (
	"mime"
	"net/http"

	"github.com/vikstrous/go-swagger/errors"
)

// ContentType parses a content type header
func ContentType(headers http.Header) (string, string, *errors.ParseError) {
	ct := headers.Get(HeaderContentType)
	orig := ct
	if ct == "" {
		ct = DefaultMime
	}

	mt, opts, err := mime.ParseMediaType(ct)
	if err != nil {
		return "", "", errors.NewParseError(HeaderContentType, "header", orig, err)
	}

	if cs, ok := opts[charsetKey]; ok {
		return mt, cs, nil
	}

	return mt, "", nil
}
