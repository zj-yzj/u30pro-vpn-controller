//go:build diagnostic

package manager

import (
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestDiagnosticErrorsDoNotReturnPrivateDetails(t *testing.T) {
	for _, err := range []error{&url.Error{Op: "Get", URL: "https://SECRET", Err: x509.UnknownAuthorityError{}}, &net.DNSError{Err: "SECRET", Name: "SECRET"}, &net.OpError{Op: "dial", Err: errors.New("SECRET")}, errors.New("SECRET")} {
		category, typ := diagnosticError(err)
		if category == "" || typ == "" || strings.Contains(category, "SECRET") || strings.Contains(typ, "SECRET") {
			t.Fatal("diagnostic leaked input")
		}
	}
	category, _ := diagnosticError(&url.Error{Op: "Get", URL: "SECRET", Err: x509.UnknownAuthorityError{}})
	if category != "tls_unknown_ca" {
		t.Fatal("TLS not classified")
	}
}
