package records

import (
	"net/http"
	"strings"
	"testing"
)

func TestHTTPRecordsRender(t *testing.T) {
	r := &HTTPRecords{
		Scheme:     "HTTPS",
		RequestURL: "https://example.com",
		FinalURL:   "https://example.com",
		StatusCode: 200,
		Status:     "200 OK",
		Proto:      "HTTP/2.0",
		Headers: http.Header{
			"Content-Type": []string{"text/html"},
			"Server":       []string{"nginx"},
		},
	}

	out := r.Render()

	for _, w := range []string{
		"[HTTPS]",
		"Status   : 200 OK",
		"Protocol : HTTP/2.0",
		"Content-Type: text/html",
		"Server: nginx",
	} {
		if !strings.Contains(out, w) {
			t.Errorf("Render() missing %q\ngot:\n%s", w, out)
		}
	}
}

func TestHTTPRecordsRenderShowsRedirect(t *testing.T) {
	r := &HTTPRecords{
		Scheme:     "HTTP",
		RequestURL: "http://example.com",
		FinalURL:   "https://example.com/home",
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
	}

	out := r.Render()

	if !strings.Contains(out, "Redirect : http://example.com -> https://example.com/home") {
		t.Errorf("Render() missing redirect line\ngot:\n%s", out)
	}
}

func TestHTTPRecordsRenderNoRedirectWhenSameURL(t *testing.T) {
	r := &HTTPRecords{
		Scheme:     "HTTP",
		RequestURL: "http://example.com",
		FinalURL:   "http://example.com",
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
	}

	if strings.Contains(r.Render(), "Redirect") {
		t.Errorf("Render() should not show a redirect when the URL is unchanged")
	}
}

func TestHTTPRecordsRenderHeadersSorted(t *testing.T) {
	r := &HTTPRecords{
		Scheme: "HTTP",
		Status: "200 OK",
		Proto:  "HTTP/1.1",
		Headers: http.Header{
			"Zeta":  []string{"z"},
			"Alpha": []string{"a"},
			"Mid":   []string{"m"},
		},
	}

	out := r.Render()
	ai := strings.Index(out, "Alpha:")
	mi := strings.Index(out, "Mid:")
	zi := strings.Index(out, "Zeta:")

	if !(ai < mi && mi < zi) {
		t.Errorf("headers not sorted alphabetically: Alpha=%d Mid=%d Zeta=%d\ngot:\n%s", ai, mi, zi, out)
	}
}

func TestHTTPRecordsRenderJoinsMultiValueHeaders(t *testing.T) {
	r := &HTTPRecords{
		Scheme:  "HTTP",
		Status:  "200 OK",
		Proto:   "HTTP/1.1",
		Headers: http.Header{"Set-Cookie": []string{"a=1", "b=2"}},
	}

	if !strings.Contains(r.Render(), "Set-Cookie: a=1, b=2") {
		t.Errorf("Render() should join multi-value headers with ', '\ngot:\n%s", r.Render())
	}
}

func TestHTTPRecordsRenderNoHeaders(t *testing.T) {
	r := &HTTPRecords{Scheme: "HTTP", Status: "200 OK", Proto: "HTTP/1.1"}

	if !strings.Contains(r.Render(), "(none)") {
		t.Errorf("Render() should print (none) when there are no headers\ngot:\n%s", r.Render())
	}
}
