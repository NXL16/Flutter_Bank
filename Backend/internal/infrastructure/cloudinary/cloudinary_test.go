package cloudinary

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignParametersIsStable(t *testing.T) {
	first := signParameters(
		map[string]string{
			"timestamp": "1700000000",
			"public_id": "nf-bank/avatars/user_7",
			"overwrite": "true",
		},
		"secret",
	)
	second := signParameters(
		map[string]string{
			"overwrite": "true",
			"public_id": "nf-bank/avatars/user_7",
			"timestamp": "1700000000",
		},
		"secret",
	)
	if first != second {
		t.Fatalf("signature must not depend on map iteration order")
	}
	if len(first) != 40 {
		t.Fatalf("expected SHA-1 hex signature, got %q", first)
	}
}

func TestUploadAvatarSendsSignedMultipartAndReadsSecureURL(t *testing.T) {
	const secret = "test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if err := request.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if request.FormValue("api_key") != "test-key" {
			t.Fatalf("missing API key")
		}
		params := map[string]string{
			"invalidate": request.FormValue("invalidate"),
			"overwrite":  request.FormValue("overwrite"),
			"public_id":  request.FormValue("public_id"),
			"timestamp":  request.FormValue("timestamp"),
		}
		if request.FormValue("signature") != signParameters(params, secret) {
			t.Fatalf("invalid upload signature")
		}
		file, _, err := request.FormFile("file")
		if err != nil {
			t.Fatalf("missing file: %v", err)
		}
		defer file.Close()
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"secure_url":"https://res.cloudinary.com/demo/avatar.jpg"}`)
	}))
	defer server.Close()

	client := NewClient("demo", "test-key", secret)
	client.uploadURL = server.URL
	client.http = server.Client()
	avatarURL, err := client.UploadAvatar(
		context.Background(),
		7,
		bytes.NewBufferString("image"),
		"avatar.jpg",
		"image/jpeg",
	)
	if err != nil {
		t.Fatalf("upload avatar: %v", err)
	}
	if avatarURL != "https://res.cloudinary.com/demo/avatar.jpg" {
		t.Fatalf("unexpected URL: %s", avatarURL)
	}
}
