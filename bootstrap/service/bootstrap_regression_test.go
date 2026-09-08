package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"

	"database/sql"
	"database/sql/driver"
	"github.com/BurntSushi/toml"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5"
	manticore "github.com/manticoresoftware/manticoresearch-go"
	"github.com/nsqio/go-nsq"
	"github.com/redis/go-redis/v9"
	"github.com/taosdata/driver-go/v3/taosWS"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func regressionLogger(t *testing.T) {
	old := resource.LoggerService
	resource.LoggerService = zap.NewNop()
	t.Cleanup(func() { resource.LoggerService = old })
}

func TestDirectoryProbePreservesExistingFiles(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprint(symlink), func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, ".write_test")
			if symlink {
				target = filepath.Join(dir, "application-data")
				if err := os.Symlink(target, filepath.Join(dir, ".write_test")); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(target, []byte("keep me"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := ValidateDirectoryPermissions(dir); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "keep me" {
				t.Fatalf("probe changed existing file: %q, %v", data, err)
			}
			matches, _ := filepath.Glob(filepath.Join(dir, ".write_test-*"))
			if len(matches) != 0 {
				t.Fatalf("temporary files leaked: %v", matches)
			}
		})
	}
}

type policyWriteTrap struct {
	persist.Adapter
	writes int
}

func (a *policyWriteTrap) SavePolicy(model.Model) error {
	a.writes++
	return errors.New("unexpected save")
}
func (a *policyWriteTrap) AddPolicy(string, string, []string) error {
	a.writes++
	return errors.New("unexpected add")
}
func (a *policyWriteTrap) RemovePolicy(string, string, []string) error {
	a.writes++
	return errors.New("unexpected remove")
}

func TestCasbinValidationAndCloseNeverWritePolicies(t *testing.T) {
	regressionLogger(t)
	m, err := model.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act\n")
	if err != nil {
		t.Fatal(err)
	}
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.AddPolicy("test_user", "/test/resource", "GET"); err != nil {
		t.Fatal(err)
	}
	adapter := &policyWriteTrap{}
	e.SetAdapter(adapter)
	e.EnableAutoSave(true)
	if err := validateCasbinEnforcer(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(e.GetPolicy()) != 1 {
		t.Fatal("validation removed the existing policy")
	}
	old := resource.Enforcer
	resource.Enforcer = e
	t.Cleanup(func() { resource.Enforcer = old })
	if err := CloseCasbin(context.Background()); err != nil {
		t.Fatal(err)
	}
	if adapter.writes != 0 {
		t.Fatalf("unexpected policy writes: %d", adapter.writes)
	}
}

type readOnlyRedisHook struct {
	commands []string
	err      error
}

func (h *readOnlyRedisHook) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("network forbidden") }
}
func (h *readOnlyRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *readOnlyRedisHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.commands = append(h.commands, cmd.Name())
		if cmd.Name() != "ping" {
			return fmt.Errorf("unexpected command: %s", cmd.Name())
		}
		cmd.SetErr(h.err)
		return h.err
	}
}
func TestRedisHealthCheckOnlyPings(t *testing.T) {
	for _, failed := range []bool{false, true} {
		hook := &readOnlyRedisHook{}
		if failed {
			hook.err = errors.New("authentication failed")
		}
		client := redis.NewClient(&redis.Options{Addr: "unused.invalid:0", MaxRetries: -1})
		defer client.Close()
		client.AddHook(hook)
		err := testRedisConnection(context.Background(), client)
		if (err != nil) != failed {
			t.Fatalf("unexpected health result: %v", err)
		}
		if strings.Join(hook.commands, ",") != "ping" {
			t.Fatalf("health check commands: %v", hook.commands)
		}
	}
}

type regressionTransport func(*http.Request) (*http.Response, error)

