package main

import (
	"actionfy/db"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type createTemplateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
	IsPublic    bool            `json:"is_public"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
}

type deployBlueprintRequest struct {
	TemplateID  string            `json:"template_id"`
	WorkspaceID string            `json:"workspace_id"`
	Variables   map[string]string `json:"variables"`
}

type blueprintTask struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	TaskType            string          `json:"task_type"`
	AgentPrompt         string          `json:"agent_prompt"`
	NativeCode          string          `json:"native_code"`
	TriggerType         string          `json:"trigger_type"`
	TriggerConfig       json.RawMessage `json:"trigger_config"`
	RequiresApproval    bool            `json:"requires_approval"`
	MissedTaskPolicy    string          `json:"missed_task_policy"`
	DependsOn           string          `json:"depends_on"` // Temporary ID from blueprint
	TriggerOnCompletion bool            `json:"trigger_on_completion"`
	BranchCondition     json.RawMessage `json:"branch_condition"`
	IsBundleRoot        bool            `json:"is_bundle_root"`
}

func apiDeployBlueprintHandler(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	var req deployBlueprintRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request payload"})
	}

	var templateID pgtype.UUID
	if err := parseUUID(req.TemplateID, &templateID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid template_id"})
	}

	template, err := queries.GetTemplateByIDRaw(c.Request().Context(), templateID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Template not found"})
		}
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch template"})
	}

	var tasks []blueprintTask
	if err := json.Unmarshal(template.Config, &tasks); err != nil {
		// Try unmarshaling as a single task if array fails
		var singleTask blueprintTask
		if err2 := json.Unmarshal(template.Config, &singleTask); err2 == nil {
			tasks = []blueprintTask{singleTask}
		} else {
			return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Invalid template configuration"})
		}
	}

	var workspaceID pgtype.UUID
	if req.WorkspaceID != "" {
		if err := parseUUID(req.WorkspaceID, &workspaceID); err != nil {
			return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid workspace_id"})
		}
	}

	// Start transaction
	tx, err := dbPool.Begin(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to start transaction"})
	}
	defer tx.Rollback(c.Request().Context())

	qtx := queries.WithTx(tx)
	idMap := make(map[string]pgtype.UUID)
	createdTasks := make([]db.Task, 0, len(tasks))

	for _, bt := range tasks {
		prompt := bt.AgentPrompt
		for k, v := range req.Variables {
			prompt = strings.ReplaceAll(prompt, fmt.Sprintf("{{%s}}", k), v)
		}

		var dependsOnTaskID pgtype.UUID
		if bt.DependsOn != "" {
			if uuid, ok := idMap[bt.DependsOn]; ok {
				dependsOnTaskID = uuid
			} else {
				log.Printf("Warning: blueprint task '%s' depends on unknown ID '%s'. Ensure tasks are in topological order.", bt.Name, bt.DependsOn)
			}
		}

		policy := bt.MissedTaskPolicy
		if policy == "" {
			policy = "run_immediately"
		}

		task, err := qtx.CreateTask(c.Request().Context(), db.CreateTaskParams{
			UserID:              userID,
			Name:                bt.Name,
			TriggerType:         pgtype.Text{String: bt.TriggerType, Valid: true},
			TriggerConfig:       bt.TriggerConfig,
			AgentPrompt:         prompt,
			WorkspaceID:         workspaceID,
			TaskType:            pgtype.Text{String: bt.TaskType, Valid: bt.TaskType != ""},
			NativeCode:          pgtype.Text{String: bt.NativeCode, Valid: bt.NativeCode != ""},
			MissedTaskPolicy:    pgtype.Text{String: policy, Valid: true},
			RequiresApproval:    pgtype.Bool{Bool: bt.RequiresApproval, Valid: true},
			DependsOnTaskID:     dependsOnTaskID,
			TriggerOnCompletion: pgtype.Bool{Bool: bt.TriggerOnCompletion, Valid: true},
			BranchCondition:     bt.BranchCondition,
			IsBundleRoot:        pgtype.Bool{Bool: bt.IsBundleRoot, Valid: true},
			NextRun:             pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			log.Printf("Failed to create task in blueprint: %v", err)
			return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to deploy blueprint task"})
		}

		if bt.ID != "" {
			idMap[bt.ID] = task.ID
		}
		createdTasks = append(createdTasks, task)
	}

	// Increment uses
	if _, err := qtx.IncrementTemplateUses(c.Request().Context(), templateID); err != nil {
		log.Printf("Warning: failed to increment template uses for %s: %v", req.TemplateID, err)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to commit deployment"})
	}

	return c.JSON(http.StatusCreated, APIResponse{Success: true, Data: createdTasks})
}

func handleCreateTemplate(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var req createTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	var workspaceID pgtype.UUID
	if req.WorkspaceID != "" {
		if err := parseUUID(req.WorkspaceID, &workspaceID); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid workspace_id format"})
		}

		// Check workspace access
		hasAccess, err := queries.CheckWorkspaceAccess(c.Request().Context(), db.CheckWorkspaceAccessParams{
			ID:      workspaceID,
			OwnerID: userID,
		})
		if err != nil || !hasAccess {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: No access to workspace"})
		}
	}

	template, err := queries.CreateTemplate(c.Request().Context(), db.CreateTemplateParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Config:      req.Config,
		IsPublic:    pgtype.Bool{Bool: req.IsPublic, Valid: true},
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create template"})
	}

	return c.JSON(http.StatusCreated, template)
}

func handleListPublicTemplates(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	search := c.QueryParam("search")
	searchParam := "%" + search + "%"

	templates, err := queries.ListPublicTemplates(c.Request().Context(), searchParam)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to list public templates"})
	}
	if templates == nil {
		templates = []db.Template{}
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: templates})
}

func handleIncrementTemplateUses(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	templateIDStr := c.Param("id")
	var templateID pgtype.UUID
	if err := parseUUID(templateIDStr, &templateID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid template ID"})
	}

	uses, err := queries.IncrementTemplateUses(c.Request().Context(), templateID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: "Template not found"})
		}
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to increment uses"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]int32{"uses_count": uses.Int32}})
}
