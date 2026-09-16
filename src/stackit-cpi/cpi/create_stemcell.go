package cpi

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/* Arguments:
- image_path [String]: Path to the stemcell image extracted from the stemcell tarball on a local filesystem.
- cloud_properties [Hash]: Cloud properties hash extracted from the stemcell tarball.

Result:
- stemcell_cid [String]: Cloud ID of the created stemcell (e.g. stemcells in AWS CPI are made into AMIs so cid .would be ami-83fdflf)
*/

// CreateStemcell handles the "create_stemcell" method of the RPC
func (c *CPI) CreateStemcell(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.CreateStemcellArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		return lib.WrapErrorInResponse(fmt.Errorf("failed to unmarshal arguments: %w", err), req.GetLoggingContext())
	}

	// Extract the stemcell archive to get the root.img file (for full stemcells)
	c.Log.Debugf("Extracting stemcell archive at path: %s", args.Path)
	extractedPath, extractErr := c.extractImage(args.Path)
	if extractErr != nil {
		c.Log.Errorf("Failed to extract stemcell archive: %v", extractErr)

		return lib.WrapErrorInResponse(fmt.Errorf("failed to extract stemcell archive: %w", extractErr), req.GetLoggingContext())

	}

	c.Log.Debugf("Successfully extracted root.img to: %s", extractedPath)

	// Prepare image metadata tags
	// Regex for keys: ^(?=.{1,63}$)([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$
	// Regex for tags: ^(?=.{0,63}$)(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])*$
	tags := map[string]any{
		"version":                 args.Info.Version,
		"os_type":                 args.Info.OsType,
		"os_distro":               args.Info.OsDistro,
		"architecture":            args.Info.Architecture,
		lib.CreatedByBoshLabelKey: lib.CreatedByBoshLabelValue,
		"description":             "bosh-stemcell",
		"director_uuid":           req.Context.DirectorUUID,
		"creation_date":           time.Now().UTC().Format("20060102_150405"),
		// Values that can be passed are limited to what BOSH passes
	}

	// Image Name is a combination of stemcell name and version, no separator to align with openstack stemcell format.
	imageName := fmt.Sprintf("%s-%s", args.Info.Name, args.Info.Version)

	c.Log.Info(fmt.Sprintf("stemcell image: name: %s, version: %s, format: %s, os: %s, distro: %s",
		imageName, args.Info.Version, args.Info.DiskFormat, args.Info.OsType, args.Info.OsDistro))

	dist := iaas.NewNullableString(&args.Info.OsDistro)

	// Image Config
	imgConfig := iaas.ImageConfig{
		Architecture:          utils.Ptr("x86"),
		OperatingSystem:       &args.Info.OsType,
		OperatingSystemDistro: *dist,
		Uefi:                  utils.Ptr(false),
	}

	// Create image payload with all available metadata
	createImagePayload := iaas.CreateImagePayload{
		Name:       imageName,
		Config:     &imgConfig,
		DiskFormat: args.Info.DiskFormat,
		Labels:     tags,
	}
	c.Log.Debug("Creating image",
		"name", createImagePayload.Name,
		"disk_format", createImagePayload.DiskFormat,
		"labels", createImagePayload.Labels,
		"config", createImagePayload.Config)

	imageCreateResp, err := c.SClient.CreateStemcell(createImagePayload)
	if err != nil {
		c.Log.Errorf("Failed to create image: %v", err)
		return lib.WrapErrorInResponse(fmt.Errorf("failed to create image: %w", err), req.GetLoggingContext())
	}
	if imageCreateResp.GetUploadUrl() == "" || imageCreateResp.GetId() == "" {
		return lib.WrapErrorInResponse(fmt.Errorf("received nil UploadUrl or Id from CreateImage response"), req.GetLoggingContext())
	}

	c.Log.Debugf("%q has been initially created, proceeding with upload", imageCreateResp.GetId())

	c.Log.Debug("Attempting to upload extracted image with retries.")
	if err = c.uploadImage(extractedPath, imageCreateResp.GetUploadUrl(), imageCreateResp.GetId()); err != nil {
		c.Log.Errorf("Failed to upload image: %v", err)

		// Try to clean up the partially created image
		deleteErr := c.SClient.DeleteStemcell(imageCreateResp.GetId())
		if deleteErr != nil {
			c.Log.Warnf("Failed to clean up partially created image %s: %v", imageCreateResp.GetId(), deleteErr)
		}
		return lib.WrapErrorInResponse(fmt.Errorf("failed to upload image: %w", err), req.GetLoggingContext())
	}

	// Wait for the image to be available
	c.Log.Debugf("Waiting for image %s to become available", imageCreateResp.GetId())

	// we see 504 for the get image endpoint regularly. not sure why but listing images works at the same
	// time. In general it is a retryable error and since waiting for a stemcell can take up to 50 minutes,
	// this loop tries it's best to avoid wasting peoples lifetimes

	for {
		_, err = c.SClient.WaitForStemcellCreate(imageCreateResp.GetId())
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusGatewayTimeout) {
			c.Log.Info("ignoring 504 error while waiting for stemcell", "stemcell_id", imageCreateResp.GetId())
			continue
		}
		break
	}
	if err != nil {
		err = fmt.Errorf("failed waiting for image to become available: %w", err)
		c.Log.Error(err.Error())
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// Clean up the extracted file
	if err := os.Remove(extractedPath); err != nil {
		c.Log.Warnf("Failed to clean up extracted image file %s: %v", extractedPath, err)
	}

	c.Log.Infof("Stemcell '%s' (version '%s') created successfully with ID: %s",
		args.Info.Name, args.Info.Version, imageCreateResp.GetId())

	return &lib.RPCResponse{
		Error:  nil,
		Result: imageCreateResp.GetId(),
		Log:    "success",
	}, nil
}