func (f regressionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestManticoreAuthenticatesAndRejectsUnhealthyResponses(t *testing.T) {
	regressionLogger(t)
	old := config.ManticoreConfig
	t.Cleanup(func() { config.ManticoreConfig = old })
	config.ManticoreConfig = &config.ManticoreConfigEntry{}
	cfg := &config.ManticoreConfig.Manticore
	cfg.Endpoints, cfg.Port = []string{"https://example.invalid:9308"}, 9308
	cfg.UserName, cfg.PassWord = "alice", "complex:p@ss word"
	client, err := createManticoreClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		status int
		body   string
		ok     bool
	}{
		{401, `{"error":"unauthorized"}`, false},
		{500, `{"error":"server error"}`, false},
		{200, `[{"error":"SQL failed"}]`, false},
		{200, `[{"error":{"message":"SQL failed"}}]`, false},
		{200, `[{"data":[{"1":1}],"total":1,"error":""}]`, true},
	} {
		t.Run(fmt.Sprintf("%d-%v", tc.status, tc.ok), func(t *testing.T) {
			called := false
			client.GetConfig().HTTPClient.Transport = regressionTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				user, password, ok := r.BasicAuth()
				if !ok || user != cfg.UserName || password != cfg.PassWord || r.URL.Scheme != "https" {
					t.Error("invalid TLS/basic authentication")
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != "SELECT 1" || r.URL.Path != "/sql" {
					t.Errorf("unexpected probe: %s %s", r.URL, body)
				}
				return &http.Response{StatusCode: tc.status, Status: fmt.Sprintf("%d %s", tc.status, http.StatusText(tc.status)), Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			})
			err := testManticoreConnection(context.Background(), client)
			if !called || (err == nil) != tc.ok {
				t.Fatalf("health result: %v (called=%v)", err, called)
			}
		})
	}
	cfg.Endpoints = []string{"example.invalid"}
	if _, err := createManticoreClient(context.Background()); err == nil {
		t.Fatal("accepted credentials over plain HTTP")
	}
}

func TestCronValidationHonorsAutoStart(t *testing.T) {
	regressionLogger(t)
	oldCfg, oldScheduler := config.CronConfig, resource.Corn
	t.Cleanup(func() { config.CronConfig, resource.Corn = oldCfg, oldScheduler })
	for _, autoStart := range []bool{false, true} {
		config.CronConfig = &config.CronConfigEntry{}
		config.CronConfig.Cron.TimeZone, config.CronConfig.Cron.AutoStart = "UTC", autoStart
		if err := InitCronScheduler(context.Background()); err != nil {
			t.Fatal(err)
		}
		ran := make(chan struct{}, 1)
		_, err := resource.Corn.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), gocron.NewTask(func() { ran <- struct{}{} }))
		if err != nil {
			t.Fatal(err)
		}
		if !autoStart {
			select {
			case <-ran:
				t.Fatal("scheduler started despite AutoStart=false")
			case <-time.After(30 * time.Millisecond):
			}
			resource.Corn.Start()
		}
		select {
		case <-ran:
		case <-time.After(time.Second):
			t.Fatal("started scheduler did not execute job")
		}
		if err := CloseCron(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostgresqlDSNPreservesLiteralValues(t *testing.T) {
	for _, password := range []string{"two words", "a'b\\c", "x sslmode=disable host=attacker", "@:/?#%+"} {
		cfg := &config.PostgresqlConfigEntry{}
		cfg.Postgresql.Host, cfg.Postgresql.Port = "localhost", 5432
		cfg.Postgresql.User, cfg.Postgresql.Password = "user name", password
		cfg.Postgresql.DBName, cfg.Postgresql.SSLMode = "app db", "require"
		parsed, err := pgx.ParseConfig(buildPostgresqlDSN(cfg))
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Password != password || parsed.User != cfg.Postgresql.User || parsed.Database != cfg.Postgresql.DBName || parsed.Host != "localhost" || parsed.TLSConfig == nil {
			t.Fatal("DSN values changed while parsing")
		}
	}
}

func TestNSQStopWaitsForCompletion(t *testing.T) {
	stopped, requested := make(chan int), make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := stopNSQConsumer(ctx, func() { close(requested) }, stopped)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("returned before completion: %v", err)
	}
	<-requested
	close(stopped)
	if err := stopNSQConsumer(context.Background(), func() {}, stopped); err != nil {
		t.Fatal(err)
	}
}

func TestUnconnectedNSQConsumerCanClose(t *testing.T) {
	regressionLogger(t)
	oldCfg, oldConsumer, oldHandler := config.NsqConfig, resource.NsqConsumer, nsqMessageHandler
	t.Cleanup(func() { config.NsqConfig, resource.NsqConsumer, nsqMessageHandler = oldCfg, oldConsumer, oldHandler })
	setupTestNSQConfig()
	if err := InitConsumer(context.Background()); err != nil {
		t.Fatal(err)
	}
	consumer := resource.NsqConsumer
	consumer.SetLogger(nil, nsq.LogLevelError)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := CloseNsq(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-consumer.StopChan:
	default:
		t.Fatal("consumer still running")
	}
}

