package sts

import (
	"context"
	"os"
	"testing"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/soap"
)

func TestClientIssue(t *testing.T) {
	url := os.Getenv("GOVC_URL") // TODO: GOVMOMI_STS_URL
	if url == "" {
		t.SkipNow()
	}

	u, err := soap.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}

	password, _ := u.User.Password()

	security := SecurityHeaderType{
		UsernameToken: &UsernameTokenType{
			Username: u.User.Username(),
			Password: password,
		},
	}

	ctx := context.Background()

	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		t.Fatal(err)
	}

	sts, err := NewClient(ctx, c.Client)
	if err != nil {
		t.Fatal(err)
	}

	res, err := sts.Issue(ctx, security)
	if err != nil {
		t.Fatal(err)
	}

	if res.RequestedSecurityToken.Assertion.ID == "" {
		t.Fatal("no Assertion ID")
	}
}
