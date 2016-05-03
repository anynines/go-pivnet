package main_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/go-pivnet"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

const (
	apiPrefix = "/api/v2"
	apiToken  = "some-api-token"
)

var _ = Describe("pivnet cli", func() {
	var (
		server *ghttp.Server
		host   string

		product pivnet.Product

		releases []pivnet.Release
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		host = server.URL()

		product = pivnet.Product{
			ID:   1234,
			Slug: "some-product-slug",
			Name: "some-product-name",
		}

		releases = []pivnet.Release{
			{
				ID:          1234,
				Version:     "version 0.2.3",
				Description: "Some release with some description.",
			},
			{
				ID:          2345,
				Version:     "version 0.3.4",
				Description: "Another release with another description.",
			},
		}

	})

	runMainWithArgs := func(args ...string) *gexec.Session {
		args = append(
			args,
			fmt.Sprintf("--api-token=%s", apiToken),
			fmt.Sprintf("--host=%s", host),
		)

		_, err := fmt.Fprintf(GinkgoWriter, "Running command: %v\n", args)
		Expect(err).NotTo(HaveOccurred())

		command := exec.Command(pivnetBinPath, args...)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		return session
	}

	Describe("Displaying help", func() {
		It("displays help with '-h'", func() {
			session := runMainWithArgs("-h")

			Eventually(session, executableTimeout).Should(gexec.Exit())
			Expect(session.Err).Should(gbytes.Say("Usage"))
		})

		It("displays help with '--help'", func() {
			session := runMainWithArgs("--help")

			Eventually(session, executableTimeout).Should(gexec.Exit())
			Expect(session.Err).Should(gbytes.Say("Usage"))
		})
	})

	Describe("Displaying version", func() {
		It("displays version with '-v'", func() {
			session := runMainWithArgs("-v")

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say("dev"))
		})

		It("displays version with '--version'", func() {
			session := runMainWithArgs("--version")

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say("dev"))
		})
	})

	Describe("printing as json", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s", apiPrefix, product.Slug),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, product),
				),
			)
		})

		It("prints as json", func() {
			session := runMainWithArgs(
				"--format=json",
				"product",
				"--product-slug", product.Slug)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))

			var receivedProduct pivnet.Product
			err := json.Unmarshal(session.Out.Contents(), &receivedProduct)
			Expect(err).NotTo(HaveOccurred())

			Expect(receivedProduct.Slug).To(Equal(product.Slug))
		})
	})

	Describe("printing as yaml", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s", apiPrefix, product.Slug),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, product),
				),
			)
		})

		It("prints as yaml", func() {
			session := runMainWithArgs(
				"--format=yaml",
				"product",
				"--product-slug", product.Slug)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))

			var receivedProduct pivnet.Product
			err := yaml.Unmarshal(session.Out.Contents(), &receivedProduct)
			Expect(err).NotTo(HaveOccurred())

			Expect(receivedProduct.Slug).To(Equal(product.Slug))
		})
	})

	Describe("User groups", func() {
		var (
			args []string
		)

		BeforeEach(func() {
			args = []string{"user-groups"}
		})

		Context("when product slug and release version are not provided", func() {
			It("displays all user groups", func() {
				userGroups := []pivnet.UserGroup{
					{
						ID:   1234,
						Name: "Some user group",
					},
					{
						ID:   2345,
						Name: "Another user group",
					},
				}

				userGroupsResponse := pivnet.UserGroups{
					UserGroups: userGroups,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf(
								"%s/user_groups",
								apiPrefix,
							),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, userGroupsResponse),
					),
				)

				session := runMainWithArgs(args...)

				Eventually(session, executableTimeout).Should(gexec.Exit(0))
				Expect(session).Should(gbytes.Say(userGroups[0].Name))
			})
		})

		Context("when product slug and release version are provided", func() {
			BeforeEach(func() {
				args = append(
					args,
					"--product-slug", product.Slug,
					"--release-version", releases[0].Version,
				)
			})

			It("displays user groups for the provided product slug and release version", func() {
				releasesResponse := pivnet.ReleasesResponse{
					Releases: releases,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
					),
				)

				userGroups := []pivnet.UserGroup{
					{
						ID:   1234,
						Name: "Some user group",
					},
					{
						ID:   2345,
						Name: "Another user group",
					},
				}

				userGroupsResponse := pivnet.UserGroups{
					UserGroups: userGroups,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf(
								"%s/products/%s/releases/%d/user_groups",
								apiPrefix,
								product.Slug,
								releases[0].ID,
							),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, userGroupsResponse),
					),
				)

				session := runMainWithArgs(args...)

				Eventually(session, executableTimeout).Should(gexec.Exit(0))
				Expect(session).Should(gbytes.Say(userGroups[0].Name))
			})
		})

		Context("when only product slug is provided", func() {
			BeforeEach(func() {
				args = append(
					args,
					"--product-slug", product.Slug,
				)
			})

			It("exits with error", func() {
				session := runMainWithArgs(args...)

				Eventually(session, executableTimeout).Should(gexec.Exit(1))
				Expect(server.ReceivedRequests()).To(HaveLen(0))
			})
		})

		Context("when only release version is provided", func() {
			BeforeEach(func() {
				args = append(
					args,
					"--release-version", releases[0].Version,
				)
			})

			It("exits with error", func() {
				session := runMainWithArgs(args...)

				Eventually(session, executableTimeout).Should(gexec.Exit(1))
				Expect(server.ReceivedRequests()).To(HaveLen(0))
			})
		})
	})

	Describe("product-files", func() {
		var (
			productFiles []pivnet.ProductFile
		)

		BeforeEach(func() {
			productFiles = []pivnet.ProductFile{
				pivnet.ProductFile{
					ID:   1234,
					Name: "some-product-file",
				},
				pivnet.ProductFile{
					ID:   2345,
					Name: "some-other-product-file",
				},
			}

		})

		It("displays product files", func() {
			response := pivnet.ProductFilesResponse{
				ProductFiles: productFiles,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/product_files",
							apiPrefix,
							product.Slug,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)

			session := runMainWithArgs(
				"product-files",
				"--product-slug", product.Slug,
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say(productFiles[0].Name))
			Expect(session).Should(gbytes.Say(productFiles[1].Name))
		})

		Context("when release version is provided", func() {
			BeforeEach(func() {
				releasesResponse := pivnet.ReleasesResponse{
					Releases: releases,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
					),
				)

				response := pivnet.ProductFilesResponse{
					ProductFiles: productFiles,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases/%d/product_files",
								apiPrefix,
								product.Slug,
								releases[0].ID,
							),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, response),
					),
				)
			})

			It("displays product files for release", func() {
				session := runMainWithArgs(
					"product-files",
					"--product-slug", product.Slug,
					"--release-version", releases[0].Version,
				)

				Eventually(session, executableTimeout).Should(gexec.Exit(0))
				Expect(session).Should(gbytes.Say(productFiles[0].Name))
				Expect(session).Should(gbytes.Say(productFiles[1].Name))
			})
		})
	})

	Describe("product-file", func() {
		var (
			productFile pivnet.ProductFile
		)

		BeforeEach(func() {
			productFile = pivnet.ProductFile{
				ID:   1234,
				Name: "some-product-file",
			}
		})

		It("displays product file", func() {
			response := pivnet.ProductFileResponse{
				ProductFile: productFile,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/product_files/%d",
							apiPrefix,
							product.Slug,
							productFile.ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)

			session := runMainWithArgs(
				"product-file",
				"--product-slug", product.Slug,
				"--product-file-id", strconv.Itoa(productFile.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say(productFile.Name))
		})

		Context("when release version is provided", func() {
			BeforeEach(func() {
				releasesResponse := pivnet.ReleasesResponse{
					Releases: releases,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
					),
				)
			})

			It("displays product files for release", func() {
				response := pivnet.ProductFileResponse{
					ProductFile: productFile,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases/%d/product_files/%d",
								apiPrefix,
								product.Slug,
								releases[0].ID,
								productFile.ID,
							),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, response),
					),
				)

				session := runMainWithArgs(
					"product-file",
					"--product-slug", product.Slug,
					"--release-version", releases[0].Version,
					"--product-file-id", strconv.Itoa(productFile.ID),
				)

				Eventually(session, executableTimeout).Should(gexec.Exit(0))
				Expect(session).Should(gbytes.Say(productFile.Name))
			})
		})
	})

	Describe("add product-file", func() {
		var (
			productFile pivnet.ProductFile
		)

		BeforeEach(func() {
			releasesResponse := pivnet.ReleasesResponse{
				Releases: releases,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
				),
			)

			productFile = pivnet.ProductFile{
				ID:   1234,
				Name: "some-product-file",
			}

			response := pivnet.ProductFileResponse{
				ProductFile: productFile,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"PATCH",
						fmt.Sprintf(
							"%s/products/%s/releases/%d/add_product_file",
							apiPrefix,
							product.Slug,
							releases[0].ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusNoContent, response),
				),
			)
		})

		It("adds product file", func() {
			session := runMainWithArgs(
				"add-product-file",
				"--product-slug", product.Slug,
				"--release-version", releases[0].Version,
				"--product-file-id", strconv.Itoa(productFile.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
		})
	})

	Describe("remove product-file", func() {
		var (
			productFile pivnet.ProductFile
		)

		BeforeEach(func() {
			releasesResponse := pivnet.ReleasesResponse{
				Releases: releases,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
				),
			)

			productFile = pivnet.ProductFile{
				ID:   1234,
				Name: "some-product-file",
			}

			response := pivnet.ProductFileResponse{
				ProductFile: productFile,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"PATCH",
						fmt.Sprintf(
							"%s/products/%s/releases/%d/remove_product_file",
							apiPrefix,
							product.Slug,
							releases[0].ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusNoContent, response),
				),
			)
		})

		It("removes product file", func() {
			session := runMainWithArgs(
				"remove-product-file",
				"--product-slug", product.Slug,
				"--release-version", releases[0].Version,
				"--product-file-id", strconv.Itoa(productFile.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
		})
	})

	Describe("delete product-file", func() {
		var (
			productFile pivnet.ProductFile
		)

		BeforeEach(func() {
			productFile = pivnet.ProductFile{
				ID:   1234,
				Name: "some-product-file",
			}

			response := pivnet.ProductFileResponse{
				ProductFile: productFile,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"DELETE",
						fmt.Sprintf(
							"%s/products/%s/product_files/%d",
							apiPrefix,
							product.Slug,
							productFile.ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)
		})

		It("deletes product file", func() {
			session := runMainWithArgs(
				"delete-product-file",
				"--product-slug", product.Slug,
				"--product-file-id", strconv.Itoa(productFile.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
		})
	})

	Describe("file-groups", func() {
		var (
			fileGroups []pivnet.FileGroup
		)

		BeforeEach(func() {
			fileGroups = []pivnet.FileGroup{
				pivnet.FileGroup{
					ID:   1234,
					Name: "some-file-group",
				},
				pivnet.FileGroup{
					ID:   2345,
					Name: "some-other-file-group",
				},
			}

		})

		It("displays product files", func() {
			response := pivnet.FileGroupsResponse{
				FileGroups: fileGroups,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/file_groups",
							apiPrefix,
							product.Slug,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)

			session := runMainWithArgs(
				"file-groups",
				"--product-slug", product.Slug,
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say(fileGroups[0].Name))
			Expect(session).Should(gbytes.Say(fileGroups[1].Name))
		})

		Context("when release version is provided", func() {
			BeforeEach(func() {
				releasesResponse := pivnet.ReleasesResponse{
					Releases: releases,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
					),
				)

				response := pivnet.FileGroupsResponse{
					FileGroups: fileGroups,
				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest(
							"GET",
							fmt.Sprintf("%s/products/%s/releases/%d/file_groups",
								apiPrefix,
								product.Slug,
								releases[0].ID,
							),
						),
						ghttp.RespondWithJSONEncoded(http.StatusOK, response),
					),
				)
			})

			It("displays file groups for release", func() {
				session := runMainWithArgs(
					"file-groups",
					"--product-slug", product.Slug,
					"--release-version", releases[0].Version,
				)

				Eventually(session, executableTimeout).Should(gexec.Exit(0))
				Expect(session).Should(gbytes.Say(fileGroups[0].Name))
				Expect(session).Should(gbytes.Say(fileGroups[1].Name))
			})
		})
	})

	Describe("file-group", func() {
		var (
			fileGroup pivnet.FileGroup
		)

		BeforeEach(func() {
			fileGroup = pivnet.FileGroup{
				ID:   1234,
				Name: "some-product-file",
			}

			response := fileGroup

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/file_groups/%d",
							apiPrefix,
							product.Slug,
							fileGroup.ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)
		})

		It("displays file group", func() {
			session := runMainWithArgs(
				"file-group",
				"--product-slug", product.Slug,
				"--file-group-id", strconv.Itoa(fileGroup.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say(fileGroup.Name))
		})
	})

	Describe("delete file-group", func() {
		var (
			fileGroup pivnet.FileGroup
		)

		BeforeEach(func() {
			fileGroup = pivnet.FileGroup{
				ID:   1234,
				Name: "some-file-group",
			}

			response := fileGroup

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"DELETE",
						fmt.Sprintf(
							"%s/products/%s/file_groups/%d",
							apiPrefix,
							product.Slug,
							fileGroup.ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, response),
				),
			)
		})

		It("deletes file group", func() {
			session := runMainWithArgs(
				"delete-file-group",
				"--product-slug", product.Slug,
				"--file-group-id", strconv.Itoa(fileGroup.ID),
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
		})
	})

	Describe("Release upgrade paths", func() {
		It("displays release upgrade paths for the provided product slug and release version", func() {
			releasesResponse := pivnet.ReleasesResponse{
				Releases: releases,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf("%s/products/%s/releases", apiPrefix, product.Slug),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, releasesResponse),
				),
			)

			releaseUpgradePaths := []pivnet.ReleaseUpgradePath{
				{
					Release: pivnet.UpgradePathRelease{
						ID:      1234,
						Version: "Some version",
					},
				},
				{
					Release: pivnet.UpgradePathRelease{
						ID:      2345,
						Version: "Another version",
					},
				},
			}

			releaseUpgradePathsResponse := pivnet.ReleaseUpgradePathsResponse{
				ReleaseUpgradePaths: releaseUpgradePaths,
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(
						"GET",
						fmt.Sprintf(
							"%s/products/%s/releases/%d/upgrade_paths",
							apiPrefix,
							product.Slug,
							releases[0].ID,
						),
					),
					ghttp.RespondWithJSONEncoded(http.StatusOK, releaseUpgradePathsResponse),
				),
			)

			session := runMainWithArgs(
				"release-upgrade-paths",
				"--product-slug", product.Slug,
				"--release-version", releases[0].Version,
			)

			Eventually(session, executableTimeout).Should(gexec.Exit(0))
			Expect(session).Should(gbytes.Say(releaseUpgradePaths[0].Release.Version))
		})
	})
})
