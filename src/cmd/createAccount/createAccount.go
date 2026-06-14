// cmd/createAccount
package createAccount

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DwifteJB/aplsonic/src/applemusic"
	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const loginURL = "https://music.apple.com"

// TODO: we need a easier way, maybe have users do tokens themselves, for now rod is fineee
// also, it would be good to verify cookies on every login tbf

func CMD(otherArgs []string) {
	dsn := config.GenerateDSN()
	if err := db.Connect(dsn); err != nil {
		panic(err)
	}

	cookies, err := captureAppleCookies()
	if err != nil {
		panic(fmt.Errorf("cookie capture failed: %w", err))
	}

	cookieContent := applemusic.RenderNetscape(cookies)

	if err := os.WriteFile("cookies.txt", []byte(cookieContent), 0600); err != nil {
		panic(fmt.Errorf("failed to write cookies.txt: %w", err))
	}
	fmt.Println("Cookies saved to cookies.txt")

	if err := createUser(cookieContent); err != nil {
		panic(fmt.Errorf("failed to create user: %w", err))
	}
}

func captureAppleCookies() ([]applemusic.Cookie, error) {
	fmt.Println("Opening Apple Music - sign in, then close the browser window when done.")

	u := launcher.New().Headless(false).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(loginURL)

	// wait until a media token appears
	for {
		time.Sleep(500 * time.Millisecond)
		raw, err := page.Cookies([]string{})
		if err != nil {
			break
		}
		for _, c := range raw {
			if c.Name == "media-user-token" {
				fmt.Println("Sign-in detected.")
				goto done
			}
		}
	}

	// did not know golang had goto lol
done:
	raw, err := page.Cookies([]string{})
	if err != nil {
		return nil, fmt.Errorf("could not read cookies: %w", err)
	}

	entries := make([]applemusic.Cookie, 0, len(raw))
	for _, c := range raw {
		entries = append(entries, applemusic.Cookie{
			Domain:  c.Domain,
			Path:    c.Path,
			Secure:  c.Secure,
			Expires: int64(c.Expires),
			Name:    c.Name,
			Value:   c.Value,
		})
	}
	return entries, nil
}

func createUser(cookieContent string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Username: ")
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("Email: ")
	scanner.Scan()
	email := strings.TrimSpace(scanner.Text())

	fmt.Print("Password: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)

	sum := sha256.Sum256([]byte(password + salt))
	hashed := hex.EncodeToString(sum[:])

	user := &schema.User{
		Username:      username,
		Email:         email,
		Password:      hashed,
		Salt:          salt,
		TokenPassword: password,
		AppleCookies:  cookieContent,
	}

	if result := db.DB.Create(user); result.Error != nil {
		return fmt.Errorf("db insert failed: %w", result.Error)
	}

	fmt.Printf("Account created (id=%d)\n", user.ID)
	return nil
}
