package cmd

import (
	"crypto/x509"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/circleci/ex/rootcerts"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_UpdateDefaultTransport(t *testing.T) {
	ctx := testcontext.Background()

	cleanup := func() {
		combinedOnce = sync.Once{}           // reset
		systemCertPool = x509.SystemCertPool // reset
	}

	t.Run("combined cert pools", func(t *testing.T) {
		t.Cleanup(cleanup)
		err := UpdateDefaultTransport(ctx)
		assert.NilError(t, err)

		wantCertPool, err := x509.SystemCertPool()
		assert.NilError(t, err)

		for _, c := range rootcerts.CertsByTrust(rootcerts.ServerTrustedDelegator) {
			wantCertPool.AddCert(c.X509Cert())
		}

		transport, ok := http.DefaultTransport.(*http.Transport)
		assert.Check(t, ok)
		assert.Check(t, transport.TLSClientConfig.RootCAs.Equal(wantCertPool))
	})

	t.Run("no system cert pool", func(t *testing.T) {
		t.Cleanup(cleanup)
		systemCertPool = func() (*x509.CertPool, error) {
			return x509.NewCertPool(), nil
		}

		err := UpdateDefaultTransport(ctx)
		assert.NilError(t, err)

		wantCertPool := rootcerts.ServerCertPool()

		transport, ok := http.DefaultTransport.(*http.Transport)
		assert.Check(t, ok)
		assert.Check(t, transport.TLSClientConfig.RootCAs.Equal(wantCertPool))
	})

	t.Run("system cert pool error", func(t *testing.T) {
		t.Cleanup(cleanup)
		systemCertPool = func() (*x509.CertPool, error) {
			return nil, errors.New("something happened")
		}

		err := UpdateDefaultTransport(ctx)
		assert.Check(t, cmp.ErrorContains(err, "something happened"))
	})
}
