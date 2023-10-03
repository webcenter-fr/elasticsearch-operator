package kibanaapi

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	"github.com/disaster37/kb-handler/v8/mocks"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	"go.uber.org/mock/gomock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *KibanaapiControllerTestSuite) TestKibanaLogstashPipelineReconciler() {
	key := types.NamespacedName{
		Name:      "t-pipeline-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	pipeline := &kibanaapicrd.LogstashPipeline{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, pipeline, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateLogstashPipelineStep(),
		doUpdateLogstashPipelineStep(),
		doDeleteLogstashPipelineStep(),
	}
	testCase.PreTest = doMockLogstashPipeline(t.mockKibanaHandler)

	testCase.Run()
}
func doMockLogstashPipeline(mockKB *mocks.MockKibanaHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {

		isCreated := false
		isUpdated := false

		mockKB.EXPECT().LogstashPipelineGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*kbapi.LogstashPipeline, error) {

			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {

					resp := &kbapi.LogstashPipeline{
						Description: "test",
						Pipeline:    "fake",
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &kbapi.LogstashPipeline{
						Description: "test",
						Pipeline:    "fake",
					}
					return resp, nil
				} else {
					resp := &kbapi.LogstashPipeline{
						Description: "test2",
						Pipeline:    "fake",
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockKB.EXPECT().LogstashPipelineDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *kbapi.LogstashPipeline) (*patch.PatchResult, error) {
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

		mockKB.EXPECT().LogstashPipelineUpdate(gomock.Any()).AnyTimes().DoAndReturn(func(pipeline *kbapi.LogstashPipeline) error {
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

		mockKB.EXPECT().LogstashPipelineDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateLogstashPipelineStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new logstash pipeline %s/%s ===", key.Namespace, key.Name)

			pipeline := &kibanaapicrd.LogstashPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: kibanaapicrd.LogstashPipelineSpec{
					KibanaRef: shared.KibanaRef{
						ManagedKibanaRef: &shared.KibanaManagedRef{
							Name: "test",
						},
					},
					Description: "test",
					Pipeline:    "fake",
				},
			}
			if err = c.Create(context.Background(), pipeline); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			pipeline := &kibanaapicrd.LogstashPipeline{}
			isCreated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, pipeline); err != nil {
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
				t.Fatalf("Failed to get kibana logstash pipeline: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, common.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *pipeline.Status.IsSync)

			return nil
		},
	}
}

func doUpdateLogstashPipelineStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update logstash pipeline %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Logstash pipeline is null")
			}
			pipeline := o.(*kibanaapicrd.LogstashPipeline)

			pipeline.Spec.Description = "test2"
			if err = c.Update(context.Background(), pipeline); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			pipeline := &kibanaapicrd.LogstashPipeline{}
			isUpdated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, pipeline); err != nil {
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
				t.Fatalf("Failed to get kibana logstash pipeline: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, common.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *pipeline.Status.IsSync)

			return nil
		},
	}
}

func doDeleteLogstashPipelineStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete logstash pipeline %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Logstash pipeline is null")
			}
			pipeline := o.(*kibanaapicrd.LogstashPipeline)

			wait := int64(0)
			if err = c.Delete(context.Background(), pipeline, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			pipeline := &kibanaapicrd.LogstashPipeline{}
			isDeleted := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, pipeline); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Kibana logstash pipeline stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)

			return nil
		},
	}
}
