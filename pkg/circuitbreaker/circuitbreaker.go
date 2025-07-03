package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// State 熔断器状态
type State int

const (
	// StateClosed 关闭状态 - 正常工作
	StateClosed State = iota
	// StateOpen 开启状态 - 熔断开启，拒绝请求
	StateOpen
	// StateHalfOpen 半开状态 - 允许少量请求通过以测试服务是否恢复
	StateHalfOpen
)

// String 返回状态的字符串表示
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config 熔断器配置
type Config struct {
	Name          string                                  // 熔断器名称
	MaxRequests   uint32                                  // 半开状态下允许的最大请求数
	Interval      time.Duration                           // 统计窗口时间
	Timeout       time.Duration                           // 熔断器开启后的超时时间
	ReadyToTrip   func(counts Counts) bool                // 判断是否应该熔断的函数
	OnStateChange func(name string, from State, to State) // 状态变化回调
	IsSuccessful  func(err error) bool                    // 判断请求是否成功的函数
}

// Counts 统计信息
type Counts struct {
	Requests             uint32 // 总请求数
	TotalSuccesses       uint32 // 总成功数
	TotalFailures        uint32 // 总失败数
	ConsecutiveSuccesses uint32 // 连续成功数
	ConsecutiveFailures  uint32 // 连续失败数
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time

	logger *zap.Logger
}

// 熔断器指标
var (
	circuitBreakerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests handled by circuit breaker",
		},
		[]string{"name", "state", "result"},
	)

	circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of circuit breaker (0=CLOSED, 1=OPEN, 2=HALF_OPEN)",
		},
		[]string{"name"},
	)

	circuitBreakerFailureRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_failure_rate",
			Help: "Current failure rate of circuit breaker",
		},
		[]string{"name"},
	)
)

func init() {
	prometheus.MustRegister(circuitBreakerRequests, circuitBreakerState, circuitBreakerFailureRate)
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          cfg.Name,
		maxRequests:   cfg.MaxRequests,
		interval:      cfg.Interval,
		timeout:       cfg.Timeout,
		readyToTrip:   cfg.ReadyToTrip,
		isSuccessful:  cfg.IsSuccessful,
		onStateChange: cfg.OnStateChange,
		state:         StateClosed,
		expiry:        time.Now().Add(cfg.Interval),
	}

	// 设置默认值
	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval == 0 {
		cb.interval = 60 * time.Second
	}
	if cb.timeout == 0 {
		cb.timeout = 60 * time.Second
	}
	if cb.readyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	}
	if cb.isSuccessful == nil {
		cb.isSuccessful = defaultIsSuccessful
	}

	// 初始化指标
	circuitBreakerState.WithLabelValues(cb.name).Set(float64(cb.state))

	return cb
}

// SetLogger 设置日志记录器
func (cb *CircuitBreaker) SetLogger(logger *zap.Logger) {
	cb.logger = logger
}

// Execute 执行函数，如果熔断器开启则返回错误
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "rejected").Inc()
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, cb.isSuccessful(err))

	// 记录指标
	if err != nil {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "failure").Inc()
	} else {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "success").Inc()
	}

	return result, err
}

// ExecuteWithContext 带上下文的执行函数
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, req func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "rejected").Inc()
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req(ctx)
	cb.afterRequest(generation, cb.isSuccessful(err))

	// 记录指标
	if err != nil {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "failure").Inc()
	} else {
		circuitBreakerRequests.WithLabelValues(cb.name, cb.state.String(), "success").Inc()
	}

	return result, err
}

// State 返回当前状态
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts 返回当前统计信息
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, errors.New("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, errors.New("too many requests")
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}

	// 更新失败率指标
	if cb.counts.Requests > 0 {
		failureRate := float64(cb.counts.TotalFailures) / float64(cb.counts.Requests)
		circuitBreakerFailureRate.WithLabelValues(cb.name).Set(failureRate)
	}
}

// currentState 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.onSuccess()

	if state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
		cb.setState(StateClosed, now)
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.onFailure()

	switch state {
	case StateClosed:
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	if cb.logger != nil {
		cb.logger.Info("Circuit breaker state changed",
			zap.String("name", cb.name),
			zap.String("from", prev.String()),
			zap.String("to", state.String()),
		)
	}

	// 更新状态指标
	circuitBreakerState.WithLabelValues(cb.name).Set(float64(state))
}

// toNewGeneration 开始新的统计周期
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// onRequest 请求计数
func (c *Counts) onRequest() {
	c.Requests++
}

// onSuccess 成功计数
func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// onFailure 失败计数
func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// clear 清空计数
func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// 默认的熔断判断函数
func defaultReadyToTrip(counts Counts) bool {
	return counts.Requests >= 20 && counts.TotalFailures > counts.TotalSuccesses
}

// 默认的成功判断函数
func defaultIsSuccessful(err error) bool {
	return err == nil
}
