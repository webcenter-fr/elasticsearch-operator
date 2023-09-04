package elasticsearch

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SystemUserCondition common.ConditionName = "SystemUserReady"
	SystemUserPhase     common.PhaseName     = "systemUser"
)

type SystemUserReconciler struct {
	common.Reconciler
}

func NewSystemUserReconciler(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, log *logrus.Entry) controller.K8sPhaseReconciler {
	return &SystemUserReconciler{
		Reconciler: common.Reconciler{
			Recorder: recorder,
			Log: log.WithFields(logrus.Fields{
				"phase": "systemUser",
			}),
			Name:   "systemUser",
			Client: client,
			Scheme: scheme,
		},
	}
}

// Configure permit to init condition
func (r *SystemUserReconciler) Configure(ctx context.Context, req ctrl.Request, resource client.Object) (res ctrl.Result, err error) {
	return r.StdConfigure(ctx, req, resource, SystemUserCondition, SystemUserPhase)
}

// Read existing users
func (r *SystemUserReconciler) Read(ctx context.Context, resource client.Object, data map[string]any) (res ctrl.Result, err error) {
	o := resource.(*elasticsearchcrd.Elasticsearch)
	userList := &elasticsearchapicrd.UserList{}
	s := &corev1.Secret{}

	// Read current system users
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", o.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return res, errors.Wrap(err, "Error when generate label selector")
	}
	if err = r.Client.List(ctx, userList, &client.ListOptions{Namespace: o.Namespace, LabelSelector: labelSelectors}); err != nil {
		return res, errors.Wrapf(err, "Error when read system users")
	}
	data["currentObjects"] = userList.Items

	// Read secret that store credentials
	if err = r.Client.Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCredentials(o)}, s); err != nil {
		return res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCredentials(o))
	}

	// Generate expected users
	expectedUsers, err := BuildUserSystem(o, s)
	if err != nil {
		return res, errors.Wrap(err, "Error when generate system users")
	}
	data["expectedObjects"] = expectedUsers

	return res, nil
}

// Diff permit to check if users are up to date
func (r *SystemUserReconciler) Diff(ctx context.Context, resource client.Object, data map[string]interface{}) (diff controller.K8sDiff, res ctrl.Result, err error) {
	return r.Reconciler.StdListDiff(ctx, resource, data)
}

// OnError permit to set status condition on the right state and record error
func (r *SystemUserReconciler) OnError(ctx context.Context, resource client.Object, data map[string]any, currentErr error) (res ctrl.Result, err error) {
	return r.StdOnError(ctx, resource, data, currentErr, SystemUserCondition, SystemUserPhase)
}

// OnSuccess permit to set status condition on the right state is everithink is good
func (r *SystemUserReconciler) OnSuccess(ctx context.Context, resource client.Object, data map[string]any, diff controller.K8sDiff) (res ctrl.Result, err error) {
	return r.StdOnSuccess(ctx, resource, data, diff, SystemUserCondition, SystemUserPhase)
}
