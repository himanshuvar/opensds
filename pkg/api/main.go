// Copyright 2019 The OpenSDS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
This module implements a entry into the OpenSDS northbound REST service.
*/

package api

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"github.com/opensds/opensds/pkg/api/filter/accesslog"
	"github.com/opensds/opensds/pkg/api/filter/auth"
	"github.com/opensds/opensds/pkg/api/filter/context"
	cfg "github.com/opensds/opensds/pkg/utils/config"
	"github.com/opensds/opensds/pkg/utils/constants"

	// Load the API routers
	_ "github.com/opensds/opensds/pkg/api/routers"
)

const (
	AddressIdx = iota
	PortIdx
)

func Run(dockApiServerCfg cfg.OsdsDock) {

	if dockApiServerCfg.HTTPSEnabled {
		if dockApiServerCfg.BeegoHTTPSCertFile == "" || dockApiServerCfg.BeegoHTTPSKeyFile == "" {
			fmt.Println("If https is enabled in hotpot, please ensure key file and cert file of the hotpot are not empty.")
			return
		}
		// beego https config
		beego.BConfig.Listen.EnableHTTP = false
		beego.BConfig.Listen.EnableHTTPS = true
		strs := strings.Split(dockApiServerCfg.RestEndpoint, ":")
		beego.BConfig.Listen.HTTPSAddr = strs[AddressIdx]
		beego.BConfig.Listen.HTTPSPort, _ = strconv.Atoi(strs[PortIdx])
		beego.BConfig.Listen.HTTPSCertFile = dockApiServerCfg.BeegoHTTPSCertFile
		beego.BConfig.Listen.HTTPSKeyFile = dockApiServerCfg.BeegoHTTPSKeyFile
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			},
		}

		beego.BeeApp.Server.TLSConfig = tlsConfig
	}

	beego.BConfig.Listen.ServerTimeOut = dockApiServerCfg.BeegoServerTimeOut
	beego.BConfig.CopyRequestBody = true
	beego.BConfig.EnableErrorsShow = false
	beego.BConfig.EnableErrorsRender = false
	beego.BConfig.WebConfig.AutoRender = false
	// insert some auth rules
	pattern := fmt.Sprintf("/%s/*", constants.APIVersion)
	beego.InsertFilter(pattern, beego.BeforeExec, context.Factory())
	beego.InsertFilter(pattern, beego.BeforeExec, auth.Factory())
	beego.InsertFilter("*", beego.BeforeExec, accesslog.Factory())
	//beego.InsertFilter("*", beego.BeforeExec, validation.Factory(dockApiServerCfg.ApiSpecPath))

	// start service
	beego.Run(dockApiServerCfg.RestEndpoint)
}
