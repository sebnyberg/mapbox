package mapbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context/ctxhttp"
)

type UpsertTilesetRequest struct {
	Name   string        `json:"name"`
	Recipe TilesetRecipe `json:"recipe"`
}

// TilesetRecipe contains instructions on how the tileset should be generated.
type TilesetRecipe struct {
	Version int                           `json:"version"`
	Layers  map[string]TilesetRecipeLayer `json:"layers"`
}

// TilesetRecipeLayer contains settings for a single soruce layer.
type TilesetRecipeLayer struct {
	// Source is the URI to a tileset source, on the format:
	// mapbox://tileset-source/{username}/{tilesetName}
	Source string `json:"source"`

	// Min and MaxZoom sets the interval for which the layer is visible.
	MinZoom int `json:"minzoom"`
	MaxZoom int `json:"maxzoom"`
}

type UpdateTilesetErrResponse struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors"`
}

// UpsertTileset creates a tileset if it does not exist. If it exists,
// it will be patched with the provided recipe.
func (c *Client) UpsertTileset(ctx context.Context, tileset string, recipe TilesetRecipe) error {
	if len(tileset) == 0 {
		return fmt.Errorf("%w failed: tileset is required", ErrValidation)
	}
	if !strings.HasPrefix(tileset, c.username) {
		tileset = c.username + "." + tileset
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(recipe); err != nil {
		return fmt.Errorf("%w failure to parse json, err: %v", ErrUnexpected, err)
	}

	url := baseURL + "/tilesets/v1/" + tileset + "?access_token=" + c.accessToken

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("%w error, failed to create http request: %v", ErrUnexpected, err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := ctxhttp.Do(ctx, c.httpClient, req)
	if err != nil {
		return fmt.Errorf("tileset upload %w failed, err: %v", ErrOperation, err)
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Parse error
	var jsonResp UpdateTilesetErrResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return fmt.Errorf("%w of tileset update response failed, err: %v", ErrParse, err)
	}

	// BadRequest is returned when there is a resource conflict, in which case
	// the message contains the string "already exists".
	if strings.Contains(jsonResp.Message, "already exists") {
		return c.UpdateTilesetRecipe(ctx, tileset, recipe)
	}

	return errors.New(jsonResp.Message + ", errors: " + strings.Join(jsonResp.Errors, ","))
}

// UpdateTilesetRecipe replaces an existing recipe for the provided tileset.
func (c *Client) UpdateTilesetRecipe(ctx context.Context, tileset string, recipe TilesetRecipe) error {
	if len(tileset) == 0 {
		return fmt.Errorf("%w failed: tileset is required", ErrValidation)
	}
	if !strings.HasPrefix(tileset, c.username) {
		tileset = c.username + "." + tileset
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(recipe); err != nil {
		return fmt.Errorf("%w failure to parse json, err: %v", ErrUnexpected, err)
	}

	url := baseURL + "/tilesets/v1/" + tileset + "/recipe?access_token=" + c.accessToken
	req, err := http.NewRequest(http.MethodPatch, url, &body)
	if err != nil {
		return fmt.Errorf("%w error, failed to create http request: %v", ErrUnexpected, err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := ctxhttp.Do(ctx, c.httpClient, req)
	if err != nil {
		return fmt.Errorf("upload recipe %w failed, err: %v", ErrOperation, err)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	var jsonResp UpdateTilesetErrResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return fmt.Errorf("%w of tileset update response failed, err: %v", ErrParse, err)
	}

	return errors.New(jsonResp.Message + ", errors: " + strings.Join(jsonResp.Errors, ","))
}

// PublishTilesetJob is a pollable resource that returns the status of a publish job.
type PublishTilesetJob struct {
	Message string
	JobID   string
	Tileset string

	client *Client
}

type PublishTilesetResponse struct {
	Message string `json:"message"`
	JobID   string `json:"jobId"`
	// MTS is in beta and they've apparently made a mistake calling "job_id"
	// "jobId". JobIDSnakeCased is used to populate the JobID field in case
	// they fix their mistake in the future.
	JobIDSnakeCased string `json:"job_id,-"`
}

// PublishTileset publishes the provided tileset and returns a job that can
// be polled to check whether the job has finished or not.
func (c *Client) PublishTileset(ctx context.Context, tileset string) (PublishTilesetJob, error) {
	var job PublishTilesetJob
	if len(tileset) == 0 {
		return job, fmt.Errorf("%w failed: tileset is required", ErrValidation)
	}
	if !strings.HasPrefix(tileset, c.username) {
		tileset = c.username + "." + tileset
	}

	url := baseURL + "/tilesets/v1/" + tileset + "/publish?access_token=" + c.accessToken
	req, err := http.NewRequest(http.MethodPatch, url, nil)
	if err != nil {
		return job, fmt.Errorf("%w error, failed to create http request: %v", ErrUnexpected, err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := ctxhttp.Do(ctx, c.httpClient, req)
	if err != nil {
		return job, fmt.Errorf("publish tileset %w failed, err: %v", ErrOperation, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return job, fmt.Errorf("tileset %v %w", tileset, ErrNotFound)
	}

	var jsonResp PublishTilesetResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return job, fmt.Errorf("%w of publish tileset response failed, err: %v", ErrParse, err)
	}

	job.Message = jsonResp.Message
	job.JobID = jsonResp.JobID
	job.Tileset = tileset
	job.client = c

	// Check if Mapbox has renamed the field to be consistent.
	if len(jsonResp.JobID) == 0 && len(jsonResp.JobIDSnakeCased) != 0 {
		job.JobID = jsonResp.JobIDSnakeCased
	}
	if len(job.JobID) == 0 {
		fmt.Println("failed to fetch job ID from response")
	}

	return job, nil
}

type PublishJobStage string

const (
	PublishJobStageQueued     PublishJobStage = "queued"
	PublishJobStageProcessing                 = "processing"
	PublishJobStageSuccess                    = "success"
	PublishJobStageFailed                     = "failed"
)

type PollPublishJobResponse struct {
	ID          string                 `json:"id"`
	Stage       PublishJobStage        `json:"stage"`
	Created     int                    `json:"created"`
	CreatedNice string                 `json:"created_nice"`
	Published   int                    `json:"published"`
	TilesetID   string                 `json:"tileset_id"`
	Errors      []interface{}          `json:"errors"`
	Warnings    []interface{}          `json:"warnings"`
	LayerStats  map[string]interface{} `json:"layer_stats"`
}

// Poll returns the the status for a publish job.
func (j *PublishTilesetJob) Poll(ctx context.Context) (*PollPublishJobResponse, error) {
	url := baseURL + "/tilesets/v1/" + j.Tileset + "/jobs/" + j.JobID +
		"?access_token=" + j.client.accessToken

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w error, failed to create http request: %v", ErrUnexpected, err)
	}

	resp, err := ctxhttp.Do(ctx, j.client.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("poll job %w failed, err: %v", ErrOperation, err)
	}

	var jsonResp PollPublishJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, fmt.Errorf("%w of job status response failed, err: %v", ErrParse, err)
	}

	return &jsonResp, nil
}
