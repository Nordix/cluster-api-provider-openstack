//go:build e2e

/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	capi_framework "sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"

	shared "sigs.k8s.io/cluster-api-provider-openstack/test/e2e/shared"
)

var (
	capoRelease012 string
	capoRelease013 string
	capoRelease014 string
	capiRelease110 string
	capiRelease111 string
	capiRelease112 string
)

// NOTE: clusterctl v1.10 cannot handle RuntimeExtensionProvider in the local
// filesystem repository created by the CAPI v1.13 test framework. And clusterctl
// v1.12 refuses to operate against v1beta1 management clusters. Therefore, we
// need to install ORC outside of the normal clusterctl init/upgrade by using
// hooks instead.
//
// To also exercise the ORC upgrade path, we install ORC v1.0.2 (the version the
// old CAPO releases were tested against) in PreInit and the current ORC version
// in PreUpgrade. Both are installed by directly applying the upstream install
// manifest, since clusterctl v1.10 cannot install ORC as a RuntimeExtensionProvider
// from the local filesystem repository.
//
// CAPO v0.13 was compiled against CAPI v1.11, which promoted the IPAM API from
// exp/ipam (v1alpha1) to api/ipam (v1beta2). This means CAPO v0.13 requires
// ipam.cluster.x-k8s.io/v1beta2 CRDs at startup, which are not available in
// CAPI v1.10 (the last v1beta1 release). Therefore the v0.13 upgrade test uses
// CAPI v1.11 (v1beta2 contract) as the starting point. This makes the test a
// v1beta2->v1beta2 management cluster upgrade rather than v1beta1->v1beta2, but
// it still validates the CAPO v0.13->current upgrade path.
//
// The dedicated CAPO v0.14 + ORC upgrade test below uses clusterctl v1.12 with
// CAPI v1.12 (creating a v1beta2 management cluster) so that clusterctl can
// properly install and upgrade ORC as a RuntimeExtensionProvider via the normal
// init/upgrade flow.

// orcInitVersion is the ORC version installed alongside the old CAPO release in
// PreInit. v1.0.2 is the version the v0.12/v0.13/v0.14 CAPO releases were tested
// against (see the previous InitWithRuntimeExtensionProviders entry).
const orcInitVersion = "v1.0.2"

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.12=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease012, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.12")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease012 = "v" + capoRelease012
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease110, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.10")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease110 = "v" + capiRelease110
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                       e2eCtx.E2EConfig,
			ClusterctlConfigPath:            e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:           e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                  e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                     false,
			InitWithBinary:                  "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease110 + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:       "v1beta1",
			InitWithInfrastructureProviders: []string{"openstack:" + capoRelease012},
			InitWithCoreProvider:            "cluster-api:" + capiRelease110,
			InitWithBootstrapProviders:      []string{"kubeadm:" + capiRelease110},
			InitWithControlPlaneProviders:   []string{"kubeadm:" + capiRelease110},
			// Pass an explicit (non-nil) empty slice so the upstream framework does
			// NOT fall back to the latest v1beta1 RuntimeExtensionProvider from the
			// e2e config (openstack-resource-controller:v1.0.2) and try to install
			// it via `clusterctl init`. clusterctl v1.10 cannot handle a
			// RuntimeExtensionProvider in the local filesystem repository, so we
			// instead install ORC ourselves in the PreInit hook below.
			InitWithRuntimeExtensionProviders: []string{},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			UseKindForManagementCluster:       true,
			// Install ORC v1.0.2 before clusterctl init
			PreInit: func(managementClusterProxy capi_framework.ClusterProxy) {
				installORC(context.Background(), managementClusterProxy, orcInitVersion)
			},
			// Upgrade ORC to the current version before clusterctl upgrade
			PreUpgrade: func(managementClusterProxy capi_framework.ClusterProxy) {
				installLatestORC(context.Background(), managementClusterProxy, e2eCtx.E2EConfig)
			},
		}
	})
})

