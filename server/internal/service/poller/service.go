package poller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dstotijn/go-notion"
	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
	"github.com/layababa/tg_todo/server/internal/service/task"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
)

// Poller syncs data from Notion to local DB
type Poller struct {
	groupRepo     repository.GroupRepository
	taskService   *task.Service
	notionService *notionsvc.Service // For listing bounds
	notionClient  func(token string) pkgnotion.Client

	encryptionKey string
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewPoller creates a new poller
func NewPoller(
	groupRepo repository.GroupRepository,
	taskService *task.Service,
	notionService *notionsvc.Service,
	encryptionKey string,
) *Poller {
	return &Poller{
		groupRepo:     groupRepo,
		taskService:   taskService,
		notionService: notionService,
		encryptionKey: encryptionKey,
		notionClient:  pkgnotion.NewClient,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		slog.Info("Notion Poller started", "interval", "1m")

		// Initial poll
		p.pollAll(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopChan:
				return
			case <-ticker.C:
				p.pollAll(ctx)
			}
		}
	}()
}

// Stop stops the poller
func (p *Poller) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

func (p *Poller) pollAll(ctx context.Context) {
	// 1. List all active bindings
	// We need a way to list all bound databases.
	// GroupRepository might not support listing all groups efficiently, but we can list groups with non-null NotionDBID?
	// The current GroupRepository interface has GetByID and GetByChatID.
	// We might need ListWithNotionBindings() in GroupRepository.
	// For now let's assume we can add it or reuse something.
	// Ah, I don't have ListAll in GroupRepository.
	// I'll implement a scan logic or add it.
	// Let's assume we add ListAllActiveBindings to GroupRepo.
	// Wait, I just modified TaskRepo. I should modifying GroupRepo too if needed.
	// Let's check GroupRepo.

	// Assuming p.groupRepo.ListWithBindings(ctx)
	// If not available, I'll log TODO and skip for this iteration of implementation until I add it.
	// Actually, this is critical.

	slog.Info("Notion Poller: Polling cycle started")

	groups, err := p.groupRepo.ListWithActiveBindings(ctx)
	if err != nil {
		slog.Error("Notion Poller: Failed to list groups", "error", err)
		return
	}

	for _, g := range groups {
		if g.DatabaseID == nil || g.NotionAccessToken == "" {
			continue
		}

		if err := p.pollDatabase(ctx, g); err != nil {
			slog.Error("Notion Poller: Failed to poll database", "group_id", g.ID, "db_id", *g.DatabaseID, "error", err)
		}
	}
}

func (p *Poller) pollDatabase(ctx context.Context, group models.Group) error {
	// Decrypt token
	token, err := security.Decrypt(group.NotionAccessToken, p.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt token: %w", err)
	}

	client := p.notionClient(token)
	dbID := *group.DatabaseID

	// Query Modified Pages (e.g., last 2 min to be safe?)
	// Or just query all open tasks?
	// Better to query recent modifications.
	// Notion Filter: "Last edited time" is on or after (now - 2m).

	cutoff := time.Now().Add(-2 * time.Minute)

	query := &notion.DatabaseQuery{
		Filter: &notion.DatabaseQueryFilter{
			Timestamp: "last_edited_time",
			DatabaseQueryPropertyFilter: notion.DatabaseQueryPropertyFilter{
				LastEditedTime: &notion.DatePropertyFilter{
					OnOrAfter: &cutoff,
				},
			},
		},
		PageSize: 100,
	}

	resp, err := client.QueryDatabase(ctx, dbID, query)
	if err != nil {
		return fmt.Errorf("query notion: %w", err)
	}

	for _, page := range resp.Results {
		if err := p.syncPage(ctx, page, dbID); err != nil {
			slog.Error("Notion Poller: Failed to sync page", "page_id", page.ID, "error", err)
		}
	}

	return nil
}

func (p *Poller) syncPage(ctx context.Context, page notion.Page, dbID string) error {
	// Extract props
	props := page.Properties.(notion.DatabasePageProperties)

	// Title
	var title string
	if prop, ok := props["Name"]; ok && len(prop.Title) > 0 {
		title = prop.Title[0].PlainText
	} else {
		return nil // Skip empty title?
	}

	// Status
	var status string
	if prop, ok := props["Status"]; ok && prop.Status != nil {
		status = prop.Status.Name
	}
	// Fallback to "To Do" if status missing? Or ignore?
	if status == "" {
		status = "To Do"
	}

	// URL
	notionURL := page.URL

	// Assignees?? (Skipped for now)

	// Assignees?? (Skipped for now)

	return p.taskService.SyncTaskFromNotion(ctx, page.ID, dbID, title, status, notionURL, nil, page.Archived)
}
