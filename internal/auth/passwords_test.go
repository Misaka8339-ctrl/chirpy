package auth

import "testing"

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: "correct-password",
			hash:     hash,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrong-password",
			hash:     hash,
			want:     false,
		},
		{
			name:     "invalid stored hash",
			password: "correct-password",
			hash:     "unset",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tt.password, tt.hash)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}

			if match != tt.want {
				t.Fatalf("match = %v, want %v", match, tt.want)
			}
		})
	}
}
