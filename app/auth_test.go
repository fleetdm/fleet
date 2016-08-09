package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateVC(t *testing.T) {
	db := openTestDB(t)

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	vc := GenerateVC(user)
	if !vc.IsAdmin() {
		t.Fatal("User is not an admin")
	}

}

func TestVC(t *testing.T) {
	db := openTestDB(t)
	r := createEmptyTestServer(db)

	user, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	admin, err := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	r.GET("/admin_login", func(c *gin.Context) {
		sm := NewSessionManager(c)
		sm.MakeSessionForUserID(admin.ID)
		err := sm.Save()
		if err != nil {
			t.Fatal(err.Error())
		}
		c.JSON(200, nil)
	})

	r.GET("/user_login", func(c *gin.Context) {
		sm := NewSessionManager(c)
		sm.MakeSessionForUserID(user.ID)
		err := sm.Save()
		if err != nil {
			t.Fatal(err.Error())
		}
		c.JSON(200, nil)
	})

	r.GET("/admin", func(c *gin.Context) {
		vc := VC(c)
		if !vc.IsAdmin() {
			t.Fatal("Not admin")
		}
		c.String(200, "OK")
	})

	r.GET("/user", func(c *gin.Context) {
		vc := VC(c)
		if vc.IsAdmin() {
			t.Fatal("Not user")
		}
		c.String(200, "OK")
	})

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/admin_login", nil)
	r.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/admin", nil)
	req2.Header.Set("Cookie", res1.Header().Get("Set-Cookie"))
	r.ServeHTTP(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/user_login", nil)
	r.ServeHTTP(res3, req3)

	res4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/user", nil)
	req4.Header.Set("Cookie", res3.Header().Get("Set-Cookie"))
	r.ServeHTTP(res4, req4)

}

func TestIsUserID(t *testing.T) {
	db := openTestDB(t)

	user1, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	vc := GenerateVC(user1)

	if !vc.IsUserID(user1.ID) {
		t.Fatal("IsUserID failed on same user object")
	}

	if vc.IsUserID(user1.ID + 1) {
		t.Fatal("IsUserID passed for incorrect ID")
	}
}

func TestCanPerformActionsOnUser(t *testing.T) {
	db := openTestDB(t)

	user1, err := NewUser(db, "marpaia", "foobar", "mike@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	user2, err := NewUser(db, "zwass", "foobar", "zwass@kolide.co", false, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	admin, err := NewUser(db, "admin", "foobar", "admin@kolide.co", true, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	adminVC := GenerateVC(admin)
	user1VC := GenerateVC(user1)

	if !adminVC.CanPerformWriteActionOnUser(user1) || !adminVC.CanPerformWriteActionOnUser(user2) {
		t.Fatal("Admin should be able to perform writes on users")
	}

	if !adminVC.CanPerformReadActionOnUser(user1) || !adminVC.CanPerformReadActionOnUser(user2) {
		t.Fatal("Admin should be able to perform reads on users")
	}

	if user1VC.CanPerformWriteActionOnUser(user2) {
		t.Fatal("user1 shouldn't be able to perform writes on user2")
	}

	if !user1VC.CanPerformReadActionOnUser(user2) {
		t.Fatal("user1 should be able to perform reads on user2")
	}

}
