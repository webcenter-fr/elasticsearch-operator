package elasticsearchapi

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	"go.uber.org/mock/gomock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestRoleReconciler() {
	key := types.NamespacedName{
		Name:      "t-role-" + helper.RandomString(10),
		Namespace: "default",
	}
	data := map[string]any{}

	testCase := test.NewTestCase[*elasticsearchapicrd.Role](t.T(), t.k8sClient, key, 5*time.Second, data)
	testCase.Steps = []test.TestStep[*elasticsearchapicrd.Role]{
		doCreateRoleStep(),
		doUpdateRoleStep(),
		doDeleteRoleStep(),
	}
	testCase.PreTest = doMockRole(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockRole(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreated := false
		isUpdated := false

		mockES.EXPECT().RoleGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*eshandler.XPackSecurityRole, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {

					resp := &eshandler.XPackSecurityRole{
						RunAs: []string{"test"},
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &eshandler.XPackSecurityRole{
						RunAs: []string{"test"},
					}
					return resp, nil
				} else {
					resp := &eshandler.XPackSecurityRole{
						RunAs: []string{"test2"},
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().RoleDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *eshandler.XPackSecurityRole) (*patch.PatchResult, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return &patch.PatchResult{
						Patch: []byte("fake change"),
					}, nil
				} else {
					return &patch.PatchResult{}, nil
				}
			case "update":
				if !isUpdated {
					return &patch.PatchResult{
						Patch: []byte("fake change"),
					}, nil
				} else {
					return &patch.PatchResult{}, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().RoleUpdate(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, policy *eshandler.XPackSecurityRole) error {
			switch *stepName {
			case "create":
				isCreated = true
				data["isCreated"] = true
				return nil
			case "update":
				isUpdated = true
				data["isUpdated"] = true
				return nil
			}

			return nil
		})

		mockES.EXPECT().RoleDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateRoleStep() test.TestStep[*elasticsearchapicrd.Role] {
	return test.TestStep[*elasticsearchapicrd.Role]{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			logrus.Infof("=== Add new role %s/%s ===\n\n", key.Namespace, key.Name)

			role := &elasticsearchapicrd.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.RoleSpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					RunAs: []string{"test"},
				},
			}
			if err = c.Create(context.Background(), role); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
			isCreated := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, role); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isCreated"]; ok {
					isCreated = b.(bool)
				}
				if !isCreated || role.GetStatus().GetObservedGeneration() == 0 {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get elasticsearch role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *role.Status.IsSync)

			return nil
		},
	}
}

func doUpdateRoleStep() test.TestStep[*elasticsearchapicrd.Role] {
	return test.TestStep[*elasticsearchapicrd.Role]{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			logrus.Infof("=== Update role %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Role is null")
			}

			data["lastGeneration"] = o.GetStatus().GetObservedGeneration()
			o.Spec.RunAs = []string{"test2"}
			if err = c.Update(context.Background(), o); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
			isUpdated := false

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, role); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isUpdated"]; ok {
					isUpdated = b.(bool)
				}
				if !isUpdated || lastGeneration == role.GetStatus().GetObservedGeneration() {
					return errors.New("Not yet updated")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get elasticsearch role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *role.Status.IsSync)

			return nil
		},
	}
}

func doDeleteRoleStep() test.TestStep[*elasticsearchapicrd.Role] {
	return test.TestStep[*elasticsearchapicrd.Role]{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			logrus.Infof("=== Delete role %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Role is null")
			}

			wait := int64(0)
			if err = c.Delete(context.Background(), o, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.Role, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
			isDeleted := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, role); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Elasticsearch role stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
