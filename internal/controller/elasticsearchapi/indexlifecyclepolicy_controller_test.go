package elasticsearchapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	"go.uber.org/mock/gomock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestIndexLifecyclePolicyReconciler() {
	key := types.NamespacedName{
		Name:      "t-ilm-" + helper.RandomString(10),
		Namespace: "default",
	}
	data := map[string]any{}

	testCase := test.NewTestCase[*elasticsearchapicrd.IndexLifecyclePolicy](t.T(), t.k8sClient, key, 5*time.Second, data)
	testCase.Steps = []test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy]{
		doCreateILMStep(),
		doUpdateILMStep(),
		doDeleteILMStep(),
	}
	testCase.PreTest = doMockILM(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockILM(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreated := false
		isUpdated := false

		mockES.EXPECT().ILMGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*olivere.XPackIlmGetLifecycleResponse, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {
					rawPolicy := `
								{
									"policy": {
										"phases": {
											"warm": {
												"min_age": "10d",
												"actions": {
													"forcemerge": {
														"max_num_segments": 1
													}
												}
											},
											"delete": {
												"min_age": "31d",
												"actions": {
													"delete": {
														"delete_searchable_snapshot": true
													}
												}
											}
										}
									}
								}`
					resp := &olivere.XPackIlmGetLifecycleResponse{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}

					return resp, nil
				}
			case "update":
				if !isUpdated {
					rawPolicy := `
						{
							"policy": {
								"phases": {
									"warm": {
										"min_age": "10d",
										"actions": {
											"forcemerge": {
												"max_num_segments": 1
											}
										}
									},
									"delete": {
										"min_age": "31d",
										"actions": {
											"delete": {
												"delete_searchable_snapshot": true
											}
										}
									}
								}
							}
						}`
					resp := &olivere.XPackIlmGetLifecycleResponse{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}
					return resp, nil
				} else {
					rawPolicy := `
						{
							"policy": {
								"phases": {
									"warm": {
										"min_age": "30d",
										"actions": {
											"forcemerge": {
												"max_num_segments": 1
											}
										}
									},
									"delete": {
										"min_age": "31d",
										"actions": {
											"delete": {
												"delete_searchable_snapshot": true
											}
										}
									}
								}
							}
						}`
					resp := &olivere.XPackIlmGetLifecycleResponse{}
					if err := json.Unmarshal([]byte(rawPolicy), resp); err != nil {
						panic(err)
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().ILMDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *olivere.XPackIlmGetLifecycleResponse) (*patch.PatchResult, error) {
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

		mockES.EXPECT().ILMUpdate(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, policy *olivere.XPackIlmGetLifecycleResponse) error {
			switch *stepName {
			case "create":
				data["isCreated"] = true
				isCreated = true
				return nil
			case "update":
				data["isUpdated"] = true
				isUpdated = true
				return nil
			}

			return nil
		})

		mockES.EXPECT().ILMDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateILMStep() test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy] {
	return test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy]{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) (err error) {
			logrus.Infof("=== Add new ILM policy %s/%s ===\n\n", key.Namespace, key.Name)
			ilm := &elasticsearchapicrd.IndexLifecyclePolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.IndexLifecyclePolicySpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					Policy: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicy{
						Phases: elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhases{
							Warm: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhasesPhase{
								MinAge: ptr.To("10d"),
								Actions: apis.MapAny{
									Data: map[string]any{
										"forcemerge": map[string]any{
											"max_num_segments": 1,
										},
									},
								},
							},
							Delete: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhasesPhase{
								MinAge: ptr.To("31d"),
								Actions: apis.MapAny{
									Data: map[string]any{
										"delete": map[string]any{
											"delete_searchable_snapshot": true,
										},
									},
								},
							},
						},
					},
				},
			}
			if err = c.Create(context.Background(), ilm); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) (err error) {
			ilm := &elasticsearchapicrd.IndexLifecyclePolicy{}
			isCreated := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, ilm); err != nil {
					t.Fatal("ILM object not found")
				}
				if b, ok := data["isCreated"]; ok {
					isCreated = b.(bool)
				}
				if !isCreated || ilm.GetStatus().GetObservedGeneration() == 0 {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get ILM: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(ilm.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *ilm.Status.IsSync)

			return nil
		},
	}
}

func doUpdateILMStep() test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy] {
	return test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy]{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) (err error) {
			logrus.Infof("=== Update ILM policy %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("ILM is null")
			}

			data["lastGeneration"] = o.GetStatus().GetObservedGeneration()
			o.Spec.Policy = &elasticsearchapicrd.IndexLifecyclePolicySpecPolicy{
				Phases: elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhases{
					Warm: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhasesPhase{
						MinAge: ptr.To("30d"),
						Actions: apis.MapAny{
							Data: map[string]any{
								"forcemerge": map[string]any{
									"max_num_segments": 1,
								},
							},
						},
					},
					Delete: &elasticsearchapicrd.IndexLifecyclePolicySpecPolicyPhasesPhase{
						MinAge: ptr.To("31d"),
						Actions: apis.MapAny{
							Data: map[string]any{
								"delete": map[string]any{
									"delete_searchable_snapshot": true,
								},
							},
						},
					},
				},
			}
			if err = c.Update(context.Background(), o); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) error {
			ilm := &elasticsearchapicrd.IndexLifecyclePolicy{}
			isUpdated := false

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, ilm); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isUpdated"]; ok {
					isUpdated = b.(bool)
				}
				if !isUpdated || lastGeneration == ilm.GetStatus().GetObservedGeneration() {
					return errors.New("Not yet updated")
				}
				return nil
			}, time.Second*30, time.Second*1)

			if err != nil || isTimeout {
				return errors.Wrapf(err, "Failed to get ILM")
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(ilm.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *ilm.Status.IsSync)

			return nil
		},
	}
}

func doDeleteILMStep() test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy] {
	return test.TestStep[*elasticsearchapicrd.IndexLifecyclePolicy]{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) (err error) {
			logrus.Infof("=== Delete ILM policy %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("ILM is null")
			}

			wait := int64(0)
			if err = c.Delete(context.Background(), o, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *elasticsearchapicrd.IndexLifecyclePolicy, data map[string]any) (err error) {
			ilm := &elasticsearchapicrd.IndexLifecyclePolicy{}
			isDeleted := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, ilm); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)

			if err != nil || isTimeout {
				return errors.Wrapf(err, "ILM not deleted")
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