// CAPO v0.13 requires ipam.cluster.x-k8s.io/v1beta2 (introduced in CAPI v1.11)
// and therefore cannot run against CAPI v1.10.
var _ = Describe("When testing clusterctl upgrades for CAPO (v0.13=>current) and ORC (v1.0.2=>current) with CAPI v1beta2 [clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease013, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.13")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease013 = "v" + capoRelease013
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		// CAPI v1.11 is the first release with ipam.cluster.x-k8s.io/v1beta2 CRDs,
		// which CAPO v0.13 requires.
		capiRelease111, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.11")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease111 = "v" + capiRelease111
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                       e2eCtx.E2EConfig,
			ClusterctlConfigPath:            e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:           e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                  e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                     false,
			InitWithBinary:                  "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease111 + "/clusterctl-{OS}-{ARCH}",
			InitWithInfrastructureProviders: []string{"openstack:" + capoRelease013},
			InitWithCoreProvider:            "cluster-api:" + capiRelease111,
			InitWithBootstrapProviders:      []string{"kubeadm:" + capiRelease111},
			InitWithControlPlaneProviders:   []string{"kubeadm:" + capiRelease111},
			// Explicit empty slice: we install ORC manually via PreInit/PreUpgrade
			// hooks rather than via clusterctl, because it is not confirmed whether
			// clusterctl v1.11 handles RuntimeExtensionProviders in the local
			// filesystem repository format created by the CAPI v1.13 test framework.
			InitWithRuntimeExtensionProviders: []string{},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			UseKindForManagementCluster:       true,
			PreInit: func(managementClusterProxy capi_framework.ClusterProxy) {
				installORC(context.Background(), managementClusterProxy, orcInitVersion)
			},
			PreUpgrade: func(managementClusterProxy capi_framework.ClusterProxy) {
				installLatestORC(context.Background(), managementClusterProxy, e2eCtx.E2EConfig)
			},
		}
	})
})

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.14=>current) and ORC (v1.0.2=>current)[clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease014, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.14")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease014 = "v" + capoRelease014
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease110, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.10")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease110 = "v" + capiRelease110
	})

	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                       e2eCtx.E2EConfig,
			ClusterctlConfigPath:            e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:           e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                  e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                     false,
			InitWithBinary:                  "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease110 + "/clusterctl-{OS}-{ARCH}",
			InitWithProvidersContract:       "v1beta1",
			InitWithInfrastructureProviders: []string{"openstack:" + capoRelease014},
			InitWithCoreProvider:            "cluster-api:" + capiRelease110,
			InitWithBootstrapProviders:      []string{"kubeadm:" + capiRelease110},
			InitWithControlPlaneProviders:   []string{"kubeadm:" + capiRelease110},
			// See the v0.12 test above: explicit empty slice prevents the framework
			// from auto-installing ORC v1.0.2 via clusterctl v1.10 (which can't
			// handle RuntimeExtensionProviders in the local filesystem repository).
			InitWithRuntimeExtensionProviders: []string{},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			UseKindForManagementCluster:       true,
			PreInit: func(managementClusterProxy capi_framework.ClusterProxy) {
				installORC(context.Background(), managementClusterProxy, orcInitVersion)
			},
			PreUpgrade: func(managementClusterProxy capi_framework.ClusterProxy) {
				installLatestORC(context.Background(), managementClusterProxy, e2eCtx.E2EConfig)
			},
		}
	})
})

var _ = Describe("When testing clusterctl upgrades for CAPO (v0.14=>current) and ORC (v1.0.2=>current) with CAPI v1beta2 [clusterctl-upgrade]", func() {
	BeforeEach(func(ctx context.Context) {
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capoRelease014, err = clusterctl.ResolveRelease(ctx, "go://github.com/kubernetes-sigs/cluster-api-provider-openstack@v0.14")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPO")
		capoRelease014 = "v" + capoRelease014
		// Note: This gives the version without the 'v' prefix, so we need to add it below.
		capiRelease112, err = capi_e2e.GetStableReleaseOfMinor(ctx, "1.12")
		Expect(err).ToNot(HaveOccurred(), "failed to get stable release of CAPI")
		capiRelease112 = "v" + capiRelease112
	})

	// Here we use clusterctl v1.12 with CAPI v1.12 to create a v1beta2 management
	// cluster. This allows us to also install ORC as RuntimeExtensionProvider and test
	// upgrading both CAPO (v0.14=>current) and ORC (v1.0.2=>current).
	capi_e2e.ClusterctlUpgradeSpec(context.TODO(), func() capi_e2e.ClusterctlUpgradeSpecInput {
		return capi_e2e.ClusterctlUpgradeSpecInput{
			E2EConfig:                         e2eCtx.E2EConfig,
			ClusterctlConfigPath:              e2eCtx.Environment.ClusterctlConfigPath,
			BootstrapClusterProxy:             e2eCtx.Environment.BootstrapClusterProxy,
			ArtifactFolder:                    e2eCtx.Settings.ArtifactFolder,
			SkipCleanup:                       false,
			InitWithBinary:                    "https://github.com/kubernetes-sigs/cluster-api/releases/download/" + capiRelease112 + "/clusterctl-{OS}-{ARCH}",
			InitWithInfrastructureProviders:   []string{"openstack:" + capoRelease014},
			InitWithCoreProvider:              "cluster-api:" + capiRelease112,
			InitWithBootstrapProviders:        []string{"kubeadm:" + capiRelease112},
			InitWithControlPlaneProviders:     []string{"kubeadm:" + capiRelease112},
			MgmtFlavor:                        shared.FlavorDefault,
			WorkloadFlavor:                    shared.FlavorCapiV1Beta1,
			InitWithKubernetesVersion:         e2eCtx.E2EConfig.MustGetVariable(shared.KubernetesKindVersion),
			InitWithRuntimeExtensionProviders: []string{"openstack-resource-controller:v1.0.2"},
			UseKindForManagementCluster:       true,
		}
	})
})

