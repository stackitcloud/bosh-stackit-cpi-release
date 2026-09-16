package integration

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	. "github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"go.yaml.in/yaml/v3"
)

var configFile *os.File

var _ = Describe("Stackit Stages", func() {
	var err error
	BeforeEach(func() {
		configFile, err = os.CreateTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		defer configFile.Close()
		os.Args = []string{"", configFile.Name()}
	})
	Describe("Targeting the default PROD stage", func() {
		var iaasSDKClient *iaas.APIClient
		When("stage profile json is not provided", func() {
			It("the cpi will initialize a stackit client for PROD endpoints", func() {
				testCPI := CustomCPI(
					WithDefaults(),
					WithProject(conf.Prod),
					WithDefaultLogger(),
				)
				iaasSDKClient = conf.Prod.GetStackitIaasClient()
				Expect(testCPI.InitAPIClient()).To(Succeed())
				_, err = iaasSDKClient.DefaultAPI.ListImages(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).Execute()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("the service account json was not provided", func() {
			It("will provide a helpful error message", func() {
				config := map[string]any{
					"stackit": map[string]any{
						"vice_account_json": "",
						"project_id":        conf.Qa.ProjectID,
						"region":            conf.Prod.Region,
					},
				}
				writeConfig(config)

				_, err = cpi.New(configFile.Name())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("cpi.Config: service_account_json is required")))
			})
		})
	})

	Describe("Targeting a non default PROD stage", func() {
		var iaasSDKClient *iaas.APIClient
		When("the Stage Profile is complete", func() {
			It("the cpi will initialize a stackit client for the non prod stage", func() {
				config := map[string]any{
					"stackit": map[string]any{
						"service_account_json": conf.Qa.GetSaKeyString(),
						"stage_profile_json":   conf.Qa.GetStageProfileString(),
						"project_id":           conf.Qa.ProjectID,
						"region":               conf.Qa.Region,
					},
				}
				writeConfig(config)

				testCPI, err := cpi.New(configFile.Name())
				iaasSDKClient = conf.Qa.GetStackitIaasClient()
				Expect(err).ToNot(HaveOccurred())

				images, err := iaasSDKClient.DefaultAPI.ListImages(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(images.GetItems()).ToNot(BeEmpty())
			})
		})
		When("the Stage Profile is misses required values", func() {
			When("the iaas endpoint is missing", func() {
				It("will indicate what is missing", func() {
					badProfileString := `{
  						"token_custom_endpoint": "https://service-account.api.qa.stackit.cloud/token"
					}`
					config := map[string]any{
						"stackit": map[string]any{
							"service_account_json": conf.Prod.GetSaKeyString(),
							"stage_profile_json":   badProfileString,
							"region":               conf.Prod.Region,
							"project_id":           conf.Prod.ProjectID,
						},
					}
					writeConfig(config)

					_, err = cpi.New(configFile.Name())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("expected non empty value for `iaas_custom_endpoint` in stackit profile json")))
				})
			})
			When("the token endpoint is missing", func() {
				It("will indicate what is missing", func() {
					badProfileString := `{
  						"iaas_custom_endpoint": "https://iaas.api.eu01.qa.stackit.cloud"
					}`
					config := map[string]any{
						"stackit": map[string]any{
							"service_account_json": conf.Prod.GetSaKeyString(),
							"stage_profile_json":   badProfileString,
							"region":               "eu01",
							"project_id":           conf.Qa.ProjectID,
						},
					}
					writeConfig(config)

					_, err = cpi.New(configFile.Name())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("expected non empty value for `token_custom_endpoint` in stackit profile json")))
				})
			})
		})
	})
})

func writeConfig(config map[string]any) {
	configBytes, err := yaml.Marshal(config)
	Expect(err).ToNot(HaveOccurred())
	Expect(os.WriteFile(configFile.Name(), configBytes, 0o644)).To(Succeed())
}
