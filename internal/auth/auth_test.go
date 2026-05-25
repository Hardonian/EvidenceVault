package auth

import (
	"net/http"
	"os"
	"testing"
)

func BenchmarkFromRequestAPIKey(b *testing.B) {
	os.Setenv("APP_ENV", "production")
	os.Setenv("API_KEYS", "key1:tenant1,key2:tenant2,key3:tenant3,key4:tenant4,key5:tenant5")

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "key3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FromRequest(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
