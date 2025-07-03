package main

import (
	"fmt"
	"log"
	"time"

	"github.com/xiebingnote/go-gin-project/pkg/circuitbreaker"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("🔧 简单熔断器使用示例")
	fmt.Println("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	// 创建 logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("创建 logger 失败: %v", err)
	}

	// 创建熔断器
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:        "example-service",
		MaxRequests: 3,                // 半开状态下允许3个请求
		Interval:    10 * time.Second, // 10秒统计窗口
		Timeout:     5 * time.Second,  // 熔断5秒后尝试恢复
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			// 当请求数>=5且失败率>=60%时熔断
			return counts.Requests >= 5 &&
				   float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6
		},
		OnStateChange: func(name string, from circuitbreaker.State, to circuitbreaker.State) {
			fmt.Printf("🔄 熔断器状态变化: %s -> %s\n", from.String(), to.String())
		},
	})
	cb.SetLogger(logger)

	fmt.Println("✅ 熔断器创建成功")

	// 示例1: 正常请求
	fmt.Println("\n📋 示例1: 发送正常请求")
	for i := 1; i <= 3; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			// 模拟成功的服务调用
			return fmt.Sprintf("成功响应 %d", i), nil
		})

		if err != nil {
			fmt.Printf("❌ 请求 %d 失败: %v\n", i, err)
		} else {
			fmt.Printf("✅ 请求 %d 成功: %v\n", i, result)
		}
	}

	// 示例2: 失败请求触发熔断
	fmt.Println("\n📋 示例2: 发送失败请求触发熔断")
	for i := 1; i <= 6; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			// 模拟失败的服务调用
			return nil, fmt.Errorf("服务错误 %d", i)
		})

		if err != nil {
			fmt.Printf("❌ 请求 %d 失败: %v\n", i, err)
		} else {
			fmt.Printf("✅ 请求 %d 成功: %v\n", i, result)
		}
	}

	// 显示当前状态
	state := cb.State()
	counts := cb.Counts()
	fmt.Printf("\n📊 当前熔断器状态: %s\n", state.String())
	fmt.Printf("📊 统计信息: 总请求=%d, 成功=%d, 失败=%d\n",
		counts.Requests, counts.TotalSuccesses, counts.TotalFailures)

	// 示例3: 熔断状态下的请求被拒绝
	fmt.Println("\n📋 示例3: 熔断状态下请求被拒绝")
	for i := 1; i <= 3; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			return "这个不会被执行", nil
		})

		if err != nil {
			fmt.Printf("❌ 请求 %d 被拒绝: %v\n", i, err)
		} else {
			fmt.Printf("✅ 请求 %d 成功: %v\n", i, result)
		}
	}

	// 示例4: 等待恢复并发送成功请求
	fmt.Println("\n📋 示例4: 等待熔断器恢复")
	fmt.Println("⏳ 等待5秒让熔断器进入半开状态...")
	time.Sleep(5200 * time.Millisecond)

	fmt.Println("🔧 发送恢复请求...")
	for i := 1; i <= 4; i++ {
		result, err := cb.Execute(func() (interface{}, error) {
			// 模拟恢复后的成功请求
			return fmt.Sprintf("恢复成功 %d", i), nil
		})

		if err != nil {
			fmt.Printf("❌ 恢复请求 %d 失败: %v\n", i, err)
		} else {
			fmt.Printf("✅ 恢复请求 %d 成功: %v\n", i, result)
		}
	}

	// 最终状态
	finalState := cb.State()
	finalCounts := cb.Counts()
	fmt.Printf("\n📊 最终熔断器状态: %s\n", finalState.String())
	fmt.Printf("📊 最终统计信息: 总请求=%d, 成功=%d, 失败=%d\n",
		finalCounts.Requests, finalCounts.TotalSuccesses, finalCounts.TotalFailures)

	fmt.Println("\n🎉 熔断器示例演示完成!")
	fmt.Println("\n💡 关键要点:")
	fmt.Println("  1. 熔断器在失败率过高时自动开启保护")
	fmt.Println("  2. 开启状态下所有请求被快速拒绝")
	fmt.Println("  3. 超时后自动进入半开状态测试恢复")
	fmt.Println("  4. 恢复成功后自动关闭熔断器")
}