package admin

import (
	"fmt"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
)


// checks all users' Apple tokens and updates their status in the database. warns if a token is expiring soon.
func StartTokenMonitor() {
	hours := config.AppConfig.TokenCheckHours
	if hours <= 0 {
		fmt.Println("token monitor: disabled (token_check_hours <= 0)")
		return
	}
	interval := time.Duration(hours) * time.Hour
	fmt.Printf("token monitor: checking Apple tokens every %dh (warn < %dd)\n", hours, config.AppConfig.TokenWarnDays)
	go func() {
		time.Sleep(10 * time.Second) // let startup settle, then first pass
		for {
			runTokenCheck()
			time.Sleep(interval)
		}
	}()
}

func runTokenCheck() {
	var users []schema.User
	if err := db.DB.Find(&users).Error; err != nil {
		fmt.Printf("token monitor: could not list users: %v\n", err)
		return
	}
	warn := time.Duration(config.AppConfig.TokenWarnDays) * 24 * time.Hour

	for i := range users {
		u := &users[i]
		if !applemusic.HasMediaUserToken(u.AppleCookies) {
			continue
		}

		if u.AppleTokenExpiresAt != nil && warn > 0 && time.Until(*u.AppleTokenExpiresAt) < warn {
			if attemptRenew(u) {
				fmt.Printf("token monitor: silently renewed token for %q\n", u.Username)
			}
		}

		status, errMsg := validate(u.AppleCookies)
		now := time.Now()
		updates := map[string]interface{}{
			"apple_token_status":          status,
			"apple_token_last_checked_at": now,
			"apple_token_last_error":      errMsg,
		}
		if exp, ok := applemusic.MediaTokenExpiry(u.AppleCookies); ok {
			updates["apple_token_expires_at"] = exp
			u.AppleTokenExpiresAt = &exp
		}
		db.DB.Model(u).Updates(updates)

		switch {
		case status == "expired":
			fmt.Printf("token monitor: %q token EXPIRED !!! needs replenish\n", u.Username)
		case u.AppleTokenExpiresAt != nil && warn > 0 && u.AppleTokenExpiresAt.Before(now.Add(warn)):
			days := int(time.Until(*u.AppleTokenExpiresAt).Hours() / 24)
			fmt.Printf("token monitor: %q token expiring in ~%dd !!! replenish soon\n", u.Username, days)
		}
	}
}
