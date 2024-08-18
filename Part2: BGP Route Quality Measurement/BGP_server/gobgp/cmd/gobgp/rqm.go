// Copyright (C) 2015 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"net"

	"github.com/spf13/cobra"

	api "github.com/osrg/gobgp/v3/api"
)

func showRQMServer(args []string) error {
	servers := make([]*api.Rqm, 0)
	stream, err := client.ListRQM(ctx, &api.ListRQMRequest{})
	if err != nil {
		fmt.Println(err)
		return err
	}
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		servers = append(servers, r.Server)
	}

	for _, r := range servers {
		
			fmt.Printf("Address: %s, Port: %d\n", r.Conf.Address, r.Conf.Port)
			for _, rule := range r.Rule {
				fmt.Printf("  Rule community: %s - time limit: %d - sampling interval: %d\n", rule.Community, rule.TimeLimit, rule.SamplingInterval)
			}
		
	}
	return nil
}

func newRQMCmd() *cobra.Command {
	rqmCmd := &cobra.Command{
		Use: cmdRQM,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 || len(args) == 1 {

				showRQMServer(args)
				return
			} else if len(args) != 2 {
				exitWithError(fmt.Errorf("usage: gobgp rqm <ip address> [add|reset|enable|disable|delete]"))
			}
			addr := net.ParseIP(args[0])
			if addr == nil {
				exitWithError(fmt.Errorf("invalid ip address: %s", args[0]))
			}
			var err error
			switch args[1] {
			case "add":
				_, err = client.AddRQM(ctx, &api.AddRQMRequest{
					Address: addr.String(),
					Port:    178,
				})
			case "reset":
				_, err = client.ResetRQM(ctx, &api.ResetRQMRequest{
					Address: addr.String(),
				})
			case "enable":
				_, err = client.EnableRQM(ctx, &api.EnableRQMRequest{
					Address: addr.String(),
				})
			case "disable":
				_, err = client.DisableRQM(ctx, &api.DisableRQMRequest{
					Address: addr.String(),
				})
			case "delete":
				_, err = client.DeleteRQM(ctx, &api.DeleteRQMRequest{
					Address: addr.String(),
					Port:    323,
				})
			default:
				exitWithError(fmt.Errorf("unknown operation: %s", args[1]))
			}
			if err != nil {
				exitWithError(err)
			}
		},
	}

	return rqmCmd
}
