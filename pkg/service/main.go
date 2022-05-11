// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020-2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/autodiscovery"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/autoevent"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/clients"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/messaging"
	"os"

	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/handlers"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/interfaces"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"

	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/common"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/container"
	"github.com/gorilla/mux"
)

var instanceName string

func Main(serviceName string, serviceVersion string, proto interface{}, ctx context.Context, cancel context.CancelFunc, router *mux.Router) {
	startupTimer := startup.NewStartUpTimer(serviceName)

	additionalUsage :=
		"    -i, --instance                  Provides a service name suffix which allows unique instance to be created\n" +
			"                                    If the option is provided, service name will be replaced with \"<name>_<instance>\"\n"
	sdkFlags := flags.NewWithUsage(additionalUsage)
	sdkFlags.FlagSet.StringVar(&instanceName, "instance", "", "")
	sdkFlags.FlagSet.StringVar(&instanceName, "i", "", "")
	sdkFlags.Parse(os.Args[1:])

	serviceName = setServiceName(serviceName)
	ds = &DeviceService{}
	ds.Initialize(serviceName, serviceVersion, proto)

	ds.flags = sdkFlags

	ds.dic = di.NewContainer(di.ServiceConstructorMap{
		container.ConfigurationName: func(get di.Get) interface{} {
			return ds.config
		},
		container.DeviceServiceName: func(get di.Get) interface{} {
			return ds.deviceService
		},
		container.ProtocolDriverName: func(get di.Get) interface{} {
			return ds.driver
		},
		container.ProtocolDiscoveryName: func(get di.Get) interface{} {
			return ds.discovery
		},
		container.DeviceValidatorName: func(get di.Get) interface{} {
			return ds.validator
		},
	})

	httpServer := handlers.NewHttpServer(router, true)

	bootstrap.Run(
		ctx,
		cancel,
		sdkFlags,
		ds.ServiceName,
		common.ConfigStemDevice,
		ds.config,
		startupTimer,
		ds.dic,
		true,
		[]interfaces.BootstrapHandler{
			httpServer.BootstrapHandler,
			messaging.BootstrapHandler,
			clients.BootstrapHandler,
			handlers.NewClientsBootstrap().BootstrapHandler,
			autoevent.BootstrapHandler,
			NewBootstrap(router).BootstrapHandler,
			autodiscovery.BootstrapHandler,
			handlers.NewStartMessage(serviceName, serviceVersion).BootstrapHandler,
		})

	ds.Stop(false)
}

func setServiceName(name string) string {
	envValue := os.Getenv(common.EnvInstanceName)
	if len(envValue) > 0 {
		instanceName = envValue
	}

	if len(instanceName) > 0 {
		name = name + "_" + instanceName
	}

	return name
}