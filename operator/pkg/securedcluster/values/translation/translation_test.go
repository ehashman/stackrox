package translation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/images"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	testingUtils "github.com/stackrox/rox/operator/pkg/values/testing"
	"github.com/stackrox/rox/operator/pkg/values/translation"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TranslationTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestTranslation(t *testing.T) {
	suite.Run(t, new(TranslationTestSuite))
}

func (s *TranslationTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *TranslationTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s TranslationTestSuite) TestImageOverrides() {
	s.envIsolator.Setenv(images.Scanner.EnvVar(), "stackrox/scanner:1.0.0")
	s.envIsolator.Setenv(images.ScannerDB.EnvVar(), "stackrox/scanner-db:1.0.0")

	obj := platform.SecuredCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      "secured-cluster",
		},
	}
	u, err := toUnstructured(obj)
	s.Require().NoError(err)

	translator := Translator{client: newFakeClientWithInitBundle(s.T())}

	vals, err := translator.Translate(context.Background(), u)
	s.Require().NoError(err)

	scannerImage, err := vals.PathValue("image.scanner.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner:1.0.0", scannerImage)

	scannerDbImage, err := vals.PathValue("image.scannerDb.fullRef")
	s.Require().NoError(err)
	s.Equal("stackrox/scanner-db:1.0.0", scannerDbImage)
}

func TestReadBaseValues(t *testing.T) {
	_, err := chartutil.ReadValues(baseValuesYAML)
	assert.NoError(t, err)
}

func TestTranslateShouldCreateConfigFingerprint(t *testing.T) {
	sc := platform.SecuredCluster{
		Spec: platform.SecuredClusterSpec{
			ClusterName: "my-cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "stackrox",
			Name:      "my-secured-cluster",
		},
	}

	u, err := toUnstructured(sc)
	require.NoError(t, err)

	translator := Translator{client: newFakeClientWithInitBundle(t)}
	vals, err := translator.Translate(context.Background(), u)
	require.NoError(t, err)

	testingUtils.AssertPathValueMatches(t, vals, regexp.MustCompile("[0-9a-f]{32}"), "meta.configFingerprintOverride")
}

