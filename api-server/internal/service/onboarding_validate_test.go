package service

import "testing"

func TestValidateSSHTarget(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		port    int
		user    string
		wantErr bool
	}{
		{"valid ip", "10.0.0.5", 22, "ubuntu", false},
		{"valid hostname", "node-1.cluster.local", 2222, "root", false},
		{"empty host", "", 22, "ubuntu", true},
		{"port too low", "10.0.0.5", 0, "ubuntu", true},
		{"port too high", "10.0.0.5", 70000, "ubuntu", true},
		{"host with space / injection", "10.0.0.5 rm -rf /", 22, "ubuntu", true},
		{"host with semicolon", "1.2.3.4;reboot", 22, "ubuntu", true},
		{"username with shell metachars", "10.0.0.5", 22, "root;rm -rf /", true},
		{"username with space", "10.0.0.5", 22, "ub untu", true},
		{"empty username", "10.0.0.5", 22, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSSHTarget("nodeIP", tc.host, tc.port, tc.user)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateSSHTarget(%q,%d,%q) err=%v wantErr=%v", tc.host, tc.port, tc.user, err, tc.wantErr)
			}
		})
	}
}
