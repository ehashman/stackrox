//+kubebuilder:object:generate=true

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MiscSpec defines miscellaneous settings for custom resources.
type MiscSpec struct {
	// Set this to true to have the operator create SecurityContextConstraints (SCCs) for the operands. This
	// isn't usually needed, and may interfere with other workloads.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Create SecurityContextConstraints for Operand"
	CreateSCCs *bool `json:"createSCCs,omitempty"`
}

// CustomizeSpec defines customizations to apply.
type CustomizeSpec struct {
	// Custom labels to set on all managed objects.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Labels map[string]string `json:"labels,omitempty"`
	// Custom annotations to set on all managed objects.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Annotations map[string]string `json:"annotations,omitempty"`

	// Custom environment variables to set on managed pods' containers.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Environment Variables"
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`
}

// DeploymentSpec defines settings that affect a deployment.
type DeploymentSpec struct {
	// Allows overriding the default resource settings for this component. Please consult the documentation
	// for an overview of default resource requirements and a sizing guide.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"},order=100
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// If you want this component to only run on specific nodes, you can configure a node selector here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Selector",order=101
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If you want this component to only run on specific nodes, you can configure tolerations of tainted nodes.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:tolerations"},order=102
	Tolerations []*corev1.Toleration `json:"tolerations,omitempty"`
}

// StackRoxCondition defines a condition for a StackRox custom resource.
type StackRoxCondition struct {
	Type    ConditionType   `json:"type"`
	Status  ConditionStatus `json:"status"`
	Reason  ConditionReason `json:"reason,omitempty"`
	Message string          `json:"message,omitempty"`

	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// ConditionType is a type of values of condition type.
type ConditionType string

// ConditionStatus is a type of values of condition status.
type ConditionStatus string

// ConditionReason is a type of values of condition reason.
type ConditionReason string

// These are the allowed values for StackRoxCondition fields.
const (
	ConditionInitialized    ConditionType = "Initialized"
	ConditionDeployed       ConditionType = "Deployed"
	ConditionReleaseFailed  ConditionType = "ReleaseFailed"
	ConditionIrreconcilable ConditionType = "Irreconcilable"

	StatusTrue    ConditionStatus = "True"
	StatusFalse   ConditionStatus = "False"
	StatusUnknown ConditionStatus = "Unknown"

	ReasonInstallSuccessful   ConditionReason = "InstallSuccessful"
	ReasonUpgradeSuccessful   ConditionReason = "UpgradeSuccessful"
	ReasonUninstallSuccessful ConditionReason = "UninstallSuccessful"
	ReasonInstallError        ConditionReason = "InstallError"
	ReasonUpgradeError        ConditionReason = "UpgradeError"
	ReasonReconcileError      ConditionReason = "ReconcileError"
	ReasonUninstallError      ConditionReason = "UninstallError"
)

// StackRoxRelease describes the Helm "release" that was most recently applied.
type StackRoxRelease struct {
	Version string `json:"version,omitempty"`
}

// AdditionalCA defines a certificate for an additional Certificate Authority.
type AdditionalCA struct {
	// Must be a valid file basename
	Name string `json:"name"`
	// PEM format
	Content string `json:"content"`
}

// TLSConfig defines common TLS-related settings for all components.
type TLSConfig struct {
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Additional CAs"
	AdditionalCAs []AdditionalCA `json:"additionalCAs,omitempty"`
}

// LocalSecretReference is a reference to a secret within the same namespace.
type LocalSecretReference struct {
	// The name of the referenced secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	Name string `json:"name"`
}

// ScannerAnalyzerComponent describes the analyzer component
type ScannerAnalyzerComponent struct {
	// Controls the number of analyzer replicas and autoscaling.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Scaling *ScannerAnalyzerScaling `json:"scaling,omitempty"`

	DeploymentSpec `json:",inline"`
}

// GetScaling returns scaling config even if receiver is nil
func (s *ScannerAnalyzerComponent) GetScaling() *ScannerAnalyzerScaling {
	if s == nil {
		return nil
	}
	return s.Scaling
}

// ScannerAnalyzerScaling defines replication settings of the analyzer.
type ScannerAnalyzerScaling struct {
	// When enabled, the number of analyzer replicas is managed dynamically based on the load, within the limits
	// specified below.
	//+kubebuilder:validation:Default=Enabled
	//+kubebuilder:default=Enabled
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling",order=1
	AutoScaling *AutoScalingPolicy `json:"autoScaling,omitempty"`

	// When autoscaling is disabled, the number of replicas will always be configured to match this value.
	//+kubebuilder:validation:Default=3
	//+kubebuilder:default=3
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Default Replicas",order=2
	Replicas *int32 `json:"replicas,omitempty"`

	//+kubebuilder:validation:Default=2
	//+kubebuilder:default=2
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling Minimum Replicas",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.autoScaling:Enabled"}
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	//+kubebuilder:validation:Default=5
	//+kubebuilder:default=5
	//+kubebuilder:validation:Minimum=1
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Autoscaling Maximum Replicas",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.autoScaling:Enabled"}
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

// AutoScalingPolicy is a type for values of spec.scannerSpec.replicas.autoScaling.
//+kubebuilder:validation:Enum=Enabled;Disabled
type AutoScalingPolicy string

const (
	// ScannerAutoScalingEnabled means that scanner autoscaling should be enabled.
	ScannerAutoScalingEnabled AutoScalingPolicy = "Enabled"
	// ScannerAutoScalingDisabled means that scanner autoscaling should be disabled.
	ScannerAutoScalingDisabled AutoScalingPolicy = "Disabled"
)
