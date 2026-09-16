package cpi_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

var r, w, actualStdin *os.File

func setupStdIn(value string) {
	os.Stdin = r

	_, err := io.WriteString(w, value)
	Expect(err).ToNot(HaveOccurred())
	Expect(w.Close()).To(Succeed())
}

var (
	configFile *os.File
	testCPI    *cpi.CPI
	err        error
)

func writeTestConfig(value string) {
	Expect(os.WriteFile(configFile.Name(), []byte(value), 0o644)).To(Succeed())
}

var _ = Describe("providing inputs", Ordered, func() {
	Describe("calling the CPI", func() {
		BeforeEach(func() {
			logfile, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			logger, err := lib.NewCPILogger(logfile.Name(), slog.LevelInfo, 1, 0, false)
			Expect(err).ToNot(HaveOccurred())
			testCPI = &cpi.CPI{
				Log: logger,
			}
			configFile, err = os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
		})

		Describe("parsing the configFile", func() {
			When("when the config is valid", func() {
				It("uses defaults for optional fields", func() {
					writeTestConfig(
						`---
stackit:
  service_account_json: '{"type":"service_account"}'
  project_id: test-project-id`)

					err := testCPI.InitConfig(configFile.Name())
					Expect(err).ToNot(HaveOccurred())

					Expect(testCPI.Config.RegionID).To(Equal("eu01"))
					Expect(testCPI.Config.LogLevel).To(Equal("info"))
					Expect(testCPI.Config.Timeout).To(Equal(600))
					Expect(testCPI.Config.RetryCount).To(Equal(3))
					Expect(testCPI.Config.DefaultRootVolumeType).To(Equal("storage_premium_perf2"))
					Expect(testCPI.Config.DefaultPersistentVolumeType).To(Equal("storage_premium_perf2"))
				})
			})

			When("the config is invalid", func() {
				When("the specified path doesn't exist", func() {
					It("will provide a helpful error for the missing path", func() {
						err := testCPI.InitConfig("/non_existing_garbage_path.yml")
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
					})
				})
				When("the file contains invalid yaml", func() {
					It("will point out that the yaml couldn't be deserialized", func() {
						writeTestConfig(
							`---
stackit:
  project_id; test-project-id`,
						)
						err := testCPI.InitConfig(configFile.Name())
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("failed to unmarshal")))
					})
				})
				When("required values are missing", func() {
					It("will provide a helpful error for service_account_json missing", func() {
						writeTestConfig(
							`---
stackit:
  project_id: test-project-id`,
						)
						err := testCPI.InitConfig(configFile.Name())
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("service_account_json is required")))
					})
					It("will provide a helpful error for project_id missing", func() {
						writeTestConfig(
							`---
stackit:
  service_account_json: '{"type":"service_account"}'`,
						)
						err := testCPI.InitConfig(configFile.Name())
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("project_id is required")))
					})
				})
			})

			When("the log level is changed", func() {
				It("should reflect that in the settings", func() {
					writeTestConfig(
						`---
stackit:
  service_account_json: '{"type":"service_account"}'
  project_id: test-project-id
  log_level: error
`)

					err := testCPI.InitConfig(configFile.Name())
					Expect(err).ToNot(HaveOccurred())
					Expect(testCPI.Config.LogLevel).To(Equal("error"))
				})
			})
		})

		Describe("parsing stdin", func() {
			BeforeAll(func() {
				actualStdin = os.Stdin
			})
			BeforeEach(func() {
				var err error
				r, w, err = os.Pipe()
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				os.Stdin = actualStdin
				r.Close()
			})

			When("the cpi receives valid input", func() {
				It("will properly deserialize it", func() {
					setupStdIn(`{"method": "info", "arguments": []}`)

					req, err := cpi.ParseStdIn()
					Expect(err).ToNot(HaveOccurred())
					Expect(req.Method == "info")
				})
				It("will properly deserialize it", func() {
					setupStdIn(`{"method": "create_vm", "arguments": ["one","two","three"]}`)

					req, err := cpi.ParseStdIn()
					Expect(err).ToNot(HaveOccurred())
					Expect(req.Method == "create_vm")
					Expect(req.Arguments).To(Equal(json.RawMessage(`["one","two","three"]`)))
				})
			})
			When("the cpi receives invalid input", func() {
				When("the input is bad json", func() {
					It("will provide a helpful error", func() {
						setupStdIn(`not a json`)

						_, err := cpi.ParseStdIn()
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("bad input:")))
					})
				})
			})
		})
	})
})
