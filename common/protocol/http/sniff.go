package http

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
)

type version byte

const (
	HTTP1 version = iota
	HTTP2
)

type SniffHeader struct {
	version version
	host    string
}

func (h *SniffHeader) Protocol() string {
	switch h.version {
	case HTTP1:
		return "http1"
	case HTTP2:
		return "http2"
	default:
		return "unknown"
	}
}

func (h *SniffHeader) Domain() string {
	return h.host
}

var (
	spaceByte     = []byte(" ")
	schemeSepByte = []byte("://")
	crlfByte      = []byte("\r\n")
	colonByte     = []byte(":")
	hostKeyByte   = []byte("host")
	httpVerPrefix = []byte("HTTP/1.")

	errNotHTTP     = errors.New("not an HTTP")
	ErrNoHostFound = errors.New("no Host header found")
)

var tcharTable = func() (t [256]bool) {
	for c := 'a'; c <= 'z'; c++ {
		t[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		t[c] = true
	}
	for _, c := range []byte("!#$%&'*+-.^_`|~") {
		t[c] = true
	}
	return
}()

func isValidMethodToken(b []byte) bool {
	for _, c := range b {
		if !tcharTable[c] {
			return false
		}
	}
	return true
}

func SniffHTTP(b []byte, c context.Context) (*SniffHeader, error) {
	attrs := make(map[string]string)

	method, _, found := bytes.Cut(b, spaceByte)
	switch {
	case !found && isValidMethodToken(b):
		return nil, common.ErrNoClue
	case !found || !isValidMethodToken(method) || len(method) == 0:
		return nil, errNotHTTP
	}

	req, afterReqLine, found := bytes.Cut(b, crlfByte)
	if !found {
		return nil, common.ErrNoClue
	}
	if len(req) < 12 {
		return nil, errNotHTTP
	}

	_, rest, ok1 := bytes.Cut(req, spaceByte)
	uri, ver, ok2 := bytes.Cut(rest, spaceByte)
	if !ok1 || !ok2 {
		return nil, errNotHTTP
	}
	if !bytes.HasPrefix(ver, httpVerPrefix) {
		return nil, errNotHTTP
	}
	if len(uri) == 0 {
		return nil, errNotHTTP
	}

	sh := &SniffHeader{
		version: HTTP1,
	}

	// Parse request line
	// Request line is like this
	// "GET /homo/114514 HTTP/1.1"
	attrs[":method"] = string(method)
	if uri[0] == '/' {
		attrs[":path"] = string(uri)
	}

	if uri[0] != '/' && uri[0] != '*' {
		if _, afterScheme, found := bytes.Cut(uri, schemeSepByte); found {
			uri = afterScheme

			var path string
			if i := bytes.IndexAny(uri, "/?#"); i >= 0 {
				path = string(uri[i:])
				if path[0] != '/' {
					path = "/" + path
				}
				uri = uri[:i]
			} else {
				path = "/"
			}
			attrs[":path"] = path

			if i := bytes.LastIndexByte(uri, '@'); i >= 0 {
				uri = uri[i+1:]
			}
		}

		if host, ok := sniffHost(uri); ok {
			sh.host = host
		}
	}

	rest = afterReqLine
	for {
		line, tail, found := bytes.Cut(rest, crlfByte)
		if !found {
			return nil, common.ErrNoClue
		}
		if len(line) == 0 {
			break
		}
		rest = tail

		key, value, found := bytes.Cut(line, colonByte)
		if !found {
			continue
		}

		value = bytes.TrimSpace(value)
		if sh.host == "" && bytes.EqualFold(key, hostKeyByte) {
			if host, ok := sniffHost(value); ok {
				sh.host = host
			}
		}
		attrs[strings.ToLower(string(key))] = string(value) // Put header in attribute
	}

	// If content.Attributes have information, that means it comes from HTTP inbound PlainHTTP mode.
	// It will set attributes, so skip it.
	content := session.ContentFromContext(c)
	if content != nil && len(content.Attributes) == 0 {
		content.Attributes = attrs
	}

	if sh.host == "" {
		return nil, ErrNoHostFound
	}
	return sh, nil
}

func sniffHost(raw []byte) (string, bool) {
	dest, err := ParseHost(string(raw), net.Port(80))
	if err != nil {
		return "", false
	}
	return dest.Address.String(), true
}
