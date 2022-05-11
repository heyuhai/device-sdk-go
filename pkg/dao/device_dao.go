package dao

// TODO: 新增dao
import (
	"database/sql"
	"edgeGateway/internal/core/devicesrvs/modbus/driver"
	"edgeGateway/internal/pkg/common/deviceregister"
	"edgeGateway/internal/pkg/db/mysql"
	edgeGatewayDi "edgeGateway/internal/pkg/di"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/config"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/internal/container"
	"edgeGateway/internal/pkg/edgexsdk/device-sdk-go/pkg/protocols"
	"edgeGateway/internal/pkg/logger"
	"edgeGateway/internal/pkg/util"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

const queryDeviceSqlCommon = "SELECT d.id, d.name, d.hwPlatformId, d.productName, d.status, d.online, d.description, d.deviceAddress, d.slaveId, d.appName, ac.protocolType, ac.baudRate, ac.dataBit, ac.stopBit, ac.check, dp.serviceName, dp.propertyName FROM t_device as d INNER JOIN t_app_config as ac ON d.appName = ac.appName LEFT JOIN t_device_property AS dp ON d.hwPlatformId = dp.hwPlatformId where dp.serviceName != \"\" AND dp.propertyName != \"\" "

type DeviceDao struct {
	Id            int64
	HwPlatformId  string
	Name          string
	ProductName   string
	Status        int64
	Online        int64
	Description   string
	DeviceAddress string
	SlaveId       string
	AppName       string
	ProtocolType  int64
	BaudRate      string
	DataBit       string
	StopBit       string
	Check         string
	ServiceName   string
	PropertyName  string
}

// QueryByProtocolType 查询设备跟点表信息
func (d *DeviceDao) QueryByProtocolType(dic *di.Container, protocolType int) ([]dtos.Device, []dtos.DeviceProfile, error) {
	mysqlClient := mysql.MysqlClientNameFrom(dic.Get)
	c := container.ConfigurationFrom(dic.Get)

	// d.status = 1 and d.online = 1 默认启动时加载状态成功的设备
	sqlStr := queryDeviceSqlCommon + " AND ac.protocolType = ?"

	devices, err := d.AddDevice(mysqlClient, c, sqlStr, protocolType)
	if err != nil {
		return nil, nil, err
	}
	if len(devices) == 0 {
		return nil, nil, errors.New("len(devices) is zero")
	}

	var deviceNames []interface{}
	for _, device := range devices {
		deviceNames = append(deviceNames, device.Name)
	}
	if len(deviceNames) == 0 {
		return nil, nil, errors.New("len(deviceNames) is zero")
	}

	// 查询点表属性
	modelDao := new(ModelDao)
	deviceProfile, err := modelDao.QueryModelByDeviceNames(mysqlClient, deviceNames)
	if err != nil {
		return nil, nil, err
	}
	if len(deviceProfile) == 0 {
		return nil, nil, errors.New("len(deviceProfile) is zero")
	}
	return devices, deviceProfile, nil
}

func (d *DeviceDao) QueryByGatewayByIds(dic *edgeGatewayDi.Container, ids []interface{}) ([]dtos.Device, []dtos.DeviceProfile, error) {
	mysqlClient := mysql.MysqlClientNameFrom(dic.Get)

	// d.status = 1 and d.online = 1 默认启动时加载状态成功的设备
	sqlStr := queryDeviceSqlCommon + " AND d.hwPlatformId IN (?" + strings.Repeat(",?", len(ids)-1) + ")"

	devices, err := d.AddDevice(mysqlClient, nil, sqlStr, ids...)
	if err != nil {
		return nil, nil, err
	}
	if len(devices) == 0 {
		return nil, nil, errors.New("len(devices) is zero")
	}

	var deviceNames []interface{}
	for _, device := range devices {
		deviceNames = append(deviceNames, device.Name)
	}
	if len(deviceNames) == 0 {
		return nil, nil, errors.New("len(deviceNames) is zero")
	}

	// 查询点表属性
	modelDao := new(ModelDao)
	deviceProfile, err := modelDao.QueryModelByDeviceNames(mysqlClient, deviceNames)
	if err != nil {
		return nil, nil, err
	}
	if len(deviceProfile) == 0 {
		return nil, nil, errors.New("len(deviceProfile) is zero")
	}
	return devices, deviceProfile, nil
}

func (d *DeviceDao) QueryDeviceByGatewayByAppNames(dic *edgeGatewayDi.Container, appNames []interface{}) ([]dtos.Device, error) {
	mysqlClient := mysql.MysqlClientNameFrom(dic.Get)

	// d.status = 1 and
	sqlStr := queryDeviceSqlCommon + " AND d.appName IN (?" + strings.Repeat(",?", len(appNames)-1) + ")"

	return d.AddDevice(mysqlClient, nil, sqlStr, appNames...)
}

func (d *DeviceDao) AddDevice(mysqlClient *mysql.MysqlClient, c *config.ConfigurationStruct, sqlStr string, whereSqlArgs ...interface{}) ([]dtos.Device, error) {
	// var (
	// 	whereSqlArgs []interface{}
	// )
	//
	// whereSqlArgs = append(whereSqlArgs, protocolType)

	dataDuplicates := make(map[string]DeviceDao)
	autoEvents := make(map[string][]dtos.AutoEvent)
	err := mysqlClient.QueryRowsOp(sqlStr, func(rows *sql.Rows) error {
		for rows.Next() {
			var id, protocolType, statusVal, online sql.NullInt64
			var name, hwPlatformId, productName, description, deviceAddress, slaveId, appName, baudRate, dataBit, stopBit, check, serviceName, propertyName sql.NullString
			err := rows.Scan(&id, &name, &hwPlatformId, &productName, &statusVal, &online, &description, &deviceAddress, &slaveId, &appName, &protocolType, &baudRate, &dataBit, &stopBit, &check, &serviceName, &propertyName)
			if err != nil {
				return err
			}
			deviceName := hwPlatformId.String
			// deviceName不能包含中文，不然有问题
			if util.IsChineseChar(deviceName) {
				logger.Warn("hwPlatformId IsChineseChar Error", deviceName)
				continue
			}
			dataDuplicates[deviceName] = DeviceDao{
				Id:            id.Int64,
				HwPlatformId:  hwPlatformId.String,
				Name:          name.String,
				ProductName:   productName.String,
				Status:        statusVal.Int64,
				Online:        online.Int64,
				Description:   description.String,
				DeviceAddress: deviceAddress.String,
				SlaveId:       slaveId.String,
				AppName:       appName.String,
				ProtocolType:  protocolType.Int64,
				BaudRate:      baudRate.String,
				DataBit:       dataBit.String,
				StopBit:       stopBit.String,
				Check:         check.String,
				ServiceName:   serviceName.String,
				PropertyName:  propertyName.String,
			}
			// AutoEvents 默认x分钟上送当前设备关联的所有资产
			if serviceName.String != "" && propertyName.String != "" {
				var autoEventsOnchange bool
				autoEventsInterval := "5m"
				if c != nil {
					autoEventsInterval = c.CustomDeviceSrv.Device.AutoEventsInterval
					if autoEventsInterval == "" {
						autoEventsInterval = "5m"
					}
					autoEventsOnchange = c.CustomDeviceSrv.Device.AutoEventsOnChange
				}

				autoEvents[deviceName] = append(autoEvents[deviceName], dtos.AutoEvent{
					Interval:   autoEventsInterval, // 1s 1m 1h ?
					OnChange:   autoEventsOnchange,
					SourceName: util.GetEdgeXResourceName(serviceName.String, propertyName.String),
				})
			}
		}
		return nil
	}, whereSqlArgs...)
	if err != nil {
		return nil, err
	}

	protocolsTimeout := "5"
	protocolsIdleTimeout := "5"
	if c != nil {
		protocolsTimeout = c.CustomDeviceSrv.Device.ProtocolsTimeout
		if protocolsTimeout == "" {
			protocolsTimeout = "5"
		}
		protocolsIdleTimeout = c.CustomDeviceSrv.Device.ProtocolsIdleTimeout
		if protocolsIdleTimeout == "" {
			protocolsIdleTimeout = "5"
		}
	}

	var devices []dtos.Device
	for _, item := range dataDuplicates {
		deviceProtocols := protocols.ModbusRtuProtocols{
			Address:  item.DeviceAddress,
			BaudRate: item.BaudRate,
			DataBits: item.DataBit,
			StopBits: item.StopBit,
			UnitID:   item.SlaveId,
			Parity:   item.Check,
		}
		deviceProtocols.Parity = deviceProtocols.FormatParity()
		deviceProtocols.Timeout = protocolsTimeout
		deviceProtocols.IdleTimeout = protocolsIdleTimeout
		autoEventsItem, ok := autoEvents[item.HwPlatformId]
		if !ok {
			autoEventsItem = nil
		}

		operatingState := models.Down
		adminState := models.Locked
		if item.Status == 1 {
			operatingState = models.Up
		}
		if item.Online == 1 {
			adminState = models.Unlocked
		}

		device := dtos.Device{
			Name:           item.HwPlatformId,
			Description:    item.Description,
			ProfileName:    item.HwPlatformId,
			AutoEvents:     autoEventsItem,
			Protocols:      deviceProtocols.Struct2MapProtocolProperties(),
			ServiceName:    protocols.ProtocolType(item.ProtocolType).GetServiceName(),
			AdminState:     adminState,
			OperatingState: operatingState,
		}
		devices = append(devices, device)
	}

	return devices, nil
}

type ModelDao struct {
	Id              int64
	ProductName     string
	ServiceName     string
	PropertyName    string
	RegistryAddress string
	RegisterLength  string
	DataType        string
	Operation       string
	Method          string
	Max             string
	Min             string
	FunctionCode    string
}

func (d *ModelDao) QueryModelByDeviceNames(mysqlClient *mysql.MysqlClient, deviceNames []interface{}) ([]dtos.DeviceProfile, error) {
	if mysqlClient == nil {
		return nil, errors.New("mysqlClient is nil")
	}
	if len(deviceNames) == 0 {
		return nil, errors.New("deviceNames length is 0")
	}

	sqlModelStr := "select id, hwPlatformId, serviceName, propertyName, registryAddress, registerLength, dataType, operation, method, max, min, functionCode from t_device_property where hwPlatformId IN (?" + strings.Repeat(",?", len(deviceNames)-1) + ")"

	return d.QueryModel(mysqlClient, nil, sqlModelStr, deviceNames...)
}

func (d *ModelDao) QueryModel(mysqlClient *mysql.MysqlClient, c *config.ConfigurationStruct, sqlModelStr string, whereSqlArgs ...interface{}) ([]dtos.DeviceProfile, error) {
	// productName, serviceName+propertyName
	deviceProfileMap := make(map[string][]ModelDao)
	err := mysqlClient.QueryRowsOp(sqlModelStr, func(rows *sql.Rows) error {
		for rows.Next() {
			var id sql.NullInt64
			var hwPlatformId, serviceName, propertyName, registryAddress, registerLength, dataType, operation, method, max, min, functionCode sql.NullString
			err := rows.Scan(&id, &hwPlatformId, &serviceName, &propertyName, &registryAddress, &registerLength, &dataType, &operation, &method, &max, &min, &functionCode)
			if err != nil {
				return err
			}
			deviceProfileMap[hwPlatformId.String] = append(deviceProfileMap[hwPlatformId.String], ModelDao{
				ProductName:     hwPlatformId.String,
				ServiceName:     serviceName.String,
				PropertyName:    propertyName.String,
				RegistryAddress: registryAddress.String,
				RegisterLength:  registerLength.String,
				DataType:        dataType.String,
				Operation:       operation.String, // TODO: 暂没用
				Method:          method.String,
				Max:             max.String,
				Min:             min.String,
				FunctionCode:    functionCode.String,
			})
		}
		return nil
	}, whereSqlArgs...)
	if err != nil {
		return nil, err
	}

	var deviceProfiles []dtos.DeviceProfile
	for name, items := range deviceProfileMap {
		deviceProfile := dtos.DeviceProfile{}
		deviceProfile.Name = name
		// deviceProfile.Model = protocols.ProtocolType(protocolType).String()
		deviceProfile.Description = ""
		var deviceResources []dtos.DeviceResource
		for _, model := range items {
			attr := make(map[string]interface{})
			functionCode, err := strconv.Atoi(model.FunctionCode)
			if err != nil {
				logger.Warn("FunctionCode strconv int Error", model.FunctionCode, model.Id)
				continue
			}
			registryAddress, err := strconv.ParseUint(model.RegistryAddress, 16, 32)
			if err != nil {
				logger.Warn("RegistryAddress strconv 16进制 uint Error", model.RegistryAddress, model.Id)
				continue
			}
			registerLength, err := strconv.ParseUint(model.RegisterLength, 16, 32)
			if err != nil {
				logger.Warn("RegisterLength strconv 16进制 uint Error", model.RegisterLength, model.Id)
				continue
			}
			attr[driver.PRIMARY_TABLE] = deviceregister.PrimaryTableFormatOpMod(functionCode)
			attr[driver.STARTING_ADDRESS] = registryAddress
			attr[driver.STRING_REGISTER_SIZE] = registerLength
			attr[driver.Operation] = model.Operation
			deviceResources = append(deviceResources, dtos.DeviceResource{
				// Description: model.,
				Name:     util.GetEdgeXResourceName(model.ServiceName, model.PropertyName),
				IsHidden: false,
				Properties: dtos.ResourceProperties{
					ValueType:    model.DataType,
					ReadWrite:    model.Method,
					Minimum:      model.Min,
					Maximum:      model.Max,
					DefaultValue: "",
					Mask:         "",
					Shift:        "",
					Scale:        "",
					Offset:       "",
					Base:         "",
					Assertion:    "",
					MediaType:    "",
				},
				Attributes: attr,
			})
		}
		deviceProfile.DeviceResources = deviceResources
		// TODO: DeviceCommands?
		// deviceProfile.DeviceCommands

		deviceProfiles = append(deviceProfiles, deviceProfile)
	}

	return deviceProfiles, nil
}
