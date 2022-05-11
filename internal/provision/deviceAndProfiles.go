package provision

import (
	"context"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/cache"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/container"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/dao"
	"edgeGateway/internal/pkg/logger"
	"fmt"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/errors"
	"github.com/google/uuid"
)

// LoadDeviceAndProfilesByMysql 从mysql读取信息
// TODO: mysql
func LoadDeviceAndProfilesByMysql(protocolType int, dic *di.Container) errors.EdgeX {
	deviceList, deviceProfiles, err := new(dao.DeviceDao).QueryByProtocolType(dic, protocolType)
	if err != nil {
		logger.Warn("Failed to LoadDeviceAndProfilesByMysql the pre-defined deviceAndProfiles: %v", err)
		return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("Failed to LoadDeviceAndProfilesByMysql the pre-defined deviceAndProfiles"), err)
	}

	// 先增加物模型
	var addProfilesReq []requests.DeviceProfileRequest
	dpc := bootstrapContainer.DeviceProfileClientFrom(dic.Get)
	logger.Info("Loading pre-defined profiles from mysql")
	for _, profile := range deviceProfiles {
		res, err := dpc.DeviceProfileByName(context.Background(), profile.Name)
		if err == nil {
			logger.Info("Profile %s exists, using the existing one", profile.Name)
			_, exist := cache.Profiles().ForName(profile.Name)
			if !exist {
				err = cache.Profiles().Add(dtos.ToDeviceProfileModel(res.Profile))
				if err != nil {
					return errors.NewCommonEdgeX(errors.KindServerError, fmt.Sprintf("failed to cache the profile %s", res.Profile.Name), err)
				}
			}
		} else {
			logger.Info("Profile %s not found in Metadata, adding it ...", profile.Name)
			req := requests.NewDeviceProfileRequest(profile)
			addProfilesReq = append(addProfilesReq, req)
		}
	}

	if len(addProfilesReq) > 0 {
		ctx := context.WithValue(context.Background(), common.CorrelationHeader, uuid.NewString()) // nolint:staticcheck
		_, edgexErr := dpc.Add(ctx, addProfilesReq)
		if edgexErr != nil {
			return edgexErr
		}
	}

	// 后增加设备
	var addDevicesReq []requests.AddDeviceRequest
	serviceName := container.DeviceServiceFrom(dic.Get).Name
	logger.Info("Loading pre-defined devices from mysql")
	for _, device := range deviceList {
		if _, ok := cache.Devices().ForName(device.Name); ok {
			logger.Info("Device %s exists, using the existing one", device.Name)
		} else {
			logger.Info("Device %s not found in Metadata, adding it ...", device.Name)
			device.ServiceName = serviceName
			// device.AdminState = models.Unlocked
			// device.OperatingState = models.Up
			req := requests.NewAddDeviceRequest(device)
			addDevicesReq = append(addDevicesReq, req)
		}
	}

	if len(addDevicesReq) > 0 {
		dc := bootstrapContainer.DeviceClientFrom(dic.Get)
		ctx := context.WithValue(context.Background(), common.CorrelationHeader, uuid.NewString()) // nolint: staticcheck
		_, edgexErr := dc.Add(ctx, addDevicesReq)
		return edgexErr
	}

	return nil
}
