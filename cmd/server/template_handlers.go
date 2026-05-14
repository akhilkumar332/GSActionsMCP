package main

import (
	"encoding/json"
	"net/http"
	"schedule-mcp/db"

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
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to increment uses"})
	}
	
	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]int32{"uses_count": uses.Int32}})
}
