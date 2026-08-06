package whatsapp

import "testing"

func TestCanonicalizeE164(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid Colombian mobile",
			input:   "+573001234567",
			want:    "+573001234567",
			wantErr: false,
		},
		{
			name:    "Colombian mobile without + prefix",
			input:   "573001234567",
			want:    "+573001234567",
			wantErr: false,
		},
		{
			name:    "non-Colombian number",
			input:   "+14155551234",
			want:    "+14155551234",
			wantErr: true,
		},
		{
			name:    "invalid Colombian prefix (starts with 2)",
			input:   "+57300123456",
			want:    "+57300123456",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "+57300",
			want:    "+57300",
			wantErr: true,
		},
		{
			name:    "with spaces",
			input:   " +573001234567 ",
			want:    "+573001234567",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalizeE164(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanonicalizeE164() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CanonicalizeE164() = %v, want %v", got, tt.want)
			}
		})
	}
}
