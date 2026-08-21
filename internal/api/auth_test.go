package api

import (
	"os"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestAuthorizedWithConfiguredToken(t *testing.T) {
	t.Setenv("FIRECRACKER_STUDIO_TOKEN", "test-token")
	server := &Server{}

	tests := []struct {
		name       string
		authority  string
		queryToken string
		want       bool
	}{
		{name: "missing token", want: false},
		{name: "wrong token", authority: "Bearer wrong-token", want: false},
		{name: "query token is rejected", queryToken: "test-token", want: false},
		{name: "correct bearer token", authority: "Bearer test-token", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Request.Header.SetMethod(fasthttp.MethodGet)
			ctx.Request.URI().SetPath("/api/v1/vms")
			if tt.authority != "" {
				ctx.Request.Header.Set("Authorization", tt.authority)
			}
			if tt.queryToken != "" {
				ctx.Request.URI().QueryArgs().Set("access_token", tt.queryToken)
			}
			if got := server.authorized(&ctx); got != tt.want {
				t.Fatalf("authorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthorizedWithoutConfiguredAuth(t *testing.T) {
	if err := os.Unsetenv("FIRECRACKER_STUDIO_TOKEN"); err != nil {
		t.Fatal(err)
	}
	server := &Server{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.URI().SetPath("/api/v1/vms")
	if !server.authorized(&ctx) {
		t.Fatal("authorized() = false, want true when no authentication is configured")
	}
}
