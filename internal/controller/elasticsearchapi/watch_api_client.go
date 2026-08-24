package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type watchApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler]
}

func newWatchApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler] {
	return &watchApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.Watch, *eshandler.XPackWatch, eshandler.ElasticsearchHandler](client),
	}
}

func (h *watchApiClient) Build(o *elasticsearchapicrd.Watch) (watch *eshandler.XPackWatch, err error) {
	w := eshandler.XPackWatch{}

	if o.Spec.ThrottlePeriod != "" {
		w["throttle_period"] = o.Spec.ThrottlePeriod
	}

	if o.Spec.ThrottlePeriodInMillis != 0 {
		w["throttle_period_in_millis"] = o.Spec.ThrottlePeriodInMillis
	}

	if o.Spec.Trigger != nil {
		w["trigger"] = o.Spec.Trigger.Data
	}

	if o.Spec.Input != nil {
		w["input"] = o.Spec.Input.Data
	}

	if o.Spec.Condition != nil {
		w["condition"] = o.Spec.Condition.Data
	}

	if o.Spec.Transform != nil {
		w["transform"] = o.Spec.Transform.Data
	}

	if o.Spec.Actions != nil {
		w["actions"] = o.Spec.Actions.Data
	}

	if o.Spec.Metadata != nil {
		w["metadata"] = o.Spec.Metadata.Data
	}

	return &w, nil
}

func (h *watchApiClient) Get(o *elasticsearchapicrd.Watch) (object *eshandler.XPackWatch, err error) {
	return h.Client().WatchGet(o.GetExternalName())
}

func (h *watchApiClient) Create(object *eshandler.XPackWatch, o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchUpdate(o.GetExternalName(), object)
}

func (h *watchApiClient) Update(object *eshandler.XPackWatch, o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchUpdate(o.GetExternalName(), object)
}

func (h *watchApiClient) Delete(o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchDelete(o.GetExternalName())
}

func (h *watchApiClient) Diff(currentOject *eshandler.XPackWatch, expectedObject *eshandler.XPackWatch, originalObject *eshandler.XPackWatch, o *elasticsearchapicrd.Watch, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().WatchDiff(currentOject, expectedObject, originalObject)
}
