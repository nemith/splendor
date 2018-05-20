package main // import "lambvanity"

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	xPrefix      = "/"                         // prefix after the domain but before the the project name
	xDomain      = "go.brbe.net"               // What domain to use for the vanity import
	xPkgIndexURL = "https://github.com/nemith" // Where to redirect when no package name is given
)

// map of vanity import name to the github project.
var pkgMap = map[string]string{
	"blowme":   "nemith/blowme",
	"goline":   "nemith/goline",
	"he":       "nemith/he",
	"mipples":  "nemith/mipples",
	"tictac":   "nemith/tictac",
	"tvdb":     "nemith/tvdb",
	"splendor": "nemith/splendor",
}

type lambdaAPIGatewayHandler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(domain, prefix, redirectURL string, pkgMap map[string]string) lambdaAPIGatewayHandler {
	return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		if req.HTTPMethod != http.MethodGet {
			return httpError(req, "Method not allowed", http.StatusMethodNotAllowed)
		}

		head, tail := strings.TrimPrefix(req.Path, prefix), ""
		if i := strings.Index(head, "/"); i != -1 {
			head, tail = head[:i], head[i:]
		}

		if head == "" {
			return httpRedirect(req, redirectURL, http.StatusTemporaryRedirect)
		}

		repo, ok := pkgMap[head]
		if !ok {
			return httpNotFound(req)
		}

		data := struct {
			Domain, Prefix, Head, Tail, Repo string
		}{domain, prefix, head, tail, repo}

		body := &bytes.Buffer{}
		if err := respTmpl.Execute(body, data); err != nil {
			log.Printf("failed to render template: %v", err)
		}

		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Cache-Control": "public, max-age=300",
			},
			Body: body.String(),
		}, nil
	}
}

func main() {
	lambda.Start(handler(xDomain, xPrefix, xPkgIndexURL, pkgMap))
}

// Meta tags are in a special format:
//    <meta name="go-import" content="prefix vcs repo-root">
//    <meta mame="go-source" content="prefix home directory file">
//
// https://golang.org/cmd/go/#hdr-Remote_import_paths
// https://github.com/golang/gddo/wiki/Source-Code-Links
var respTmpl = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Domain}}{{.Prefix}}{{.Head}} git https://github.com/{{.Repo}}">
<meta name="go-source" content="{{.Domain}}{{.Prefix}}{{.Head}} https://github.com/{{.Repo}}/ https://github.com/{{.Repo}}/tree/master{/dir} https://github.com/{{.Repo}}/blob/master{/dir}/{file}#L{line}">
<meta http-equiv="refresh" content="0; url=https://godoc.org/{{.Domain}}{{.Prefix}}{{.Head}}{{.Tail}}">
</head>
<body>
Nothing to see here; <a href="https://godoc.org/{{.Domain}}{{.Prefix}}{{.Head}}{{.Tail}}">move along</a>.
</body>
</html>
`))
