package elasticsearchapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/golang/mock/gomock"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestSnapshotLifecyclePolicyReconciler() {
	key := types.NamespacedName{
		Name:      "t-slm-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, slm, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateSLMStep(),
		doUpdateSLMStep(),
		doDeleteSLMStep(),
	}
	testCase.PreTest = doMockSLM(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockSLM(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreated := false
		isUpdated := false

		mockES.EXPECT().SnapshotRepositoryGet(gomock.Any()).AnyTimes().Return(&olivere.SnapshotRepositoryMetaData{
			Type: "url",
		}, nil)

		mockES.EXPECT().SLMGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*eshandler.SnapshotLifecyclePolicySpec, error) {

			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {
					rawPolicy := `
					{
						"schedule": "0 30 1 * * ?",
						"name": "<daily-snap-{now/d}>",
						"repository": "my_repository",
						"config": {
						  "indices": ["data-*", "important"],
						  "ignore_unavailable": false,
						  "include_global_state": false
						},
						"retention": {
						  "expire_after": "30d",
						  "min_count": 5,
						  "max_count": 50
						}
					}`
					resp := &eshandler.SnapshotLifecyclePolicySpec{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					rawPolicy := `
					{
						"schedule": "0 30 1 * * ?",
						"name": "<daily-snap-{now/d}>",
						"repository": "my_repository",
						"config": {
						  "indices": ["data-*", "important"],
						  "ignore_unavailable": false,
						  "include_global_state": false
						},
						"retention": {
						  "expire_after": "30d",
						  "min_count": 5,
						  "max_count": 50
						}
					}`
					resp := &eshandler.SnapshotLifecyclePolicySpec{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}
					return resp, nil
				} else {
					rawPolicy := `
					{
						"schedule": "0 30 1 * * ?",
						"name": "<daily-snap-{now/d}>",
						"repository": "my_repository",
						"config": {
						  "indices": ["data-*", "important"],
						  "ignore_unavailable": false,
						  "include_global_state": false
						},
						"retention": {
						  "expire_after": "30d",
						  "min_count": 6,
						  "max_count": 50
						}
					}`
					resp := &eshandler.SnapshotLifecyclePolicySpec{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().SLMDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *eshandler.SnapshotLifecyclePolicySpec) (*patch.PatchResult, error) {
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

		mockES.EXPECT().SLMUpdate(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, policy *eshandler.SnapshotLifecyclePolicySpec) error {
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

		mockES.EXPECT().SLMDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateSLMStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new SLM policy %s/%s ===", key.Namespace, key.Name)

			slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.SnapshotLifecyclePolicySpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					Schedule:   "0 30 1 * * ?",
					Name:       "<daily-snap-{now/d}>",
					Repository: "my_repository",
					Config: elasticsearchapicrd.SLMConfig{
						Indices:            []string{"data-*", "important"},
						IgnoreUnavailable:  false,
						IncludeGlobalState: false,
					},
					Retention: &elasticsearchapicrd.SLMRetention{
						ExpireAfter: "30d",
						MinCount:    5,
						MaxCount:    50,
					},
				},
			}
			if err = c.Create(context.Background(), slm); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{}
			isCreated := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, slm); err != nil {
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
				t.Fatalf("Failed to get SLM: %s", err.Error())
			}

			assert.True(t, condition.IsStatusConditionPresentAndEqual(slm.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *slm.Status.IsSync)

			return nil
		},
	}
}

func doUpdateSLMStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update SLM policy %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("SLM is null")
			}
			slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

			slm.Spec.Retention.MinCount = 6
			if err = c.Update(context.Background(), slm); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{}
			isUpdated := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, slm); err != nil {
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
				t.Fatalf("Failed to get SLM: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(slm.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *slm.Status.IsSync)

			return nil
		},
	}
}

func doDeleteSLMStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete SLM policy %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("SLM is null")
			}
			slm := o.(*elasticsearchapicrd.SnapshotLifecyclePolicy)

			wait := int64(0)
			if err = c.Delete(context.Background(), slm, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}
			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			slm := &elasticsearchapicrd.SnapshotLifecyclePolicy{}
			isDeleted := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, slm); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("SLM stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
