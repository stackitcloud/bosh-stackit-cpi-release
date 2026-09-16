/*
Package integration_test contains our integration tests for the Stackit CPI impementation
*/
package integration

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
)

var (
	conf      TestConfig
	pubIpTags map[string]any
)

var _ = BeforeSuite(func() {
	// check if TEST_CONFIG_PATH was provded via ENV
	testConfigPath := os.Getenv("TEST_CONFIG_PATH")
	Expect(testConfigPath).ToNot(BeEmpty())
	configBytes, err := os.ReadFile(testConfigPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(json.Unmarshal(configBytes, &conf)).To(Succeed())
	pubIpTags = map[string]any{
		"name": "test",
	}
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
