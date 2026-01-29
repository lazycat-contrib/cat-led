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
	BulbStyle string `json:"bulb_style" binding:"required,oneof=classic lava vintage liquid lightbulb analog"`
}

// UserPreferenceResponse represents the user preference data returned to the client.
type UserPreferenceResponse struct {
	UserID    string `json:"user_id"`
	BulbStyle string `json:"bulb_style"`
	UpdatedAt string `json:"updated_at"`
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
		UserID:    pref.UserID,
		BulbStyle: pref.BulbStyle,
		UpdatedAt: pref.UpdatedAt.Format(time.RFC3339),
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

	var req UserPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := scheduleUseCase.GetClient()
	pref, err := client.UserPreference.
		Query().
		Where(userpreference.UserID(basicInfo.UserId)).
		Only(ctx)

	now := time.Now()

	if err != nil {
		if ent.IsNotFound(err) {
			// Create new preference
			pref, err = client.UserPreference.
				Create().
				SetUserID(basicInfo.UserId).
				SetBulbStyle(req.BulbStyle).
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user preference"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user preference"})
			return
		}
	} else {
		// Update existing preference
		pref, err = pref.Update().
			SetBulbStyle(req.BulbStyle).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user preference"})
			return
		}
	}

	c.JSON(http.StatusOK, UserPreferenceResponse{
		UserID:    pref.UserID,
		BulbStyle: pref.BulbStyle,
		UpdatedAt: pref.UpdatedAt.Format(time.RFC3339),
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
			return client.UserPreference.
				Create().
				SetUserID(userID).
				SetBulbStyle("classic").
				SetCreatedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
		}
		return nil, err
	}

	return pref, nil
}
