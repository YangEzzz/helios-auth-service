package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFCookieMatchesThirtyDaySession(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := CSRF(c, true); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CSRFCookie {
		t.Fatal("expected CSRF cookie")
	}
	cookie := cookies[0]
	if cookie.MaxAge != 30*24*60*60 || !cookie.Secure || cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected CSRF cookie policy: max-age=%d secure=%t httponly=%t samesite=%v", cookie.MaxAge, cookie.Secure, cookie.HttpOnly, cookie.SameSite)
	}
}
