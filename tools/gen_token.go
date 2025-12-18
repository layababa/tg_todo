package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

func main() {
	botToken := "demo-token"
	
	// Data to sign
	params := map[string]string{
		"query_id": "test_query_id",
		"user":     `{"id":12345,"first_name":"Test","last_name":"User","username":"testuser","language_code":"en"}`,
		"auth_date": fmt.Sprintf("%d", time.Now().Unix()),
	}

	// 1. Create data check string
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	dataCheckString := strings.Join(parts, "\n")

	// 2. Compute secret key
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	// 3. Compute hash
	h2 := hmac.New(sha256.New, secretKey)
	h2.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(h2.Sum(nil))

	// 4. Construct final string
	var finalParts []string
	for k, v := range params {
		finalParts = append(finalParts, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	finalParts = append(finalParts, fmt.Sprintf("hash=%s", hash))
	
	fmt.Println(strings.Join(finalParts, "&"))
}
