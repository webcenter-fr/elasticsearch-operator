package common

import (
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
)

const (
	RequeuedDuration = time.Minute * 1
)

type Reconciler struct {
	Recorder record.EventRecorder
	Log      *logrus.Entry
}

/*
type CompareResource struct {
	Current  client.Object
	Expected client.Object
	Diff     *controller.Diff
}
*/
