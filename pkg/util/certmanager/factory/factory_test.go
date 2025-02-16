/*
Copyright 2022 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package factory

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openyurtio/openyurt/pkg/projectinfo"
	"github.com/openyurtio/openyurt/pkg/yurttunnel/constants"
)

const (
	failed  = "\u2717"
	succeed = "\u2713"
)

var cs = &fake.Clientset{}
var fc = &factory{
	clientset: cs,
}

func TestNewCertManagerFactory(t *testing.T) {
	cs := &fake.Clientset{}
	tests := []struct {
		name      string
		clientSet kubernetes.Interface
		expect    CertManagerFactory
	}{
		{
			"normal",
			cs,
			fc,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			t.Parallel()
			t.Logf("\tTestCase: %s", tt.name)
			{
				get := NewCertManagerFactory(cs)

				if !reflect.DeepEqual(get, tt.expect) {
					t.Fatalf("\t%s\texpect %v, but get %v", failed, tt.expect, get)
				}
				t.Logf("\t%s\texpect %v, get %v", succeed, tt.expect, get)

			}
		}
		t.Run(tt.name, tf)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *CertManagerConfig
		expect error
	}{
		{
			"normal",
			&CertManagerConfig{
				IPs:      []net.IP{},
				DNSNames: []string{},
				IPGetter: func() ([]net.IP, error) {
					dynamicIPs := []net.IP{}
					return dynamicIPs, nil
				},
				ComponentName:  projectinfo.GetServerName(),
				CertDir:        "",
				SignerName:     certificatesv1.KubeletServingSignerName,
				CommonName:     fmt.Sprintf("system:node:%s", constants.YurtTunnelServerNodeName),
				Organizations:  []string{user.NodesGroup},
				ForServerUsage: true,
			},
			nil,
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			t.Parallel()
			t.Logf("\tTestCase: %s", tt.name)
			{
				_, get := fc.New(tt.cfg)

				if !reflect.DeepEqual(get, tt.expect) {
					t.Fatalf("\t%s\texpect %v, but get %v", failed, tt.expect, get)
				}
				t.Logf("\t%s\texpect %v, get %v", succeed, tt.expect, get)

			}
		}
		t.Run(tt.name, tf)
	}
}
