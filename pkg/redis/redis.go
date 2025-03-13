package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/resource"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// SetValue sets a key-value pair in Redis with no expiration time.
//
// It uses the Redis client to set the key "test" with the value "test".
// If the operation fails, it returns the error.
func SetValue() error {
	// Set the key "test" to the value "test" with no expiration time.
	err := resource.RedisClient.Set(ctx, "test", "test", 0).Err()
	if err != nil {
		// Return the error if the set operation fails.
		return err
	}

	// Return nil if the operation was successful.
	return nil
}

// GetValue gets a value from Redis.
//
// It uses the Redis client to get the value for the key "test". If the key
// does not exist, it returns redis.Nil. If there is an error during the get
// operation, it returns the error. If the operation is successful, it prints
// out the value.
func GetValue() error {
	// Get the value for the key "test".
	val, err := resource.RedisClient.Get(ctx, "test").Result()
	if errors.Is(err, redis.Nil) {
		// Log an error if the key does not exist.
		resource.LoggerService.Error(fmt.Sprintf("redis get failed: %v", err))
		return err
	} else if err != nil {
		// Log an error if there is an error during the get operation.
		resource.LoggerService.Error(fmt.Sprintf("redis get failed: %v", err))
		return err
	}

	// Print out the value if the operation was successful.
	fmt.Println("val:", val)

	// Return nil if the operation was successful.
	return nil
}

// ListValue demonstrates various Redis list operations.
//
// It performs a series of operations on a Redis list identified by the key
// "test-list". It pushes elements to the list, pops an element from the list,
// and retrieves the remaining elements. It returns an error if any operation fails.
func ListValue() error {
	listKey := "test-list"

	// Push elements "test1", "test2", and "test3" to the end of the list.
	err := resource.RedisClient.RPush(ctx, listKey, "test1", "test2", "test3").Err()
	if err != nil {
		return err
	}

	// Push elements "test4" and "test5" to the start of the list.
	err = resource.RedisClient.LPush(ctx, listKey, "test4", "test5").Err()
	if err != nil {
		return err
	}

	// Pop an element from the start of the list.
	//resource.RedisClient.RPop(ctx, listKey).Result()
	task, err := resource.RedisClient.LPop(ctx, listKey).Result()
	if err != nil {
		return err
	}
	fmt.Println("task:", task) // Print the popped element.

	// Retrieve all elements from the list.
	tasks, err := resource.RedisClient.LRange(ctx, listKey, 0, -1).Result()
	if err != nil {
		return err
	}
	fmt.Println("tasks:", tasks) // Print the remaining elements in the list.

	return nil
}

// HashValue demonstrates various Redis hash operations.
//
// It performs a series of operations on a Redis hash identified by the key
// "test-hash". It sets multiple field-value pairs, retrieves a specific field,
// and retrieves all field-value pairs. It returns an error if any operation fails.
func HashValue() error {
	hashKey := "test-hash"

	// Set field-value pairs "test1": 1 and "test2": 2 in the hash.
	err := resource.RedisClient.HSet(ctx, hashKey, map[string]interface{}{
		"test1": 1,
		"test2": 2,
	}).Err()
	if err != nil {
		return err
	}

	// Retrieve the value associated with field "test1".
	val, err := resource.RedisClient.HGet(ctx, hashKey, "test1").Result()
	if err != nil {
		return err
	}
	fmt.Println("val:", val) // Print the retrieved value.

	// Retrieve all field-value pairs from the hash.
	allValue, err := resource.RedisClient.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return err
	}
	fmt.Println("all value:", allValue) // Print all field-value pairs.

	return nil
}

// PublishValue publishes a message to a Redis pub/sub channel.
//
// It publishes a message with the value "test" to the channel identified by
// the key "test". It returns an error if the publish operation fails.
func PublishValue() error {
	err := resource.RedisClient.Publish(ctx, "test", "test").Err()
	if err != nil {
		return err
	}
	return nil
}

// SubscribeValue subscribes to a Redis pub/sub channel and prints all messages
// it receives.
//
// It subscribes to the channel identified by the key "test" and prints all
// messages it receives. It blocks until the context is canceled and then
// returns.
func SubscribeValue() error {
	// Subscribe to the channel identified by the key "test".
	sub := resource.RedisClient.Subscribe(ctx, "test")
	ch := sub.Channel()

	// Print all messages received from the channel.
	for v := range ch {
		fmt.Println("subscribe:", v.Payload)
	}

	return nil
}

// SetNXValue uses the SETNX command to set a value to a key only if the key
// does not exist. It is used to implement a lock mechanism.
//
// It sets the value "test" to the key "test" only if the key does not exist.
// If the key already exists, it prints a message indicating that the lock
// cannot be acquired. Otherwise, it prints a message indicating that the
// lock is acquired and then releases the lock by deleting the key.
func SetNXValue() error {
	lockKey := "test"
	ok, err := resource.RedisClient.SetNX(ctx, lockKey, "test", 0).Result()
	if err != nil {
		return err
	}
	if ok {
		fmt.Println("Get lock success")

		// todo: do something

		// release lock
		resource.RedisClient.Del(ctx, lockKey)
	} else {
		fmt.Println("Get lock failed")
	}

	return nil
}
