package service_test

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/kofuk/premises/internal/entity/web"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

//go:embed testdata/world.tar.zst
var worldArchive []byte

var _ = Describe("WorldService", func() {
	var (
		sut  *service.WorldService
		ctrl *gomock.Controller
	)

	BeforeEach(func() {
		httpmock.Activate(GinkgoTB())

		ctrl = gomock.NewController(GinkgoT())

		sut = service.NewWorldService("https://premises.local", "key", http.DefaultClient)

		httpmock.RegisterResponder(http.MethodGet, "https://premises.local/_/world/latest-id/world",
			httpmock.NewJsonResponderOrPanic(http.StatusOK, web.SuccessfulResponse[any]{
				Success: true,
				Data: web.GetLatestWorldIDResponse{
					WorldID: "latest-world.tar.zst",
				},
			}),
		)
	})

	It("should return correct latest resource ID", func() {
		resourceID, err := sut.GetLatestResourceID(GinkgoT().Context(), "world")
		Expect(err).To(BeNil())
		Expect(resourceID).To(Equal("latest-world.tar.zst"))
	})

	Describe("run inside a temporary environment", func() {
		var (
			tmpDir      string
			dataDir     string
			envProvider *env.MockEnvProvider
		)

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			dataDir = GinkgoT().TempDir()

			os.MkdirAll(filepath.Join(dataDir, "gamedata/world"), 0o755)

			envProvider = env.NewMockEnvProvider(ctrl)
			envProvider.EXPECT().GetTempDir().AnyTimes().Return(tmpDir)
			envProvider.EXPECT().GetDataPath("gamedata/world").AnyTimes().Return(filepath.Join(dataDir, "gamedata/world"))
			envProvider.EXPECT().GetDataPath("gamedata").AnyTimes().Return(filepath.Join(dataDir, "gamedata"))
		})

		It("should download and extract a world", func() {
			httpmock.RegisterResponder(http.MethodPost, "https://premises.local/_/world/download-url",
				httpmock.NewJsonResponderOrPanic(http.StatusCreated, web.SuccessfulResponse[any]{
					Success: true,
					Data: web.CreateWorldDownloadURLResponse{
						URL: "https://s3.premises.local/download",
					},
				}),
			)
			httpmock.RegisterResponder(http.MethodGet, "https://s3.premises.local/download",
				httpmock.NewBytesResponder(http.StatusOK, worldArchive),
			)

			err := sut.DownloadWorld(GinkgoT().Context(), "latest-world.tar.zst", envProvider)
			Expect(err).To(BeNil())

			Expect(filepath.Join(dataDir, "gamedata/world/level.dat")).To(BeARegularFile())
		})

		It("should archive and upload world", func() {
			httpmock.RegisterResponder(http.MethodPost, "https://premises.local/_/world/upload-url",
				httpmock.NewJsonResponderOrPanic(http.StatusCreated, web.SuccessfulResponse[any]{
					Success: true,
					Data: web.CreateWorldUploadURLResponse{
						URL:     "https://s3.premises.local/upload",
						WorldID: "uploaded-world.tar.zst",
					},
				}),
			)
			httpmock.RegisterResponder(http.MethodPut, "https://s3.premises.local/upload",
				httpmock.NewStringResponder(http.StatusOK, ""),
			)
			os.WriteFile(filepath.Join(dataDir, "gamedata/world/level.dat"), []byte("level"), 0o644)
			os.WriteFile(filepath.Join(dataDir, "gamedata/world/foo.txt"), []byte("foo"), 0o644)

			resourceID, err := sut.UploadWorld(GinkgoT().Context(), "foo", envProvider)
			Expect(err).To(BeNil())
			Expect(resourceID).To(Equal("uploaded-world.tar.zst"))
		})
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WorldService Suite")
}
