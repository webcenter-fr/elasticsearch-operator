package elasticsearch

import (
	"context"
	"errors"
	"testing"
	"time"

	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/es-handler/v9/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestES() *elasticsearchcrd.Elasticsearch {
	return &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{Name: "master", Roles: []string{"master"}},
				{Name: "data", Roles: []string{"data"}},
			},
		},
	}
}

func newTestState(es *elasticsearchcrd.Elasticsearch, handler eshandler.ElasticsearchHandler) *UpgradeState {
	return &UpgradeState{
		ES:        es,
		ESHandler: handler,
		Logger:    logrus.NewEntry(logrus.New()),
	}
}

func TestIsPredicateDisabled(t *testing.T) {
	t.Run("nil annotations", func(t *testing.T) {
		es := newTestES()
		assert.False(t, IsPredicateDisabled(es, "cluster_health_not_red"))
	})

	t.Run("wildcard disables all", func(t *testing.T) {
		es := newTestES()
		es.Annotations = map[string]string{
			elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation: "*",
		}
		assert.True(t, IsPredicateDisabled(es, "cluster_health_not_red"))
		assert.True(t, IsPredicateDisabled(es, "one_master_at_a_time"))
		assert.True(t, IsPredicateDisabled(es, "anything"))
	})

	t.Run("specific name", func(t *testing.T) {
		es := newTestES()
		es.Annotations = map[string]string{
			elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation: "cluster_health_not_red",
		}
		assert.True(t, IsPredicateDisabled(es, "cluster_health_not_red"))
		assert.False(t, IsPredicateDisabled(es, "one_master_at_a_time"))
	})

	t.Run("comma-separated with spaces", func(t *testing.T) {
		es := newTestES()
		es.Annotations = map[string]string{
			elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation: " cluster_health_not_red , one_master_at_a_time ",
		}
		assert.True(t, IsPredicateDisabled(es, "cluster_health_not_red"))
		assert.True(t, IsPredicateDisabled(es, "one_master_at_a_time"))
		assert.False(t, IsPredicateDisabled(es, "skip_terminating_pods"))
	})

	t.Run("empty annotation value", func(t *testing.T) {
		es := newTestES()
		es.Annotations = map[string]string{
			elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation: "",
		}
		assert.False(t, IsPredicateDisabled(es, "cluster_health_not_red"))
	})
}

func TestApplyPredicatesClusterHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("red health blocks upgrade", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "red"}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "cluster_health_not_red")
		assert.Contains(t, reason, "RED")
	})

	t.Run("green health passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})
}

func TestApplyPredicatesRequireStartedReplica(t *testing.T) {
	ctx := context.Background()

	t.Run("no started replica blocks upgrade", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{
			"test-data-es-0": {{Index: "logs", Shard: "0", Primary: true}},
		}, nil)
		mockES.EXPECT().CountStartedReplicas(gomock.Any(), "logs", "0", "test-data-es-0").Return(1, 0, nil)
		mockES.EXPECT().CountStartedReplicas(gomock.Any(), "logs", "0", "").Return(1, 0, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "no STARTED replica")
	})

	t.Run("started replica passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{
			"test-data-es-0": {{Index: "logs", Shard: "0", Primary: true}},
		}, nil)
		mockES.EXPECT().CountStartedReplicas(gomock.Any(), "logs", "0", "test-data-es-0").Return(1, 1, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})

	t.Run("empty shards map passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})
}

func TestApplyPredicatesOneMasterAtATime(t *testing.T) {
	ctx := context.Background()

	t.Run("master pod blocked when another master is terminating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		state.DeletedPods = []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "test-master-es-1"}},
		}
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-master-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "another master")
	})

	t.Run("data pod passes even when a master is terminating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		state.DeletedPods = []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "test-master-es-1"}},
		}
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})

	t.Run("master pod passes when no other master is terminating", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-master-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})
}

func TestApplyPredicatesSkipTerminating(t *testing.T) {
	ctx := context.Background()

	t.Run("terminating pod is blocked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		now := metav1.Time{Time: time.Now()}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-data-es-0",
				DeletionTimestamp: &now,
			},
		}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "terminating")
	})
}

func TestApplyPredicatesDisabled(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled cluster_health_not_red predicate is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "red"}, nil).AnyTimes()
		mockES.EXPECT().GetShardsByNode(gomock.Any()).Return(map[string][]eshandler.ShardInfo{}, nil)

		es := newTestES()
		es.Annotations = map[string]string{
			elasticsearchcrd.ElasticsearchDisableUpgradePredicatesAnnotation: "cluster_health_not_red",
		}
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, reason, err := ApplyPredicates(ctx, pod, state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})
}

func TestApplyPredicatesError(t *testing.T) {
	ctx := context.Background()

	t.Run("cluster health error is propagated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(nil, errors.New("connection refused"))

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, _, err := ApplyPredicates(ctx, pod, state)
		assert.False(t, ok)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cluster_health_not_red")
	})

	t.Run("nil cluster health is propagated as error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		mockES.EXPECT().ClusterHealth().Return(nil, nil)

		es := newTestES()
		state := newTestState(es, mockES)
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-data-es-0"}}

		ok, _, err := ApplyPredicates(ctx, pod, state)
		assert.False(t, ok)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cluster_health_not_red")
	})
}
