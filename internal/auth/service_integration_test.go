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

    if _,err:=service.CreateTelegramLinkChallenge(ctx,registered.User.ID);err!=nil { t.Fatalf("create Telegram link challenge: %v",err) }
    challenge,err:=service.CreateTelegramLinkChallenge(ctx,registered.User.ID)
    if err!=nil { t.Fatalf("create replacement Telegram link challenge: %v",err) }
    telegramID:=time.Now().UnixNano()
    identity:=TelegramIdentity{ID:telegramID,Username:"integration_user",FirstName:"Integration"}
    if err:=service.CompleteTelegramLink(ctx,challenge.Token,identity);err!=nil { t.Fatalf("complete Telegram link: %v",err) }
    linked,err:=service.TelegramIdentity(ctx,registered.User.ID)
    if err!=nil || linked.ID==nil || *linked.ID!=telegramID || linked.Username!="integration_user" {
        t.Fatalf("unexpected linked Telegram identity: %+v, err=%v",linked,err)
    }
    if err:=service.CompleteTelegramLink(ctx,challenge.Token,identity);!errors.Is(err,ErrTelegramChallengeExpired) {
        t.Fatalf("replay challenge error = %v, want ErrTelegramChallengeExpired",err)
    }
    secondEmail:=fmt.Sprintf("auth-second-%d@example.test",time.Now().UnixNano())
    second,err:=service.Register(ctx,secondEmail,"correct-horse-battery","Second User","","")
    if err!=nil { t.Fatalf("register second user: %v",err) }
    secondChallenge,err:=service.CreateTelegramLinkChallenge(ctx,second.User.ID)
    if err!=nil { t.Fatalf("create second Telegram challenge: %v",err) }
    if err:=service.CompleteTelegramLink(ctx,secondChallenge.Token,identity);!errors.Is(err,ErrTelegramAlreadyLinked) {
        t.Fatalf("duplicate Telegram ID error = %v, want ErrTelegramAlreadyLinked",err)
    }
    if err:=service.UnlinkTelegramIdentity(ctx,registered.User.ID);err!=nil { t.Fatalf("unlink Telegram identity: %v",err) }
    unlinked,err:=service.TelegramIdentity(ctx,registered.User.ID)
    if err!=nil || unlinked.ID!=nil { t.Fatalf("Telegram identity still linked after unlink: %+v, err=%v",unlinked,err) }
}
