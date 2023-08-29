package elasticsearchapi

import (
	"context"
	"testing"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/es-handler/v8/mocks"
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
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchapiControllerTestSuite) TestLicenseReconciler() {
	key := types.NamespacedName{
		Name:      "t-license-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	license := &elasticsearchapicrd.License{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, license, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doEnableBasicLicenseStep(),
		doDeleteBasicLicenseStep(),
		doUpdateToEnterpriseLicenseStep(),
		doUpdateEnterpriseLicenseStep(),
		doDeleteEnterpriseLicenseStep(),
	}
	testCase.PreTest = doMockLicense(t.mockElasticsearchHandler)

	testCase.Run()
}

func doMockLicense(mockES *mocks.MockElasticsearchHandler) func(stepName *string, data map[string]any) error {
	return func(stepName *string, data map[string]any) (err error) {
		isCreatedBasicLicense := false
		isUpdatedToEnterpriseLicense := false
		isUpdatedEnterpriseLicense := false

		mockES.EXPECT().LicenseGet().AnyTimes().DoAndReturn(func() (*olivere.XPackInfoLicense, error) {
			switch *stepName {
			case "create_basic_license":
				if !isCreatedBasicLicense {
					return nil, nil
				} else {
					return &olivere.XPackInfoLicense{
						UID:  "test",
						Type: "basic",
					}, nil
				}
			case "update_to_enterprise_license":
				if !isUpdatedToEnterpriseLicense {
					return &olivere.XPackInfoLicense{
						UID:  "test",
						Type: "basic",
					}, nil
				} else {
					return &olivere.XPackInfoLicense{
						UID:  "test",
						Type: "gold",
					}, nil
				}
			case "update_enterprise_license":
				if !isUpdatedEnterpriseLicense {
					return &olivere.XPackInfoLicense{
						UID:  "test",
						Type: "basic",
					}, nil
				} else {
					return &olivere.XPackInfoLicense{
						UID:  "test2",
						Type: "gold",
					}, nil
				}
			}

			return nil, nil
		})

		mockES.EXPECT().LicenseDiff(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(actual, expected *olivere.XPackInfoLicense) bool {
			switch *stepName {
			case "create_basic_license":
				if !isCreatedBasicLicense {
					return true
				} else {
					return false
				}
			case "update_to_enterprise_license":
				if !isUpdatedToEnterpriseLicense {
					return true
				} else {
					return false
				}
			case "update_enterprise_license":
				if !isUpdatedEnterpriseLicense {
					return true
				} else {
					return false
				}
			}

			return false
		})

		mockES.EXPECT().LicenseEnableBasic().AnyTimes().DoAndReturn(func() error {
			switch *stepName {
			case "create_basic_license":
				if !isCreatedBasicLicense {
					data["isCreatedBasicLicense"] = true
					return nil
				} else {
					return nil
				}
			case "delete_enterprise_license":
				data["isDeleted"] = true
				return nil
			}

			return nil
		})

		mockES.EXPECT().LicenseUpdate(gomock.Any()).AnyTimes().DoAndReturn(func(license string) error {
			switch *stepName {
			case "update_to_enterprise_license":
				data["isUpdatedToEnterpriseLicense"] = true
				return nil
			case "update_enterprise_license":
				data["isUpdatedEnterpriseLicense"] = true
				return nil
			}

			return nil
		})

		return nil
	}
}

func doEnableBasicLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "create_basic_license",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Enable basic license %s/%s ===", key.Namespace, key.Name)

			license := &elasticsearchapicrd.License{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.LicenseSpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					SecretRef: &core.LocalObjectReference{
						Name: key.Name,
					},
					Basic: pointer.Bool(true),
				},
			}

			if err = c.Create(context.Background(), license); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			license := &elasticsearchapicrd.License{}

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, license); err != nil {
					t.Fatal(err)
				}
				if !condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionTrue) {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get License: %s", err.Error())
			}
			assert.Empty(t, license.Status.ExpireAt)
			assert.Empty(t, license.Status.LicenseChecksum)
			assert.Equal(t, "basic", license.Status.LicenseType)
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, license.Status.Sync)

			return nil
		},
	}
}

func doDeleteBasicLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "delete_basic_license",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete basic license %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("License is null")
			}
			license := o.(*elasticsearchapicrd.License)

			wait := int64(0)
			if err = c.Delete(context.Background(), license, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			license := &elasticsearchapicrd.License{}
			isDeleted := true

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, license); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("License stil exist: %s", err.Error())
			}

			assert.True(t, isDeleted)

			return nil
		},
	}
}

func doUpdateToEnterpriseLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "update_to_enterprise_license",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update to enterprise %s/%s ===", key.Namespace, key.Name)

			licenseJson := `
			{
				"license": {
					"uid": "test",
					"type": "gold",
					"issue_date_in_millis": 1629849600000,
					"expiry_date_in_millis": 1661990399999,
					"max_nodes": 15,
					"issued_to": "test",
					"issuer": "API",
					"signature": "test",
					"start_date_in_millis": 1629849600000
				}
			}
			`
			secret := &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Data: map[string][]byte{
					"license": []byte(licenseJson),
				},
			}
			license := &elasticsearchapicrd.License{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapicrd.LicenseSpec{
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: "test",
						},
					},
					SecretRef: &core.LocalObjectReference{
						Name: key.Name,
					},
					Basic: pointer.Bool(false),
				},
			}
			if err = c.Create(context.Background(), secret); err != nil {
				return err
			}
			if err = c.Create(context.Background(), license); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			license := &elasticsearchapicrd.License{}

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, license); err != nil {
					t.Fatal(err)
				}
				if !condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionTrue) {
					return errors.New("Not yet created")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get License: %s", err.Error())
			}
			assert.NotEmpty(t, license.Status.ExpireAt)
			assert.NotEmpty(t, license.Status.LicenseChecksum)
			assert.Equal(t, "gold", license.Status.LicenseType)
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, license.Status.Sync)

			return nil
		},
	}
}

func doUpdateEnterpriseLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "update_enterprise_license",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update license %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("License is null")
			}
			license := o.(*elasticsearchapicrd.License)
			data["licenseChecksum"] = license.Status.LicenseChecksum

			secret := &core.Secret{}

			licenseJson := `
			{
				"license": {
					"uid": "test2",
					"type": "gold",
					"issue_date_in_millis": 1629849600000,
					"expiry_date_in_millis": 1661990399999,
					"max_nodes": 15,
					"issued_to": "test",
					"issuer": "API",
					"signature": "test",
					"start_date_in_millis": 1629849600000
				}
			}
			`
			if err := c.Get(context.Background(), key, secret); err != nil {
				return err
			}
			secret.Data["license"] = []byte(licenseJson)
			if err := c.Update(context.Background(), secret); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			license := &elasticsearchapicrd.License{}
			isUpdated := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, license); err != nil {
					t.Fatal(err)
				}
				if b, ok := data["isUpdatedEnterpriseLicense"]; ok {
					isUpdated = b.(bool)
				}
				if !isUpdated {
					return errors.New("Not yet updated")
				}
				return nil
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Failed to get License: %s", err.Error())
			}
			assert.NotEmpty(t, license.Status.ExpireAt)
			assert.NotEqual(t, data["licenseChecksum"], license.Status.LicenseChecksum)
			assert.Equal(t, "gold", license.Status.LicenseType)
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, LicenseCondition, metav1.ConditionTrue))
			assert.True(t, condition.IsStatusConditionPresentAndEqual(license.Status.Conditions, common.ReadyCondition, metav1.ConditionTrue))
			assert.True(t, license.Status.Sync)

			return nil
		},
	}
}

func doDeleteEnterpriseLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "delete_enterprise_license",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete enterprise license %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("License is null")
			}
			license := o.(*elasticsearchapicrd.License)

			wait := int64(0)
			if err = c.Delete(context.Background(), license, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			license := &elasticsearchapicrd.License{}
			isDeleted := false

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, license); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("License stil exist: %s", err.Error())
			}
			assert.True(t, isDeleted)
			return nil
		},
	}
}
