package flyio

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// flyReplies answers every GraphQL request with the reply its query names.
type flyReplies struct{ expiry, renew string }

func (f flyReplies) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	reply := f.expiry
	if strings.Contains(string(body), "renewCertificate") {
		reply = f.renew
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(bytes.NewBufferString(reply)), Request: req}, nil
}

func useFly(t *testing.T, f flyReplies) {
	t.Helper()
	saved := httpClient
	httpClient = &http.Client{Transport: f}
	t.Cleanup(func() { httpClient = saved })
}

// A renewal counts as done only when the GraphQL reply says so: a reply that
// carries errors, or reports no success, is a failure.
func TestRenewCertReadsTheReply(t *testing.T) {
	for name, reply := range map[string]string{
		"errors":                `{"data":null,"errors":[{"message":"certificate not found"}]}`,
		"no success":            `{"data":{"renewCertificate":{"success":false}}}`,
		"no field":              `{"data":{}}`,
		"errors beside success": `{"data":{"renewCertificate":{"success":true}},"errors":[{"message":"rate limited"}]}`,
	} {
		useFly(t, flyReplies{renew: reply})
		if err := renewCert(context.Background(), "tok", "app", "example.com"); err == nil {
			t.Fatalf("%s: the renewal counted as done", name)
		}
	}
	useFly(t, flyReplies{renew: `{"data":{"renewCertificate":{"success":true}}}`})
	if err := renewCert(context.Background(), "tok", "app", "example.com"); err != nil {
		t.Fatalf("a reported success: %v", err)
	}
}
