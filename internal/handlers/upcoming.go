package handlers

import (
	"cmp"
	"slices"
	"time"

	"cat-led/internal/ent"
	"github.com/gin-gonic/gin"
)

type upcomingEvent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Operation string    `json:"operation"`
	At        time.Time `json:"at"`
}

// nextScheduleTime uses the scheduler's server-local calendar and weekday rules.
func nextScheduleTime(s *ent.Schedule, now time.Time) time.Time {
	if !s.Enabled {
		return time.Time{}
	}
	// Walk actual minutes so repeated and skipped DST wall times follow the
	// same clock as the scheduler. Eight calendar days cover every weekday.
	end := now.AddDate(0, 0, 8)
	for at := now.Truncate(time.Minute).Add(time.Minute); at.Before(end); at = at.Add(time.Minute) {
		if at.Hour() == s.Hour && at.Minute() == s.Minute &&
			(len(s.WeekDays) == 0 || slices.Contains(s.WeekDays, int(at.Weekday()))) {
			return at
		}
	}

	return time.Time{}
}

func upcomingSchedules(all []*ent.Schedule, userID string, now time.Time) []upcomingEvent {
	events := make([]upcomingEvent, 0)
	for _, s := range all {
		if s.Creator != userID && !s.AllowEditByOthers {
			continue
		}
		if at := nextScheduleTime(s, now); !at.IsZero() {
			events = append(events, upcomingEvent{ID: s.ID.String(), Name: s.Name, Operation: string(s.Operation), At: at})
		}
	}
	slices.SortFunc(events, func(a, b upcomingEvent) int { return cmp.Compare(a.At.Unix(), b.At.Unix()) })
	return events
}

func GetUpcomingEvents(c *gin.Context) {
	if !requireScheduleUseCase(c) {
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	all, err := scheduleUseCase.GetAllSchedules(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": "获取定时任务失败"})
		return
	}
	now := time.Now()
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"server_time": now, "events": upcomingSchedules(all, userID, now)})
}
