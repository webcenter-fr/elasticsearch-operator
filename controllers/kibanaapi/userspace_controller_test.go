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

func (t *KibanaapiControllerTestSuite) TestKibanaUserSpaceReconciler() {
	key := types.NamespacedName{
		Name:      "t-space-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	space := &kibanaapicrd.UserSpace{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, space, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateUserSpaceStep(),
		doUpdateUserSpaceStep(),
		doDeleteUserSpaceStep(),
		doCreateUserSpaceWithObjectsStep(),
		doDeleteUserSpaceStep(),
	}
	testCase.PreTest = doMockUserSpace(t.mockKibanaHandler)

	testCase.Run()
}
func doMockUserSpace(mockKB *mocks.MockKibanaHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {

		isCreated := false
		isUpdated := false
		isCreatedWithObject := false

		mockKB.EXPECT().UserSpaceGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*kbapi.KibanaSpace, error) {

			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {

					resp := &kbapi.KibanaSpace{
						Name:        "test",
						ID:          name,
						Description: "test",
					}
					return resp, nil
				}
			case "createWithObject":
				if !isCreatedWithObject {
					return nil, nil
				} else {

					resp := &kbapi.KibanaSpace{
						Name:        "test",
						ID:          name,
						Description: "test",
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &kbapi.KibanaSpace{
						Name:        "test",
						ID:          name,
						Description: "test",
					}
					return resp, nil
				} else {
					resp := &kbapi.KibanaSpace{
						Name:        "test",
						ID:          name,
						Description: "test2",
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockKB.EXPECT().UserSpaceDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *kbapi.KibanaSpace) (*patch.PatchResult, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return &patch.PatchResult{
						Patch: []byte("fake change"),
					}, nil
				} else {
					return &patch.PatchResult{}, nil
				}
			case "createWithObject":
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

		mockKB.EXPECT().UserSpaceCreate(gomock.Any()).AnyTimes().DoAndReturn(func(space *kbapi.KibanaSpace) error {
			switch *stepName {
			case "create":
				isCreated = true
				data["isCreated"] = true
				return nil
			case "createWithObject":
				isCreatedWithObject = true
				data["isCreatedWithObject"] = true
				return nil
			}

			return nil

		})

		mockKB.EXPECT().UserSpaceUpdate(gomock.Any()).AnyTimes().DoAndReturn(func(space *kbapi.KibanaSpace) error {
			isUpdated = true
			data["isUpdated"] = true
			return nil
		})

		mockKB.EXPECT().UserSpaceDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		mockKB.EXPECT().UserSpaceCopyObject(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(originSpace string, copySpec *kbapi.KibanaSpaceCopySavedObjectParameter) error {
			switch *stepName {
			case "createWithObject":
				data["isCopyObject"] = true
				return nil
			}

			return nil
		})

		return nil
	}
}

func doCreateUserSpaceStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new user space %s/%s ===", key.Namespace, key.Name)

			space := &kibanaapicrd.UserSpace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: kibanaapicrd.UserSpaceSpec{
					KibanaRef: shared.KibanaRef{
						ManagedKibanaRef: &shared.KibanaManagedRef{
							Name: "test",
						},
					},
					Description: "test",
					Name:        "test",
				},
			}
			if err = c.Create(context.Background(), space); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			space := &kibanaapicrd.UserSpace{}
			isCreated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, space); err != nil {
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
				t.Fatalf("Failed to get kibana user space: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, UserSpaceCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, space.Status.Sync)

			return nil
		},
	}
}

func doCreateUserSpaceWithObjectsStep() test.TestStep {
	return test.TestStep{
		Name: "createWithObject",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new user space with object %s/%s ===", key.Namespace, key.Name)

			space := &kibanaapicrd.UserSpace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: kibanaapicrd.UserSpaceSpec{
					KibanaRef: shared.KibanaRef{
						ManagedKibanaRef: &shared.KibanaManagedRef{
							Name: "test",
						},
					},
					Description: "test",
					Name:        "test",
					KibanaUserSpaceCopies: []kibanaapicrd.KibanaUserSpaceCopy{
						{
							OriginUserSpace: "default",
							KibanaObjects: []kibanaapicrd.KibanaSpaceObjectParameter{
								{
									Type: "index-pattern",
									ID:   "fake",
								},
							},
						},
					},
				},
			}
			if err = c.Create(context.Background(), space); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			space := &kibanaapicrd.UserSpace{}
			isCreated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, space); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isCreatedWithObject"]; ok {
					isCreated = b.(bool)
				}
				if !isCreated {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get kibana user space: %s", err.Error())
			}

			isCopyObject := false
			if b, ok := data["isCopyObject"]; ok {
				isCopyObject = b.(bool)
			}
			assert.True(t, isCopyObject)
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, UserSpaceCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, space.Status.Sync)

			return nil
		},
	}
}

func doUpdateUserSpaceStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update user space %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("User space is null")
			}
			space := o.(*kibanaapicrd.UserSpace)

			space.Spec.Description = "test2"
			if err = c.Update(context.Background(), space); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			space := &kibanaapicrd.UserSpace{}
			isUpdated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, space); err != nil {
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
				t.Fatalf("Failed to get kibana user space: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, UserSpaceCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(space.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, space.Status.Sync)

			return nil
		},
	}
}

func doDeleteUserSpaceStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete user space %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("User space is null")
			}
			space := o.(*kibanaapicrd.UserSpace)

			wait := int64(0)
			if err = c.Delete(context.Background(), space, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			space := &kibanaapicrd.UserSpace{}
			isDeleted := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, space); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Kibana user space stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
