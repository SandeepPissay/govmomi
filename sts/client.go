/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sts

import (
	"context"
	"encoding/xml"
	"time"

	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

type Client struct {
	*soap.Client
}

func NewClient(ctx context.Context, c *vim25.Client) (*Client, error) {
	// TODO: OptionManager config.vpxd.sso.sts.uri
	sc := c.Client.NewServiceClient("/sts/STSService/vsphere.local", "urn:oasis:names:tc:SAML:2.0:assertion")

	return &Client{sc}, nil
}

func (c *Client) Issue(ctx context.Context, security SecurityHeaderType) (*RequestSecurityTokenResponseType, error) {
	created := Date{time.Now().UTC()}

	security.Timestamp = &TimestampType{
		Created: created,
		Expires: Date{created.Add(time.Minute * 10)},
	}

	header := struct {
		XMLName  xml.Name            `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
		Security *SecurityHeaderType `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd Security"`
	}{
		Security: &security,
	}

	req := RequestSecurityTokenType{
		TokenType:   c.Namespace,
		RequestType: "http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue",
		Lifetime: &LifetimeType{
			Created: created,
			Expires: Date{created.Add(time.Minute * 30)},
		},
		Renewing: &RenewingType{
			Allow: false,
			OK:    false,
		},
		KeyType:            "http://docs.oasis-open.org/ws-sx/ws-trust/200512/Bearer",
		SignatureAlgorithm: "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
		Delegatable:        true,
	}

	ctx = c.HeaderContext(ctx, header)
	ctx = c.ActionContext(ctx, "http://docs.oasis-open.org/ws-sx/ws-trust/200512/RST/Issue")
	res, err := RequestSecurityToken(ctx, c, &req)
	if err != nil {
		return nil, err
	}

	return res.RequestSecurityTokenResponse, nil
}
