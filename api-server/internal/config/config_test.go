package config

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "auth disabled allows defaults",
			cfg:  Config{AuthEnabled: false, JWTSecret: defaultJWTSecret, AdminPassword: defaultAdminPassword},
		},
		{
			name:    "auth enabled rejects default jwt secret",
			cfg:     Config{AuthEnabled: true, JWTSecret: defaultJWTSecret, AdminPassword: "strong-pass"},
			wantErr: true,
		},
		{
			name:    "auth enabled rejects empty jwt secret",
			cfg:     Config{AuthEnabled: true, JWTSecret: "", AdminPassword: "strong-pass"},
			wantErr: true,
		},
		{
			name:    "auth enabled rejects default password",
			cfg:     Config{AuthEnabled: true, JWTSecret: "a-real-secret", AdminPassword: defaultAdminPassword},
			wantErr: true,
		},
		{
			name: "auth enabled accepts strong values",
			cfg:  Config{AuthEnabled: true, JWTSecret: "a-real-secret", AdminPassword: "strong-pass"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
