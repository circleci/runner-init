package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"sync"

	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/rootcerts"
)

var (
	systemCertPool = x509.SystemCertPool

	combinedCertPool *x509.CertPool
	combinedOnce     sync.Once
)

// UpdateDefaultTransport updates the configuration for http.DefaultTransport
// to use the root CA certificates from the system cert pool and those defined in `rootcerts` when used as an HTTP client.
//
// It will return an error if the DefaultTransport is not actually an *http.Transport.
func UpdateDefaultTransport(ctx context.Context) (err error) {
	combinedOnce.Do(func() {
		combinedCertPool, err = systemCertPool()
		if err != nil {
			err := o11y.NewWarning(err.Error())
			o11y.LogError(ctx, "Unable to load system cert pool; using internal cert pool", err)
			combinedCertPool = x509.NewCertPool()
		}
		for _, c := range rootcerts.CertsByTrust(rootcerts.ServerTrustedDelegator) {
			// Merge the system and internal cert pools
			combinedCertPool.AddCert(c.X509Cert())
		}
	})

	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{
				RootCAs:    combinedCertPool,
				MinVersion: tls.VersionTLS12,
			}
		} else {
			t.TLSClientConfig.RootCAs = combinedCertPool
		}
	} else {
		return errors.New("http.DefaultTransport is not an *http.Transport")
	}
	return err
}