// defaultExtractImage extracts a gzip tar archive and returns the path to the root.img file
func (c *CPI) extractImage(archivePath string) (string, error) {
	// Validate the archive path
	archivePath = filepath.Clean(archivePath)
	if !filepath.IsAbs(archivePath) {
		return "", fmt.Errorf("archive path must be absolute")
	}

	// Open the gzip archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open archive file: %w", err)
	}
	defer func() { _ = archiveFile.Close() }()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Create a temporary file to extract files
	tmpFile, err := os.CreateTemp(os.TempDir(), "root.img")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer tmpFile.Close()

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", fmt.Errorf("error reading tar archive: %w", err)
		}

		// The BOSH Director untars the downloaded stemcell tarball, and provides the CPI a tarball named "Image" inside it
		// We're looking for the root.img file in it
		if header.Name == "root.img" || filepath.Base(header.Name) == "root.img" {
			// Copy the contents to the output file with a size limit (20GB max)
			limitedReader := io.LimitReader(tarReader, lib.StemcellMaxSize)
			if _, err := io.Copy(tmpFile, limitedReader); err != nil {
				return "", fmt.Errorf("failed to extract root.img file: %w", err)
			}

			c.Log.Debugf("Extracted root.img to %s", tmpFile.Name())
			break // We found what we needed
		}
	}

	return tmpFile.Name(), nil
}

func (c *CPI) uploadImage(imagePath, uploadURL, imageID string) error {
	// Validate the image path
	imagePath = filepath.Clean(imagePath)
	if !filepath.IsAbs(imagePath) {
		return fmt.Errorf("image path must be absolute")
	}

	// Open the file for streaming instead of loading all into memory
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Config.LRPTimeout)*time.Second)
	defer cancel()
	// Create request with file stream as body
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	// Set appropriate timeouts based on file size
	client := &http.Client{
		Timeout: time.Duration(c.Config.LRPTimeout) * time.Second,
	}

	c.Log.Debugf("Uploading image to '%s' (%d bytes)...", imageID, fileInfo.Size())
	return c.uploadWithRetries(client, req)
}

func (c *CPI) uploadWithRetries(uploadClient *http.Client, r *http.Request) (err error) {
	for attempt := range c.Config.RetryCount + 1 {
		c.Log.Debug("starting", "attempt", attempt+1)
		var resp *http.Response
		resp, err = uploadClient.Do(r)

		if err == nil {
			c.Log.Debug("success")
			return nil
		}
		c.Log.Debugf("failed uploading with: %s", err)
		err = oapierror.GenericOpenAPIError{
			StatusCode: resp.StatusCode,
			Body:       []byte(err.Error()),
		}
		if !lib.IsRetryableOpenAPIError(err) {
			c.Log.Debug("non recoverable failure", "attempt", attempt, "error", err)
			return fmt.Errorf("cpi.defaultUploadImage(): upload failed with status: %s", resp.Status)
		}
		c.Log.Debug("recoverable failure", "attempt", attempt, "error", err)
	}
	return err
}
