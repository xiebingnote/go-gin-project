package httpserver

import (
	"github.com/gin-gonic/gin"
)

// NewServer returns a new gin.Engine that can be used to start a web server.
//
// The returned gin.Engine is configured to serve the controllers registered in
// the controller package at "/web/api/xxx/v1". This URL path is used by the
// controllers to handle incoming requests.
//
// Users of NewServer should call the Run method on the returned gin.Engine to
// start the web server. The Run method will block until the web server is
// interrupted or an error occurs.
//
// The returned gin.Engine has been configured with the default gin.Logger and
// gin.Recovery middleware. The gin.Logger middleware logs each incoming
// request and response. The gin.Recovery middleware recovers from any panics
// that occur during the handling of a request and logs the panic.
func NewServer() *gin.Engine {
	// Create a new gin.Engine with the default middleware.
	// The default middleware consists of the gin.Logger and gin.Recovery
	// middleware.
	router := gin.Default()

	// Register the controllers with the gin.Engine.
	// The controllers are registered at "/web/api/xxx/v1".
	Router(router.Group("/web/api/project"))

	// Return the configured gin.Engine.
	// Users of NewServer should call the Run method on the returned gin.Engine
	// to start the web server.
	return router
}
