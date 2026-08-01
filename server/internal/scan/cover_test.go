package scan

import "testing"

func TestCoverExt(t *testing.T) {
	type testCase struct {
		name string
		data []byte
		mime string
		want string
	}

	tests := []testCase{
		{
			name: "png by header",
			data: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
			mime: "image/jpeg",
			want: ".png",
		},
		{
			name: "jpeg by header",
			data: []byte{0xff, 0xd8, 0xff, 0xdb},
			mime: "image/png",
			want: ".jpg",
		},
		{
			name: "webp by header",
			data: []byte("RIFF1234WEBP"),
			mime: "image/jpeg",
			want: ".webp",
		},
		{
			name: "fallback by mime",
			data: []byte("not-an-image-header"),
			mime: "image/gif",
			want: ".gif",
		},
		{
			name: "unknown",
			data: []byte("unknown"),
			mime: "unknown",
			want: ".bin",
		},
	}

	for _, tc := range tests {
		if got := coverExt(tc.data, tc.mime); got != tc.want {
			t.Fatalf("%s: want %s got %s", tc.name, tc.want, got)
		}
	}
}
