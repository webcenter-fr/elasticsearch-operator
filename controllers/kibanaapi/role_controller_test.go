package kibanaapi

import (
	"context"
	"testing"
	"time"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/disaster37/kb-handler/v8/mocks"
	"github.com/disaster37/kb-handler/v8/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *KibanaapiControllerTestSuite) TestKibanaRoleReconciler() {
	key := types.NamespacedName{
		Name:      "t-role-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	role := &kibanaapicrd.Role{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, role, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateRoleStep(),
		doUpdateRoleStep(),
		doDeleteRoleStep(),
	}
	testCase.PreTest = doMockRole(t.mockKibanaHandler)

	testCase.Run()
}
func doMockRole(mockKB *mocks.MockKibanaHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {

		isCreated := false
		isUpdated := false

		mockKB.EXPECT().RoleGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*kbapi.KibanaRole, error) {

			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {

					resp := &kbapi.KibanaRole{
						Name: name,
						Elasticsearch: &kbapi.KibanaRoleElasticsearch{
							RunAs: []string{"test"},
						},
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &kbapi.KibanaRole{
						Name: name,
						Elasticsearch: &kbapi.KibanaRoleElasticsearch{
							RunAs: []string{"test"},
						},
					}
					return resp, nil
				} else {
					resp := &kbapi.KibanaRole{
						Name: name,
						Elasticsearch: &kbapi.KibanaRoleElasticsearch{
							RunAs: []string{"test2"},
						},
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockKB.EXPECT().RoleDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *kbapi.KibanaRole) (*patch.PatchResult, error) {
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

		mockKB.EXPECT().RoleUpdate(gomock.Any()).AnyTimes().DoAndReturn(func(role *kbapi.KibanaRole) error {
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

		mockKB.EXPECT().RoleDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateRoleStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new role %s/%s ===", key.Namespace, key.Name)

			role := &kibanaapicrd.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: kibanaapicrd.RoleSpec{
					KibanaRef: shared.KibanaRef{
						ManagedKibanaRef: &shared.KibanaManagedRef{
							Name: "test",
						},
					},
					Elasticsearch: &kibanaapicrd.KibanaRoleElasticsearch{
						RunAs: []string{"test"},
					},
				},
			}
			if err = c.Create(context.Background(), role); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &kibanaapicrd.Role{}
			isCreated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, role); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isCreated"]; ok {
					isCreated = b.(bool)
				}
				if !isCreated {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get kibana role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, RoleCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, role.Status.Sync)

			return nil
		},
	}
}

func doUpdateRoleStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update role %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Role is null")
			}
			role := o.(*kibanaapicrd.Role)

			role.Spec.Elasticsearch.RunAs = []string{"test2"}
			if err = c.Update(context.Background(), role); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &kibanaapicrd.Role{}
			isUpdated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, role); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isUpdated"]; ok {
					isUpdated = b.(bool)
				}
				if !isUpdated {
					return errors.New("Not yet updated")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get kibana role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, RoleCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, role.Status.Sync)

			return nil
		},
	}
}

func doDeleteRoleStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete role %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Role is null")
			}
			role := o.(*kibanaapicrd.Role)

			wait := int64(0)
			if err = c.Delete(context.Background(), role, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &kibanaapicrd.Role{}
			isDeleted := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
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
				t.Fatalf("Kibana role stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}