package handlers

import (
	"context"
	"net/http"
	"time"

	"cat-led/internal/ent"
	"cat-led/internal/ent/userpreference"

	"github.com/gin-gonic/gin"
)

// UserPreferenceRequest represents the request body for updating user preferences.
type UserPreferenceRequest struct {
	BulbStyle        *string `json:"bulb_style" binding:"omitempty,oneof=classic lava vintage liquid lightbulb analog single-led neon-switch fox-daynight"`
	ShowSchedules    *bool   `json:"show_schedules"`
	RemindersEnabled *bool   `json:"reminders_enabled"`
	ReminderMinutes  *int    `json:"reminder_minutes" binding:"omitempty,min=1,max=1440"`
}

// UserPreferenceResponse represents the user preference data returned to the client.
type UserPreferenceResponse struct {
	UserID           string `json:"user_id"`
	BulbStyle        string `json:"bulb_style"`
	UpdatedAt        string `json:"updated_at"`
	ShowSchedules    bool   `json:"show_schedules"`
	RemindersEnabled bool   `json:"reminders_enabled"`
	ReminderMinutes  int    `json:"reminder_minutes"`
}

// GetUserPreference returns the current user's preference settings.
func GetUserPreference(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	ctx := c.Request.Context()
	basicInfo := extractBasicInfo(c)

	if basicInfo.UserId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	client := scheduleUseCase.GetClient()
	pref, err := getUserPreferenceOrCreate(ctx, client, basicInfo.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user preference"})
		return
	}

	c.JSON(http.StatusOK, UserPreferenceResponse{
		UserID:           pref.UserID,
		BulbStyle:        pref.BulbStyle,
		UpdatedAt:        pref.UpdatedAt.Format(time.RFC3339),
		ShowSchedules:    pref.ShowSchedules,
		RemindersEnabled: pref.RemindersEnabled,
		ReminderMinutes:  pref.ReminderMinutes,
	})
}

// UpdateUserPreference updates the current user's preference settings.
func UpdateUserPreference(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}

	ctx := c.Request.Context()
	basicInfo := extractBasicInfo(c)

	if basicInfo.UserId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var req UserPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := scheduleUseCase.GetClient()
	pref, err := getUserPreferenceOrCreate(ctx, client, basicInfo.UserId)
	if err == nil {
		pref, err = pref.Update().
			SetNillableBulbStyle(req.BulbStyle).
			SetNillableShowSchedules(req.ShowSchedules).
			SetNillableRemindersEnabled(req.RemindersEnabled).
			SetNillableReminderMinutes(req.ReminderMinutes).
			SetUpdatedAt(time.Now()).Save(ctx)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user preference"})
		return
	}

	c.JSON(http.StatusOK, UserPreferenceResponse{
		UserID:           pref.UserID,
		BulbStyle:        pref.BulbStyle,
		UpdatedAt:        pref.UpdatedAt.Format(time.RFC3339),
		ShowSchedules:    pref.ShowSchedules,
		RemindersEnabled: pref.RemindersEnabled,
		ReminderMinutes:  pref.ReminderMinutes,
	})
}

// getUserPreferenceOrCreate gets existing preference or creates a default one.
func getUserPreferenceOrCreate(ctx context.Context, client *ent.Client, userID string) (*ent.UserPreference, error) {
	pref, err := client.UserPreference.
		Query().
		Where(userpreference.UserID(userID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// Create default preference
			now := time.Now()
			created, createErr := client.UserPreference.
				Create().
				SetUserID(userID).
				SetBulbStyle("classic").
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			if ent.IsConstraintError(createErr) {
				return client.UserPreference.Query().Where(userpreference.UserID(userID)).Only(ctx)
			}
			return created, createErr
		}
		return nil, err
	}

	return pref, nil
}
