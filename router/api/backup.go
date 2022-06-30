package api

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	echo "github.com/labstack/echo/v4"
	"github.com/swisscom/backman/config"
	"github.com/swisscom/backman/log"
	"github.com/swisscom/backman/service"
)

// swagger:model Backup
type Backup struct {
	Service Service `json:"Service"`
	Files   []File  `json:"Files"`
}
type File struct {
	Key          string    `json:"Key"`
	Filepath     string    `json:"Filepath"`
	Filename     string    `json:"Filename"`
	Size         int64     `json:"Size"`
	LastModified time.Time `json:"LastModified"`
}

// swagger:response Backups
type Backups []Backup

func getAPIBackup(backup service.Backup) Backup {
	files := make([]File, 0)
	for _, file := range backup.Files {
		files = append(files, File{
			Key:          file.Key,
			Filepath:     file.Filepath,
			Filename:     file.Filename,
			Size:         file.Size,
			LastModified: file.LastModified,
		})
	}
	return Backup{
		Service: getAPIService(backup.Service),
		Files:   files,
	}
}

// swagger:route GET /api/v1/backups backup listBackups
// Lists all backup objects.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   200: Backups
func (h *Handler) ListBackups(c echo.Context) error {
	serviceType := c.QueryParam("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}

	b, err := service.GetBackups(serviceType, serviceName)
	if err != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	}

	backups := make(Backups, 0)
	for _, backup := range b {
		backups = append(backups, getAPIBackup(backup))
	}
	return c.JSON(http.StatusOK, backups)
}

// swagger:route GET /api/v1/backup/{service_type}/{service_name} backup getBackups
// Returns a full backup object for given service.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   200: Backup
func (h *Handler) GetBackups(c echo.Context) error {
	serviceType := c.QueryParam("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}

	backups, err := service.GetBackups(serviceType, serviceName)
	if err != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	}

	// there should only be 1 backup struct in there since we specified serviceName
	if len(backups) != 1 {
		return c.JSON(http.StatusNotFound, fmt.Errorf("backups not found"))
	}
	return c.JSON(http.StatusOK, getAPIBackup(backups[0]))
}

// swagger:route GET /api/v1/backup/{service_type}/{service_name}/{filename} backup getBackup
// Returns a single backup file object for given service.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   200: Backup
func (h *Handler) GetBackup(c echo.Context) error {
	serviceType := c.Param("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}
	filename, err := url.QueryUnescape(c.Param("file"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filename: %v", err))
	}

	backup, err := service.GetBackup(serviceType, serviceName, filename)
	if err != nil {
		return c.JSON(http.StatusNotFound, err.Error())
	}
	if len(backup.Files) == 0 || len(backup.Files[0].Filename) == 0 {
		return c.JSON(http.StatusNotFound, fmt.Errorf("file not found"))
	}
	return c.JSON(http.StatusOK, getAPIBackup(*backup))
}

// swagger:route POST /api/v1/backup/{service_type}/{service_name} backup createBackup
// Creates a new backup for given service.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   202: Service
func (h *Handler) CreateBackup(c echo.Context) error {
	serviceType := c.Param("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}

	if !config.IsValidServiceType(serviceType) {
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("unsupported service type: %s", serviceType))
	}

	serviceInstance := service.GetService(serviceType, serviceName)
	if len(serviceInstance.Name) == 0 {
		err := fmt.Errorf("could not find service [%s] to backup", serviceName)
		log.Errorf("%v", err)
		return c.JSON(http.StatusNotFound, err.Error())
	}

	go func() { // async
		if err := service.CreateBackup(serviceInstance); err != nil {
			log.Errorf("requested backup for service [%s] failed: %v", serviceName, err)
		}
	}()
	return c.JSON(http.StatusAccepted, getAPIService(serviceInstance))
}

// swagger:route GET /api/v1/backup/{service_type}/{service_name}/{filename}/download backup downloadBackup
// Download a backup file for given service.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   200:
func (h *Handler) DownloadBackup(c echo.Context) error {
	serviceType := c.Param("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}
	filename, err := url.QueryUnescape(c.Param("file"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filename: %v", err))
	}

	reader, err := service.ReadBackup(serviceType, serviceName, filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Stream(http.StatusOK, "application/gzip", reader)
}

// swagger:route DELETE /api/v1/backup/{service_type}/{service_name}/{filename} backup deleteBackup
// Deletes a backup file from S3 for given service.
//
// produces:
// - application/json
//
// schemes: http, https
//
// responses:
//   204:
func (h *Handler) DeleteBackup(c echo.Context) error {
	serviceType := c.Param("service_type")
	serviceName, err := url.QueryUnescape(c.Param("service_name"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid service name: %v", err))
	}
	filename, err := url.QueryUnescape(c.Param("file"))
	if err != nil {
		log.Errorf("%v", err)
		return c.JSON(http.StatusBadRequest, fmt.Sprintf("invalid filename: %v", err))
	}

	if err := service.DeleteBackup(serviceType, serviceName, filename); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
