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
	"github.com/netapp/trident/utils"
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
	odu "github.com/opensds/opensds/contrib/drivers/utils"
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
		tu.GenerateFakePools(1), volumes)

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

	md.storageDriver.CreatePrepare(volConfig)

	md.storageDriver.Volumes[volConfig.CloneSourceVolume] = fakestorage.Volume{
		Name:          volConfig.CloneSourceVolume,
		PhysicalPool: "pool-0",
	}
	if err := md.storageDriver.CreateClone(volConfig); err != nil {
		err = fmt.Errorf("problem creating clone volume '%s': %v",
			volConfig.Name, err)
		return err
	}
	return nil
}

func (md *SANStorageDriverMock) Destroy(name string) error {

	if err := md.storageDriver.Destroy(name); err != nil {
		err = fmt.Errorf("problem deleting volume '%s': %v",
			name, err)
		return err
	}
	return nil
}

func (md *SANStorageDriverMock) resize(volConfig *storage.VolumeConfig, sizeBytes uint64) error {

	if err := md.storageDriver.Resize(volConfig,sizeBytes); err != nil {
		err = fmt.Errorf("problem resizing volume '%s': %v",
			volConfig.Name, err)
		return err
	}
	return nil
}

func (md *SANStorageDriverMock) publish(name string, publishInfo *utils.VolumePublishInfo) error {

	if err := md.storageDriver.Publish(name, publishInfo); err != nil {
		err = fmt.Errorf("problem publishing connection '%s': %v",
			name, err)
		return err
	}
	return nil
}

func (md *SANStorageDriverMock) CreateSnapshot(snapConfig *storage.SnapshotConfig) (*storage.Snapshot, error) {

	md.storageDriver.Volumes[snapConfig.VolumeInternalName] = fakestorage.Volume{
		Name:          snapConfig.VolumeInternalName,
		//PhysicalPool: "pool-0",
	}
	snapshot, err := md.storageDriver.CreateSnapshot(snapConfig);
	if err != nil {
		err = fmt.Errorf("problem creating volume snapshot '%s': %v",
			snapConfig.Name, err)
		return nil, err
	}
	return snapshot, nil
}

func (md *SANStorageDriverMock) DeleteSnapshot(snapConfig *storage.SnapshotConfig) error {

	if err := md.storageDriver.DeleteSnapshot(snapConfig); err != nil {
		err = fmt.Errorf("problem creating volume snapshot '%s': %v",
			snapConfig.Name, err)
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
		SnapshotId:   "3769855c-a102-11e7-b772-17b880d2f537",
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
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		var snapName = getSnapshotName(opt.GetSnapshotId())
		var volName = opt.GetMetadata()["volume"]
		var name = getVolumeName(opt.GetId())

		volConfig = d.GetVolumeConfig(name, opt.GetSize())
		volConfig.CloneSourceVolumeInternal = volName
		volConfig.CloneSourceVolume = volName
		volConfig.CloneSourceSnapshot = snapName
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
		t.Errorf("Expected %+v, Got %+v\n", expected, vol)
	}
	t.Logf("Expected %+v, Got %+v\n", expected, vol)

	//assert.Equalf(t, vol, expected, "The two values should be the same.")
}

func TestPullVolume(t *testing.T) {
	var d = GetSANDriver(t)
	volId := "e1bb066c-5ce7-46eb-9336-25508cee9f71"
	vol, err := d.PullVolume(volId)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method PullVolume has not been implemented yet"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to pull volume:", err)
	}
	if vol != nil{
		t.Errorf("Expected %+v, Got %+v\n", nil, vol)
	}
}

func TestDeleteVolume(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.DeleteVolumeOpts{
		Id: "e1bb066c-5ce7-46eb-9336-25508cee9f71",
	}
	// Save current function and restore at the end:
	old := destroy
	defer func() { destroy = old;}()
	destroy = func(name string) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		name = getVolumeName(opt.GetId())
		err = md.Destroy(name)
		if err != nil {
			t.Error(err)
		}
		return nil
	}
	if err := d.DeleteVolume(opt); err != nil {
		t.Error("Failed to delete volume:", err)
	}
	t.Logf("volume (%s) deleted successfully.", opt.GetId())
}

