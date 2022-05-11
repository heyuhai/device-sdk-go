// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"edgeGateway/internal/pkg/db/mysql"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/provision"
	"edgeGateway/internal/pkg/logger"
	"fmt"
	"sync"
	"time"

	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/startup"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
	"github.com/gorilla/mux"

	"github.com/edgexfoundry/device-sdk-go/v2/internal/cache"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
)

// Bootstrap contains references to dependencies required by the BootstrapHandler.
type Bootstrap struct {
	router *mux.Router
}

// NewBootstrap is a factory method that returns an initialized Bootstrap receiver struct.
func NewBootstrap(router *mux.Router) *Bootstrap {
	return &Bootstrap{
		router: router,
	}
}

func (b *Bootstrap) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup, startupTimer startup.Timer, dic *di.Container) (success bool) {
	ds.UpdateFromContainer(b.router, dic)
	ds.ctx = ctx
	ds.wg = wg
	ds.controller.InitRestRoutes()

	err := cache.InitCache(ds.ServiceName, dic)
	if err != nil {
		ds.LoggingClient.Errorf("Failed to init cache: %v", err)
		return false
	}

	if ds.AsyncReadings() {
		ds.asyncCh = make(chan *models.AsyncValues, ds.config.Device.AsyncBufferSize)
		go ds.processAsyncResults(ctx, wg, dic)
	}
	if ds.DeviceDiscovery() {
		ds.deviceCh = make(chan []models.DiscoveredDevice, 1)
		go ds.processAsyncFilterAndAdd(ctx, wg)
	}

	e := ds.driver.Initialize(ds.LoggingClient, ds.asyncCh, ds.deviceCh)
	if e != nil {
		ds.LoggingClient.Errorf("Failed to init ProtocolDriver: %v", e)
		return false
	}
	ds.initialized = true

	err = ds.selfRegister()
	if err != nil {
		ds.LoggingClient.Errorf("Failed to register service on Metadata: %v", err)
		return false
	}

	// TODO: 增加容器注入依赖
	var err1 error
	// 初始化日志
	loggerConfig, err1 := logger.InitLogger(ds.config.Logger)
	if err != nil {
		logger.Fatal("Failed to initialize logs. Procedure！error: %+v", err)
		ds.LoggingClient.Errorf("Failed to initialize logs. Procedure！error: %+v", err1)
		return false
	}
	logger.SetAppName(ds.config.Systeminfo.APP)

	// 初始化mysql数据库
	mysqlConfig := ds.config.Mysql
	dataSourceName := mysqlConfig.Name + ":" + mysqlConfig.Password +
		"@(" + fmt.Sprintf("%s:%d", mysqlConfig.Host, mysqlConfig.Port) + ")/" + mysqlConfig.DatabaseName + "?charset=utf8"
	mysqlClient, err1 := mysql.NewMysqlDb("mysql", dataSourceName)
	if err1 != nil {
		logger.Fatal("mysql initialization failed. Procedure ！error: %+v", err1)
		ds.LoggingClient.Errorf("mysql initialization failed. Procedure ！error: %+v", err1)
		return false
	}
	if mysqlClient != nil {
		logger.Warn("Mysql initial success !!!")
	}
	mysqlClient.SetMaxOpenConns(mysqlConfig.MaxOpenConns)
	mysqlClient.SetMaxIdleConns(mysqlConfig.MaxIdleConns)
	mysqlClient.SetConnMaxLifetime(time.Minute * time.Duration(mysqlConfig.ConnMaxLifetime))

	dic.Update(di.ServiceConstructorMap{
		mysql.MysqlClientName: func(get di.Get) interface{} {
			return mysqlClient
		},
		logger.LogConfigName: func(get di.Get) interface{} {
			return loggerConfig
		},
	})

	// TODO: 修改加载配置 改成 读取数据库
	// // err = provision.LoadProfiles(ds.config.Device.ProfilesDir, dic)
	// err = provision.LoadProfilesByMysql(ds.config.CustomDeviceSrv.ProtocolType, dic)
	// if err != nil {
	// 	logger.Fatal("Failed to create the pre-defined device profiles: %v", err)
	// 	ds.LoggingClient.Errorf("Failed to create the pre-defined device profiles: %v", err)
	// 	return false
	// }
	// // err = provision.LoadDevices(ds.config.Device.DevicesDir, dic)
	// err = provision.LoadDevicesByMysql(ds.config.CustomDeviceSrv.ProtocolType, dic)
	// if err != nil {
	// 	logger.Fatal("Failed to create the pre-defined devices: %v", err)
	// 	ds.LoggingClient.Errorf("Failed to create the pre-defined devices: %v", err)
	// 	return false
	// }
	err = provision.LoadDeviceAndProfilesByMysql(ds.config.CustomDeviceSrv.ProtocolType, dic)
	if err != nil {
		logger.Fatal("Failed to create the pre-defined: %v", err)
		ds.LoggingClient.Errorf("Failed to create the pre-defined: %v", err)
		return false
	}

	ds.manager.StartAutoEvents()

	return true
}
