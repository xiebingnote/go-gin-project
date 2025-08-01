package api

import (
	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/request"
	resp "github.com/xiebingnote/go-gin-project/library/response"
	"github.com/xiebingnote/go-gin-project/model/service/flink/cdc"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateFlinkCDCConfig creates a YAML file for Flink CDC to MySQL
//
// This function is an API handler that creates a YAML configuration file used to
// establish a CDC pipeline from a MySQL source to a StarRocks sink. The
// configuration includes source database details, sink database details, and
// pipeline settings. The YAML file is stored in the directory specified by
// StarRocksConfig.FileDir with a file name derived from the request's job name.
//
// @Summary Create Flink CDC to MySQL YAML configuration file
// @Tags FlinkCDC
// @Accept  json
// @Produce  json
// @Param   CreateFlinkCdcToMySQLYamlReq body     request.CreateFlinkCdcToMySQLYamlReq true "CreateFlinkCdcToMySQLYamlReq"
// @Success 200 {object} resp.Response
// @Failure 400 {object} resp.Response
// @Failure 500 {object} resp.Response
// @Router /flink/cdc [post]
func CreateFlinkCDCConfig(c *gin.Context) {
	// Generate a new request ID for tracking
	reqID := uuid.NewString()

	// Parse the request body into the CreateFlinkCdcToMySQLYamlReq struct
	info := request.CreateFlinkCdcToMySQLYamlReq{}
	err := c.BindJSON(&info)
	if err != nil {
		// Respond with a 400 error if the request body is invalid
		resp.NewErrResp(c, 400, err.Error(), reqID)
		return
	}

	// Check if the file already exists
	if cdc.IsFileExist(info.JobName) {
		resp.NewErrResp(c, 400, "file already exist", reqID)
	}

	// Call the service layer to create the YAML configuration file
	err = cdc.CreateFlinkCdcToMySQLYaml(info)
	if err != nil {
		// Respond with a 500 error if there is an issue creating the YAML file
		resp.ErrorRestResp(500, err.Error(), reqID)
		return
	}

	// Respond with a success message if the file is created successfully
	resp.NewOKResp(c, "", reqID)
}

// AddFlinkCDCJob handles the addition of a new Flink CDC job
//
// This function is an API handler that accepts a JSON request to add a new
// Flink CDC job. It binds the request to the AddFlinkCdcJobReq struct, executes
// the Flink CDC command using the provided file name, and returns the result
// or an error response.
//
// @Summary Add Flink CDC Job
// @Tags FlinkCDC
// @Accept  json
// @Produce  json
// @Param   AddFlinkCdcJobReq body     request.AddFlinkCdcJobReq true "AddFlinkCdcJobReq"
// @Success 200 {object} resp.Response
// @Failure 400 {object} resp.Response
// @Failure 500 {object} resp.Response
// @Router /flink/cdc/job [post]
func AddFlinkCDCJob(c *gin.Context) {
	// Generate a new request ID for tracking
	reqID := uuid.NewString()

	// Parse the request body into the AddFlinkCdcJobReq struct
	info := request.AddFlinkCdcJobReq{}
	err := c.BindJSON(&info)
	if err != nil {
		// Respond with a 400 error if the request body is invalid
		resp.NewErrResp(c, 400, err.Error(), reqID)
		return
	}

	// Execute the Flink CDC command with the provided file name
	res, err := cdc.ExecFlinkCDC(info.FileName)
	if err != nil {
		// Respond with a 500 error if there is an issue executing the command
		resp.ErrorRestResp(500, err.Error(), reqID)
		return
	}

	// Respond with a success message and the command result
	resp.NewOKResp(c, string(res), reqID)
}

// GetFlinkCDCConfigInfo gets the YAML content of a Flink CDC configuration file
//
// This function is an API handler that accepts a query parameter 'file_name'
// and returns the YAML content of the file specified by the parameter.
//
// @Summary Get Flink CDC configuration file content
// @Tags FlinkCDC
// @Produce  json
// @Param   file_name query     string true "file_name"
// @Success 200 {object} resp.Response
// @Failure 400 {object} resp.Response
// @Failure 500 {object} resp.Response
// @Router /flink/cdc/config [get]
func GetFlinkCDCConfigInfo(c *gin.Context) {
	reqID := uuid.NewString()
	name := c.Query("file_name")
	isExist := cdc.IsFileExist(name)
	if !isExist {
		resp.NewErrResp(c, 400, "file not exist", reqID)
	}
	info, err := cdc.GetFileContent(name)
	if err != nil {
		// Respond with a 500 error if there is an issue reading the file
		resp.ErrorRestResp(500, err.Error(), reqID)
		return
	}

	// Respond with a success message and the file content
	resp.NewOKResp(c, info, reqID)
}

// GetFlinkCDCConfigList retrieves the list of Flink CDC configuration files.
//
// This function is an API handler that fetches the list of YAML configuration files
// present in the directory specified by StarRocksConfig.FileDir.
//
// @Summary Get list of Flink CDC configuration files
// @Tags FlinkCDC
// @Produce json
// @Success 200 {object} resp.Response
// @Failure 500 {object} resp.Response
// @Router /flink/cdc/configs [get]
func GetFlinkCDCConfigList(c *gin.Context) {
	// Generate a new request ID for tracking
	reqID := uuid.NewString()

	// Fetch the list of configuration files from the specified directory
	fileList, err := cdc.GetFileList(config.StarRocksConfig.StarRocks.FileDir)
	if err != nil {
		// Respond with a 500 error if there is an issue fetching the file list
		resp.ErrorRestResp(500, err.Error(), "")
		return
	}

	// Respond with a success message and the list of files
	resp.NewOKResp(c, fileList, reqID)
}

// DeleteFlinkCDCConfig deletes a Flink CDC configuration file
//
// This function is an API handler that accepts a query parameter 'file_name'
// and deletes the file specified by the parameter.
//
// @Summary Delete Flink CDC configuration file
// @Tags FlinkCDC
// @Produce  json
// @Param   file_name query     string true "file_name"
// @Success 200 {object} resp.Response
// @Failure 500 {object} resp.Response
// @Router /flink/cdc/config [delete]
func DeleteFlinkCDCConfig(c *gin.Context) {
	reqID := uuid.NewString()
	name := c.Query("file_name")
	err := cdc.DeleteFile(name)
	if err != nil {
		// Respond with a 500 error if there is an issue deleting the file
		resp.ErrorRestResp(500, err.Error(), reqID)
		return
	}

	// Respond with a success message
	resp.NewOKResp(c, "", reqID)
}
