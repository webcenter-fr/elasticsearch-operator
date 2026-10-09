package elasticsearch

import (
	"context"
	"errors"
	"testing"

	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/es-handler/v9/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSupportsNodeShutdown(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"7.15.1", false},
		{"7.15.2", true},
		{"7.15.2-SNAPSHOT", true},
		{"7.16.0", true},
		{"7.17.9", true},
		{"8.0.0", true},
		{"8.11.0-alpha1", true},
		{"9.0.0", true},
		{"7.14.0", false},
		{"6.8.0", false},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.expected, supportsNodeShutdown(tt.version))
		})
	}
}

func TestRollingRestartOrchestratorAccessors(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockES := mocks.NewMockElasticsearchHandler(ctrl)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	logger := logrus.NewEntry(logrus.New())

	orch := &RollingRestartOrchestrator{
		esHandler:    mockES,
		k8sClient:    k8sClient,
		logger:       logger,
		strategy:     StrategyNodeShutdown,
		esVersion:    "8.11.0",
		nodeNameToID: map[string]string{"pod-0": "n1"},
	}

	assert.Equal(t, StrategyNodeShutdown, orch.Strategy())
	assert.Equal(t, "8.11.0", orch.ESVersion())
}

func TestPrepareForRollingRestart(t *testing.T) {
	ctx := context.Background()

	t.Run("allocation-filter disables allocation and flushes", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		mockES.EXPECT().DisableReplicaShardsAllocationTransient().Return(nil)
		mockES.EXPECT().DisableRoutingRebalanceTransient().Return(nil)
		mockES.EXPECT().Flush(ctx).Return(nil)

		err := orch.PrepareForRollingRestart(ctx)
		assert.NoError(t, err)
	})

	t.Run("allocation-filter flush error is non-blocking", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		mockES.EXPECT().DisableReplicaShardsAllocationTransient().Return(nil)
		mockES.EXPECT().DisableRoutingRebalanceTransient().Return(nil)
		mockES.EXPECT().Flush(ctx).Return(errors.New("flush failed"))

		err := orch.PrepareForRollingRestart(ctx)
		assert.NoError(t, err)
	})

	t.Run("allocation-filter propagates disable replica error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		mockES.EXPECT().DisableReplicaShardsAllocationTransient().Return(errors.New("disable failed"))

		err := orch.PrepareForRollingRestart(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to disable replica allocation")
	})

	t.Run("node-shutdown makes no ES calls", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyNodeShutdown,
			esVersion: "8.11.0",
		}

		err := orch.PrepareForRollingRestart(ctx)
		assert.NoError(t, err)
	})
}

func TestCompleteRollingRestart(t *testing.T) {
	ctx := context.Background()

	t.Run("allocation-filter re-enables allocation", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		mockES.EXPECT().EnableShardAllocationTransient().Return(nil)
		mockES.EXPECT().EnableRoutingRebalanceTransient().Return(nil)

		err := orch.CompleteRollingRestart(ctx)
		assert.NoError(t, err)
	})

	t.Run("node-shutdown cleans up completed restart shutdowns only", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyNodeShutdown,
			esVersion: "8.11.0",
		}

		mockES.EXPECT().GetAllNodeShutdowns(ctx).Return([]eshandler.NodeShutdownInfo{
			{NodeID: "n1", Type: "restart", Status: eshandler.ShutdownComplete},
			{NodeID: "n2", Type: "restart", Status: eshandler.ShutdownInProgress},
			{NodeID: "n3", Type: "remove", Status: eshandler.ShutdownComplete},
		}, nil)
		mockES.EXPECT().DeleteNodeShutdown(ctx, "n1").Return(nil)

		err := orch.CompleteRollingRestart(ctx)
		assert.NoError(t, err)
	})
}

func TestRequestNodeShutdown(t *testing.T) {
	ctx := context.Background()

	t.Run("node-shutdown requests shutdown for known pod", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		mockES.EXPECT().PutNodeShutdown(ctx, "n1", eshandler.ShutdownTypeRestart, "rolling-restart").Return(nil)

		err := orch.RequestNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
	})

	t.Run("node-shutdown errors for unknown pod", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		err := orch.RequestNodeShutdown(ctx, "pod-unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node ID not found")
	})

	t.Run("allocation-filter is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		err := orch.RequestNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
	})
}

func TestWaitForNodeShutdown(t *testing.T) {
	ctx := context.Background()

	t.Run("nil shutdown info means complete", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		mockES.EXPECT().GetNodeShutdown(ctx, "n1").Return(nil, nil)

		complete, err := orch.WaitForNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
		assert.True(t, complete)
	})

	t.Run("COMPLETE status means complete", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		mockES.EXPECT().GetNodeShutdown(ctx, "n1").Return(&eshandler.NodeShutdownInfo{
			NodeID: "n1",
			Status: eshandler.ShutdownComplete,
		}, nil)

		complete, err := orch.WaitForNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
		assert.True(t, complete)
	})

	t.Run("IN_PROGRESS status means not complete", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		mockES.EXPECT().GetNodeShutdown(ctx, "n1").Return(&eshandler.NodeShutdownInfo{
			NodeID: "n1",
			Status: eshandler.ShutdownInProgress,
		}, nil)

		complete, err := orch.WaitForNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
		assert.False(t, complete)
	})

	t.Run("unknown pod returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		complete, err := orch.WaitForNodeShutdown(ctx, "pod-unknown")
		assert.Error(t, err)
		assert.False(t, complete)
	})

	t.Run("allocation-filter returns true immediately", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		complete, err := orch.WaitForNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
		assert.True(t, complete)
	})
}

