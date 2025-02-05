package servers

import (
	"context"
	"errors"
	"net/http"
	"project/servers/httpserver"
)

// Start starts a new HTTP server.
//
// The server is started in a new goroutine, and the returned channel will
// receive any error that occurs while starting or running the server. If the
// server is closed intentionally, the channel will be closed without sending
// an error.
//
// The provided context is used to cancel the server when the context is
// canceled. This is useful for cleaning up resources when the server is
// intentionally closed.
//
// The returned *http.Server is the server that was started. It can be used to
// close the server with its Shutdown method.
func Start(ctx context.Context) (*http.Server, <-chan error) {
	router := httpserver.NewServer()
	// Create a new HTTP server with the specified address and default handler.
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Create a channel to receive errors from the server.
	errChan := make(chan error, 1)

	// Start the server in a new goroutine.
	go func() {
		// ListenAndServe starts the HTTP server.
		// If an error occurs and it's not due to the server being intentionally closed,
		// send the error to errChan.
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		// Close the error channel when the server stops.
		close(errChan)
	}()

	// Return the server and error channel.
	return server, errChan
}
