package elasticsearchapi

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/golang/mock/gomock"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
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

func (t *ElasticsearchapiControllerTestSuite) TestSnapshotRepositoryReconciler() {
	key := types.NamespacedName{
		Name:      "t-snapshotrepository-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	sr := &elasticsearchapicrd.SnapshotRepository{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, sr, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateSnapshotRepositoryStep(),
		doUpdateSnapshotRepositoryStep(),
		doDeleteSnapshotRepositoryStep(),
	}
	testCase.PreTest = doMockSnapshotRepository(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockSnapshotRepository(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreated := false
		isUpdated := false

		mockES.EXPECT().SnapshotRepositoryGet(gomock.Any()).AnyTimes().DoAndReturn(func(name string) (*olivere.SnapshotRepositoryMetaData, error) {
			switch *stepName {
			case "create":
				if !isCreated {
					return nil, nil
				} else {
					resp := &olivere.SnapshotRepositoryMetaData{
						Type: "url",
						Settings: map[string]any{
							"url": "http://fake",
						},
					}
					return resp, nil
				}
			case "update":
				if !isUpdated {
					resp := &olivere.SnapshotRepositoryMetaData{
						Type: "url",
						Settings: map[string]any{
							"url": "http://fake",
						},
					}
					return resp, nil
				} else {
					resp := &olivere.SnapshotRepositoryMetaData{
						Type: "url",
						Settings: map[string]any{
							"url": "http://fake2",
						},
					}
					return resp, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().SnapshotRepositoryDiff(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected, original *olivere.SnapshotRepositoryMetaData) (*patch.PatchResult, error) {
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

		mockES.EXPECT().SnapshotRepositoryUpdate(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(name string, policy *olivere.SnapshotRepositoryMetaData) error {
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

		mockES.EXPECT().SnapshotRepositoryDelete(gomock.Any()).AnyTimes().DoAndReturn(func(name string) error {
			data["isDeleted"] = true
			return nil
		})

		return nil
	}
}

func doCreateSnapshotRepositoryStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new snapshot repository %s/%s ===", key.Namespace, key.Name)

			repo := &elasticsearchapicrd.SnapshotRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.SnapshotRepositorySpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					Type: "url",
					Settings: `
					{
						"url" : "http://fake"
					}
					`,
				},
			}
			if err = c.Create(context.Background(), repo); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			repo := &elasticsearchapicrd.SnapshotRepository{}
			isCreated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, repo); err != nil {
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
				t.Fatalf("Failed to get Snapshot repository: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(repo.Status.Conditions, common.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *repo.Status.IsSync)

			return nil
		},
	}
}

func doUpdateSnapshotRepositoryStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update snapshot repository %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Snapshot repo is null")
			}
			repo := o.(*elasticsearchapicrd.SnapshotRepository)

			repo.Spec.Settings = `
				{
					"url" : "http://fake2"
				}
			`
			if err = c.Update(context.Background(), repo); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			repo := &elasticsearchapicrd.SnapshotRepository{}
			isUpdated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, repo); err != nil {
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
				t.Fatalf("Failed to get Snapshot repository: %s", err.Error())
			}
			assert.True(t, condition.IsStatusConditionPresentAndEqual(repo.Status.Conditions, common.ReadyCondition.String(), metav1.ConditionTrue))
			assert.True(t, *repo.Status.IsSync)

			return nil
		},
	}
}

func doDeleteSnapshotRepositoryStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete snapshot repository %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Snapshot repo is null")
			}
			repo := o.(*elasticsearchapicrd.SnapshotRepository)

			wait := int64(0)
			if err = c.Delete(context.Background(), repo, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			repo := &elasticsearchapicrd.SnapshotRepository{}
			isDeleted := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, repo); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Snapshot repository stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)
			return nil
		},
	}
}