func TestClearNodeShutdown(t *testing.T) {
	ctx := context.Background()

	t.Run("known pod deletes shutdown", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		mockES.EXPECT().DeleteNodeShutdown(ctx, "n1").Return(nil)

		err := orch.ClearNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
	})

	t.Run("unknown pod is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler:    mockES,
			k8sClient:    k8sClient,
			logger:       logger,
			strategy:     StrategyNodeShutdown,
			esVersion:    "8.11.0",
			nodeNameToID: map[string]string{"pod-0": "n1"},
		}

		err := orch.ClearNodeShutdown(ctx, "pod-unknown")
		assert.NoError(t, err)
	})

	t.Run("allocation-filter is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		logger := logrus.NewEntry(logrus.New())

		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyAllocationFilter,
			esVersion: "7.14.0",
		}

		err := orch.ClearNodeShutdown(ctx, "pod-0")
		assert.NoError(t, err)
	})
}

func TestCheckPredicates(t *testing.T) {
	ctx := context.Background()

	newOrchestrator := func(ctrl *gomock.Controller, objs ...client.Object) (*RollingRestartOrchestrator, *mocks.MockElasticsearchHandler) {
		mockES := mocks.NewMockElasticsearchHandler(ctrl)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objs...).Build()
		logger := logrus.NewEntry(logrus.New())
		orch := &RollingRestartOrchestrator{
			esHandler: mockES,
			k8sClient: k8sClient,
			logger:    logger,
			strategy:  StrategyNodeShutdown,
			esVersion: "8.11.0",
		}
		return orch, mockES
	}

	newStsAndPod := func() (*appv1.StatefulSet, *corev1.Pod) {
		sts := &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "es",
				Namespace: "default",
			},
			Spec: appv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "es"},
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "es-0",
				Namespace: "default",
				Labels:    map[string]string{"app": "es"},
			},
		}
		return sts, pod
	}

	t.Run("red cluster health blocks restart", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		orch, mockES := newOrchestrator(ctrl)

		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "red"}, nil)

		ok, reason, err := orch.CheckPredicates(ctx, &appv1.StatefulSetList{})
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "RED")
	})

	t.Run("green health and empty sts list passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		orch, mockES := newOrchestrator(ctrl)

		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(ctx).Return(map[string][]eshandler.ShardInfo{}, nil)

		ok, reason, err := orch.CheckPredicates(ctx, &appv1.StatefulSetList{})
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})

	t.Run("green health with started replica passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		sts, pod := newStsAndPod()
		orch, mockES := newOrchestrator(ctrl, sts, pod)

		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(ctx).Return(map[string][]eshandler.ShardInfo{
			"es-0": {
				{Index: "my-index", Shard: "0", Primary: true, State: "STARTED", NodeName: "es-0"},
			},
		}, nil)
		mockES.EXPECT().CountStartedReplicas(ctx, "my-index", "0", "es-0").Return(1, 1, nil)

		stsList := &appv1.StatefulSetList{Items: []appv1.StatefulSet{*sts}}

		ok, reason, err := orch.CheckPredicates(ctx, stsList)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})

	t.Run("no started replica with replicas configured blocks restart", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		sts, pod := newStsAndPod()
		orch, mockES := newOrchestrator(ctrl, sts, pod)

		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(ctx).Return(map[string][]eshandler.ShardInfo{
			"es-0": {
				{Index: "my-index", Shard: "0", Primary: true, State: "STARTED", NodeName: "es-0"},
			},
		}, nil)
		mockES.EXPECT().CountStartedReplicas(ctx, "my-index", "0", "es-0").Return(1, 0, nil)
		mockES.EXPECT().CountStartedReplicas(ctx, "my-index", "0", "").Return(1, 0, nil)

		stsList := &appv1.StatefulSetList{Items: []appv1.StatefulSet{*sts}}

		ok, reason, err := orch.CheckPredicates(ctx, stsList)
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Contains(t, reason, "no STARTED replica")
	})

	t.Run("no replicas at all passes", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		sts, pod := newStsAndPod()
		orch, mockES := newOrchestrator(ctrl, sts, pod)

		mockES.EXPECT().ClusterHealth().Return(&esapi.ClusterHealthResponse{Status: "green"}, nil)
		mockES.EXPECT().GetShardsByNode(ctx).Return(map[string][]eshandler.ShardInfo{
			"es-0": {
				{Index: "my-index", Shard: "0", Primary: true, State: "STARTED", NodeName: "es-0"},
			},
		}, nil)
		mockES.EXPECT().CountStartedReplicas(ctx, "my-index", "0", "es-0").Return(0, 0, nil)
		mockES.EXPECT().CountStartedReplicas(ctx, "my-index", "0", "").Return(0, 0, nil)

		stsList := &appv1.StatefulSetList{Items: []appv1.StatefulSet{*sts}}

		ok, reason, err := orch.CheckPredicates(ctx, stsList)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, reason)
	})
}