func (s TranslationTestSuite) TestTranslate() {
	s.envIsolator.Setenv(features.LocalImageScanning.EnvVar(), "true")
	t := s.T()

	type args struct {
		client ctrlClient.Client
		sc     platform.SecuredCluster
	}

	scannerComponentPolicy := platform.LocalScannerComponentAutoSense
	scannerAutoScalingPolicy := platform.ScannerAutoScalingEnabled

	scannerReplicas := int32(7)
	scannerMinReplicas := int32(6)
	scannerMaxReplicas := int32(8)

	// TODO(ROX-7647): Add sensor, collector and compliance tests
	tests := map[string]struct {
		args args
		want chartutil.Values
	}{
		"minimal spec": {
			args: args{
				client: newFakeClientWithInitBundle(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
			},
		},
		"local scanner autosense suppression": {
			args: args{
				client: newFakeClientWithInitBundleAndCentral(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": true,
				},
			},
		},
		"local scanner autosense no suppression": {
			args: args{
				client: newFakeClientWithInitBundle(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
					Spec: platform.SecuredClusterSpec{
						ClusterName: "test-cluster",
					},
				},
			},
			want: chartutil.Values{
				"clusterName":   "test-cluster",
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": true,
					},
					"listenOnCreates": true,
					"listenOnUpdates": true,
				},
				"scanner": map[string]interface{}{
					"disable": false,
				},
			},
		},
		"complete spec": {
			args: args{
				client: newFakeClientWithInitBundle(t),
				sc: platform.SecuredCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-secured-cluster",
						Namespace: "stackrox",
					},
					Spec: platform.SecuredClusterSpec{
						ClusterName:     "test-cluster",
						CentralEndpoint: "central.test:443",
						Sensor: &platform.SensorComponentSpec{
							DeploymentSpec: platform.DeploymentSpec{
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
							},
						},
						AdmissionControl: &platform.AdmissionControlComponentSpec{
							ListenOnCreates:      pointer.BoolPtr(true),
							ListenOnUpdates:      pointer.BoolPtr(false),
							ListenOnEvents:       pointer.BoolPtr(true),
							ContactImageScanners: platform.ScanIfMissing.Pointer(),
							TimeoutSeconds:       pointer.Int32Ptr(4),
							Bypass:               platform.BypassBreakGlassAnnotation.Pointer(),
							DeploymentSpec: platform.DeploymentSpec{
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1502m"),
										v1.ResourceMemory: resource.MustParse("1002Mi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1501m"),
										v1.ResourceMemory: resource.MustParse("1001Mi"),
									},
								},
								NodeSelector: map[string]string{
									"admission-ctrl-node-selector1": "admission-ctrl-node-selector-val1",
									"admission-ctrl-node-selector2": "admission-ctrl-node-selector-val2",
								},
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
							},
						},
						ClusterLabels: map[string]string{
							"my-label1": "value1",
							"my-label2": "value2",
						},
						ImagePullSecrets: []platform.LocalSecretReference{
							{Name: "image-pull-secrets-secret1"},
							{Name: "image-pull-secrets-secret2"},
						},
						TLS: &platform.TLSConfig{
							AdditionalCAs: []platform.AdditionalCA{
								{Name: "ca1-name", Content: "ca1-content"},
								{Name: "ca2-name", Content: "ca2-content"},
							},
						},
						AuditLogs: &platform.AuditLogsSpec{
							Collection: platform.AuditLogsCollectionEnabled.Pointer(),
						},
						PerNode: &platform.PerNodeSpec{
							Collector: &platform.CollectorContainerSpec{
								ImageFlavor: platform.ImageFlavorRegular.Pointer(),
								Collection:  platform.CollectionEBPF.Pointer(),
							},
							TaintToleration: platform.TaintTolerate.Pointer(),
							Compliance: &platform.ContainerSpec{
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1504m"),
										v1.ResourceMemory: resource.MustParse("1004Mi"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("1503m"),
										v1.ResourceMemory: resource.MustParse("1003Mi"),
									},
								},
							},
						},
						Scanner: &platform.LocalScannerComponentSpec{
							ScannerComponent: &scannerComponentPolicy,
							Analyzer: &platform.ScannerAnalyzerComponent{
								Scaling: &platform.ScannerAnalyzerScaling{
									AutoScaling: &scannerAutoScalingPolicy,
									Replicas:    &scannerReplicas,
									MinReplicas: &scannerMinReplicas,
									MaxReplicas: &scannerMaxReplicas,
								},
								DeploymentSpec: platform.DeploymentSpec{
									NodeSelector: map[string]string{
										"scanner-node-selector-label1": "scanner-node-selector-value1",
										"scanner-node-selector-label2": "scanner-node-selector-value2",
									},
									Tolerations: []*v1.Toleration{
										{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
										{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
									},
									Resources: &v1.ResourceRequirements{
										Limits: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("50"),
											v1.ResourceMemory: resource.MustParse("60"),
										},
										Requests: v1.ResourceList{
											v1.ResourceCPU:    resource.MustParse("70"),
											v1.ResourceMemory: resource.MustParse("80"),
										},
									},
								},
							},
							DB: &platform.DeploymentSpec{
								NodeSelector: map[string]string{
									"scanner-db-node-selector-label1": "scanner-db-node-selector-value1",
									"scanner-db-node-selector-label2": "scanner-db-node-selector-value2",
								},
								Tolerations: []*v1.Toleration{
									{Key: "node.stackrox.io", Value: "false", Operator: v1.TolerationOpEqual},
									{Key: "node-role.kubernetes.io/infra", Value: "", Operator: v1.TolerationOpExists},
								},
								Resources: &v1.ResourceRequirements{
									Limits: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("90"),
										v1.ResourceMemory: resource.MustParse("100"),
									},
									Requests: v1.ResourceList{
										v1.ResourceCPU:    resource.MustParse("110"),
										v1.ResourceMemory: resource.MustParse("120"),
									},
								},
							},
						},
						Customize: &platform.CustomizeSpec{
							Labels: map[string]string{
								"customize-label1": "customize-label1-value",
								"customize-label2": "customize-label2-value",
							},
							Annotations: map[string]string{
								"customize-annotation1": "customize-annotation1-value",
								"customize-annotation2": "customize-annotation2-value",
							},
							EnvVars: []v1.EnvVar{
								{
									Name:  "customize-env-var1",
									Value: "customize-env-var1-value",
								},
								{
									Name:  "customize-env-var2",
									Value: "customize-env-var2-value",
								},
							},
						},
						Misc: &platform.MiscSpec{
							CreateSCCs: pointer.BoolPtr(true),
						},
					},
				},
			},
			want: chartutil.Values{
				"clusterName":     "test-cluster",
				"centralEndpoint": "central.test:443",
				"clusterLabels": map[string]interface{}{
					"my-label1": "value1",
					"my-label2": "value2",
				},
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []string{
						"image-pull-secrets-secret1",
						"image-pull-secrets-secret2",
					},
				},
				"additionalCAs": map[string]interface{}{
					"ca1-name": "ca1-content",
					"ca2-name": "ca2-content",
				},
				"sensor": map[string]interface{}{
					"tolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
				},
				"admissionControl": map[string]interface{}{
					"dynamic": map[string]interface{}{
						"enforceOnCreates": true,
						"enforceOnUpdates": false,
						"scanInline":       true,
						"disableBypass":    false,
						"timeout":          4,
					},
					"listenOnCreates": true,
					"listenOnUpdates": false,
					"listenOnEvents":  true,
					"nodeSelector": map[string]interface{}{
						"admission-ctrl-node-selector1": "admission-ctrl-node-selector-val1",
						"admission-ctrl-node-selector2": "admission-ctrl-node-selector-val2",
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "1502m",
							"memory": "1002Mi",
						}, "requests": map[string]interface{}{
							"cpu":    "1501m",
							"memory": "1001Mi",
						},
					},
					"tolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
				},
				"auditLogs": map[string]interface{}{
					"disableCollection": false,
				},
				"scanner": map[string]interface{}{
					"disable":  false,
					"replicas": int32(7),
					"autoscaling": map[string]interface{}{
						"disable":     false,
						"minReplicas": int32(6),
						"maxReplicas": int32(8),
					},
					"nodeSelector": map[string]string{
						"scanner-node-selector-label1": "scanner-node-selector-value1",
						"scanner-node-selector-label2": "scanner-node-selector-value2",
					},
					"tolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
					"dbNodeSelector": map[string]string{
						"scanner-db-node-selector-label1": "scanner-db-node-selector-value1",
						"scanner-db-node-selector-label2": "scanner-db-node-selector-value2",
					},
					"dbTolerations": []map[string]interface{}{
						{
							"key":      "node.stackrox.io",
							"operator": "Equal",
							"value":    "false",
						}, {
							"key":      "node-role.kubernetes.io/infra",
							"operator": "Exists",
						},
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "50",
							"memory": "60",
						},
						"requests": map[string]interface{}{
							"cpu":    "70",
							"memory": "80",
						},
					},
					"dbResources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "90",
							"memory": "100",
						},
						"requests": map[string]interface{}{
							"cpu":    "110",
							"memory": "120",
						},
					},
				},
				"ca":            map[string]string{"cert": "ca central content"},
				"createSecrets": false,
				"customize": map[string]interface{}{
					"annotations": map[string]string{
						"customize-annotation1": "customize-annotation1-value",
						"customize-annotation2": "customize-annotation2-value",
					},
					"labels": map[string]string{
						"customize-label1": "customize-label1-value",
						"customize-label2": "customize-label2-value",
					},
					"envVars": map[string]interface{}{
						"customize-env-var1": map[string]interface{}{
							"value": "customize-env-var1-value",
						},
						"customize-env-var2": map[string]interface{}{
							"value": "customize-env-var2-value",
						},
					},
				},
				"collector": map[string]interface{}{
					"collectionMethod":        "EBPF",
					"disableTaintTolerations": false,
					"slimMode":                false,
					"complianceResources": map[string]interface{}{
						"limits": map[string]interface{}{
							"cpu":    "1504m",
							"memory": "1004Mi",
						}, "requests": map[string]interface{}{
							"cpu":    "1503m",
							"memory": "1003Mi",
						},
					},
				},
				"system": map[string]interface{}{
					"createSCCs": true,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wantAsValues, err := translation.ToHelmValues(tt.want)
			require.NoError(t, err, "error in test specification: cannot translate `want` specification to Helm values")

			translator := Translator{client: tt.args.client}
			got, err := translator.translate(context.Background(), tt.args.sc)
			require.NoError(t, err)

			// Remove config fingerprint as it changes as the test case changes
			_, err = got.PathValue("meta.configFingerprintOverride")
			require.NoError(t, err)
			delete(got["meta"].(map[string]interface{}), "configFingerprintOverride")
			if len(got["meta"].(map[string]interface{})) == 0 {
				delete(got, "meta")
			}

			// TODO(ROX-8466): Remove if feature flag gets enabled
			if buildinfo.ReleaseBuild {
				delete(wantAsValues, "scanner")
			}

			assert.Equal(t, wantAsValues, got)
		})
	}
}

func toUnstructured(sc platform.SecuredCluster) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&sc)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

func newFakeClientWithInitBundle(t *testing.T) ctrlClient.Client {
	return testutils.NewFakeClientBuilder(t,
		createSecret(sensorTLSSecretName),
		createSecret(collectorTLSSecretName),
		createSecret(admissionControlTLSSecretName)).Build()
}

func newFakeClientWithInitBundleAndCentral(t *testing.T) ctrlClient.Client {
	return testutils.NewFakeClientBuilder(t,
		createSecret(sensorTLSSecretName),
		createSecret(collectorTLSSecretName),
		createSecret(admissionControlTLSSecretName),
		&platform.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "a-central",
				Namespace: "stackrox",
			},
		}).Build()
}

func createSecret(name string) *v1.Secret {
	serviceName := strings.TrimSuffix(name, "-tls")

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "stackrox",
		},
		Data: map[string][]byte{
			"ca.pem":                                []byte(`ca central content`),
			fmt.Sprintf("%s-key.pem", serviceName):  []byte(`key content`),
			fmt.Sprintf("%s-cert.pem", serviceName): []byte(`cert content`),
		},
	}
}
