package tests

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	"kubevirt.io/containerized-data-importer/tests/framework"
	"kubevirt.io/containerized-data-importer/tests/utils"
)

var _ = Describe("Transport Tests", func() {

	const (
		secretPrefix        = "transport-e2e-sec"
		targetFile          = "tinyCore.iso"
		targetQCOWFile      = "tinyCore.qcow2"
		targetQCOWImage     = "tinycoreqcow2"
		targetRawImage      = "tinycoreqcow2"
		targetArchivedImage = "tinycoreisotar"
		sizeCheckPod        = "size-checker"
	)

	var (
		ns  string
		f   = framework.NewFrameworkOrDie("transport", framework.Config{SkipNamespaceCreation: false})
		c   = f.K8sClient
		sec *v1.Secret
	)

	BeforeEach(func() {
		ns = f.Namespace.Name
		By(fmt.Sprintf("Waiting for all \"%s/%s\" deployment replicas to be Ready", f.CdiInstallNs, utils.FileHostName))
		utils.WaitForDeploymentReplicasReadyOrDie(c, f.CdiInstallNs, utils.FileHostName)
	})

	// it() is the body of the test and is executed once per Entry() by DescribeTable()
	// closes over c and ns
	it := func(ep, file, accessKey, secretKey, source, certConfigMap string, insecureRegistry, shouldSucceed bool) {

		var (
			err error // prevent shadowing
		)

		pvcAnn := map[string]string{
			controller.AnnEndpoint: ep + "/" + file,
			controller.AnnSecret:   "",
			controller.AnnSource:   source,
		}

		if accessKey != "" || secretKey != "" {
			By(fmt.Sprintf("Creating secret for endpoint %s", ep))
			if accessKey == "" {
				accessKey = utils.AccessKeyValue
			}
			if secretKey == "" {
				secretKey = utils.SecretKeyValue
			}
			stringData := make(map[string]string)
			stringData[common.KeyAccess] = accessKey
			stringData[common.KeySecret] = secretKey

			sec, err = utils.CreateSecretFromDefinition(c, utils.NewSecretDefinition(nil, stringData, nil, ns, secretPrefix))
			Expect(err).NotTo(HaveOccurred(), "Error creating test secret")
			pvcAnn[controller.AnnSecret] = sec.Name
		}

		if certConfigMap != "" {
			n, err := utils.CopyConfigMap(c, f.CdiInstallNs, certConfigMap, ns, "")
			Expect(err).To(BeNil())
			pvcAnn[controller.AnnCertConfigMap] = n
		}

		if insecureRegistry {
			err = utils.SetInsecureRegistry(c, f.CdiInstallNs, ep)
			Expect(err).To(BeNil())
			defer utils.ClearInsecureRegistry(c, f.CdiInstallNs)
		}

		By(fmt.Sprintf("Creating PVC with endpoint annotation %q", pvcAnn[controller.AnnEndpoint]))
		pvc, err := utils.CreatePVCFromDefinition(c, ns, utils.NewPVCDefinition("transport-e2e", "20Mi", pvcAnn, nil))
		Expect(err).NotTo(HaveOccurred(), "Error creating PVC")

		if shouldSucceed {
			By("Verify PVC status annotation says succeeded")
			found, err := utils.WaitPVCPodStatusSucceeded(f.K8sClient, pvc)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			By("Verifying PVC is not empty")
			Expect(framework.VerifyPVCIsEmpty(f, pvc)).To(BeFalse(), fmt.Sprintf("Found 0 imported files on PVC %q", pvc.Namespace+"/"+pvc.Name))

			pod, err := utils.CreateExecutorPodWithPVC(c, sizeCheckPod, ns, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(utils.WaitTimeoutForPodReady(c, sizeCheckPod, ns, 90*time.Second)).To(Succeed())

			switch pvcAnn[controller.AnnSource] {
			case controller.SourceHTTP, controller.SourceRegistry:
				if file != targetFile {
					command := `expSize=20971520; haveSize=$(wc -c < /pvc/disk.img); (( $expSize == $haveSize )); echo $?`
					exitCode, _ := f.ExecShellInPod(pod.Name, ns, command)
					// A 0 exitCode should indicate that $expSize == $haveSize
					Expect(strconv.Atoi(exitCode)).To(BeZero())
				}
			}
		} else {
			By("Verify PVC status annotation says failed")
			found, err := utils.WaitPVCPodStatusFailed(f.K8sClient, pvc)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())

			By("Verifying PVC is empty")
			Expect(framework.VerifyPVCIsEmpty(f, pvc)).To(BeTrue(), fmt.Sprintf("Found 0 imported files on PVC %q", pvc.Namespace+"/"+pvc.Name))
		}
	}

	httpNoAuthEp := fmt.Sprintf("http://%s:%d", utils.FileHostName+"."+f.CdiInstallNs, utils.HTTPNoAuthPort)
	httpsNoAuthEp := fmt.Sprintf("https://%s:%d", utils.FileHostName+"."+f.CdiInstallNs, utils.HTTPSNoAuthPort)
	httpAuthEp := fmt.Sprintf("http://%s:%d", utils.FileHostName+"."+f.CdiInstallNs, utils.HTTPAuthPort)
	registryNoAuthEp := fmt.Sprintf("docker://%s", utils.RegistryHostName+"."+f.CdiInstallNs)
	altRegistryNoAuthEp := fmt.Sprintf("docker://%s.%s:%d", utils.RegistryHostName, f.CdiInstallNs, 5000)
	DescribeTable("Transport Test Table", it,
		Entry("should connect to http endpoint without credentials", httpNoAuthEp, targetFile, "", "", controller.SourceHTTP, "", false, true),
		Entry("should connect to http endpoint with credentials", httpAuthEp, targetFile, utils.AccessKeyValue, utils.SecretKeyValue, controller.SourceHTTP, "", false, true),
		Entry("should not connect to http endpoint with invalid credentials", httpAuthEp, targetFile, "invalid", "invalid", controller.SourceHTTP, "", false, false),
		Entry("should connect to QCOW http endpoint without credentials", httpNoAuthEp, targetQCOWFile, "", "", controller.SourceHTTP, "", false, true),
		Entry("should connect to QCOW http endpoint with credentials", httpAuthEp, targetQCOWFile, utils.AccessKeyValue, utils.SecretKeyValue, controller.SourceHTTP, "", false, true),
		Entry("should succeed to import from registry when image contains valid qcow file", registryNoAuthEp, targetQCOWImage, "", "", controller.SourceRegistry, "cdi-docker-registry-host-certs", false, true),
		Entry("should succeed to import from registry when image contains valid qcow file", registryNoAuthEp, targetQCOWImage, "", "", controller.SourceRegistry, "", true, true),
		Entry("should succeed to import from registry when image contains valid qcow file", altRegistryNoAuthEp, targetQCOWImage, "", "", controller.SourceRegistry, "", true, true),
		Entry("should fail no certs", registryNoAuthEp, targetQCOWImage, "", "", controller.SourceRegistry, "", false, false),
		Entry("should fail bad certs", registryNoAuthEp, targetQCOWImage, "", "", controller.SourceRegistry, "cdi-file-host-certs", false, false),
		Entry("should succeed to import from registry when image contains valid raw file", registryNoAuthEp, targetRawImage, "", "", controller.SourceRegistry, "cdi-docker-registry-host-certs", false, true),
		Entry("should succeed to import from registry when image contains valid archived raw file", registryNoAuthEp, targetArchivedImage, "", "", controller.SourceRegistry, "cdi-docker-registry-host-certs", false, true),
		Entry("should not connect to https endpoint without cert", httpsNoAuthEp, targetFile, "", "", controller.SourceHTTP, "", false, false),
		Entry("should connect to https endpoint with cert", httpsNoAuthEp, targetFile, "", "", controller.SourceHTTP, "cdi-file-host-certs", false, true),
		Entry("should not connect to https endpoint with bad cert", httpsNoAuthEp, targetFile, "", "", controller.SourceHTTP, "cdi-docker-registry-host-certs", false, false),
	)
})
