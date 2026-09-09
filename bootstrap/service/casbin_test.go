package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	gormlogger "gorm.io/gorm/logger"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// setupTestLoggerForCasbin initializes a test logger for testing purposes
func setupTestLoggerForCasbin() {
	logger, _ := zap.NewDevelopment()
	resource.LoggerService = logger
}

// setupTestMySQLClient creates a mock MySQL client for testing
func setupTestMySQLClient(t *testing.T) error {
	db := sql.OpenDB(casbinTestConnector{})
	t.Cleanup(func() { _ = db.Close() })
	resource.MySQLClient = &gorm.DB{Config: &gorm.Config{ConnPool: db}}
	return nil
}

type casbinTestConnector struct{ driver.Connector }

func (casbinTestConnector) Connect(context.Context) (driver.Conn, error) {
	return casbinTestConn{}, nil
}
func (casbinTestConnector) Driver() driver.Driver { return nil }

type casbinTestConn struct{ driver.Conn }

func (casbinTestConn) Close() error               { return nil }
func (casbinTestConn) Ping(context.Context) error { return nil }

// createTestCasbinConfig creates a temporary Casbin configuration file for testing
func createTestCasbinConfig() (string, error) {
	configContent := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

	tmpFile, err := os.CreateTemp("", "casbin_test_*.conf")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// TestValidateCasbinDependencies tests the ValidateCasbinDependencies function with
// various scenarios of invalid and valid dependencies.
//
// The tests cover the following scenarios:
// - Nil MySQL client
// - Nil logger service
// - Missing config file
// - Valid dependencies with a temporary config file
//
// The function is expected to return an error if any of the dependencies are not
// initialized or if the config file is not found. Otherwise, it should return nil.
func TestValidateCasbinDependencies(t *testing.T) {
	setupTestLoggerForCasbin()

	tests := []struct {
		name        string
		setupFunc   func() error
		cleanupFunc func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil mysql client",
			setupFunc: func() error {
				resource.MySQLClient = nil
				return nil
			},
			expectError: true,
			errorMsg:    "mysql client is not initialized",
		},
		{
			name: "nil logger service",
			setupFunc: func() error {
				if err := setupTestMySQLClient(t); err != nil {
					return err
				}
				resource.LoggerService = nil
				return nil
			},
			cleanupFunc: func() {
				resource.MySQLClient = nil
			},
			expectError: true,
			errorMsg:    "logger service is not initialized",
		},
		{
			name: "missing config file",
			setupFunc: func() error {
				setupTestLoggerForCasbin()
				if err := setupTestMySQLClient(t); err != nil {
					return err
				}
				// Set non-existent config path
				os.Setenv("CASBIN_CONFIG_PATH", "/non/existent/path.conf")
				return nil
			},
			cleanupFunc: func() {
				resource.MySQLClient = nil
				os.Unsetenv("CASBIN_CONFIG_PATH")
			},
			expectError: true,
			errorMsg:    "casbin configuration file not found: /non/existent/path.conf",
		},
		{
			name: "valid dependencies",
			setupFunc: func() error {
				setupTestLoggerForCasbin()
				if err := setupTestMySQLClient(t); err != nil {
					return err
				}

				// Create temporary config file
				configPath, err := createTestCasbinConfig()
				if err != nil {
					return err
				}
				os.Setenv("CASBIN_CONFIG_PATH", configPath)
				return nil
			},
			cleanupFunc: func() {
				if configPath := os.Getenv("CASBIN_CONFIG_PATH"); configPath != "" {
					os.Remove(configPath)
					os.Unsetenv("CASBIN_CONFIG_PATH")
				}
				resource.MySQLClient = nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			err := validateCasbinDependencies(context.Background())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestGetCasbinConfigPath tests that the `getCasbinConfigPath` function returns the correct casbin config path.
//
// The test cases cover the following scenarios:
//
// 1. No environment variable set: the function should return the default config path.
// 2. Environment variable set to a custom path: the function should return the custom path.
func TestGetCasbinConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default path",
			envValue: "",
			expected: "./conf/service/casbin.conf",
		},
		{
			name:     "custom path from env",
			envValue: "/custom/path/casbin.conf",
			expected: "/custom/path/casbin.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv("CASBIN_CONFIG_PATH", tt.envValue)
			} else {
				os.Unsetenv("CASBIN_CONFIG_PATH")
			}

			// Cleanup
			defer func() {
				if tt.envValue != "" {
					os.Unsetenv("CASBIN_CONFIG_PATH")
				}
			}()

			result := getCasbinConfigPath()
			if result != tt.expected {
				t.Errorf("Expected path '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCloseCasbin_NoEnforcer tests the CloseCasbin function when no enforcer is set.
//
// The test ensures that no error is returned when the enforcer is not set.
func TestCloseCasbin_NoEnforcer(t *testing.T) {
	setupTestLoggerForCasbin()

	// Ensure no enforcer is set
	resource.Enforcer = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := CloseCasbin(ctx)
	if err != nil {
		t.Errorf("Expected no error when closing with no enforcer, got: %v", err)
	}
}

// TestInitCasbinEnforcer_WithValidSetup tests the InitEnforcer function with all dependencies
// properly set up.
//
// The test ensures that no panic is triggered when the function is called with a valid
// setup.
//
// This test requires a proper test environment setup, so it is skipped if the test
// can't create a valid test environment.
func TestInitCasbinEnforcer_WithValidSetup(t *testing.T) {
	setupTestLoggerForCasbin()

	// This test will only pass if all dependencies are properly set up
	// Skip if we can't create a proper test environment
	t.Skip("Skipping integration test - requires proper test environment setup")

	if err := setupTestMySQLClient(t); err != nil {
		t.Fatalf("Failed to setup test MySQL client: %v", err)
	}
	defer func() {
		resource.MySQLClient = nil
	}()

	// Create temporary config file
	configPath, err := createTestCasbinConfig()
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(configPath)

	os.Setenv("CASBIN_CONFIG_PATH", configPath)
	defer os.Unsetenv("CASBIN_CONFIG_PATH")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should not panic with valid setup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitEnforcer panicked: %v", r)
		}
	}()

	InitEnforcer(ctx)

	// Clean up
	if resource.Enforcer != nil {
		_ = CloseCasbin(ctx)
	}
}

// The fake driver exercises the actual GORM adapter without a live database.
type casbinInitConnector struct {
	hook func(context.Context, string) error
}

func (c casbinInitConnector) Connect(context.Context) (driver.Conn, error) {
	return &casbinInitConn{hook: c.hook}, nil
}
func (casbinInitConnector) Driver() driver.Driver { return nil }

type casbinInitConn struct {
	driver.Conn
	hook func(context.Context, string) error
}

func (*casbinInitConn) Close() error { return nil }
func (c *casbinInitConn) Ping(ctx context.Context) error {
	return c.hook(ctx, "ping")
}
func (c *casbinInitConn) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	var stage string
	var rows casbinInitRows
	switch {
	case query == "SELECT DATABASE()":
		stage = "database"
		rows = casbinInitRows{columns: []string{"database"}, values: [][]driver.Value{{"audit"}}}
	case strings.Contains(query, "information_schema.tables"):
		stage = "table"
		rows = casbinInitRows{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}
	case strings.Contains(query, "`casbin_rule`"):
		stage = "policy"
		rows = casbinInitRows{columns: []string{"id", "p_type", "v0", "v1", "v2", "v3", "v4", "v5"}}
	default:
		return nil, fmt.Errorf("unexpected SQL query: %s", query)
	}
	if err := c.hook(ctx, stage); err != nil {
		return nil, err
	}
	return &rows, nil
}
func (c *casbinInitConn) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	if err := c.hook(ctx, "write"); err != nil {
		return nil, err
	}
	return casbinInitResult{}, nil
}

type casbinInitResult struct{}

func (casbinInitResult) LastInsertId() (int64, error) { return 1, nil }
func (casbinInitResult) RowsAffected() (int64, error) { return 1, nil }

type casbinInitRows struct {
	columns []string
	values  [][]driver.Value
}

func (r *casbinInitRows) Columns() []string { return r.columns }
func (*casbinInitRows) Close() error        { return nil }
func (r *casbinInitRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}

func setupCasbinInitialization(t *testing.T, hook func(context.Context, string) error) *gorm.DB {
	t.Helper()
	regressionLogger(t)
	previousDB, previousEnforcer := resource.MySQLClient, resource.Enforcer
	t.Cleanup(func() { resource.MySQLClient, resource.Enforcer = previousDB, previousEnforcer })
	t.Setenv("CASBIN_CONFIG_PATH", "../../conf/service/casbin.conf")
	sqlDB := sql.OpenDB(casbinInitConnector{hook: hook})
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
		DisableAutomaticPing: true, SkipDefaultTransaction: true,
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	resource.MySQLClient, resource.Enforcer = db, nil
	return db
}

func TestCasbinInitializationAlreadyCanceled(t *testing.T) {
	previousDB, previousLogger := resource.MySQLClient, resource.LoggerService
	t.Cleanup(func() { resource.MySQLClient, resource.LoggerService = previousDB, previousLogger })
	resource.MySQLClient, resource.LoggerService = nil, nil
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := InitCasbinEnforcer(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("initialization error=%v, want context.Canceled", err)
	}
	if _, err := createCasbinAdapter(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("adapter error=%v, want context.Canceled", err)
	}
	if _, err := createCasbinEnforcer(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("enforcer error=%v, want context.Canceled", err)
	}
}

func TestCasbinInitializationWaitsForCanceledDatabaseWork(t *testing.T) {
	for _, phase := range []string{"ping", "table", "policy"} {
		t.Run(phase, func(t *testing.T) {
			entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			setupCasbinInitialization(t, func(ctx context.Context, stage string) error {
				if stage != phase {
					return ctx.Err()
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Error("database operation has no initialization deadline")
				}
				close(entered)
				select {
				case <-ctx.Done():
				case <-release:
					return errors.New("test interrupted database operation")
				}
				close(canceled)
				// Model a driver that needs time to finish after observing cancellation.
				<-release
				return ctx.Err()
			})
			ctx, cancel := context.WithCancel(context.Background())
			result, finished := make(chan error, 1), make(chan struct{})
			go func() {
				defer close(finished)
				result <- InitCasbinEnforcer(ctx)
			}()
			t.Cleanup(func() {
				cancel()
				unblock()
				select {
				case <-finished:
				case <-time.After(2 * time.Second):
					t.Error("initialization did not finish")
				}
			})
			select {
			case <-entered:
			case err := <-result:
				t.Fatalf("initialization returned before %s: %v", phase, err)
			case <-time.After(2 * time.Second):
				t.Fatal("database operation did not start")
			}
			cancel()
			select {
			case <-canceled:
			case <-time.After(2 * time.Second):
				t.Fatal("database operation did not receive cancellation")
			}
			select {
			case err := <-result:
				t.Fatalf("initialization returned with database work still active: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			unblock()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("initialization did not return after database work finished")
			}
			if resource.Enforcer != nil {
				t.Fatal("canceled initialization published an enforcer")
			}
			if err := CloseMySQLContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if resource.MySQLClient != nil {
				t.Fatal("database cleanup did not clear the shared client")
			}
		})
	}
}

func TestCasbinRuntimePoliciesSurviveInitializationContextCancellation(t *testing.T) {
	var policyCtx context.Context
	writes := 0
	client := setupCasbinInitialization(t, func(ctx context.Context, stage string) error {
		if stage == "policy" && policyCtx == nil {
			policyCtx = ctx
		}
		if stage == "write" {
			writes++
		}
		return ctx.Err()
	})
	originalContext := client.Statement.Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := InitCasbinEnforcer(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if policyCtx == nil || !errors.Is(policyCtx.Err(), context.Canceled) {
		t.Fatal("startup policy load did not use the canceled initialization context")
	}
	if client.Statement.Context != originalContext || originalContext.Err() != nil {
		t.Fatal("initialization changed the shared MySQL context")
	}
	if added, err := resource.Enforcer.AddPolicy("admin", "/audit", "GET"); err != nil || !added {
		t.Fatalf("runtime AutoSave failed: added=%v, err=%v", added, err)
	}
	if writes != 1 {
		t.Fatalf("runtime AutoSave issued %d writes, want 1", writes)
	}
	if err := resource.Enforcer.LoadPolicy(); err != nil {
		t.Fatalf("runtime policy reload failed: %v", err)
	}
}
