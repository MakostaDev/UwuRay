package http_test

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/common"
	. "github.com/xtls/xray-core/common/protocol/http"
)

func TestHTTPHeaders(t *testing.T) {
	cases := []struct {
		input  string
		domain string
		err    bool
	}{
		{
			input: "GET /tutorials/other/top-20-mysql-best-practices/ HTTP/1.1\r\n" +
				"Host: net.tutsplus.com\r\n" +
				"User-Agent: Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.1.5) Gecko/20091102 Firefox/3.5.5 (.NET CLR 3.5.30729)\r\n\r\n",
			domain: "net.tutsplus.com",
		},
		{
			input: "POST /foo.php HTTP/1.1\r\n" +
				"Host: localhost\r\n\r\n",
			domain: "localhost",
		},
		{
			input: "X№ /foo.php HTTP/1.1\r\n" +
				"Host: localhost\r\n\r\n",
			domain: "",
			err:    true,
		},
		{
			input: "GET /foo.php HTTP/1.1\r\n" +
				"User-Agent: x\r\n" +
				"Content-Length: 43\r\n\r\n" +
				"Host: localhost\r\n" +
				"first_name=John",
			domain: "",
			err:    true,
		},
		{
			input:  "GET /tutorials/other/top-20-mysql-best-practices/ HTTP/1.1",
			domain: "",
			err:    true,
		},
		{
			input: "GET http://example.com HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET http://example.com/some/path HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET http://example.com?x=1 HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET http://example.com#frag HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET http://example.com:443/path HTTP/1.1\r\n" +
				"Host: test.example.com\r\n" +
				"Content-Type: text/html\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "CONNECT example.com:443 HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "OPTIONS * HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "test.example.com",
		},
		{
			input: "гет /path HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "",
			err:    true,
		},
		{
			input: "GET / HTTP/1.1\r\n" +
				"Host: [1EeT:S2:A2:tTt::1]:8080\r\n\r\n",
			domain: "1eet:s2:a2:ttt::1",
		},
		{
			input: "GET / HTTP/1.1\r\n" +
				"Host: example.com.\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET http://chicken:leg@example.com/path HTTP/1.1\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input: "GET / HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Host: test.example.com\r\n\r\n",
			domain: "example.com",
		},
		{
			input:  "",
			domain: "",
			err:    true,
		},
		{
			input:  "POS",
			domain: "",
			err:    true,
		},
	}

	for _, test := range cases {
		header, err := SniffHTTP([]byte(test.input), context.TODO())
		if test.err {
			if err == nil {
				t.Errorf("Expect error but nil, in test: %v", test)
			}
		} else {
			if err != nil {
				t.Errorf("Expect no error but actually %s in test %v", err.Error(), test)
				continue
			}
			if header.Domain() != test.domain {
				t.Error("expected domain ", test.domain, " but got ", header.Domain())
				continue
			}

			piece := make([]byte, 0, len(test.input))
			for i := 0; i < len(test.input); i++ {
				b := test.input[i]
				piece = append(piece, b)

				header, err := SniffHTTP(piece, context.TODO())

				if i == len(test.input)-1 {
					if err != nil {
						t.Errorf("Expect no error but actually %s in test %v", err.Error(), test)
						continue
					}
					if header.Domain() != test.domain {
						t.Errorf("expected domain %s, but got %s", test.domain, header.Domain())
					}
				} else if err == nil {
					t.Errorf("Expect error but nil, in test: %v piece: %v", test, string(piece))
				} else if err != common.ErrNoClue {
					t.Errorf("Expect error but not ErrNoClue %s, but got error: %v piece: %v", test.domain, err, string(piece))
				}
			}
		}
	}
}
