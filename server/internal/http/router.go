package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zz/tg_todo/server/internal/auth"
	"github.com/zz/tg_todo/server/internal/task"
)

// NewRouter wires the HTTP routes needed by Mini App 前端.
func NewRouter(taskSvc *task.Service, validator *auth.Validator, serviceToken string) http.Handler {
	mux := http.NewServeMux()
	authWrapper := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			initData, err := authorizeRequest(r, validator, serviceToken)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
				return
			}
			if initData != nil {
				ctx := context.WithValue(r.Context(), initDataContextKey, initData)
				next(w, r.WithContext(ctx))
				return
			}
			next(w, r)
		}
	}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/tasks", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := taskSvc.List(r.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var payload createTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			input, err := payload.toInput()
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			actor := actorFromContext(r.Context())
			if actor != nil {
				input.Creator = *actor
			} else if strings.TrimSpace(input.Creator.ID) == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "creator is required"})
				return
			}
			created, err := taskSvc.Create(r.Context(), input)
			if err != nil {
				if errors.Is(err, task.ErrInvalidInput) {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Location", "/tasks/"+created.ID)
			writeJSON(w, http.StatusCreated, created)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/tasks/", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")
		if id == "" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			task, err := taskSvc.Get(r.Context(), id)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, task)
		case http.MethodPatch:
			var payload struct {
				Title  *string      `json:"title"`
				Status *task.Status `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			actor := actorFromContext(r.Context())
			updated, err := taskSvc.Update(r.Context(), id, actor, payload.Title, payload.Status)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			actor := actorFromContext(r.Context())
			if err := taskSvc.Delete(r.Context(), id, actor); err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusNoContent, nil)
		default:
			http.NotFound(w, r)
		}
	}))

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

type contextKey string

var initDataContextKey = contextKey("telegramInitData")

func authorizeRequest(r *http.Request, validator *auth.Validator, serviceToken string) (*auth.InitData, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if token := strings.TrimPrefix(authHeader, "Bearer "); token != authHeader {
		if serviceToken != "" && strings.TrimSpace(token) == serviceToken {
			return nil, nil
		}
	}
	if validator == nil {
		return nil, fmt.Errorf("auth disabled")
	}
	initDataHeader := strings.TrimSpace(r.Header.Get("X-Telegram-Init-Data"))
	if initDataHeader == "" {
		return nil, fmt.Errorf("missing telegram init data")
	}
	data, err := validator.Validate(initDataHeader)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func actorFromContext(ctx context.Context) *task.Person {
	value := ctx.Value(initDataContextKey)
	if value == nil {
		return nil
	}
	initData, ok := value.(*auth.InitData)
	if !ok || initData == nil {
		return nil
	}
	if initData.User.ID == 0 {
		return nil
	}
	return &task.Person{
		ID:          strconv.FormatInt(initData.User.ID, 10),
		DisplayName: userDisplayName(initData.User),
		Username:    initData.User.Username,
		AvatarURL:   initData.User.PhotoURL,
	}
}

func userDisplayName(user auth.TelegramUser) string {
	name := strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " "))
	if name != "" {
		return name
	}
	if user.Username != "" {
		return user.Username
	}
	return strconv.FormatInt(user.ID, 10)
}

type personPayload struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username"`
	AvatarURL   string `json:"avatarUrl"`
}

func (p personPayload) toPerson() task.Person {
	return task.Person{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Username:    p.Username,
		AvatarURL:   p.AvatarURL,
	}
}

type telegramMessagePayload struct {
	ChatID    int64 `json:"chatId"`
	MessageID int64 `json:"messageId"`
}

type createTaskRequest struct {
	Title            string                  `json:"title"`
	Description      *string                 `json:"description"`
	Creator          *personPayload          `json:"creator"`
	Assignees        []personPayload         `json:"assignees"`
	SourceMessageURL *string                 `json:"sourceMessageUrl"`
	Status           *task.Status            `json:"status"`
	TelegramMessage  *telegramMessagePayload `json:"telegramMessage"`
}

func (req createTaskRequest) toInput() (task.CreateInput, error) {
	input := task.CreateInput{
		Title:            req.Title,
		Description:      req.Description,
		SourceMessageURL: req.SourceMessageURL,
	}
	if req.Creator != nil {
		input.Creator = req.Creator.toPerson()
	}
	if req.Status != nil {
		input.Status = *req.Status
	}
	if len(req.Assignees) > 0 {
		input.Assignees = make([]task.Person, 0, len(req.Assignees))
		for _, person := range req.Assignees {
			input.Assignees = append(input.Assignees, person.toPerson())
		}
	}
	if req.TelegramMessage != nil {
		input.TelegramMessage = &task.TelegramMessageRef{
			ChatID:    req.TelegramMessage.ChatID,
			MessageID: req.TelegramMessage.MessageID,
		}
	}
	return input, nil
}