// installLatestORC downloads and applies the install manifest for the current/latest version of
// the OpenStack Resource Controller (ORC) to the management cluster.
//
// The ORC version is derived from the e2e config (the latest version with contract v1beta2), so no
// version hardcoding is required here — updating the provider entry in the e2e config is sufficient.
//
// After this function returns, the upgrade spec's own post-upgrade controller wait logic will handle
// waiting for the ORC deployment to become available. This works because ORC resources carry the
// "cluster.x-k8s.io/provider" label (added by the ORC kustomization), which causes them to be
// discovered by framework.GetControllerDeployments.
func installLatestORC(ctx context.Context, proxy capi_framework.ClusterProxy, e2eConfig *clusterctl.E2EConfig) {
	// GetProviderLatestVersionsByContract returns strings in the format "provider-name:version",
	// e.g. "openstack-resource-controller:v2.5.0".
	orcVersionStrings := e2eConfig.GetProviderLatestVersionsByContract("v1beta2", "openstack-resource-controller")
	Expect(orcVersionStrings).ToNot(BeEmpty(),
		"No ORC version with v1beta2 contract found in e2e config; cannot install ORC for upgrade")

	parts := strings.SplitN(orcVersionStrings[0], ":", 2)
	Expect(parts).To(HaveLen(2),
		"Unexpected ORC provider version string format (expected 'name:version'): %q", orcVersionStrings[0])
	orcVersion := parts[1]

	installORC(ctx, proxy, orcVersion)
}

// installORC downloads and applies the upstream install manifest for the given version of the
// OpenStack Resource Controller (ORC) to the management cluster.
//
// This is used to install ORC outside of the normal clusterctl init/upgrade flow, which is needed
// for the upgrade tests that use clusterctl v1.10: clusterctl v1.10 cannot handle
// RuntimeExtensionProvider in the local filesystem repository format used by the CAPI test
// framework. CAPO has required ORC CRDs to be present at controller startup since v0.12, so the
// initial ORC install must happen BEFORE clusterctl init runs the old CAPO controller (i.e. from
// the PreInit hook).
func installORC(ctx context.Context, proxy capi_framework.ClusterProxy, orcVersion string) {
	By(fmt.Sprintf("Installing ORC %s on the management cluster", orcVersion))

	orcInstallURL := fmt.Sprintf(
		"https://github.com/k-orc/openstack-resource-controller/releases/download/%s/install.yaml",
		orcVersion,
	)
	By(fmt.Sprintf("Downloading ORC %s install manifest from %s", orcVersion, orcInstallURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orcInstallURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred(), "Failed to create HTTP request for ORC install manifest")

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // closed below via defer
	Expect(err).ToNot(HaveOccurred(),
		"Failed to download ORC %s install manifest from %s", orcVersion, orcInstallURL)
	defer resp.Body.Close()

	Expect(resp.StatusCode).To(Equal(http.StatusOK),
		"Unexpected HTTP status %d when downloading ORC install manifest from %s", resp.StatusCode, orcInstallURL)

	orcManifest, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred(), "Failed to read ORC install manifest response body")

	By(fmt.Sprintf("Applying ORC %s install manifest to the management cluster", orcVersion))
	Expect(proxy.CreateOrUpdate(ctx, orcManifest)).To(Succeed(),
		"Failed to apply ORC %s install manifest to management cluster", orcVersion)
}
