package main

import (
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func htmlEscape(s string) string {
	return htmlReplacer.Replace(s)
}

func httpRedirect(req events.APIGatewayProxyRequest, url string, code int) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Location": url,
	}

	if req.HTTPMethod == "GET" || req.HTTPMethod == "HEAD" {
		headers["Content-Type"] = "text/html; charset=utf-8"
	}

	var body string
	if req.HTTPMethod == "GET" {
		body = "<a href=\"" + htmlEscape(url) + "\">" + http.StatusText(code) + "</a>.\n"

	}

	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers:    headers,
		Body:       body,
	}, nil
}

func httpError(req events.APIGatewayProxyRequest, error string, code int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Content-Type":           "text/plain; charset=utf-8",
			"X-Context_type-Options": "nosniff",
		},
		StatusCode: code,
		Body:       error,
	}, nil
}

func httpNotFound(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return httpError(req, "404 page not found", http.StatusNotFound)
}
