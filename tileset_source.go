package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateTilesetSource creates a new tileset source.
// This function simply calls PutTilesetSource.
//
// The provided JSON path should point to a file containing one GeoJSON
// feature object per line.
func (c *Client) CreateTilesetSource(ctx context.Context, tilesetID string, jsonReader io.Reader) (NewTilesetSourceResponse, error) {
	return c.PutTilesetSource(ctx, tilesetID, jsonReader)
}

type NewTilesetSourceResponse struct {
	// File size in bytes.
	FileSizeBytes int `json:"file_size"`

	// Number of files in tileset.
	Files int `json:"files"`

	// Unique identifier for the tileset source.
	ID string `json:"id"`

	// Total size in bytes of all the files in the tileset source.
	SourceSize int `json:"source_size"`
}

// PutTilesetSource replaces a tileset source with new data, or creates
// a new tileset source if it does not already exist.
//
// The provided path should point to a file containing one GeoJSON
// feature object per line.
func (c *Client) PutTilesetSource(ctx context.Context, tilesetID string, jsonReader io.Reader) (NewTilesetSourceResponse, error) {
	url := baseURL +
		"/tilesets/v1/sources/" + c.username + "/" + tilesetID +
		"?access_token=" + c.accessToken

	var jsonResp NewTilesetSourceResponse
	resp, err := putMultipart(ctx, c.httpClient, url, "filenamedoesntmatter", jsonReader)
	if err != nil {
		return jsonResp, fmt.Errorf("upload %w failed, %v", ErrOperation, err)
	}

	if resp.StatusCode != http.StatusOK {
		return jsonResp, fmt.Errorf("upload %w failed, non-200 response: %v", ErrOperation, resp.StatusCode)
	}

	if err = json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return jsonResp, fmt.Errorf("%w failed, err: %v", ErrParse, err)
	}

	return jsonResp, nil
}
