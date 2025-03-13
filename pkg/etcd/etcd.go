package etcd

import (
	"context"
	"fmt"

	"github.com/xiebingnote/go-gin-project/library/resource"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var ctx = context.Background()

// PutValue puts a value into etcd.
//
// It creates a background context and puts the value into etcd with the key
// "test". If the put operation is successful, it returns nil. Otherwise, it
// returns an error.
func PutValue() error {
	// Put the value into etcd
	_, err := resource.EtcdClient.Put(ctx, "test", "test")
	if err != nil {
		return fmt.Errorf("failed to put value into etcd: %w", err)
	}

	return nil
}

// GetValue gets a value from etcd.
//
// It creates a background context and retrieves the value specified by the
// key from etcd. If the retrieval is successful, it prints out the value.
// Otherwise, it returns an error.
func GetValue() error {
	// Get the value from etcd
	resp, err := resource.EtcdClient.Get(ctx, "test")
	if err != nil {
		return err
	}

	// Print out the value
	for _, v := range resp.Kvs {
		fmt.Println("value:", string(v.Value))
	}

	return nil
}

// DeleteValue deletes a value from etcd.
//
// It creates a background context and deletes the value specified by the key
// from etcd. If the deletion is successful, it returns nil. Otherwise, it
// returns an error.
func DeleteValue() error {

	// Delete the value from etcd
	_, err := resource.EtcdClient.Delete(ctx, "test")
	if err != nil {
		return fmt.Errorf("failed to delete value from etcd: %w", err)
	}

	return nil
}

// Lease creates a lease in etcd and associates a key with the lease.
//
// It grants a lease with a TTL of 10 seconds and puts a key-value pair into etcd
// with the lease attached. It logs an error if any operation fails and returns the error.
// On success, it prints a message indicating the lease will expire in 10 seconds.
func Lease() error {
	// Grant a lease with a TTL of 10 seconds
	leaseResp, err := resource.EtcdClient.Grant(ctx, 10)
	if err != nil {
		// Log the error if lease granting fails
		resource.LoggerService.Error(fmt.Sprintf("failed to grant lease: %v", err))
		return err
	}

	// Put a key-value pair with the lease into etcd
	_, err = resource.EtcdClient.Put(ctx, "test", "test", clientv3.WithLease(leaseResp.ID))
	if err != nil {
		// Log the error if putting the key with lease fails
		resource.LoggerService.Error(fmt.Sprintf("failed to put grant lease key into etcd: %v", err))
		return err
	}

	// Print success message indicating lease expiration
	fmt.Println("lease success, will be expired in 10s")

	return nil
}

// Watch monitors changes to a key in etcd and prints out any events it receives.
//
// This function creates a watcher on the key "test" and continuously monitors for events.
// The events can include PUT, DELETE, and other types of changes to the key. Each detected
// event is printed with its type and associated key-value information.
func Watch() error {
	// Start watching the key "test" for changes
	watchChan := resource.EtcdClient.Watch(ctx, "test")

	// Iterate over the received watch results
	for watchResult := range watchChan {
		// Process each event in the watch result
		for _, event := range watchResult.Events {
			// Print the event type, key, and value
			fmt.Printf("Detected change: %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)
		}
	}

	// Return nil to indicate successful completion
	return nil
}

// Txn demonstrates how to use etcd's transaction feature to atomically
// execute a request based on the current state of the key.
//
// In this example, we attempt to create a key "test" with the value "locked".
// If the key does not exist, the transaction will succeed and the key will be
// created. If the key already exists, the transaction will fail and the key
// will not be modified.
func Txn() error {
	// The key to lock
	muteKey := "test-Lock"

	// Create a transaction conditionally putting the key if it does not exist
	txnResp, err := resource.EtcdClient.Txn(ctx).
		// If the key exists, abort the transaction
		If(clientv3.Compare(clientv3.CreateRevision(muteKey), "=", 0)).
		// If the key does not exist, create it with the value "locked"
		Then(clientv3.OpPut(muteKey, "locked")).
		// Commit the transaction
		Commit()
	if err != nil {
		fmt.Println("txn failed:", err)
		return err
	}

	// If the transaction failed, return an error
	if !txnResp.Succeeded {
		return fmt.Errorf("failed to lock key %s", muteKey)
	}

	// Print success message indicating the key was locked
	fmt.Println("lock success")

	return nil
}
