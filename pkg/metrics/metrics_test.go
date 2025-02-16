package metrics

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()
	assert.NotNil(t, collector)
}

func TestGlobalCollector(t *testing.T) {
	collector := GlobalCollector()
	assert.NotNil(t, collector)
}

func TestCounter(t *testing.T) {
	c := NewCounter("test_counter")

	assert.Equal(t, uint64(0), c.Value())
	// 测试 Inc()
	c.Inc()
	assert.Equal(t, uint64(1), c.Value())

	// 测试 Add()
	c.Add(5)
	assert.Equal(t, uint64(6), c.Value())

	// 测试 Reset()
	c.Reset()
	assert.Equal(t, uint64(0), c.Value())
}

// 测试 Gauge 类型
func TestGauge(t *testing.T) {
	g := NewGauge("test_gauge")

	assert.Equal(t, int64(0), g.Value())

	// 测试 Set()
	g.Set(10)
	assert.Equal(t, int64(10), g.Value())

	// 测试 Add()
	g.Add(5)
	assert.Equal(t, int64(15), g.Value())

	// 测试 Sub()
	g.Sub(3)
	assert.Equal(t, int64(12), g.Value())

	// 测试 Set()
	g.Set(10)
	assert.Equal(t, int64(10), g.Value())
}

// 测试 Histogram 类型
func TestHistogram(t *testing.T) {
	buckets := []float64{0.5, 1.0, 5.0, 3.0}
	h := NewHistogram("test_histogram", buckets)

	// 测试空直方图
	assert.Equal(t, uint64(0), h.Count())
	assert.Equal(t, float64(0), h.Sum())
	assert.Equal(t, 4, len(h.Buckets()))

	// 记录测试数据
	testValues := []float64{0.3, 0.6, 2.0, 10.0}
	expectedSum := 0.3 + 0.6 + 2.0 + 10.0

	for _, v := range testValues {
		h.Observe(v)
	}
	assert.Equal(t, uint64(len(testValues)), h.Count())
	assert.InEpsilon(t, expectedSum, h.Sum(), 1e-6, "sum should be close to expectedSum")

	// 验证分桶
	counts := h.Value().([]float64)
	expectedBuckets := map[float64]float64{
		0.5: 1, // <=0.5 的样本数
		1.0: 2, // <=1.0 的样本数
		3.0: 3, // <=3.0 的样本数
		5.0: 3, // <=5.0 的样本数
	}
	hbuckets := h.Buckets()
	for i, bucket := range hbuckets {
		assert.Equal(t, expectedBuckets[bucket], counts[i],
			"bucket %.1f counts should match", bucket)
	}
}

func TestSummary(t *testing.T) {
	quantiles := map[float64]float64{
		0.5: 0.05,
		0.9: 0.01,
	}
	s := NewSummary("test_summary", quantiles)

	// 记录测试数据
	testValues := []float64{5, 3, 1, 4, 2}
	for _, v := range testValues {
		s.Observe(v)
	}

	// 验证基础统计
	assert.Equal(t, uint64(len(testValues)), s.Count(), "样本数不一致")
	assert.InEpsilon(t, 15.0, s.Sum(), 1e-6, "总和不匹配")

	// 验证分位数
	expectedQuantiles := map[float64]float64{
		0.5: 3.0,
		0.9: 5.0,
	}

	quantileResults := s.Quantiles()
	for q, expected := range expectedQuantiles {
		assert.InEpsilon(t, expected, quantileResults[q], 1e-6,
			"分位数 %.1f 不匹配", q)
	}

	// 测试 Reset()
	s.Reset()
	assert.Equal(t, uint64(0), s.Count(), "Reset() 后样本数应为 0")
	assert.Equal(t, 0.0, s.Sum(), "Reset() 后总和应为 0.0")
}

func TestConcurrency(t *testing.T) {
	// Counter 并发测试
	c := NewCounter("concurrency_counter")
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()
	assert.Equal(t, uint64(1000), c.Value(), "并发 Inc() 后值不匹配")

	// Histogram 并发测试
	h := NewHistogram("concurrency_histogram", []float64{10})
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(v float64) {
			defer wg.Done()
			h.Observe(v)
		}(float64(i))
	}
	wg.Wait()
	assert.Equal(t, uint64(1000), h.Count(), "并发 Observe() 后样本数不匹配")
}

func TestCollector(t *testing.T) {
	collector := NewCollector()

	// 注册指标
	c := NewCounter("collector_counter")
	g := NewGauge("collector_gauge")
	collector.Register(c)
	collector.Register(g)

	// 验证指标获取
	assert.NotNil(t, collector.Get("collector_counter"), "应能获取已注册的计数器")
	assert.Nil(t, collector.Get("not_exist"), "不存在的指标应返回 nil")

	// 验证指标数量
	assert.Len(t, collector.Metrics(), 2, "收集器应包含 2 个指标")
}
