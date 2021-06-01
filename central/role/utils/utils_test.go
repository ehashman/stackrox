package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	labelUtils "github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func TestValidatePermissionSet(t *testing.T) {
	mockGoodID := permissionSetIDPrefix + "Tanis Half-Elven"
	mockBadID := "Tanis Half-Elven"
	mockName := "Hero of the Lance"
	mockGoodResource := "K8sRoleBinding"
	mockBadResource := "K8sWitchcraftAndWizardry"

	testCasesGood := map[string]*storage.PermissionSet{
		"id and name are set": {
			Id:   mockGoodID,
			Name: mockName,
		},
		"id, name, and a resource are set": {
			Id:   mockGoodID,
			Name: mockName,
			ResourceToAccess: map[string]storage.Access{
				mockGoodResource: storage.Access_READ_ACCESS,
			},
		},
	}

	testCasesBad := map[string]*storage.PermissionSet{
		"empty permissionSet": {},
		"id is missing": {
			Name: mockName,
		},
		"name is missing": {
			Id: mockGoodID,
		},
		"bad id": {
			Id:   mockBadID,
			Name: mockName,
		},
		"bad resource": {
			Id:   mockGoodID,
			Name: mockName,
			ResourceToAccess: map[string]storage.Access{
				mockBadResource: storage.Access_NO_ACCESS,
			},
		},
	}

	for desc, permissionSet := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidatePermissionSet(permissionSet)
			assert.NoErrorf(t, err, "permission set: '%+v'", permissionSet)
		})
	}

	for desc, permissionSet := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidatePermissionSet(permissionSet)
			assert.Errorf(t, err, "permission set: '%+v'", permissionSet)
		})
	}
}

func TestValidateSimpleAccessScope(t *testing.T) {
	mockGoodID := EnsureValidAccessScopeID("42")
	mockBadID := "42"
	mockName := "Heart of Gold"
	mockDescription := "HHGTTG"
	mockGoodRules := &storage.SimpleAccessScope_Rules{
		IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
			{
				ClusterName:   "Atomic Vector Plotter",
				NamespaceName: "Advanced Tea Substitute",
			},
		},
	}
	mockBadRules := &storage.SimpleAccessScope_Rules{
		NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
			{
				Requirements: nil,
			},
		},
	}

	testCasesGood := map[string]*storage.SimpleAccessScope{
		"id and name are set": {
			Id:   mockGoodID,
			Name: mockName,
		},
		"id, name, and namespace label selector are set": {
			Id:    mockGoodID,
			Name:  mockName,
			Rules: mockGoodRules,
		},
		"all possible fields are set": {
			Id:          mockGoodID,
			Name:        mockName,
			Description: mockDescription,
			Rules:       mockGoodRules,
		},
	}

	testCasesBad := map[string]*storage.SimpleAccessScope{
		"empty simple access scope": {},
		"id is missing": {
			Name: mockName,
		},
		"name is missing": {
			Id: mockGoodID,
		},
		"bad id": {
			Id:   mockBadID,
			Name: mockName,
		},
		"bad rules": {
			Id:    mockGoodID,
			Name:  mockName,
			Rules: mockBadRules,
		},
	}

	for desc, scope := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScope(scope)
			assert.NoErrorf(t, err, "simple access scope: '%+v'", scope)
		})
	}

	for desc, scope := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScope(scope)
			assert.Errorf(t, err, "simple access scope: '%+v'", scope)
		})
	}
}

func TestValidateSimpleAccessScopeRules(t *testing.T) {
	mockClusterName := "Infinite Improbability Drive"
	mockGoodNamespace := &storage.SimpleAccessScope_Rules_Namespace{
		ClusterName:   "Atomic Vector Plotter",
		NamespaceName: "Advanced Tea Substitute",
	}
	mockBadNamespace1 := &storage.SimpleAccessScope_Rules_Namespace{
		ClusterName: "Brownian Motion Producer",
	}
	mockBadNamespace2 := &storage.SimpleAccessScope_Rules_Namespace{
		NamespaceName: "Bambleweeny 57 Submeson Brain",
	}
	mockGoodSelector := labelUtils.LabelSelector("fleet", storage.SetBasedLabelSelector_NOT_IN, []string{"vogon"})
	mockBadSelector := &storage.SetBasedLabelSelector{
		Requirements: nil,
	}

	testCasesGood := map[string]*storage.SimpleAccessScope_Rules{
		"valid namespace label selector": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockGoodNamespace,
			},
		},
		"all possible rules are set": {
			IncludedClusters: []string{
				mockClusterName,
			},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockGoodNamespace,
			},
			ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
				mockGoodSelector,
			},
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
				mockGoodSelector,
			},
		},
	}

	testCasesBad := map[string]*storage.SimpleAccessScope_Rules{
		"namespace with missing namespace name": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace1,
			},
		},
		"namespace with missing cluster name": {
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				mockBadNamespace2,
			},
		},
		"cluster label selector with empty requirements": {
			ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		},
		"namespace label selector with empty requirements": {
			NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
				mockBadSelector,
			},
		},
	}

	for desc, scopeRules := range testCasesGood {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScopeRules(scopeRules)
			assert.NoErrorf(t, err, "simple access scope rules: '%+v'", scopeRules)
		})
	}

	for desc, scopeRules := range testCasesBad {
		t.Run(desc, func(t *testing.T) {
			err := ValidateSimpleAccessScopeRules(scopeRules)
			assert.Errorf(t, err, "simple access scope rules: '%+v'", scopeRules)
		})
	}
}
