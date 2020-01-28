// Copyright 2019 The OpenSDS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package ontap

import (
	"fmt"
	"github.com/netapp/trident/storage"
	"github.com/netapp/trident/storage_drivers/fake"
	"github.com/netapp/trident/storage_drivers/ontap/api/azgo"
	"github.com/opensds/opensds/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"reflect"
	"testing"

	tridentconfig "github.com/netapp/trident/config"
	fakestorage "github.com/netapp/trident/storage/fake"
	sa "github.com/netapp/trident/storage_attribute"
	drivers "github.com/netapp/trident/storage_drivers"
	tu "github.com/netapp/trident/storage_drivers/fake/test_utils"
	//testutils "github.com/netapp/trident/storage_drivers/fake/test_utils"
	. "github.com/opensds/opensds/contrib/drivers/utils/config"
	pb "github.com/opensds/opensds/pkg/model/proto"
	"github.com/opensds/opensds/pkg/utils/config"
)

// SANStorageDriverMock
type SANStorageDriverMock struct {
	mock.Mock
	storageDriver *fake.StorageDriver
}

func NewSANStorageDriverMock(t *testing.T) (*SANStorageDriverMock, error) {
	/*mockPools := tu.GetFakePools()
	vpool, vpools := tu.GetFakeVirtualPools()
	t.Logf("Virtual Pool {%+v} ", vpool)
	t.Logf("Virtual Pools {%+v} ", vpools)
	fakeConfig, err := fake.NewFakeStorageDriverConfigJSONWithVirtualPools("mock", tridentconfig.Block, mockPools, vpool, vpools)
	if err != nil {
		err = fmt.Errorf("Unable to construct config JSON.")
		return nil, err
	}*/
	volumes := make([]fakestorage.Volume, 0)
	fakeConfig, err := fake.NewFakeStorageDriverConfigJSON("test", tridentconfig.Block,
		tu.GenerateFakePools(2), volumes)

	// Parse the common config struct from JSON
	commonConfig, err := drivers.ValidateCommonSettings(fakeConfig)
	if err != nil {
		err = fmt.Errorf("input failed validation: %v", err)
		return nil, err
	}

	md := &SANStorageDriverMock{}

	if initializeErr := md.Initialize(
		tridentconfig.CurrentDriverContext, fakeConfig, commonConfig); initializeErr != nil {
		err = fmt.Errorf("problem initializing storage driver '%s': %v",
			commonConfig.StorageDriverName, initializeErr)
		return nil, err
	}
	t.Logf("Fake Config {%+v} ", fakeConfig)
	return md, nil
}

func (md *SANStorageDriverMock) Initialize(
	context tridentconfig.DriverContext, configJSON string, commonConfig *drivers.CommonStorageDriverConfig,
) error {

	storageDriver := &fake.StorageDriver{}

	if err := storageDriver.Initialize(
		tridentconfig.CurrentDriverContext, configJSON, commonConfig); err != nil {
		err = fmt.Errorf("problem initializing storage driver '%s': %v",
			commonConfig.StorageDriverName, err)
		return err
	}
	md.storageDriver =storageDriver
	return nil
}

func (md *SANStorageDriverMock) Create(
	volConfig *storage.VolumeConfig, storagePool *storage.Pool, volAttributes map[string]sa.Request,
) error {

	//storageDriver := &fake.StorageDriver{}

	if err := md.storageDriver.Create(volConfig, storagePool, volAttributes); err != nil {
		err = fmt.Errorf("problem creating volume '%s': %v",
			volConfig.Name, err)
		return err
	}
	return nil
}

func (md *SANStorageDriverMock) CreateClone(
	volConfig *storage.VolumeConfig,
) error {

	//storageDriver := &fake.StorageDriver{}

	if err := md.storageDriver.CreateClone(volConfig); err != nil {
		err = fmt.Errorf("problem creating volume '%s': %v",
			volConfig.Name, err)
		return err
	}
	return nil
}

func GetSANDriver(t *testing.T) *SANDriver {
	var d = &SANDriver{}
	config.CONF.OsdsDock.Backends.NetappOntapSan.ConfigPath = "testdata/netapp_ontap_san.yaml"
	// Save current function and restore at the end:
	old := initialize
	defer func() { initialize = old }()

	initialize = func(context tridentconfig.DriverContext, configJSON string, commonConfig *drivers.CommonStorageDriverConfig) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		t.Logf("Mock Driver {%+v} created successfully ", md)
		return nil
	}
	return d
}

func TestSetup(t *testing.T) {
    var d = &SANDriver{}
    config.CONF.OsdsDock.Backends.NetappOntapSan.ConfigPath = "testdata/netapp_ontap_san.yaml"

     expectedPool :=  map[string] PoolProperties{
		 "pool-0": {
			 StorageType:      "block",
			 AvailabilityZone: "default",
			 MultiAttach:      true,
			 Extras: model.StoragePoolExtraSpec{
				 DataStorage: model.DataStorageLoS{
					 ProvisioningPolicy: "Thin",
					 Compression:        false,
					 Deduplication:      false,
				 },
				 IOConnectivity: model.IOConnectivityLoS{
					 AccessProtocol: "iscsi",
				 },
			 },
		 },
	 }
     expectedBackend := BackendOptions{
		Version:           1,
		StorageDriverName: "ontap-san",
		ManagementLIF: "127.0.0.1",
		DataLIF:       "127.0.0.1",
		IgroupName:    "opensds",
		Svm:           "vserver",
		Username:      "admin",
		Password:      "password",
	}
    expectedDriver := &SANDriver{
        conf: &ONTAPConfig{
            BackendOptions: expectedBackend,
            Pool : expectedPool,
        },
	}

	// Save current function and restore at the end:
	old := initialize
	defer func() { initialize = old }()

	initialize = func(context tridentconfig.DriverContext, configJSON string, commonConfig *drivers.CommonStorageDriverConfig) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		t.Logf("Mock Driver {%+v} created successfully ", md)
		return nil
	}

	if err := d.Setup(); err != nil {
		t.Errorf("Setup lvm driver failed: %+v\n", err)
	}

	if !reflect.DeepEqual(d.conf, expectedDriver.conf) {
		t.Errorf("Expected %+v, got %+v", expectedDriver.conf, d.conf)
	}
	t.Logf("Expected %+v, got %+v", expectedDriver.conf, d.conf)
}

