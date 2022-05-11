// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"edgeGateway/internal/pkg/edgexsdk/go-mod-messaging/messaging"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
)

var MessagingClientName = di.TypeInstanceToName((*messaging.MessageClient)(nil))

func MessagingClientFrom(get di.Get) messaging.MessageClient {
	client, ok := get(MessagingClientName).(messaging.MessageClient)
	if !ok {
		return nil
	}

	return client
}
