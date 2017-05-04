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
	"reflect"
	"time"

	signature "github.com/russellhaering/goxmldsig/types"
	"github.com/vmware/govmomi/vim25/soap"
	vim "github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vim25/xml"
)

type Date struct {
	time.Time
}

func (d Date) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	s := d.Time.Format("2006-01-02T15:04:05.000Z")
	return e.EncodeElement(s, start)
}

type AttributeType struct {
	AttributeValue string `xml:"AttributeValue,omitempty,typeattr"`
}

type BinaryExchangeType struct{}

type BinarySecurityTokenType struct{}

type DelegateToType struct {
	UsernameToken *UsernameTokenType `xml:"UsernameToken,omitempty"`
}

type LifetimeType struct {
	Created Date `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Created"`
	Expires Date `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Expires"`
}

type RenewingType struct {
	Allow bool `xml:"Allow,attr"`
	OK    bool `xml:"OK,attr"`
}

type RequestSecurityTokenResponseCollectionType struct {
	RequestSecurityTokenResponse *RequestSecurityTokenResponseType `xml:"RequestSecurityTokenResponse,omitempty"`
}

func init() {
	vim.Add("RequestSecurityTokenResponseCollection", reflect.TypeOf((*RequestSecurityTokenResponseCollectionType)(nil)).Elem())
}

type RequestSecurityTokenResponseType struct {
	TokenType              string                     `xml:"TokenType,omitempty"`
	Lifetime               *LifetimeType              `xml:"Lifetime,omitempty"`
	RequestedSecurityToken RequestedSecurityTokenType `xml:"RequestedSecurityToken"`
	Renewing               *RenewingType              `xml:"Renewing,omitempty"`
	BinaryExchange         *BinaryExchangeType        `xml:"BinaryExchange,omitempty"`
	KeyType                string                     `xml:"KeyType,omitempty"`
	SignatureAlgorithm     string                     `xml:"SignatureAlgorithm,omitempty"`
	Delegatable            bool                       `xml:"Delegatable,omitempty"`
	Status                 *StatusType                `xml:"Status,omitempty"`
}

type RequestSecurityTokenType struct {
	TokenType          string              `xml:"TokenType,omitempty"`
	RequestType        string              `xml:"RequestType,omitempty"`
	Lifetime           *LifetimeType       `xml:"Lifetime,omitempty"`
	Renewing           *RenewingType       `xml:"Renewing,omitempty"`
	BinaryExchange     *BinaryExchangeType `xml:"BinaryExchange,omitempty"`
	KeyType            string              `xml:"KeyType,omitempty"`
	SignatureAlgorithm string              `xml:"SignatureAlgorithm,omitempty"`
	UseKey             vim.AnyType         `xml:"UseKey,omitempty"`
	DelegateTo         *DelegateToType     `xml:"DelegateTo,omitempty"`
	Delegatable        bool                `xml:"Delegatable,omitempty"`
	ValidateTarget     []vim.AnyType       `xml:"ValidateTarget,omitempty"`
	RenewTarget        []vim.AnyType       `xml:"RenewTarget,omitempty"`
}

type RequestedSecurityTokenType struct {
	Assertion Assertion `xml:"Assertion"`
}

type SecurityHeaderType struct {
	Timestamp           *TimestampType           `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Timestamp,omitempty"`
	UsernameToken       *UsernameTokenType       `xml:"UsernameToken,omitempty"`
	BinarySecurityToken *BinarySecurityTokenType `xml:"BinarySecurityToken,omitempty"`
}

type StatusType struct {
	Code   string `xml:"Code,omitempty"`
	Reason string `xml:"Reason,omitempty"`
}

type TimestampType struct {
	Created Date `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Created"`
	Expires Date `xml:"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd Expires"`
}

type UsernameTokenType struct {
	Username string `xml:"Username,omitempty"`
	Password string `xml:"Password,omitempty"`
}

type Assertion struct {
	XMLName      xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID           string   `xml:"ID,attr"`
	IssueInstant Date     `xml:"IssueInstant,attr"`
	Version      string   `xml:"Version,attr"`
	Issuer       struct {
		Format string `xml:"Format,attr"`
		Issuer string `xml:",innerxml"`
	} `xml:"Issuer"`
	Signature          signature.Signature
	Subject            Subject
	Conditions         Conditions
	AuthnStatement     AuthnStatement
	AttributeStatement AttributeStatement
}

type NameID struct {
	Format string `xml:"Format,attr"`
	ID     string `xml:",innerxml"`
}

type Subject struct {
	NameID              NameID
	SubjectConfirmation SubjectConfirmation
}

type SubjectConfirmation struct {
	Method                  string `xml:"Method,attr"`
	SubjectConfirmationData struct {
		NotOnOrAfter Date `xml:"NotOnOrAfter,attr"`
	}
}

type Conditions struct {
	NotBefore            Date                  `xml:",attr"`
	NotOnOrAfter         Date                  `xml:",attr"`
	ProxyRestriction     *ProxyRestriction     `xml:",omitempty"`
	RenewRestrictionType *RenewRestrictionType `xml:",omitempty"`
}

type ConditionAbstractType struct {
	Count int32 `xml:",attr"`
}

type ProxyRestriction struct {
	ConditionAbstractType
}

type RenewRestrictionType struct {
	ConditionAbstractType
}

type AuthnStatement struct {
	AuthnInstant Date `xml:",attr"`
	AuthnContext struct {
		AuthnContextClassRef string
	}
}

type AttributeValue struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

type Attribute struct {
	FriendlyName   string `xml:",attr"`
	Name           string `xml:",attr"`
	NameFormat     string `xml:",attr"`
	AttributeValue []AttributeValue
}

type AttributeStatement struct {
	Attribute []Attribute
}

type RequestSecurityTokenBody struct {
	Req    *RequestSecurityTokenType                   `xml:"http://docs.oasis-open.org/ws-sx/ws-trust/200512 RequestSecurityToken"`
	Res    *RequestSecurityTokenResponseCollectionType `xml:"http://docs.oasis-open.org/ws-sx/ws-trust/200512 RequestSecurityTokenResponseCollection,omitempty"`
	Fault_ *soap.Fault                                 `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
}

func (b *RequestSecurityTokenBody) Fault() *soap.Fault { return b.Fault_ }

func RequestSecurityToken(ctx context.Context, r soap.RoundTripper, req *RequestSecurityTokenType) (*RequestSecurityTokenResponseCollectionType, error) {
	var reqBody, resBody RequestSecurityTokenBody

	reqBody.Req = req

	if err := r.RoundTrip(ctx, &reqBody, &resBody); err != nil {
		return nil, err
	}

	return resBody.Res, nil
}
