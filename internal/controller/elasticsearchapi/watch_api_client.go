package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type watchApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler]
}

func newWatchApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler] {
	return &watchApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler](client),
	}
}

func (h *watchApiClient) Build(o *elasticsearchapicrd.Watch) (watch *olivere.XPackWatch, err error) {
	watch = &olivere.XPackWatch{
		ThrottlePeriod:         o.Spec.ThrottlePeriod,
		ThrottlePeriodInMillis: o.Spec.ThrottlePeriodInMillis,
	}

	if o.Spec.Trigger != nil {
		watch.Trigger = map[string]map[string]any{}
		for key, data := range o.Spec.Trigger.Data {
			watch.Trigger[key] = data.(map[string]any)
		}
	}

	if o.Spec.Input != nil {
		watch.Input = map[string]map[string]any{}
		for key, data := range o.Spec.Input.Data {
			watch.Input[key] = data.(map[string]any)
		}
	}

	if o.Spec.Condition != nil {
		watch.Condition = map[string]map[string]any{}
		for key, data := range o.Spec.Condition.Data {
			watch.Condition[key] = data.(map[string]any)
		}
	}

	if o.Spec.Transform != nil {
		watch.Transform = o.Spec.Condition.Data
	}

	if o.Spec.Actions != nil {
		watch.Actions = map[string]map[string]any{}
		for key, data := range o.Spec.Actions.Data {
			watch.Actions[key] = data.(map[string]any)
		}
	}

	if o.Spec.Metadata != nil {
		watch.Metadata = o.Spec.Metadata.Data
	}

	return watch, nil
}

func (h *watchApiClient) Get(o *elasticsearchapicrd.Watch) (object *olivere.XPackWatch, err error) {
	return h.Client().WatchGet(o.GetExternalName())
}

func (h *watchApiClient) Create(object *olivere.XPackWatch, o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchUpdate(o.GetExternalName(), object)
}

func (h *watchApiClient) Update(object *olivere.XPackWatch, o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchUpdate(o.GetExternalName(), object)
}

func (h *watchApiClient) Delete(o *elasticsearchapicrd.Watch) (err error) {
	return h.Client().WatchDelete(o.GetExternalName())
}

func (h *watchApiClient) Diff(currentOject *olivere.XPackWatch, expectedObject *olivere.XPackWatch, originalObject *olivere.XPackWatch, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().WatchDiff(currentOject, expectedObject, originalObject)
}
