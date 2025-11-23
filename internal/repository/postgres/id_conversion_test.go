package postgres

import "testing"

func TestStringIDToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid ID with prefix",
			input:   "u1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "valid ID without prefix",
			input:   "1",
			want:    1,
			wantErr: false,
		},
		{
			name:    "valid ID with large number",
			input:   "u12345",
			want:    12345,
			wantErr: false,
		},
		{
			name:    "valid ID zero",
			input:   "u0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid ID - non-numeric",
			input:   "uabc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid ID - empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid ID - only prefix",
			input:   "u",
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative number (parsed but may be invalid in context)",
			input:   "u-1",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "invalid ID - with spaces",
			input:   "u 1",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := stringIDToInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("stringIDToInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("stringIDToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntToStringID(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{
			name:  "positive number",
			input: 1,
			want:  "u1",
		},
		{
			name:  "zero",
			input: 0,
			want:  "u0",
		},
		{
			name:  "large number",
			input: 12345,
			want:  "u12345",
		},
		{
			name:  "single digit",
			input: 5,
			want:  "u5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := intToStringID(tt.input)
			if got != tt.want {
				t.Errorf("intToStringID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRStringIDToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid ID with prefix",
			input:   "pr-1001",
			want:    1001,
			wantErr: false,
		},
		{
			name:    "valid ID without prefix",
			input:   "1001",
			want:    1001,
			wantErr: false,
		},
		{
			name:    "valid ID with large number",
			input:   "pr-12345",
			want:    12345,
			wantErr: false,
		},
		{
			name:    "valid ID zero",
			input:   "pr-0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid ID - non-numeric",
			input:   "pr-abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid ID - empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid ID - only prefix",
			input:   "pr-",
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative number (parsed but may be invalid in context)",
			input:   "pr--1",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "invalid ID - with spaces",
			input:   "pr- 1001",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong prefix - will fail to parse",
			input:   "u-1001",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := prStringIDToInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("prStringIDToInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("prStringIDToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRIntToStringID(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{
			name:  "positive number",
			input: 1001,
			want:  "pr-1001",
		},
		{
			name:  "zero",
			input: 0,
			want:  "pr-0",
		},
		{
			name:  "large number",
			input: 12345,
			want:  "pr-12345",
		},
		{
			name:  "single digit",
			input: 5,
			want:  "pr-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := prIntToStringID(tt.input)
			if got != tt.want {
				t.Errorf("prIntToStringID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Тесты на обратную совместимость - проверяем, что преобразования обратимы
func TestIDConversionRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		userID  int
		prID    int
		userStr string
		prStr   string
	}{
		{
			name:    "round trip user ID",
			userID:  1,
			userStr: "u1",
		},
		{
			name:  "round trip PR ID",
			prID:  1001,
			prStr: "pr-1001",
		},
		{
			name:    "round trip large numbers",
			userID:  99999,
			prID:    123456,
			userStr: "u99999",
			prStr:   "pr-123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.userID != 0 {
				// Проверяем user ID: int -> string -> int
				str := intToStringID(tt.userID)
				if str != tt.userStr {
					t.Errorf("intToStringID(%d) = %v, want %v", tt.userID, str, tt.userStr)
				}

				back, err := stringIDToInt(str)
				if err != nil {
					t.Errorf("stringIDToInt(%v) error = %v", str, err)
					return
				}
				if back != tt.userID {
					t.Errorf("stringIDToInt(intToStringID(%d)) = %d, want %d", tt.userID, back, tt.userID)
				}
			}

			if tt.prID != 0 {
				// Проверяем PR ID: int -> string -> int
				str := prIntToStringID(tt.prID)
				if str != tt.prStr {
					t.Errorf("prIntToStringID(%d) = %v, want %v", tt.prID, str, tt.prStr)
				}

				back, err := prStringIDToInt(str)
				if err != nil {
					t.Errorf("prStringIDToInt(%v) error = %v", str, err)
					return
				}
				if back != tt.prID {
					t.Errorf("prStringIDToInt(prIntToStringID(%d)) = %d, want %d", tt.prID, back, tt.prID)
				}
			}
		})
	}
}
