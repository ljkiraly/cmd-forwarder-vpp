// Copyright (c) 2022-2024 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

// nolint:dupl
package tests

import (
	"context"
	"net"

	"go.fd.io/govpp/api"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	ipsecapi "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/ipsec"

	"github.com/ljkiraly/sdk/pkg/networkservice/chains/client"
	"github.com/ljkiraly/sdk/pkg/networkservice/chains/endpoint"
	"github.com/ljkiraly/sdk/pkg/networkservice/common/authorize"
	"github.com/ljkiraly/sdk/pkg/networkservice/common/mechanisms"
	"github.com/ljkiraly/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/ljkiraly/sdk/pkg/networkservice/utils/metadata"
	"github.com/ljkiraly/sdk/pkg/tools/token"

	"github.com/ljkiraly/sdk-vpp/pkg/networkservice/connectioncontext"
	"github.com/ljkiraly/sdk-vpp/pkg/networkservice/mechanisms/ipsec"
	"github.com/ljkiraly/sdk-vpp/pkg/networkservice/pinhole"
	"github.com/ljkiraly/sdk-vpp/pkg/networkservice/up"
)

type ipsecVerifiableEndpoint struct {
	ctx     context.Context
	vppConn api.Connection
	endpoint.Endpoint
}

func newIpsecVerifiableEndpoint(ctx context.Context,
	prefix1, prefix2 *net.IPNet,
	tokenGenerator token.GeneratorFunc,
	vppConn api.Connection) verifiableEndpoint {
	rv := &ipsecVerifiableEndpoint{
		ctx:     ctx,
		vppConn: vppConn,
	}
	name := "ipsecVerifiableEndpoint"
	rv.Endpoint = endpoint.NewServer(ctx,
		tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authorize.NewServer()),
		endpoint.WithAdditionalFunctionality(
			metadata.NewServer(),
			point2pointipam.NewServer(prefix1),
			point2pointipam.NewServer(prefix2),
			up.NewServer(ctx, vppConn),
			pinhole.NewServer(vppConn),
			connectioncontext.NewServer(vppConn),
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				ipsecapi.MECHANISM: ipsec.NewServer(vppConn, net.ParseIP(serverIP)),
			}),
		),
	)
	return rv
}

func (v *ipsecVerifiableEndpoint) VerifyConnection(conn *networkservice.Connection) error {
	for _, ip := range conn.GetContext().GetIpContext().GetSrcIpAddrs() {
		if err := pingVpp(v.ctx, v.vppConn, ip); err != nil {
			return err
		}
	}
	return nil
}

func (v *ipsecVerifiableEndpoint) VerifyClose(_ *networkservice.Connection) error {
	return nil
}

type ipsecVerifiableClient struct {
	ctx     context.Context
	vppConn api.Connection
	networkservice.NetworkServiceClient
}

func newIpsecVerifiableClient(
	ctx context.Context,
	sutCC grpc.ClientConnInterface,
	vppConn api.Connection,
) verifiableClient {
	return &ipsecVerifiableClient{
		ctx:     ctx,
		vppConn: vppConn,
		NetworkServiceClient: client.NewClient(
			ctx,
			client.WithName("ipsecVerifiableClient"),
			client.WithClientConn(sutCC),
			client.WithAdditionalFunctionality(
				up.NewClient(ctx, vppConn),
				connectioncontext.NewClient(vppConn),
				ipsec.NewClient(vppConn, net.ParseIP(clientIP)),
				pinhole.NewClient(vppConn),
			),
		),
	}
}

func (v *ipsecVerifiableClient) VerifyConnection(conn *networkservice.Connection) error {
	for _, ip := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
		if err := pingVpp(v.ctx, v.vppConn, ip); err != nil {
			return err
		}
	}
	return nil
}

func (v *ipsecVerifiableClient) VerifyClose(_ *networkservice.Connection) error {
	return nil
}