func TestExtendVolume(t *testing.T) {
	var d = GetSANDriver(t)

	opt := &pb.ExtendVolumeOpts{
		Id: "591c43e6-1156-42f5-9fbc-161153da185c",
		Size: int64(2),
	}

	// Save current function and restore at the end:
	old := resize
	defer func() { resize = old;}()
	resize = func(volConfig *storage.VolumeConfig, newSize uint64) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		var name = getVolumeName(opt.GetId())
		volConfig = d.GetVolumeConfig(name, opt.GetSize())

		newSize = uint64(opt.GetSize() * bytesGiB)
		err = md.resize(volConfig, newSize)
		if err != nil {
			t.Error(err)
		}
		return nil
	}

	vol, err := d.ExtendVolume(opt)
	if err != nil {
		t.Error("Failed to extend volume:", err)
	}

	if vol.Size != 2 {
		t.Errorf("Expected %+v, Got %+v\n", 2, vol.Size)
	}
	t.Logf("Got Extended Volume %+v\n", vol)
}

func TestInitializeConnection(t *testing.T) {
	var d = GetSANDriver(t)
	initiators := []*pb.Initiator{}
	initiator := &pb.Initiator{
		PortName:             "iqn.2020-01.io.opensds:example",
		Protocol:             "iscsi",
	}
	initiators = append(initiators, initiator)

	opt := &pb.CreateVolumeAttachmentOpts{
		Id:                   "591c43e6-1156-42f5-9fbc-161153da185c",
		VolumeId:             "e1bb066c-5ce7-46eb-9336-25508cee9f71",
		HostInfo:             &pb.HostInfo{
			OsType:               "linux",
			Host:                 "localhost",
			Ip:                   "127.0.0.1",
			Initiators:       initiators,
		},
		Metadata:             nil,
		DriverName:           "netapp_ontap_san",
		AccessProtocol:       "iscsi",
	}

	expected := &model.ConnectionInfo{
		DriverVolumeType: "iscsi",
		ConnectionData: map[string]interface{}{
			"target_discovered": true,
			"volumeId":          "e1bb066c-5ce7-46eb-9336-25508cee9f71",
			"volume":            "opensds_e1bb066c5ce746eb933625508cee9f71",
			"description":       "NetApp ONTAP Attachment",
			"hostName":          "localhost",
			"initiator":         "iqn.2020-01.io.opensds:example",
			"targetIQN":         []string{""},
			"targetPortal":      []string{"127.0.0.1" + ":3260"},
			"targetLun":         int32(0),
			"igroup":            "",
		},
	}

	// Save current function and restore at the end:
	old := publish
	defer func() { publish = old;}()
	publish = func(name string, publishInfo *utils.VolumePublishInfo) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		name = getVolumeName(opt.GetVolumeId())
		hostInfo := opt.GetHostInfo()
		initiator := odu.GetInitiatorName(hostInfo.GetInitiators(), opt.GetAccessProtocol())
		hostName := hostInfo.GetHost()
		publishInfo = &utils.VolumePublishInfo{
			HostIQN:  []string{initiator},
			HostIP:   []string{hostInfo.GetIp()},
			HostName: hostName,
		}

		err = md.publish(name, publishInfo)
		return nil
	}

	connectionInfo, err:= d.InitializeConnection(opt)
	if err != nil {
		t.Error("Failed to initialize connection:", err)
	}

	if !reflect.DeepEqual(connectionInfo, expected) {
		t.Errorf("Expected %+v, Got %+v\n", expected, connectionInfo)
	}
	t.Logf("Expected %+v, Got %+v\n", expected, connectionInfo)
}

func TestPullSnapshot(t *testing.T) {
	var d = GetSANDriver(t)
	snapshotId := "d1916c49-3088-4a40-b6fb-0fda18d074c3"
	snapshot, err := d.PullSnapshot(snapshotId)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method PullSnapshot has not been implemented yet"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to pull snapshot:", err)
	}
	if snapshot != nil{
		t.Errorf("Expected %+v, Got %+v\n", nil, snapshot)
	}
}

func TestInitializeSnapshotConnection(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.CreateSnapshotAttachmentOpts{}
	connectionInfo, err := d.InitializeSnapshotConnection(opt)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method InitializeSnapshotConnection has not been implemented yet."}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to initialize snapshot connection:", err)
	}
	if connectionInfo != nil{
		t.Errorf("Expected %+v, Got %+v\n", nil, connectionInfo)
	}
}

func TestTerminateSnapshotConnection(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.DeleteSnapshotAttachmentOpts{}
	err := d.TerminateSnapshotConnection(opt)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method TerminateSnapshotConnection has not been implemented yet."}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to terminate snapshot connection:", err)
	}
}

