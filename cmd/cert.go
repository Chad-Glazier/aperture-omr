package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	certDir = "data/cert/"
)

var (
	host     string
	validFor time.Duration
	isCA     bool
	rsaBits  int
)

var certCmd = &cobra.Command{
	Use:   "certify",
	Short: "Generates TLS certificates",
	Long: `Generate a self-signed X.509 certificate for the OMR's server 
to use with TLS (that is, to serve over HTTPS). Outputs to 
'` + certDir + `cert.pem' and '` + certDir + `key.pem' and will overwrite 
existing files.
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if rsaBits < 256 {
			return errors.New("rsa-bits cannot be less than 256")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := os.MkdirAll(certDir, 0755); err != nil {
			log.Fatal("error creating data/cert directory")
		}

		generateCerts()
	},
}

func init() {
	rootCmd.AddCommand(certCmd)

	certCmd.Flags().StringVar(&host,
		"host",
		"localhost,127.0.0.1",
		"comma-separated hostnames and IPs to generate a certificate\nfor",
	)
	certCmd.Flags().DurationVar(&validFor,
		"valid-for",
		time.Hour*24*365,
		"duration that certificate is valid for",
	)
	certCmd.Flags().BoolVar(&isCA,
		"ca",
		false,
		"whether this cert should be its own Certificate Authority",
	)
	certCmd.Flags().IntVar(&rsaBits,
		"rsa-bits",
		2048,
		"size of RSA key to generate",
	)
}

func generateCerts() {

	//
	// The bulk of this function has been adapted from the Go standard
	// library's "crypto/tls/generate_cert.go" file. The authors included a
	// copyright statement that I have reproduced below. I have also included
	// the referenced "LICENSE" file as a comment, since it is fairly brief
	// (and it happens to fit inside my 80-column style rule).
	//

	// Copyright 2009 The Go Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.

	/*
		Copyright 2009 The Go Authors.

		Redistribution and use in source and binary forms, with or without
		modification, are permitted provided that the following conditions are
		met:

		   * Redistributions of source code must retain the above copyright
		notice, this list of conditions and the following disclaimer.
		   * Redistributions in binary form must reproduce the above
		copyright notice, this list of conditions and the following disclaimer
		in the documentation and/or other materials provided with the
		distribution.
		   * Neither the name of Google LLC nor the names of its
		contributors may be used to endorse or promote products derived from
		this software without specific prior written permission.

		THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
		"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
		LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
		A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
		OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
		SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
		LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
		DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
		THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
		(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
		OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
	*/

	var (
		priv *rsa.PrivateKey
		err  error
	)

	priv, err = rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// ECDSA, ED25519, ML-DSA, and RSA subject keys should have the
	// DigitalSignature KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature
	// RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	keyUsage |= x509.KeyUsageKeyEncipherment

	var (
		notBefore = time.Now()
		notAfter  = notBefore.Add(validFor)

		serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)
	)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.SplitSeq(host, ",")
	for h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certDir + "cert.pem")
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %v", err)
	}
	log.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile(certDir+"key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open key.pem for writing: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing key.pem: %v", err)
	}
	log.Print("wrote key.pem\n")
}
