package telegramauth

import (
	"testing"
	"time"
)

func TestParseInitData(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		check   func(*testing.T, *InitData)
	}{
		{
			name:    "valid init data with user",
			raw:     `query_id=AAHdF6IQAAAAAN0XohDhrOrc&user=%7B%22id%22%3A99281932%2C%22first_name%22%3A%22John%22%2C%22last_name%22%3A%22Doe%22%2C%22username%22%3A%22johndoe%22%2C%22language_code%22%3A%22en%22%7D&auth_date=1234567890&hash=abcdef123456`,
			wantErr: false,
			check: func(t *testing.T, d *InitData) {
				if d.QueryID != "AAHdF6IQAAAAAN0XohDhrOrc" {
					t.Errorf("QueryID = %v, want AAHdF6IQAAAAAN0XohDhrOrc", d.QueryID)
				}
				if d.AuthDate != 1234567890 {
					t.Errorf("AuthDate = %v, want 1234567890", d.AuthDate)
				}
				if d.Hash != "abcdef123456" {
					t.Errorf("Hash = %v, want abcdef123456", d.Hash)
				}
				if d.User == nil {
					t.Fatal("User is nil")
				}
				if d.User.ID != 99281932 {
					t.Errorf("User.ID = %v, want 99281932", d.User.ID)
				}
				if d.User.FirstName != "John" {
					t.Errorf("User.FirstName = %v, want John", d.User.FirstName)
				}
				if d.User.Username != "johndoe" {
					t.Errorf("User.Username = %v, want johndoe", d.User.Username)
				}
			},
		},
		{
			name:    "valid init data with start_param",
			raw:     `query_id=test123&start_param=task_123&auth_date=1234567890&hash=xyz789`,
			wantErr: false,
			check: func(t *testing.T, d *InitData) {
				if d.StartParam != "task_123" {
					t.Errorf("StartParam = %v, want task_123", d.StartParam)
				}
			},
		},
		{
			name:    "invalid URL encoding",
			raw:     "invalid%zzurl",
			wantErr: true,
		},
		{
			name:    "invalid auth_date",
			raw:     "auth_date=not_a_number&hash=abc",
			wantErr: true,
		},
		{
			name:    "invalid user JSON",
			raw:     "user=invalid_json&hash=abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInitData(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInitData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestInitData_Validate(t *testing.T) {
	// Test case with a real-world example signature
	// Note: This is a simplified test. In production, you'd use actual Telegram data
	tests := []struct {
		name     string
		initData *InitData
		botToken string
		wantErr  error
	}{
		{
			name: "missing hash",
			initData: &InitData{
				Hash:    "",
				RawData: "auth_date=1234567890",
			},
			botToken: "test_token",
			wantErr:  ErrMissingHash,
		},
		{
			name: "invalid signature",
			initData: &InitData{
				Hash:    "invalid_hash",
				RawData: "auth_date=1234567890&hash=invalid_hash",
			},
			botToken: "test_token",
			wantErr:  ErrInvalidSignature,
		},
		{
			name: "valid signature",
			initData: func() *InitData {
				// Create a valid signature
				botToken := "test_bot_token"
				// Data check string should be sorted, without hash
				dataCheckString := "auth_date=1234567890\nquery_id=test123"

				// Compute expected hash
				secretKey := computeSecretKey(botToken)
				hash := computeHash(dataCheckString, secretKey)

				// Raw data includes hash
				fullRaw := "auth_date=1234567890&query_id=test123&hash=" + hash

				return &InitData{
					Hash:     hash,
					RawData:  fullRaw,
					AuthDate: 1234567890,
					QueryID:  "test123",
				}
			}(),
			botToken: "test_bot_token",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initData.Validate(tt.botToken)
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitData_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		authDate int64
		maxAge   time.Duration
		want     bool
	}{
		{
			name:     "not expired - recent",
			authDate: now.Add(-5 * time.Minute).Unix(),
			maxAge:   10 * time.Minute,
			want:     false,
		},
		{
			name:     "expired - old",
			authDate: now.Add(-15 * time.Minute).Unix(),
			maxAge:   10 * time.Minute,
			want:     true,
		},
		{
			name:     "expired - zero auth date",
			authDate: 0,
			maxAge:   10 * time.Minute,
			want:     true,
		},
		{
			name:     "not expired - just under threshold",
			authDate: now.Add(-9 * time.Minute).Unix(),
			maxAge:   10 * time.Minute,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InitData{AuthDate: tt.authDate}
			if got := d.IsExpired(tt.maxAge); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildDataCheckString(t *testing.T) {
	tests := []struct {
		name    string
		rawData string
		want    string
	}{
		{
			name:    "simple sorted",
			rawData: "c=3&a=1&b=2&hash=ignored",
			want:    "a=1\nb=2\nc=3",
		},
		{
			name:    "hash is excluded",
			rawData: "key1=value1&hash=should_be_excluded&key2=value2",
			want:    "key1=value1\nkey2=value2",
		},
		{
			name:    "single parameter",
			rawData: "auth_date=1234567890&hash=xyz",
			want:    "auth_date=1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &InitData{RawData: tt.rawData}
			got := d.buildDataCheckString()
			if got != tt.want {
				t.Errorf("buildDataCheckString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeSecretKey(t *testing.T) {
	botToken := "test_token"
	secretKey := computeSecretKey(botToken)

	// Should return 32 bytes (SHA256 output)
	if len(secretKey) != 32 {
		t.Errorf("computeSecretKey() length = %v, want 32", len(secretKey))
	}

	// Should be deterministic
	secretKey2 := computeSecretKey(botToken)
	if string(secretKey) != string(secretKey2) {
		t.Error("computeSecretKey() is not deterministic")
	}
}

func TestComputeHash(t *testing.T) {
	data := "test_data"
	key := []byte("test_key")

	hash := computeHash(data, key)

	// Should return hex string (64 chars for SHA256)
	if len(hash) != 64 {
		t.Errorf("computeHash() length = %v, want 64", len(hash))
	}

	// Should be deterministic
	hash2 := computeHash(data, key)
	if hash != hash2 {
		t.Error("computeHash() is not deterministic")
	}

	// Different data should produce different hash
	hash3 := computeHash("different_data", key)
	if hash == hash3 {
		t.Error("computeHash() produced same hash for different data")
	}
}