func TestCreateVolumeGroup(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.CreateVolumeGroupOpts{}
	vg, err := d.CreateVolumeGroup(opt)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method CreateVolumeGroup has not been implemented yet"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to create volume group:", err)
	}
	if vg != nil{
		t.Errorf("Expected %+v, Got %+v\n", nil, vg)
	}
}

func TestUpdateVolumeGroup(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.UpdateVolumeGroupOpts{}
	vg, err := d.UpdateVolumeGroup(opt)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method UpdateVolumeGroup has not been implemented yet"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to update volume group:", err)
	}
	if vg != nil{
		t.Errorf("Expected %+v, Got %+v\n", nil, vg)
	}
}

func TestDeleteVolumeGroup(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.DeleteVolumeGroupOpts{}
	err := d.DeleteVolumeGroup(opt)
	t.Logf("Expected Error %+v\n", err)
	expectedErr := &model.NotImplementError{"method DeleteVolumeGroup has not been implemented yet"}
	if !reflect.DeepEqual(err, expectedErr) {
		t.Error("Failed to delete volume group:", err)
	}
}

func TestCreateSnapshot(t *testing.T) {
	var d = GetSANDriver(t)

	opt := &pb.CreateVolumeSnapshotOpts{
		Id:          "d1916c49-3088-4a40-b6fb-0fda18d074c3",
		Name:        "snap001",
		Description: "volume snapshot for Netapp ontap testing",
		Size:        int64(1),
		VolumeId:    "e1bb066c-5ce7-46eb-9336-25508cee9f71",
	}
	var expected = &model.VolumeSnapshotSpec{
		BaseModel:   &model.BaseModel{Id: "d1916c49-3088-4a40-b6fb-0fda18d074c3"},
		Name:        "snap001",
		Description: "volume snapshot for Netapp ontap testing",
		Size:        int64(1),
		VolumeId:    "e1bb066c-5ce7-46eb-9336-25508cee9f71",
		Metadata: map[string]string{
			"name": "opensds_snapshot_d1916c4930884a40b6fb0fda18d074c3",
			"volume":       "opensds_e1bb066c5ce746eb933625508cee9f71",
			"creationTime": "2020-01-29T09:05:18Z",
			"size":         "0K",
		},
	}

	// Save current function and restore at the end:
	old := createSnapshot
	defer func() { createSnapshot = old;}()
	createSnapshot = func(snapConfig *storage.SnapshotConfig) (*storage.Snapshot, error) {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		var snapName = getSnapshotName(opt.GetId())
		var volName = getVolumeName(opt.GetVolumeId())

		snapConfig = d.GetSnapshotConfig(snapName, volName)

		snapshot, err := md.CreateSnapshot(snapConfig)
		if err != nil {
			t.Error(err)
		}
		snapshot.Created = "2020-01-29T09:05:18Z"
		return snapshot, nil
	}

	snapshot, err := d.CreateSnapshot(opt)
	if err != nil {
		t.Error("Failed to create volume snapshot:", err)
	}

	if !reflect.DeepEqual(snapshot, expected) {
		t.Errorf("Expected %+v, got %+v\n", expected, snapshot)
	}
	t.Logf("Expected %+v, Got %+v\n", expected, snapshot)
}

func TestDeleteSnapshot(t *testing.T) {
	var d = GetSANDriver(t)
	opt := &pb.DeleteVolumeSnapshotOpts{
		Id: "d1916c49-3088-4a40-b6fb-0fda18d074c3",
		VolumeId: "e1bb066c-5ce7-46eb-9336-25508cee9f71",
	}
	// Save current function and restore at the end:
	old := deleteSnapshot
	defer func() { deleteSnapshot = old;}()
	deleteSnapshot = func(snapConfig *storage.SnapshotConfig) error {
		// This will be called, do whatever you want to,
		// return whatever you want to
		md, err := NewSANStorageDriverMock(t)
		if err != nil {
			t.Fatalf("Unable to create mock driver.")
		}
		var snapName = getSnapshotName(opt.GetId())
		var volName = getVolumeName(opt.GetVolumeId())

		snapConfig = d.GetSnapshotConfig(snapName, volName)
		err = md.DeleteSnapshot(snapConfig)
		if err != nil {
			t.Error(err)
		}
		return nil
	}
	if err := d.DeleteSnapshot(opt); err != nil {
		t.Error("Failed to delete snapshot:", err)
	}
	t.Logf("volume (%s) deleted successfully.", opt.GetId())
}
