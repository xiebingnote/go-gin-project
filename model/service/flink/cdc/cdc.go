package cdc

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xiebingnote/go-gin-project/library/config"
	"github.com/xiebingnote/go-gin-project/library/request"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"github.com/xiebingnote/go-gin-project/model/types"

	"gopkg.in/yaml.v2"
)

// CreateFlinkCdcToMySQLYaml creates a YAML file for Flink CDC to MySQL
//
// This function generates a YAML configuration file used to establish
// a CDC pipeline from a MySQL source to a StarRocks sink. The configuration
// includes source database details, sink database details, and pipeline settings.
// The YAML file is stored in the directory specified by StarRocksConfig.FileDir
// with a file name derived from the request's job name.
func CreateFlinkCdcToMySQLYaml(info request.CreateFlinkCdcToMySQLYamlReq) error {
	// Get the JDBC URL for the sink
	sinkJdbcUrl := GetJdbcUrl()

	// Construct the load URL using StarRocks endpoints and HTTP port
	sinkLoadUrl := fmt.Sprintf("%s:%d", config.StarRocksConfig.StarRocks.EndPoints[0], config.StarRocksConfig.StarRocks.HttpPort)

	// Seed the random number generator and generate a random server ID
	rand.Seed(time.Now().UnixNano())
	serverID := rand.Intn(65535) + 1

	// Build the configuration object for StarRocks
	conf := types.StarRocksConfig{
		Source: struct {
			Type            string "yaml:\"type\""
			Hostname        string "yaml:\"hostname\""
			Port            int    "yaml:\"port\""
			Username        string "yaml:\"username\""
			Password        string "yaml:\"password\""
			Tables          string "yaml:\"tables\""
			TablesExclude   string "yaml:\"tables.exclude\""
			ServerID        int    "yaml:\"server-id\""
			ServerTimeZone  string "yaml:\"server-time-zone\""
			ScanStartupMode string "yaml:\"scan.startup.mode\""
		}{
			Type:            types.SourcdType,
			Hostname:        config.MySQLConfig.Resource.Manual.Default[0].Host,
			Port:            config.MySQLConfig.Resource.Manual.Default[0].Port,
			Username:        config.MySQLConfig.MySQL.Username,
			Password:        config.MySQLConfig.MySQL.Password,
			Tables:          info.Tables,
			TablesExclude:   types.TablesExclude,
			ServerID:        serverID,
			ServerTimeZone:  types.ServerTimeZone,
			ScanStartupMode: types.ScanStartupMode,
		},
		Sink: struct {
			Type     string "yaml:\"type\""
			Name     string "yaml:\"name\""
			JdbcURL  string "yaml:\"jdbc-url\""
			LoadURL  string "yaml:\"load-url\""
			Username string "yaml:\"username\""
			Password string "yaml:\"password\""
		}{
			Type:     types.SinkType,
			Name:     info.SinkName,
			JdbcURL:  sinkJdbcUrl,
			LoadURL:  sinkLoadUrl,
			Username: config.StarRocksConfig.StarRocks.UserName,
			Password: config.StarRocksConfig.StarRocks.PassWord,
		},
		Pipeline: struct {
			Name        string "yaml:\"name\""
			Parallelism int    "yaml:\"parallelism\""
		}{
			Name:        info.JobName,
			Parallelism: config.StarRocksConfig.StarRocks.Parallelism,
		},
	}

	// Define the file path for the YAML file
	filePath := fmt.Sprintf("%s/%s.yaml", config.StarRocksConfig.StarRocks.FileDir, info.JobName)

	// Create the YAML file
	file, err := os.Create(fmt.Sprintf(filePath))
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to create yaml file: %v", err))
		return err
	}
	defer file.Close()

	// Encode the configuration as YAML and write it to the file
	encoder := yaml.NewEncoder(file)
	if err = encoder.Encode(conf); err != nil {
		resource.LoggerService.Error(fmt.Sprintf("failed to encode yaml: %v", err))
		return err
	}

	// Return nil if no error occurred
	return nil
}

// GetJdbcUrl returns the JDBC URL for connecting to StarRocks.
// The method takes into account the number of endpoints configured in the StarRocks configuration.
// If there is only one endpoint, the method returns a JDBC URL for a single endpoint.
// If there are multiple endpoints, the method returns a JDBC URL that contains all the endpoints.
func GetJdbcUrl() string {
	// Initialize the JDBC URL as an empty string
	var jdbcUrl string

	// Check the number of endpoints in the StarRocks configuration
	switch len(config.StarRocksConfig.StarRocks.EndPoints) {
	// If there are no endpoints, return an empty string
	case 0:
		return jdbcUrl
	// If there is one endpoint, create a JDBC URL with that endpoint
	case 1:
		jdbcUrl = fmt.Sprintf("jdbc:mysql://%s:%d", config.StarRocksConfig.StarRocks.EndPoints[0], config.StarRocksConfig.StarRocks.FEPort)
	// If there are multiple endpoints, create a JDBC URL with all the endpoints
	default:
		// Initialize an array to store the JDBC URLs for each endpoint
		var jdbcArray []string
		// Iterate through the endpoints and create a JDBC URL for each one
		for i := 0; i < len(config.StarRocksConfig.StarRocks.EndPoints); i++ {
			url := fmt.Sprintf("jdbc:mysql://%s:%d", config.StarRocksConfig.StarRocks.EndPoints[i], config.StarRocksConfig.StarRocks.FEPort)
			// Append the JDBC URL for this endpoint to the array
			jdbcArray = append(jdbcArray, url)
		}
		// Create a JDBC URL that contains all the endpoints
		jdbcUrl = fmt.Sprintf("jdbc:mysql://%s", strings.Join(jdbcArray, ","))
	}

	// Return the JDBC URL
	return jdbcUrl
}

