package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SystemUserCondition shared.ConditionName = "SystemUserReady"
	SystemUserPhase     shared.PhaseName     = "systemUser"
)

type systemUserReconciler struct {
	controller.BaseReconciler
	controller.MultiPhaseStepReconcilerAction
}

func newSystemUserReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &systemUserReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			SystemUserPhase,
			SystemUserCondition,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Recorder: recorder,
			Log:      logger,
		},
	}
}

// Read existing users
func (r *systemUserReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	userList := &elasticsearchapicrd.UserList{}
	s := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	// Read current system users
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, userList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read system users")
	}
	read.SetCurrentObjects(helper.ToSliceOfObject(userList.Items))

	// Read secret that store credentials
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
	}

	// Generate expected users
	expectedUsers, err := buildSystemUsers(o, s)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate system users")
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedUsers))

	return read, res, nil
}