func TestNSQDrainWaitsForApplicationHandler(t *testing.T) {
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	h := &drainingNSQHandler{handler: nsq.HandlerFunc(func(*nsq.Message) error { close(started); <-release; return nil })}
	go func() { _ = h.HandleMessage(nil); close(finished) }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := h.drain(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain returned with active handler: %v", err)
	}
	if err := h.HandleMessage(nil); err == nil {
		t.Fatal("accepted new work during shutdown")
	}
	close(release)
	<-finished
	if err := h.drain(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type blockedSyncer struct {
	entered, release, finished chan struct{}
	err                        error
}

func (s *blockedSyncer) Write(p []byte) (int, error) { return len(p), nil }
func (s *blockedSyncer) Sync() error {
	close(s.entered)
	<-s.release
	defer close(s.finished)
	return s.err
}
func TestLoggerCloseRetainsHandleOnTimeoutAndReturnsErrors(t *testing.T) {
	regressionLogger(t)
	s := &blockedSyncer{make(chan struct{}), make(chan struct{}), make(chan struct{}), nil}
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), s, zap.InfoLevel))
	resource.LoggerService = logger
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := CloseLogger(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected result: %v", err)
	}
	if resource.LoggerService != logger {
		t.Fatal("cleared active logger")
	}
	close(s.release)
	<-s.finished
	failure := errors.New("disk flush failed")
	s = &blockedSyncer{make(chan struct{}), make(chan struct{}), make(chan struct{}), failure}
	close(s.release)
	resource.LoggerService = zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), s, zap.InfoLevel))
	if err := CloseLogger(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("lost sync failure: %v", err)
	}
}

type blockedIdleTransport struct {
	regressionTransport
	entered, release, finished chan struct{}
}

func (t *blockedIdleTransport) CloseIdleConnections() {
	close(t.entered)
	<-t.release
	close(t.finished)
}
func TestManticoreCloseTimeoutDoesNotTouchGlobalFromWorker(t *testing.T) {
	old := resource.ManticoreClient
	t.Cleanup(func() { resource.ManticoreClient = old })
	transport := &blockedIdleTransport{entered: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	cfg := manticore.NewConfiguration()
	cfg.HTTPClient = &http.Client{Transport: transport}
	client := manticore.NewAPIClient(cfg)
	resource.ManticoreClient = client
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := CloseManticore(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected close: %v", err)
	}
	if resource.ManticoreClient != client {
		t.Fatal("cleared unfinished client")
	}
	resource.ManticoreClient = nil
	close(transport.release)
	<-transport.finished
	if resource.ManticoreClient != nil {
		t.Fatal("late close changed global state")
	}
	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	resource.ManticoreClient = client
	if err := CloseManticore(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected canceled close: %v", err)
	}
}

func TestHealthWorkerStopJoinsBeforeReturning(t *testing.T) {
	var worker healthWorker
	entered, release := make(chan struct{}), make(chan struct{})
	worker.start(context.Background(), func(ctx context.Context) { close(entered); <-ctx.Done(); <-release })
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := worker.stop(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected stop: %v", err)
	}
	close(release)
	if err := worker.stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTDengineDriverAndConfiguration(t *testing.T) {
	var cfg config.TDengineConfigEntry
	if _, err := toml.DecodeFile("../../conf/service/tdengine.toml", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.TDengine.ConnectTimeout != 10*time.Second || cfg.TDengine.ConnMaxLifetime != time.Hour || cfg.TDengine.Port != 6041 {
		t.Fatal("incorrect adapter port or duration units")
	}
	cfg.TDengine.PassWord = "p@ss:/?#%+ word"
	parsed, err := taosWS.ParseDSN(buildTDengineDSN(&cfg))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Passwd != cfg.TDengine.PassWord || parsed.Net != "ws" || parsed.ReadTimeout != 30*time.Second {
		t.Fatal("incorrect websocket parameters")
	}
	db, err := sql.Open("taosWS", buildTDengineDSN(&cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
}

type lateConnector struct {
	driver.Connector
	release chan struct{}
	conn    driver.Conn
}

func (c *lateConnector) Connect(context.Context) (driver.Conn, error) {
	<-c.release
	return c.conn, nil
}

type closeTrackedConn struct {
	driver.Conn
	closed chan struct{}
}

func (c *closeTrackedConn) Close() error { close(c.closed); return nil }
func TestTDengineConnectTimeoutClosesLateConnection(t *testing.T) {
	conn := &closeTrackedConn{closed: make(chan struct{})}
	upstream := &lateConnector{release: make(chan struct{}), conn: conn}
	connector := &tdengineConnector{Connector: upstream, timeout: 20 * time.Millisecond}
	if _, err := connector.Connect(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected connect result: %v", err)
	}
	close(upstream.release)
	select {
	case <-conn.closed:
	case <-time.After(time.Second):
		t.Fatal("late connection leaked")
	}
}
