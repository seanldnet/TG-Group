package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/seanldnet/tg-group/internal/database"
)

func TestRegistrationSessionLifecyclePostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" { t.Skip("TEST_DATABASE_URL is not configured") }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Open(ctx, databaseURL)
	if err != nil { t.Fatalf("open test database: %v", err) }
	defer db.Close()
	if err := db.Migrate(ctx); err != nil { t.Fatalf("apply migrations: %v", err) }

	service := NewService(db.Pool)
	email := fmt.Sprintf("auth-%d@example.test", time.Now().UnixNano())
	registered, err := service.Register(ctx,email,"correct-horse-battery","Integration Test","","127.0.0.1:4321")
	if err != nil { t.Fatalf("register: %v",err) }
	if registered.User.Email != email || registered.User.TenantID == "" || registered.Token == "" {
		t.Fatalf("unexpected registration response: %+v",registered)
	}

	if _, err := service.Register(ctx,email,"correct-horse-battery","Duplicate","",""); !errors.Is(err,ErrEmailTaken) {
		t.Fatalf("duplicate registration error = %v, want ErrEmailTaken",err)
	}
	authenticated, err := service.Authenticate(ctx,registered.Token)
	if err != nil { t.Fatalf("authenticate registered session: %v",err) }
	if authenticated.ID != registered.User.ID || authenticated.TenantID != registered.User.TenantID {
		t.Fatalf("authenticated user mismatch: %+v",authenticated)
	}

	if err := service.Logout(ctx,registered.Token); err != nil { t.Fatalf("logout: %v",err) }
	if _, err := service.Authenticate(ctx,registered.Token); !errors.Is(err,ErrUnauthorized) {
		t.Fatalf("authenticate revoked token error = %v, want ErrUnauthorized",err)
	}

	loggedIn, err := service.Login(ctx,email,"correct-horse-battery","","127.0.0.1:4321")
	if err != nil { t.Fatalf("login: %v",err) }
	if _, err := service.Authenticate(ctx,loggedIn.Token); err != nil { t.Fatalf("authenticate login session: %v",err) }
}
