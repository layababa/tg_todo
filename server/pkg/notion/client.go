package notion

import (
	"context"
	"time"

	"github.com/dstotijn/go-notion"
)

// Client defines the interface for Notion API interactions
type Client interface {
	CreatePage(ctx context.Context, params CreatePageParams) (*notion.Page, error)
	Search(ctx context.Context, query string) (*notion.SearchResponse, error)
	GetDatabase(ctx context.Context, id string) (*notion.Database, error)
	UpdateDatabase(ctx context.Context, id string, params notion.UpdateDatabaseParams) (*notion.Database, error)
	QueryDatabase(ctx context.Context, id string, params *notion.DatabaseQuery) (*notion.DatabaseQueryResponse, error)
	UpdatePage(ctx context.Context, pageID string, params UpdatePageParams) (*notion.Page, error)
}

// CreatePageParams holds parameters for creating a page
type CreatePageParams struct {
	DatabaseID string
	Title      string
	Status     string         // "To Do", "In Progress", "Done"
	Assignees  []string       // Notion User IDs
	Children   []notion.Block // Page content (blocks)
}

// UpdatePageParams holds parameters for updating a page
type UpdatePageParams struct {
	Title  *string
	Status *string // "To Do", "In Progress", "Done"
}

// clientWrapper wraps dstotijn/go-notion client
type clientWrapper struct {
	api *notion.Client
}

// NewClient creates a new Notion client with the given access token
func NewClient(accessToken string) Client {
	return &clientWrapper{
		api: notion.NewClient(accessToken),
	}
}

// CreatePage creates a new page in the specified database
func (c *clientWrapper) CreatePage(ctx context.Context, params CreatePageParams) (*notion.Page, error) {
	// 1. Build Properties
	props := notion.DatabasePageProperties{
		"Name": notion.DatabasePageProperty{
			Title: []notion.RichText{
				{
					Text: &notion.Text{
						Content: params.Title,
					},
				},
			},
		},
		"Status": notion.DatabasePageProperty{
			Status: &notion.SelectOptions{
				Name: params.Status,
			},
		},
	}

	// 2. Use Children from params
	children := params.Children

	// 3. Create Request
	req := notion.CreatePageParams{
		ParentType:             notion.ParentTypeDatabase,
		ParentID:               params.DatabaseID,
		DatabasePageProperties: &props,
		Children:               children,
	}

	// 4. Call API
	// 4. Call API with Retry
	return Retry(ctx, func() (*notion.Page, error) {
		page, err := c.api.CreatePage(ctx, req)
		if err != nil {
			return nil, err
		}
		return &page, nil
	})
}

// Search searches for databases
func (c *clientWrapper) Search(ctx context.Context, query string) (*notion.SearchResponse, error) {
	req := &notion.SearchOpts{
		Query: query,
		Filter: &notion.SearchFilter{
			Property: "object",
			Value:    "database",
		},
		// Removed sort to avoid guessing constants. Default sort is fine for now.
	}
	return Retry(ctx, func() (*notion.SearchResponse, error) {
		resp, err := c.api.Search(ctx, req)
		if err != nil {
			return nil, err
		}
		return &resp, nil
	})
}

// GetDatabase retrieves a database by ID
func (c *clientWrapper) GetDatabase(ctx context.Context, id string) (*notion.Database, error) {
	return Retry(ctx, func() (*notion.Database, error) {
		db, err := c.api.FindDatabaseByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &db, nil
	})
}

// UpdateDatabase updates a database (e.g., adding properties)
func (c *clientWrapper) UpdateDatabase(ctx context.Context, id string, params notion.UpdateDatabaseParams) (*notion.Database, error) {
	return Retry(ctx, func() (*notion.Database, error) {
		db, err := c.api.UpdateDatabase(ctx, id, params)
		if err != nil {
			return nil, err
		}
		return &db, nil
	})
}

// QueryDatabase queries a database
func (c *clientWrapper) QueryDatabase(ctx context.Context, id string, params *notion.DatabaseQuery) (*notion.DatabaseQueryResponse, error) {
	return Retry(ctx, func() (*notion.DatabaseQueryResponse, error) {
		resp, err := c.api.QueryDatabase(ctx, id, params)
		if err != nil {
			return nil, err
		}
		return &resp, nil
	})
}

// UpdatePage updates a page properties
func (c *clientWrapper) UpdatePage(ctx context.Context, pageID string, params UpdatePageParams) (*notion.Page, error) {
	props := make(notion.DatabasePageProperties)

	if params.Title != nil {
		props["Name"] = notion.DatabasePageProperty{
			Title: []notion.RichText{
				{
					Text: &notion.Text{
						Content: *params.Title,
					},
				},
			},
		}
	}

	if params.Status != nil {
		props["Status"] = notion.DatabasePageProperty{
			Status: &notion.SelectOptions{
				Name: *params.Status,
			},
		}
	}

	req := notion.UpdatePageParams{
		DatabasePageProperties: props,
	}

	return Retry(ctx, func() (*notion.Page, error) {
		page, err := c.api.UpdatePage(ctx, pageID, req)
		if err != nil {
			return nil, err
		}
		return &page, nil
	})
}

// Retry is a helper to retry operations with exponential backoff
func Retry[T any](ctx context.Context, op func() (T, error)) (T, error) {
	var result T
	var err error

	// Max 3 retries (total 4 attempts)
	maxRetries := 3
	backoff := 500 * time.Millisecond

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(backoff):
				// Exponential backoff
				backoff *= 2
			}
		}

		result, err = op()
		if err == nil {
			return result, nil
		}

		// Check if we should retry
		// Retry on temporary network errors or 429/5xx if possible to detect.
		// For now simple retry on any error for MVP, maybe refined later.
		// TODO: Refine retry conditions.
	}
	return result, err
}
