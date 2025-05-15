package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/multiphase"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	SystemUserCondition shared.ConditionName = "SystemUserReady"
	SystemUserPhase     shared.PhaseName     = "systemUser"
)

type systemUserReconciler struct {
	multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.User]
}

func newSystemUserReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction multiphase.MultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.User]) {
	return &systemUserReconciler{
		MultiPhaseStepReconcilerAction: multiphase.NewMultiPhaseStepReconcilerAction[*elasticsearchcrd.Elasticsearch, *elasticsearchapicrd.User](
			client,
			SystemUserPhase,
			SystemUserCondition,
			recorder,
		),
	}
}

// Read existing users
func (r *systemUserReconciler) Read(ctx context.Context, o *elasticsearchcrd.Elasticsearch, data map[string]any, logger *logrus.Entry) (read multiphase.MultiPhaseRead[*elasticsearchapicrd.User], res reconcile.Result, err error) {
	userList := &elasticsearchapicrd.UserList{}
	s := &corev1.Secret{}
	read = multiphase.NewMultiPhaseRead[*elasticsearchapicrd.User]()

	// Read current system users
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client().List(ctx, userList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return read, res, errors.Wrapf(err, "Error when read system users")
	}
	read.SetCurrentObjects(helper.ToSlicePtr(userList.Items))

	// Read secret that store credentials
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
	}

	// Generate expected users
	expectedUsers, err := buildSystemUsers(o, s)
	if err != nil {
		return read, res, errors.Wrap(err, "Error when generate system users")
	}
	read.SetExpectedObjects(expectedUsers)

	return read, res, nil
}
