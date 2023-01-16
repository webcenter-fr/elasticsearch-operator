package elasticsearchapi

import (
	"context"
	"testing"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestRoleReconciler() {
	key := types.NamespacedName{
		Name:      "t-role-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	role := &elasticsearchapicrd.Role{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, role, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
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

		mockES.EXPECT().RoleDiff(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected *eshandler.XPackSecurityRole) (string, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return "fake change", nil
				} else {
					return "", nil
				}
			case "update":
				if !isUpdated {
					return "fake change", nil
				} else {
					return "", nil
				}
			}

			return "", nil
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

func doCreateRoleStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new role %s/%s ===", key.Namespace, key.Name)

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
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
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
				t.Fatalf("Failed to get elasticsearch role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, RoleCondition, metav1.ConditionTrue))

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
			role := o.(*elasticsearchapicrd.Role)

			role.Spec.RunAs = []string{"test2"}
			if err = c.Update(context.Background(), role); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
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
				t.Fatalf("Failed to get elasticsearch role: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(role.Status.Conditions, RoleCondition, metav1.ConditionTrue))

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
			role := o.(*elasticsearchapicrd.Role)

			wait := int64(0)
			if err = c.Delete(context.Background(), role, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			role := &elasticsearchapicrd.Role{}
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
				t.Fatalf("Elasticsearch role stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
