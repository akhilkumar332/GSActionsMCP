package main

import (
	"net/http"

	"actionfy/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func handleGetWorkspaces(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	workspaces, err := queries.GetUserWorkspaces(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch workspaces"})
	}
	if workspaces == nil {
		workspaces = []db.Workspace{}
	}
	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: workspaces})
}

func handleCreateWorkspace(c echo.Context) error {
	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Workspace name is required"})
	}

	workspace, err := queries.CreateWorkspace(c.Request().Context(), db.CreateWorkspaceParams{
		Name:    input.Name,
		OwnerID: userID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to create workspace"})
	}

	return c.JSON(http.StatusCreated, APIResponse{Success: true, Data: workspace})
}

func handleListWorkspaceEnvVars(c echo.Context) error {
	workspaceIDStr := c.Param("id")
	var workspaceID pgtype.UUID
	if err := parseUUID(workspaceIDStr, &workspaceID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid workspace ID"})
	}

	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	// Verify access
	hasAccess, err := queries.CheckWorkspaceAccess(c.Request().Context(), db.CheckWorkspaceAccessParams{
		ID:      workspaceID,
		OwnerID: userID,
	})
	if err != nil || !hasAccess {
		return c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: "Forbidden"})
	}

	envVars, err := queries.ListWorkspaceEnvVars(c.Request().Context(), workspaceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to fetch environment variables"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: envVars})
}

func handleUpsertWorkspaceEnvVar(c echo.Context) error {
	workspaceIDStr := c.Param("id")
	var workspaceID pgtype.UUID
	if err := parseUUID(workspaceIDStr, &workspaceID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid workspace ID"})
	}

	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	// Verify access
	hasAccess, err := queries.CheckWorkspaceAccess(c.Request().Context(), db.CheckWorkspaceAccessParams{
		ID:      workspaceID,
		OwnerID: userID,
	})
	if err != nil || !hasAccess {
		return c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: "Forbidden"})
	}

	var input struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request body"})
	}

	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Name is required"})
	}

	envVar, err := queries.UpsertWorkspaceEnvVar(c.Request().Context(), db.UpsertWorkspaceEnvVarParams{
		WorkspaceID: workspaceID,
		Name:        input.Name,
		Value:       input.Value,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to upsert environment variable"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Data: envVar})
}

func handleDeleteWorkspaceEnvVar(c echo.Context) error {
	workspaceIDStr := c.Param("id")
	var workspaceID pgtype.UUID
	if err := parseUUID(workspaceIDStr, &workspaceID); err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid workspace ID"})
	}

	name := c.Param("name")
	if name == "" {
		return c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: "Name is required"})
	}

	userID := getUserID(c)
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: "Unauthorized"})
	}

	// Verify access
	hasAccess, err := queries.CheckWorkspaceAccess(c.Request().Context(), db.CheckWorkspaceAccessParams{
		ID:      workspaceID,
		OwnerID: userID,
	})
	if err != nil || !hasAccess {
		return c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: "Forbidden"})
	}

	err = queries.DeleteWorkspaceEnvVar(c.Request().Context(), db.DeleteWorkspaceEnvVarParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to delete environment variable"})
	}

	return c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Environment variable deleted"})
}
