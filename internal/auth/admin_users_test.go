package auth

import "testing"

func TestAdminUserStatusErrorsAreDefined(t *testing.T) {
	if ErrInvalidAdminUserStatus == nil || ErrAdminSelfLock == nil || ErrAdminUserNotFound == nil {
		t.Fatal("admin user management errors must be defined")
	}
}
