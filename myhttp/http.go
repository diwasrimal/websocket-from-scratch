package myhttp

import (
	"errors"
	"fmt"
	"regexp"
)

type Request struct {
	Method   string
	Route    string
	Protocol string
	Headers  map[string]string
	Body     string

}

const httpRequestPattern = `(?s)(?P<method>\w+) (?P<route>[\w\.\d/-]+) (?P<protocol>HTTP/1\.[01])\r\n(?P<header>.*?)\r\n\r\n(?P<body>.*)?`
const httpHeadersPattern = `([\w-]+): (.*)\r\n`

func ParseRequest(msg string) (Request, error) {
	var req Request
	matches := regexp.MustCompile(httpRequestPattern).FindStringSubmatch(msg)

	// Should match 4 or 5 parts, matches contains one more part, matches[0], the string itself
	if len(matches) < 5 || len(matches) > 6 {
		return req, errors.New(fmt.Sprintf("Request matched %v patterns from HTTP request regexp", len(matches)))
	}
	req = Request{
		Method:   matches[1],
		Route:    matches[2],
		Protocol: matches[3],
		Headers:  parseHttpHeaders(matches[4]),
		Body:     matches[5],
	}
	return req, nil
}

func parseHttpHeaders(data string) map[string]string {
	headers := make(map[string]string)
	matches := regexp.MustCompile(httpHeadersPattern).FindAllStringSubmatch(data, -1)
	for _, match := range matches {
		headers[match[1]] = match[2]
	}
	return headers
}
