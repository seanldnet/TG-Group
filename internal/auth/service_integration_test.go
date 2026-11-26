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
	if registered.User.Email != email || registered.User.TenantID == "" || registered.Token == "" || registered.RefreshToken == "" {
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
	rotated,err:=service.Refresh(ctx,loggedIn.RefreshToken,"integration-test","127.0.0.1:4321")
	if err!=nil { t.Fatalf("refresh login session: %v",err) }
	if rotated.Token=="" || rotated.RefreshToken=="" || rotated.RefreshToken==loggedIn.RefreshToken {
		t.Fatalf("refresh did not rotate credentials: %+v",rotated)
	}
	if _,err:=service.Authenticate(ctx,rotated.Token);err!=nil { t.Fatalf("authenticate rotated access token: %v",err) }
	if _,err:=service.Refresh(ctx,loggedIn.RefreshToken,"integration-test","127.0.0.1:4321");!errors.Is(err,ErrRefreshReuse) {
		t.Fatalf("reused refresh token error = %v, want ErrRefreshReuse",err)
	}
	if _,err:=service.Authenticate(ctx,rotated.Token);!errors.Is(err,ErrUnauthorized) {
		t.Fatalf("authenticate session after refresh reuse error = %v, want ErrUnauthorized",err)
	}
	freshLogin,err:=service.Login(ctx,email,"correct-horse-battery","","127.0.0.1:4321")
	if err!=nil { t.Fatalf("login after refresh reuse: %v",err) }
	if err:=service.Logout(ctx,freshLogin.Token);err!=nil { t.Fatalf("logout refreshed family: %v",err) }
	if _,err:=service.Refresh(ctx,freshLogin.RefreshToken,"","");!errors.Is(err,ErrRefreshUnauthorized) {
		t.Fatalf("refresh after logout error = %v, want ErrRefreshUnauthorized",err)
	}


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
	securityEvents,err:=service.ListSecurityEvents(ctx,registered.User.ID,100)
	if err!=nil { t.Fatalf("list account security events: %v",err) }
	actions:=map[string]bool{}
	for _,event:=range securityEvents { actions[event.Action]=true }
	for _,action:=range []string{"auth.register","auth.login","auth.refresh","auth.logout","auth.telegram_link","auth.telegram_unlink"} {
		if !actions[action] { t.Errorf("security history missing %s",action) }
	}
}

func TestChangePasswordRevokesPreviousSessionsPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" { t.Skip("TEST_DATABASE_URL is not configured") }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, databaseURL)
	if err != nil { t.Fatalf("open test database: %v", err) }
	defer db.Close()
	if err := db.Migrate(ctx); err != nil { t.Fatalf("apply migrations: %v", err) }

	service := NewService(db.Pool)
	email := fmt.Sprintf("password-change-%d@example.test", time.Now().UnixNano())
	first, err := service.Register(ctx,email,"old-password-123","Password Test","","127.0.0.1:4321")
	if err != nil { t.Fatalf("register: %v",err) }
	second, err := service.Login(ctx,email,"old-password-123","","127.0.0.1:4322")
	if err != nil { t.Fatalf("second login: %v",err) }

	if _, err := service.ChangePassword(ctx,first.Token,"incorrect-password","new-password-456","","127.0.0.1:4321"); !errors.Is(err,ErrCurrentPasswordInvalid) {
		t.Fatalf("wrong current password error = %v, want ErrCurrentPasswordInvalid",err)
	}
	if _, err := service.ChangePassword(ctx,first.Token,"old-password-123","short","",""); !errors.Is(err,ErrInvalidInput) {
		t.Fatalf("weak new password error = %v, want ErrInvalidInput",err)
	}
	if _, err := service.ChangePassword(ctx,first.Token,"old-password-123","old-password-123","",""); !errors.Is(err,ErrInvalidInput) {
		t.Fatalf("unchanged password error = %v, want ErrInvalidInput",err)
	}

	changed, err := service.ChangePassword(ctx,first.Token,"old-password-123","new-password-456","integration-test","127.0.0.1:4321")
	if err != nil { t.Fatalf("change password: %v",err) }
	if changed.Token == "" || changed.RefreshToken == "" { t.Fatal("password change did not issue a replacement session") }
	if _, err := service.Authenticate(ctx,changed.Token); err != nil { t.Fatalf("authenticate replacement session: %v",err) }
	if _, err := service.Authenticate(ctx,first.Token); !errors.Is(err,ErrUnauthorized) {
		t.Fatalf("old current session error = %v, want ErrUnauthorized",err)
	}
	if _, err := service.Authenticate(ctx,second.Token); !errors.Is(err,ErrUnauthorized) {
		t.Fatalf("other device session error = %v, want ErrUnauthorized",err)
	}
	if _, err := service.Refresh(ctx,first.RefreshToken,"",""); !errors.Is(err,ErrRefreshUnauthorized) {
		t.Fatalf("old refresh token error = %v, want ErrRefreshUnauthorized",err)
	}
	if _, err := service.Login(ctx,email,"old-password-123","",""); !errors.Is(err,ErrInvalidCredentials) {
		t.Fatalf("old password login error = %v, want ErrInvalidCredentials",err)
	}
	if _, err := service.Login(ctx,email,"new-password-456","",""); err != nil {
		t.Fatalf("login with new password: %v",err)
	}
}

func TestLoginRateLimitByEmailAndIPPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" { t.Skip("TEST_DATABASE_URL is not configured") }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, databaseURL)
	if err != nil { t.Fatalf("open test database: %v",err) }
	defer db.Close()
	if err:=db.Migrate(ctx);err!=nil { t.Fatalf("apply migrations: %v",err) }

	service:=NewService(db.Pool)
	email:=fmt.Sprintf("login-limit-%d@example.test",time.Now().UnixNano())
	if _,err:=service.Register(ctx,email,"correct-password-123","Rate Limit Test","","");err!=nil {
		t.Fatalf("register rate limit user: %v",err)
	}
	const sourceIP="198.51.100.77:4321"
	for i:=0;i<loginFailureLimit;i++ {
		if _,err:=service.Login(ctx,email,"wrong-password","",""+sourceIP);!errors.Is(err,ErrInvalidCredentials) {
			t.Fatalf("failed login %d error = %v, want ErrInvalidCredentials",i+1,err)
		}
	}
	if _,err:=service.Login(ctx,email,"correct-password-123","",sourceIP);!errors.Is(err,ErrLoginRateLimited) {
		t.Fatalf("login after email threshold error = %v, want ErrLoginRateLimited",err)
	}

	otherEmail:=fmt.Sprintf("login-limit-other-%d@example.test",time.Now().UnixNano())
	if _,err:=service.Register(ctx,otherEmail,"correct-password-123","Other User","","");err!=nil {
		t.Fatalf("register second rate limit user: %v",err)
	}
	if _,err:=service.Login(ctx,otherEmail,"correct-password-123","",sourceIP);!errors.Is(err,ErrLoginRateLimited) {
		t.Fatalf("login after IP threshold error = %v, want ErrLoginRateLimited",err)
	}
	if _,err:=service.Login(ctx,email,"correct-password-123","","198.51.100.78:4321");!errors.Is(err,ErrLoginRateLimited) {
		t.Fatalf("login from another IP after email threshold error = %v, want ErrLoginRateLimited",err)
	}
	if _,err:=service.Login(ctx,otherEmail,"correct-password-123","","198.51.100.79:4321");err!=nil {
		t.Fatalf("login from a fresh email/IP pair: %v",err)
	}
}
