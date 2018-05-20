package main

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/net/html"
)

var testPkgMap = map[string]string{
	"pkg1": "me/pkg1",
	"pkg2": "org/pkg2",
	"pkg3": "org/otherreponame",
}

var testHandler = handler("go.example.com", "/x/", "github.com/go/packages", testPkgMap)

func TestHandler(t *testing.T) {
	tt := []struct {
		path                           string
		goImport, goSource, refreshURL string
	}{
		{
			path:       "/x/pkg1",
			goImport:   `go.example.com/x/pkg1 git https://github.com/me/pkg1`,
			goSource:   `go.example.com/x/pkg1 https://github.com/me/pkg1/ https://github.com/me/pkg1/tree/master{/dir} https://github.com/me/pkg1/blob/master{/dir}/{file}#L{line}`,
			refreshURL: `0; url=https://godoc.org/go.example.com/x/pkg1`,
		},
		{
			path:       "/x/pkg1/sub/pkg",
			goImport:   `go.example.com/x/pkg1 git https://github.com/me/pkg1`,
			goSource:   `go.example.com/x/pkg1 https://github.com/me/pkg1/ https://github.com/me/pkg1/tree/master{/dir} https://github.com/me/pkg1/blob/master{/dir}/{file}#L{line}`,
			refreshURL: `0; url=https://godoc.org/go.example.com/x/pkg1/sub/pkg`,
		},
		{
			path:       "/x/pkg3",
			goImport:   `go.example.com/x/pkg3 git https://github.com/org/otherreponame`,
			goSource:   `go.example.com/x/pkg3 https://github.com/org/otherreponame/ https://github.com/org/otherreponame/tree/master{/dir} https://github.com/org/otherreponame/blob/master{/dir}/{file}#L{line}`,
			refreshURL: `0; url=https://godoc.org/go.example.com/x/pkg3`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.path, func(t *testing.T) {
			req := events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       tc.path,
			}
			resp, err := testHandler(req)
			if err != nil {
				t.Fatalf("failed to run handler: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status code doesn't match (want %d, got %d)", http.StatusOK, resp.StatusCode)
			}

			body := strings.NewReader(resp.Body)
			doc := parseDoc(body)

			if doc.goImport != tc.goImport {
				t.Errorf("unexpected `go-import` meta content (want %q, got %q)", tc.goImport, doc.goImport)
			}

			if doc.goSource != tc.goSource {
				t.Errorf("unexpected `go-source` meta content (want %q, got %q)", tc.goSource, doc.goSource)
			}

			if doc.refreshURL != tc.refreshURL {
				t.Errorf("unexpected `http-refresh` meta url (want %q, got %q)", tc.refreshURL, doc.refreshURL)
			}
		})
	}
}

func TestHandlerNoPkg(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodGet,
		Path:       "/x/",
	}
	resp, err := testHandler(req)
	if err != nil {
		t.Fatalf("failed to run handler: %v", err)
	}

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("status code doesn't match (want %d, got %d)", http.StatusTemporaryRedirect, resp.StatusCode)
	}

	u, ok := resp.Headers["Location"]
	if !ok {
		t.Error("Location header missing")
	}

	wantURL := "github.com/go/packages"
	if u != wantURL {
		t.Errorf("Location header doesn't match (want: %s, got: %s)", wantURL, u)
	}

}

func TestHandlerNotFound(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodGet,
		Path:       "/x/willnevereverexist",
	}
	resp, err := testHandler(req)
	if err != nil {
		t.Fatalf("failed to run handler: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status code doesn't match (want %d, got %d)", http.StatusNotFound, resp.StatusCode)
	}
}

func TestHandlerMethods(t *testing.T) {
	tt := []struct {
		method string
		status int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusMethodNotAllowed},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodDelete, http.StatusMethodNotAllowed},
		{http.MethodTrace, http.StatusMethodNotAllowed},
		{http.MethodOptions, http.StatusMethodNotAllowed},
	}

	for _, tc := range tt {
		t.Run(tc.method, func(t *testing.T) {
			req := events.APIGatewayProxyRequest{
				HTTPMethod: tc.method,
				Path:       "/x/pkg1",
			}
			resp, err := testHandler(req)
			if err != nil {
				t.Fatalf("failed to run handler: %v", err)
			}

			if resp.StatusCode != tc.status {
				t.Errorf("status code doesn't match (want %d, got %d)", tc.status, resp.StatusCode)
			}
		})
	}
}

func getAttr(t html.Token, name string) string {
	for _, attr := range t.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func parseDoc(r io.Reader) doc {
	z := html.NewTokenizer(r)

	doc := doc{}

Loop:
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			break Loop
		case html.StartTagToken:
			t := z.Token()

			if t.Data == "meta" {
				switch getAttr(t, "name") {
				case "go-import":
					doc.goImport = getAttr(t, "content")
				case "go-source":
					doc.goSource = getAttr(t, "content")
				}

				if getAttr(t, "http-equiv") == "refresh" {
					doc.refreshURL = getAttr(t, "content")
				}
			}
		}
	}

	return doc
}

type doc struct {
	goSource, goImport, refreshURL string
}
