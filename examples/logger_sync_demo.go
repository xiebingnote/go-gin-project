package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xiebingnote/go-gin-project/bootstrap/service"
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"go.uber.org/zap"
)

// setupDemoConfig initializes configuration for the demo
func setupDemoConfig() {
	config.LogConfig = &config.LogConfigEntry{
		Log: struct {
			DefaultLevel string `toml:"DefaultLevel"`
			LogDir       string `toml:"LogDir"`
			LogFileDebug string `toml:"LogFileDebug"`
			LogFileInfo  string `toml:"LogFileInfo"`
			LogFileWarn  string `toml:"LogFileWarn"`
			LogFileError string `toml:"LogFileError"`
			MaxSize      int    `toml:"MaxSize"`
			MaxAge       int    `toml:"MaxAge"`
			MaxBackups   int    `toml:"MaxBackups"`
			LocalTime    bool   `toml:"LocalTime"`
			Compress     bool   `toml:"Compress"`
		}{
			DefaultLevel: "info",
			LogDir:       "./demo_logs",
			LogFileDebug: "debug.log",
			LogFileInfo:  "info.log",
			LogFileWarn:  "warn.log",
			LogFileError: "error.log",
			MaxSize:      100,
			MaxAge:       30,
			MaxBackups:   10,
			LocalTime:    true,
			Compress:     false,
		},
	}

	config.ServerConfig = &config.ServerConfigEntry{
		Version: struct {
			Version string `toml:"Version"`
		}{
			Version: "1.0.0-demo",
		},
	}
}

// cleanupDemoLogs removes demo log directory
func cleanupDemoLogs() {
	if config.LogConfig != nil {
		os.RemoveAll(config.LogConfig.Log.LogDir)
	}
}

func main() {
	fmt.Println("=== Logger Sync 修复演示 ===")
	fmt.Println("这个演示展示了修复后的logger如何优雅地处理同步错误")
	fmt.Println()

	// Setup configuration
	setupDemoConfig()
	defer cleanupDemoLogs()

	ctx := context.Background()

	// Initialize logger
	fmt.Println("1. 初始化logger服务...")
	err := service.InitLoggerService(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	fmt.Println("   ✓ Logger初始化成功")

	// Log some messages
	fmt.Println("\n2. 记录一些日志消息...")
	resource.LoggerService.Info("应用程序启动", zap.String("status", "starting"))
	resource.LoggerService.Info("配置加载完成", zap.String("config_file", "demo.toml"))
	resource.LoggerService.Warn("这是一个警告消息", zap.String("component", "demo"))
	resource.LoggerService.Error("这是一个错误消息", zap.String("error_type", "demo_error"))
	fmt.Println("   ✓ 日志消息已记录")

	// Test flush operation
	fmt.Println("\n3. 测试日志刷新操作...")
	err = service.FlushLogger()
	if err != nil {
		fmt.Printf("   ⚠ 刷新时出现预期的警告: %v\n", err)
	} else {
		fmt.Println("   ✓ 日志刷新完成")
	}

	// Log more messages
	fmt.Println("\n4. 记录更多日志消息...")
	for i := 0; i < 5; i++ {
		resource.LoggerService.Info("批量日志消息", 
			zap.Int("batch_id", i),
			zap.String("operation", "demo_batch"))
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("   ✓ 批量日志记录完成")

	// Test multiple flush operations
	fmt.Println("\n5. 测试多次刷新操作...")
	for i := 0; i < 3; i++ {
		err = service.FlushLogger()
		if err != nil {
			fmt.Printf("   ⚠ 第%d次刷新出现预期警告\n", i+1)
		} else {
			fmt.Printf("   ✓ 第%d次刷新成功\n", i+1)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Close logger gracefully
	fmt.Println("\n6. 优雅关闭logger服务...")
	err = service.CloseLogger(ctx)
	if err != nil {
		fmt.Printf("   ⚠ 关闭时出现错误: %v\n", err)
	} else {
		fmt.Println("   ✓ Logger服务已优雅关闭")
	}

	// Verify logger is closed
	if !service.IsLoggerInitialized() {
		fmt.Println("   ✓ Logger状态验证: 已正确关闭")
	} else {
		fmt.Println("   ✗ Logger状态验证: 未正确关闭")
	}

	// Test double close (should be safe)
	fmt.Println("\n7. 测试重复关闭操作...")
	err = service.CloseLogger(ctx)
	if err != nil {
		fmt.Printf("   ⚠ 重复关闭出现错误: %v\n", err)
	} else {
		fmt.Println("   ✓ 重复关闭操作安全完成")
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("修复要点:")
	fmt.Println("• 添加了同步错误检查和分类处理")
	fmt.Println("• 将常见的文件描述符错误标记为可忽略的警告")
	fmt.Println("• 改进了控制台核心创建，增加文件描述符有效性检查")
	fmt.Println("• 提供了更友好的错误消息和日志级别")
	fmt.Println("• 确保了logger关闭操作的健壮性")
}
