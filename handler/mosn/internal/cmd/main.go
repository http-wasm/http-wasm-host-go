/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Copied from https://github.com/mosn/mosn/blob/9bd8b14b54fd979ebcb077f13e7a18e2bcfc43cd/cmd/mosn/main/main.go
// with extensions we don't need removed to improve compile time.

package main

import (
	_ "flag"
	"os"
	"strconv"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	"github.com/urfave/cli"
	_ "mosn.io/mosn/pkg/admin/debug"
	_ "mosn.io/mosn/pkg/filter/listener/originaldst"
	_ "mosn.io/mosn/pkg/filter/network/connectionmanager"
	_ "mosn.io/mosn/pkg/filter/network/proxy"
	_ "mosn.io/mosn/pkg/filter/network/streamproxy"
	_ "mosn.io/mosn/pkg/filter/network/tunnel"
	_ "mosn.io/mosn/pkg/filter/stream/transcoder/httpconv"
	_ "mosn.io/mosn/pkg/network"
	_ "mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/router"
	_ "mosn.io/mosn/pkg/server/keeper"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	_ "mosn.io/mosn/pkg/upstream/healthcheck"
	_ "mosn.io/pkg/buffer"

	_ "github.com/http-wasm/http-wasm-host-go/handler/mosn"
)

var _ = &corev3.Pipe{}

// Version mosn version is specified by build tag, in VERSION file
var Version = ""

func main() {
	app := newMosnApp(&cmdStart)

	// ignore error so we don't exit non-zero and break gfmrun README example tests
	_ = app.Run(os.Args)
}

func newMosnApp(startCmd *cli.Command) *cli.App {
	app := cli.NewApp()
	app.Name = "mosn"
	app.Version = Version
	app.Compiled = time.Now()
	app.Copyright = "(c) " + strconv.Itoa(time.Now().Year()) + " Ant Group"
	app.Usage = "MOSN is modular observable smart netstub."
	app.Flags = cmdStart.Flags

	//commands
	app.Commands = []cli.Command{
		cmdStart,
		cmdStop,
		cmdReload,
	}

	//action
	app.Action = func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			return cli.ShowAppHelp(c)
		}

		return startCmd.Action.(func(c *cli.Context) error)(c)
	}

	return app
}
