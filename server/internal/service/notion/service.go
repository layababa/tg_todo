package notion

import (
	"context"
	"errors"
	"fmt"

	"github.com/dstotijn/go-notion"
	"github.com/layababa/tg_todo/server/internal/repository"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DatabaseSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	Icon      string `json:"icon,omitempty"`
}

type ValidationResult struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	RequiredFields []string `json:"required_fields"`
	MissingFields  []string `json:"missing_fields"`
	Compatible     bool     `json:"compatible"`
}

type Service struct {
	logger        *zap.Logger
	userRepo      repository.UserRepository
	encryptionKey string
	// Allow mocking the client creation
	ClientFactory func(token string) pkgnotion.Client
}

func NewService(logger *zap.Logger, userRepo repository.UserRepository, encryptionKey string) *Service {
	return &Service{
		logger:        logger,
		userRepo:      userRepo,
		encryptionKey: encryptionKey,
		ClientFactory: pkgnotion.NewClient,
	}
}

func (s *Service) getClient(ctx context.Context, userID string) (pkgnotion.Client, error) {
	token, err := s.userRepo.FindNotionToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find notion token: %w", err)
	}

	accessToken, err := security.Decrypt(token.AccessTokenEnc, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	return s.ClientFactory(accessToken), nil
}

func (s *Service) ListDatabases(ctx context.Context, userID string, query string) ([]DatabaseSummary, error) {
	client, err := s.getClient(ctx, userID)
	if err != nil {
		// If user hasn't connected Notion, return empty list instead of error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []DatabaseSummary{}, nil
		}
		return nil, err
	}

	resp, err := client.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("notion search failed: %w", err)
	}

	var summaries []DatabaseSummary
	for _, result := range resp.Results {
		var db *notion.Database
		switch v := result.(type) {
		case *notion.Database:
			db = v
		case notion.Database:
			db = &v
		default:
			continue
		}

		title := ""
		if len(db.Title) > 0 {
			title = db.Title[0].PlainText
		}

		icon := ""
		if db.Icon != nil {
			if db.Icon.Type == "emoji" && db.Icon.Emoji != nil {
				icon = *db.Icon.Emoji
			} else if db.Icon.Type == "external" && db.Icon.External != nil {
				icon = db.Icon.External.URL
			} else if db.Icon.Type == "file" && db.Icon.File != nil {
				icon = db.Icon.File.URL
			}
		}

		workspace := "Notion Workspace" // dynamic fetch difficult without granular parent info logic

		summaries = append(summaries, DatabaseSummary{
			ID:        db.ID,
			Name:      title,
			Workspace: workspace,
			Icon:      icon,
		})
	}

	return summaries, nil
}

func (s *Service) ValidateDatabase(ctx context.Context, userID string, dbID string) (*ValidationResult, error) {
	client, err := s.getClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	db, err := client.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	title := ""
	if len(db.Title) > 0 {
		title = db.Title[0].PlainText
	}

	required := []string{"Status", "Assignee", "Date"}
	missing := []string{}

	// Check fields
	// Status: select or status
	if prop, ok := db.Properties["Status"]; !ok {
		missing = append(missing, "Status")
	} else {
		if prop.Type != notion.DBPropTypeSelect && prop.Type != notion.DBPropTypeStatus {
			missing = append(missing, "Status (Expected Select/Status)")
		}
	}

	// Assignee: people
	if prop, ok := db.Properties["Assignee"]; !ok {
		missing = append(missing, "Assignee")
	} else {
		if prop.Type != notion.DBPropTypePeople {
			missing = append(missing, "Assignee (Expected People)")
		}
	}

	// Date: date
	if prop, ok := db.Properties["Date"]; !ok {
		missing = append(missing, "Date")
	} else {
		if prop.Type != notion.DBPropTypeDate {
			missing = append(missing, "Date (Expected Date)")
		}
	}

	return &ValidationResult{
		ID:             db.ID,
		Name:           title,
		RequiredFields: required,
		MissingFields:  missing,
		Compatible:     len(missing) == 0,
	}, nil
}

type InitResult struct {
	Initialized   bool     `json:"initialized"`
	CreatedFields []string `json:"created_fields"`
}

func (s *Service) InitializeDatabase(ctx context.Context, userID string, dbID string) (*InitResult, error) {
	client, err := s.getClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	db, err := client.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	missing := []string{}
	properties := make(map[string]*notion.DatabaseProperty)

	// Status
	if _, ok := db.Properties["Status"]; !ok {
		missing = append(missing, "Status")
		properties["Status"] = &notion.DatabaseProperty{
			Type: notion.DBPropTypeStatus,
			Status: &notion.StatusMetadata{
				Options: []notion.SelectOptions{
					{Name: "To Do", Color: "red"},
					{Name: "In Progress", Color: "blue"},
					{Name: "Done", Color: "green"},
				},
			},
		}
	}

	// Assignee
	if _, ok := db.Properties["Assignee"]; !ok {
		missing = append(missing, "Assignee")
		properties["Assignee"] = &notion.DatabaseProperty{
			Type:   notion.DBPropTypePeople,
			People: &notion.EmptyMetadata{}, // People property config is empty object
		}
	}

	// Date
	if _, ok := db.Properties["Date"]; !ok {
		missing = append(missing, "Date")
		properties["Date"] = &notion.DatabaseProperty{
			Type: notion.DBPropTypeDate,
			Date: &notion.EmptyMetadata{},
		}
	}

	if len(missing) == 0 {
		return &InitResult{Initialized: false, CreatedFields: []string{}}, nil
	}

	_, err = client.UpdateDatabase(ctx, dbID, notion.UpdateDatabaseParams{
		Properties: properties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update database properties: %w", err)
	}

	return &InitResult{
		Initialized:   true,
		CreatedFields: missing,
	}, nil
}
