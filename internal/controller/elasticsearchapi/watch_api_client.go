package elasticsearchapi

import (
	"encoding/json"

	"emperror.dev/errors"
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

	if o.Spec.Trigger != "" {
		trigger := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Trigger), &trigger); err != nil {
			return nil, errors.Wrap(err, "Error when decode trigger")
		}
		watch.Trigger = trigger
	}

	if o.Spec.Input != "" {
		input := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Input), &input); err != nil {
			return nil, errors.Wrap(err, "Error when decode input")
		}
		watch.Input = input
	}

	if o.Spec.Condition != "" {
		condition := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Condition), &condition); err != nil {
			return nil, errors.Wrap(err, "Error when decode condition")
		}
		watch.Condition = condition
	}

	if o.Spec.Transform != "" {
		transform := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Transform), &transform); err != nil {
			return nil, errors.Wrap(err, "Error when decode transform")
		}
		watch.Transform = transform
	}

	if o.Spec.Actions != "" {
		actions := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Actions), &actions); err != nil {
			return nil, errors.Wrap(err, "Error when decode actions")
		}
		watch.Actions = actions
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, errors.Wrap(err, "Error when decode metadata")
		}
		watch.Metadata = meta
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
