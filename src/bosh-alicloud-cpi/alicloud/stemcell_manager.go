/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package alicloud

import (
	"encoding/json"
	"os"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

const (
	AlicloudImageServiceTag          = "AlicloudImageService"
	AlicloudDefaultImageName         = "bosh-stemcell"
	AlicloudDefaultImageArchitecture = "x86_64"
	AlicloudDefaultImageOSType       = "linux"
	WaitForImageReadyTimeout         = 1800
	DefaultWaitForImageReadyTimeout  = 1200
	DefaultWaitForInterval           = 10
)

type StemcellManager interface {
	FindStemcellById(id string) (*ecs.Image, error)
	DeleteStemcell(id string) error
	ImportImage(args *ecs.ImportImageRequest) (string, error)
	OpenLocalFile(path string) (*os.File, error)
	WaitForImageReady(id string) error
}

type StemcellManagerImpl struct {
	config Config
	logger boshlog.Logger
	region string
}

func NewStemcellManager(config Config, logger boshlog.Logger) StemcellManager {
	return StemcellManagerImpl{
		config: config,
		logger: logger,
		region: config.OpenApi.RegionId,
	}
}

func (a StemcellManagerImpl) log(action string, err error, args interface{}, result string) {
	s, _ := json.Marshal(args)
	if err != nil {
		a.logger.Error("StemcellManager", "%s failed args=%s err=%s", action, s, err)
	} else {
		a.logger.Info("StemcellManager", "%s done! args=%s result=%s", action, s, result)
	}
}

func (a StemcellManagerImpl) FindStemcellById(id string) (*ecs.Image, error) {
	client, err := a.config.NewEcsClient()
	if err != nil {
		return nil, err
	}
	a.logger.Debug(AlicloudImageServiceTag, "Finding Alicloud Image '%s'", id)

	args := ecs.CreateDescribeImagesRequest()
	args.RegionId = a.config.OpenApi.GetRegion()
	args.ImageId = id

	images, err := client.DescribeImages(args)
	a.logger.Debug(AlicloudImageServiceTag, "Find Alicloud Images '%#v'", images)

	if err != nil {
		return nil, err
	}

	if images == nil || len(images.Images.Image) <= 0 {
		return nil, nil
	}

	return &images.Images.Image[0], nil
}

func (a StemcellManagerImpl) DeleteStemcell(id string) error {
	image, err := a.FindStemcellById(id)
	if err != nil {
		return err
	}
	if image == nil {
		return bosherr.WrapErrorf(err, "Alicloud Image '%s' does not exists", id)
	}

	a.logger.Debug(AlicloudImageServiceTag, "Deleting Alicloud Image '%s'", id)
	client, err := a.config.NewEcsClient()
	if err != nil {
		return err
	}
	args := ecs.CreateDeleteImageRequest()
	args.RegionId = a.config.OpenApi.GetRegion()
	args.ImageId = id
	_, err = client.DeleteImage(args)

	if err != nil {
		return bosherr.WrapErrorf(err, "Failed to delete Alicloud Image '%s'", id)
	}

	return nil
}

func (a StemcellManagerImpl) ImportImage(args *ecs.ImportImageRequest) (string, error) {
	client, err := a.config.NewEcsClient()
	if err != nil {
		return "", err
	}

	args.RegionId = a.config.OpenApi.GetRegion()

	resp, err := client.ImportImage(args)
	if err != nil || resp == nil {
		return "", bosherr.WrapErrorf(err, "Failed to import Alicloud Image in '%s'.", args.RegionId)
	}
	a.log("Importing Image", err, args, resp.ImageId)
	return resp.ImageId, err
}

func (a StemcellManagerImpl) OpenLocalFile(path string) (*os.File, error) {
	return os.Open(path)
}

// import image from oss may take >=15min
// we set timeout value to 30min, if needed turn it up
func (a StemcellManagerImpl) WaitForImageReady(id string) error {
	region := a.config.OpenApi.GetRegion()
	return a.WaitForImage(region, id, WaitForImageReadyTimeout)
}

//Wait Image ready
func (a StemcellManagerImpl) WaitForImage(regionId, imageId string, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultWaitForImageReadyTimeout
	}

	for {
		image, err := a.FindStemcellById(imageId)
		a.logger.Debug(AlicloudImageServiceTag, "Find Alicloud Images '%#v'", imageId)

		if err != nil {
			return err
		}

		if image == nil {
			return GetNotFoundErrorFromString(GetNotFoundMessage("ECS image", imageId))
		}

		if image.Status == "Available" {
			break
		}

		timeout = timeout - DefaultWaitForInterval
		if timeout < 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("ECS image", "Available"))
		}
		time.Sleep(DefaultWaitForInterval * time.Second)
	}
	return nil
}
