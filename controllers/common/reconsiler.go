package common

import (
	"time"

	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RequeuedDuration = time.Minute * 1
)

type Reconciler struct {
	Recorder record.EventRecorder
	Log      *logrus.Entry
}

type CompareResource struct {
	Current  client.Object
	Expected client.Object
	Diff     *controller.Diff
}
