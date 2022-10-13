/*
Copyright 2022 The KubeBlocks Authors

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

package playground

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPlayground(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PlayGround Suite")
}

var _ = BeforeSuite(func() {
	// set default cluster name to test
	clusterName = "dbctl-playground-test"
	dbClusterName = "dbctl-playground-test-cluster"

	// set fake image info
	K3sImage = "fake-k3s-image"
	K3dToolsImage = "fake-k3s-tools-image"
	K3dProxyImage = "fake-k3d-proxy-image"
})