func TestCreateVolume(t *testing.T) {
	var d = GetSANDriver(t)

	opt := &pb.CreateVolumeOpts{
		Id:          "e1bb066c-5ce7-46eb-9336-25508cee9f71",
		Name:        "testOntapVol1",
		Description: "volume for testing netapp ontap",
		Size:        int64(1),
		PoolName:    "pool-0",
	}
	var expected = &model.VolumeSpec{
		BaseModel:   &model.BaseModel{Id: "e1bb066c-5ce7-46eb-9336-25508cee9f71"},
		Name:        "testOntapVol1",
		Description: "volume for testing netapp ontap",
		Size:        int64(1),
		Identifier:  &model.Identifier{DurableName: "60a98000486e542d4f5a2f47694d684b", DurableNameFormat: "NAA"},
		Metadata: map[string]string{
			"lunPath": "/vol/opensds_e1bb066c5ce746eb933625508cee9f71/lun0",
		},
	}

	// Save current function and restore at the end:
	old := create
	old1 := LunGetSerialNumber
	defer func() { create = old; LunGetSerialNumber = old1}()

	var serialNumber = "HnT-OZ/GiMhK"
	LunGetSerialNumber = func(lunPath string) (*azgo.LunGetSerialNumberResponse, error) {
		return &azgo.LunGetSerialNumberResponse{
			Result: azgo.LunGetSerialNumberResponseResult{
				SerialNumberPtr: &serialNumber,
			},
		}, nil
	}

	create = func(volConfig *storage.VolumeConfig, storagePool *storage.Pool, volAttributes map[string]sa.Request) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		var name = getVolumeName(opt.GetId())
		volConfig = d.GetVolumeConfig(name, opt.GetSize())
		storagePool = &storage.Pool{
			Name:               opt.GetPoolName(),
			StorageClasses:     make([]string, 0),
			Attributes:         make(map[string]sa.Offer),
			InternalAttributes: make(map[string]string),
		}
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		err = md.Create(volConfig, storagePool, make(map[string]sa.Request))
		if err != nil {
			t.Error(err)
		}
		return nil
	}

	vol, err := d.CreateVolume(opt)
	if err != nil {
		t.Error("Failed to create volume:", err)
	}

	if !reflect.DeepEqual(vol, expected) {
		t.Errorf("Expected %+v, got %+v\n", expected, vol)
	}
	t.Logf("Expected %+v, got %+v\n", expected, vol)

	assert.Equalf(t, vol, expected, "The two values should be the same.")
}

func TestCreateVolumeFromSnapshot(t *testing.T) {
	var d = GetSANDriver(t)

	opt := &pb.CreateVolumeOpts{
		Id:          "e1bb066c-5ce7-46eb-9336-25508cee9f71",
		Name:        "testOntapVol1",
		Description: "volume for testing netapp ontap",
		Size:        int64(1),
		PoolName:    "pool-0",
		SnapshotId:   "3769855c-a102-11e7-b772-17b880d2f537",
	}
	var expected = &model.VolumeSpec{
		BaseModel:   &model.BaseModel{Id: "e1bb066c-5ce7-46eb-9336-25508cee9f71"},
		Name:        "testOntapVol1",
		Description: "volume for testing netapp ontap",
		Size:        int64(1),
		Identifier:  &model.Identifier{DurableName: "60a98000486e542d4f5a2f47694d684b", DurableNameFormat: "NAA"},
		Metadata: map[string]string{
			"lunPath": "/vol/opensds_e1bb066c5ce746eb933625508cee9f71/lun0",
		},
	}

	// Save current function and restore at the end:
	old := createClone
	old1 := LunGetSerialNumber
	defer func() { createClone = old; LunGetSerialNumber = old1}()

	var serialNumber = "HnT-OZ/GiMhK"
	LunGetSerialNumber = func(lunPath string) (*azgo.LunGetSerialNumberResponse, error) {
		return &azgo.LunGetSerialNumberResponse{
			Result: azgo.LunGetSerialNumberResponseResult{
				SerialNumberPtr: &serialNumber,
			},
		}, nil
	}

	createClone = func(volConfig *storage.VolumeConfig) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		var name = getVolumeName(opt.GetId())
		volConfig = d.GetVolumeConfig(name, opt.GetSize())

		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		err = md.CreateClone(volConfig)
		if err != nil {
			t.Error(err)
		}
		return nil
	}

	vol, err := d.CreateVolume(opt)
	if err != nil {
		t.Error("Failed to create volume:", err)
	}

	if !reflect.DeepEqual(vol, expected) {
		t.Errorf("Expected %+v, got %+v\n", expected, vol)
	}
	t.Logf("Expected %+v, got %+v\n", expected, vol)

	assert.Equalf(t, vol, expected, "The two values should be the same.")
}