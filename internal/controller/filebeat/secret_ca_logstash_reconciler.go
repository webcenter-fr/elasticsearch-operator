package filebeat

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/apis/shared"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/logstash"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CALogstashCondition shared.ConditionName = "CALogstashReady"
	CALogstashPhase     shared.PhaseName     = "CALogstash"
)

type caLogstashReconciler struct {
	controller.MultiPhaseStepReconcilerAction
}

func newCALogstashReconciler(client client.Client, recorder record.EventRecorder) (multiPhaseStepReconcilerAction controller.MultiPhaseStepReconcilerAction) {
	return &caLogstashReconciler{
		MultiPhaseStepReconcilerAction: controller.NewBasicMultiPhaseStepReconcilerAction(
			client,
			CALogstashPhase,
			CALogstashCondition,
			recorder,
		),
	}
}

// Read existing secret
func (r *caLogstashReconciler) Read(ctx context.Context, resource object.MultiPhaseObject, data map[string]any, logger *logrus.Entry) (read controller.MultiPhaseRead, res ctrl.Result, err error) {
	o := resource.(*beatcrd.Filebeat)
	s := &corev1.Secret{}
	sLs := &corev1.Secret{}
	read = controller.NewBasicMultiPhaseRead()

	var ls *logstashcrd.Logstash

	// Read current secret
	if err = r.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetSecretNameForCALogstash(o)}, s); err != nil {
		if !k8serrors.IsNotFound(err) {
			return read, res, errors.Wrapf(err, "Error when read secret %s", GetSecretNameForCALogstash(o))
		}
		s = nil
	}
	if s != nil {
		read.SetCurrentObjects([]client.Object{s})
	}

	if o.Spec.LogstashRef.IsManaged() {
		// Read Logstash
		ls, err = GetLogstashFromRef(ctx, r.Client(), o, o.Spec.LogstashRef)
		if err != nil {
			return read, res, errors.Wrap(err, "Error when read logstashRef")
		}
		if ls == nil {
			logger.Warn("LogstashRef not found, try latter")
			return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Check if mirror logstash pki
		if ls.Spec.Pki.IsEnabled() {
			// Read secret that store logstash pki certificates
			if err = r.Client().Get(ctx, types.NamespacedName{Namespace: ls.Namespace, Name: logstashcontrollers.GetSecretNameForPki(ls)}, sLs); err != nil {
				if !k8serrors.IsNotFound(err) {
					return read, res, errors.Wrapf(err, "Error when read secret %s", logstashcontrollers.GetSecretNameForPki(ls))
				}
				logger.Warnf("Secret not found %s/%s, try latter", ls.Namespace, logstashcontrollers.GetSecretNameForPki(ls))
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Generate expected secret
	expectedSecretCALogstashs, err := buildCALogstashSecrets(o, sLs)
	if err != nil {
		return read, res, errors.Wrapf(err, "Error when generate secret %s", GetSecretNameForCALogstash(o))
	}
	read.SetExpectedObjects(helper.ToSliceOfObject(expectedSecretCALogstashs))

	return read, res, nil
}
