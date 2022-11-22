//  Copyright Istio Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package ca

import (
	"os"
	"path/filepath"
)

import (
	kubeApiCore "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/test/cert"
	"github.com/apache/dubbo-go-pixiu/pkg/test/util/file"
	"github.com/apache/dubbo-go-pixiu/pkg/test/util/tmpl"
)

const (
	istioConfTemplate = `
[ req ]
encrypt_key = no
prompt = no
utf8 = yes
default_md = sha256
default_bits = 4096
req_extensions = req_ext
x509_extensions = req_ext
distinguished_name = req_dn
[ req_ext ]
subjectKeyIdentifier = hash
basicConstraints = critical, CA:true, pathlen:0
keyUsage = critical, digitalSignature, nonRepudiation, keyEncipherment, keyCertSign
subjectAltName=@san
[ san ]
DNS.1 = istiod.{{ .SystemNamespace }}
DNS.2 = istiod.{{ .SystemNamespace }}.svc
DNS.3 = istio-pilot.{{ .SystemNamespace }}
DNS.4 = istio-pilot.{{ .SystemNamespace }}.svc
[ req_dn ]
O = Istio
CN = Intermediate CA
`
)

// NewIstioConfig creates an extensions configuration for Istio, using the given system namespace in
// the DNS SANs.
func NewIstioConfig(systemNamespace string) (string, error) {
	return tmpl.Evaluate(istioConfTemplate, map[string]interface{}{
		"SystemNamespace": systemNamespace,
	})
}

// IntermediateCA is an intermediate CA for a single cluster.
type Intermediate struct {
	KeyFile  string
	ConfFile string
	CSRFile  string
	CertFile string
	Root     Root
}

// NewIntermediate creates a new intermediate CA for the given cluster.
func NewIntermediate(workDir, config string, root Root) (Intermediate, error) {
	ca := Intermediate{
		KeyFile:  filepath.Join(workDir, "ca-key.pem"),
		ConfFile: filepath.Join(workDir, "ca.conf"),
		CSRFile:  filepath.Join(workDir, "ca.csr"),
		CertFile: filepath.Join(workDir, "ca-cert.pem"),
		Root:     root,
	}

	// Write out the CA config file.
	if err := os.WriteFile(ca.ConfFile, []byte(config), os.ModePerm); err != nil {
		return Intermediate{}, err
	}

	// Create the key for the intermediate CA.
	if err := cert.GenerateKey(ca.KeyFile); err != nil {
		return Intermediate{}, err
	}

	// Create the CSR for the intermediate CA.
	if err := cert.GenerateCSR(ca.ConfFile, ca.KeyFile, ca.CSRFile); err != nil {
		return Intermediate{}, err
	}

	// Create the intermediate cert, signed by the root.
	if err := cert.GenerateIntermediateCert(ca.ConfFile, ca.CSRFile, root.CertFile,
		root.KeyFile, ca.CertFile); err != nil {
		return Intermediate{}, err
	}

	return ca, nil
}

// NewIstioCASecret creates a secret (named "cacerts") containing the intermediate certificate and cert chain.
// If available when Istio starts, this will be used instead of Istio's autogenerated self-signed root
// (istio-ca-secret). This can be used in a multicluster environment in order to establish a common root of
// trust between the clusters.
func (ca Intermediate) NewIstioCASecret() (*kubeApiCore.Secret, error) {
	caCert, err := file.AsString(ca.CertFile)
	if err != nil {
		return nil, err
	}
	caKey, err := file.AsString(ca.KeyFile)
	if err != nil {
		return nil, err
	}
	rootCert, err := file.AsString(ca.Root.CertFile)
	if err != nil {
		return nil, err
	}

	// Create the cert chain by concatenating the intermediate and root certs.
	certChain := caCert + rootCert

	return &kubeApiCore.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cacerts",
		},
		Data: map[string][]byte{
			"ca-cert.pem":    []byte(caCert),
			"ca-key.pem":     []byte(caKey),
			"cert-chain.pem": []byte(certChain),
			"root-cert.pem":  []byte(rootCert),
		},
	}, nil
}
