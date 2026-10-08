// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRemoteCertificates(t *testing.T) {
	name := fmt.Sprintf("tf-acc-remote-tls-%s", internal.RandomSuffix())
	cert, key := selfSignedPEM(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // Pulp drops the comment, the config must still match
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "tls" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = %[1]q
  ca_cert      = %[2]q
  client_cert  = %[3]q
  client_key   = %[4]q
}`, name, "# test CA\n"+cert, cert, key),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_remote.tls", "ca_cert", "# test CA\n"+cert),
					resource.TestCheckResourceAttr("pulp_remote.tls", "client_cert", cert),
					resource.TestCheckResourceAttr("pulp_remote.tls", "client_key", key),
				),
			},
			{ // clear -> Pulp must receive an explicit null
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "tls" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = %[1]q
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("pulp_remote.tls", "ca_cert"),
					resource.TestCheckNoResourceAttr("pulp_remote.tls", "client_cert"),
					resource.TestCheckNoResourceAttr("pulp_remote.tls", "client_key"),
				),
			},
		},
	})
}

func selfSignedPEM(t *testing.T) (cert, key string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tf-acc"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
}