// GetFileList returns a list of files in the specified directory
// path: the directory path to search for files
// Returns: a slice of file paths and an error if any
func GetFileList(path string) ([]string, error) {
	var fileList []string

	// Read the directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	// Iterate through directory entries
	for _, entry := range entries {
		// Skip directories, only include regular files
		if !entry.IsDir() {
			fileList = append(fileList, entry.Name())
		}
	}

	return fileList, nil
}

// ExecCmd executes a command in the specified directory
// dir: the directory where the command should be executed
// cmd: the command to execute
// Returns: the output of the command and an error if any
func ExecCmd(dir string, cmd string, args ...string) ([]byte, error) {
	// Create a new command;
	command := exec.Command(cmd, args...)

	// Set the working directory
	command.Dir = dir

	// Run the command and capture the output
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute command %s: %w", cmd, err)
	}

	// Return the output and any error
	return output, err
}

// ExecFlinkCDC executes the flink-cdc.sh command in the specified directory
//
// This function executes the flink-cdc.sh command in the directory specified by
// StarRocksConfig.FlinkCDC.Path and passes the full path of the YAML file
// specified by fileName as an argument to the command.
//
// path: the directory where the command should be executed
// fileName: the name of the YAML file to pass as an argument to the command
// Returns: the output of the command and an error if any
func ExecFlinkCDC(fileName string) ([]byte, error) {
	// Get the directory path and command from the StarRocks configuration
	path := config.StarRocksConfig.FlinkCDC.Path
	cmd := config.StarRocksConfig.FlinkCDC.Cmd

	// Construct the full path of the YAML file
	filePath := fmt.Sprintf("%s/%s.yaml", config.StarRocksConfig.StarRocks.FileDir, fileName)

	// Execute the command and capture the output
	return ExecCmd(path, cmd, filePath)
	//return ExecCmd(path, "ls", path)
}

// DeleteFile deletes a file in the specified directory
//
// This function deletes a YAML file from the directory specified in the StarRocks configuration.
// The file name is specified by the fileName parameter.
//
// path: the directory path to delete the file
// fileName: the name of the file to delete (without the .yaml extension)
// Returns: an error if any occurs during the deletion process
func DeleteFile(fileName string) error {
	// Construct the full path of the file by concatenating the directory path
	// and the file name
	filePath := fmt.Sprintf("%s/%s.yaml", config.StarRocksConfig.StarRocks.FileDir, fileName)

	// Delete the file
	return os.Remove(filePath)
}

// GetFileContent reads a YAML file from the directory specified in the StarRocks configuration,
// and unmarshals its content into a StarRocksConfig struct.
//
// This function is used to retrieve the content of a YAML configuration file by its name.
// It reads the file from the directory specified in the StarRocks configuration,
// unmarshals its content into a StarRocksConfig struct and returns it.
//
// fileName: the name of the file to read (without the .yaml extension)
// Returns: the content of the file as a StarRocksConfig struct and an error if any occurs during reading or unmarshalling.
func GetFileContent(fileName string) (types.StarRocksConfig, error) {
	// Initialize the StarRocksConfig struct to store the file content
	var yamlInfo types.StarRocksConfig

	// Construct the full path of the YAML file
	filePath := fmt.Sprintf("%s/%s.yaml", config.StarRocksConfig.StarRocks.FileDir, fileName)

	// Read the file content
	readBytes, err := os.ReadFile(filePath)
	if err != nil {
		// Return error if reading fails
		return yamlInfo, err
	}

	// Unmarshal the YAML content into the StarRocksConfig struct
	if err = yaml.Unmarshal(readBytes, &yamlInfo); err != nil {
		// Return error if unmarshalling fails
		return yamlInfo, err
	}

	// Return the unmarshalled content
	return yamlInfo, nil
}

// IsFileExist checks if a file exists in the specified directory
// path: the directory path to check the file
// fileName: the name of the file to check
// Returns: true if the file exists, false otherwise
func IsFileExist(fileName string) bool {
	// Get the full path of the file
	filePath := fmt.Sprintf("%s/%s.yaml", config.StarRocksConfig.StarRocks.FileDir, fileName)

	// Check if the file exists
	_, err := os.Stat(filePath)
	// Return true if the file exists, false otherwise
	return err == nil
}
