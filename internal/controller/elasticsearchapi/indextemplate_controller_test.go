package elasticsearchapi

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"go.uber.org/mock/gomock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestIndexTemplateReconciler() {
	key := types.NamespacedName{
		Name:      "t-indextemplate-" + helper.RandomString(10),
		Namespace: "default",
	}
	it := &elasticsearchapicrd.IndexTemplate{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, it, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateIndexTemplateStep(),
		doUpdateIndexTemplateStep(),
		doDeleteIndexTemplateStep(),
	}
	testCase.PreTest = doMockIndexTemplate(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockIndexTemplate(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreated := false
		isUpdated := false

		mockES.EXPECT().IndexTemplateGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*olivere.IndicesGetIndexTemplate, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {

					resp := &olivere.IndicesGetIndexTemplate{
						IndexPatterns: []string{"test"},
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &olivere.IndicesGetIndexTemplate{
						IndexPatterns: []string{"test"},
					}
					return resp, nil
				} else {
					resp := &olivere.IndicesGetIndexTemplate{
						IndexPatterns: []string{"test2"},
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().IndexTemplateDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *olivere.IndicesGetIndexTemplate) (*patch.PatchResult, error) {
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

		mockES.EXPECT().IndexTemplateUpdate(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, policy *olivere.IndicesGetIndexTemplate) error {
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

		mockES.EXPECT().IndexTemplateDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateIndexTemplateStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new index template %s/%s ===\n\n", key.Namespace, key.Name)

			template := &elasticsearchapicrd.IndexTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.IndexTemplateSpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					IndexPatterns: []string{"test"},
				},
			}
			if err = c.Create(context.Background(), template); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			template := &elasticsearchapicrd.IndexTemplate{}
			isCreated := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, template); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isCreated"]; ok {
					isCreated = b.(bool)
				}
				if !isCreated || template.GetStatus().GetObservedGeneration() == 0 {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get index template: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(template.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *template.Status.IsSync)

			return nil
		},
	}
}

func doUpdateIndexTemplateStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update index template %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Index template is null")
			}
			template := o.(*elasticsearchapicrd.IndexTemplate)

			data["lastGeneration"] = template.GetStatus().GetObservedGeneration()
			template.Spec.IndexPatterns = []string{"test2"}
			if err = c.Update(context.Background(), template); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			template := &elasticsearchapicrd.IndexTemplate{}
			isUpdated := false

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, template); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isUpdated"]; ok {
					isUpdated = b.(bool)
				}
				if !isUpdated || lastGeneration == template.GetStatus().GetObservedGeneration() {
					return errors.New("Not yet updated")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get index template: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(template.Status.Conditions, controller.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *template.Status.IsSync)

			return nil
		},
	}
}

func doDeleteIndexTemplateStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete index template %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Index template is null")
			}
			template := o.(*elasticsearchapicrd.IndexTemplate)

			wait := int64(0)
			if err = c.Delete(context.Background(), template, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			template := &elasticsearchapicrd.IndexTemplate{}
			isDeleted := false

			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, template); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Index template stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)
			return nil
		},
	}
}
