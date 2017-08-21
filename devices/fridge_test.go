package devices

import (
	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/entities"
	"github.com/smartystreets/goconvey/convey"
	"testing"

	"encoding/json"
	"reflect"
)

func TestSetDevConfigWithRedis(t *testing.T) {
	dbWorker := db.RedisClient{DbServer: entities.Server{Host: "0.0.0.0", Port: uint(6379)}}
	dbWorker.Connect()
	defer dbWorker.Close()
	fridge := Fridge{}

	fridge.Config = FridgeConfig{true, true, int64(200), int64(200)}
	b, _ := json.Marshal(fridge.Config)
	devConfig := entities.DevConfig{"00-00-00-11-11-11", b}

	convey.Convey("Should be all ok", t, func() {
		fridge.SetDevConfig("00-00-00-11-11-11:config", &devConfig, &dbWorker)
		isConfig := fridge.GetDevConfig("00-00-00-11-11-11:config", "00-00-00-11-11-11", &dbWorker)
		dbWorker.FlushAll()
		convey.So(reflect.DeepEqual(*isConfig, devConfig), convey.ShouldBeTrue)
	})
}

// Impossible to testing StreamOn and TurnedOn
func TestValidateDevData(t *testing.T) {
	dbWorker := db.RedisClient{DbServer: entities.Server{Host: "0.0.0.0", Port: uint(6379)}}
	dbWorker.Connect()
	defer dbWorker.Close()
	fridge := Fridge{}
	var devConfig entities.DevConfig
	convey.Convey("Invalid MAC. Should be false", t, func() {
		fridge.Config = FridgeConfig{true, true, int64(0), int64(0)}
		b, _ := json.Marshal(fridge.Config)
		devConfig = entities.DevConfig{"Invalid mac", b}
		valid, _ := fridge.ValidateDevData(devConfig)
		convey.So(valid, convey.ShouldBeFalse)
	})
	convey.Convey("Valid DevConfig. Should be false", t, func() {

		fridge.Config = FridgeConfig{true, true, int64(200), int64(200)}
		b, _ := json.Marshal(fridge.Config)
		devConfig = entities.DevConfig{"00-00-00-11-11-11", b}

		valid, _ := fridge.ValidateDevData(devConfig)
		convey.So(valid, convey.ShouldBeTrue)
	})
	convey.Convey("Collect Frequency should be more than 150!", t, func() {

		fridge.Config = FridgeConfig{true, true, int64(100), int64(200)}
		b, _ := json.Marshal(fridge.Config)
		devConfig = entities.DevConfig{"00-00-00-11-11-11", b}
		valid, _ := fridge.ValidateDevData(devConfig)
		convey.So(valid, convey.ShouldBeFalse)
	})
	convey.Convey("Send Frequency should be more than 150!", t, func() {
		fridge.Config = FridgeConfig{true, true, int64(200), int64(100)}
		b, _ := json.Marshal(fridge.Config)
		devConfig = entities.DevConfig{"00-00-00-11-11-11", b}
		valid, _ := fridge.ValidateDevData(devConfig)
		convey.So(valid, convey.ShouldBeFalse)
	})
	dbWorker.FlushAll()
}

func TestSmallSetDevData(t *testing.T) {
	dbWorker := db.RedisClient{DbServer: entities.Server{Host: "0.0.0.0", Port: uint(6379)}}
	dbWorker.Connect()
	defer dbWorker.Close()

	tempCam := make(map[int64]float32)
	tempCam[1] = 1.0
	tempCam[2] = 2.0

	key := "test"

	convey.Convey("", t, func() {
		expected := []string{"1:1", "2:2"}
		setCameraData(tempCam, key, &dbWorker)
		actual, _ := dbWorker.Client.ZRangeByScore(key, "-inf", "inf")
		convey.So(actual, convey.ShouldResemble, expected)
	})
	dbWorker.FlushAll()
}

func TestGetDefaultConfig(t *testing.T) {
	dbWorker := db.RedisClient{DbServer: entities.Server{Host: "0.0.0.0", Port: uint(6379)}}
	dbWorker.Connect()
	defer dbWorker.Close()

	fridge := Fridge{}

	config := FridgeConfig{
		TurnedOn:    true,
		StreamOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000,
	}

	data, _ := json.Marshal(config)
	expected := &entities.DevConfig{
		MAC:  fridge.Meta.MAC,
		Data: data,
	}

	convey.Convey("", t, func() {
		actual := fridge.GetDefaultConfig()
		convey.So(actual, convey.ShouldResemble, expected)
	})
	dbWorker.FlushAll()
}

func TestSetDevData(t *testing.T) {
	dbWorker := db.RedisClient{DbServer: entities.Server{Host: "0.0.0.0", Port: uint(6379)}}
	dbWorker.Connect()
	defer dbWorker.Close()

	tempCam := make(map[int64]float32)
	tempCam[1] = 1.0
	tempCam[2] = 2.0

	dataMap := make(map[string][]string)
	dataMap["TempCam1"] = []string{"1:1", "2:2"}
	dataMap["TempCam2"] = []string{"1:1", "2:2"}

	meta := entities.DevMeta{MAC: "00-00-00-11-11-11", Name: "name", Type: "fridge"}
	data := FridgeData{tempCam, tempCam}
	fridge := Fridge{}

	b, _ := json.Marshal(data)

	req := entities.Request{Meta: meta, Data: b}

	convey.Convey("Must bu all ok", t, func() {

		fridge.SetDevData(&req, &dbWorker)
		dbWorker.Connect()
		devParamsKey := "device:" + meta.Type + ":" + meta.Name + ":" + meta.MAC + ":params"

		actual := fridge.GetDevData(devParamsKey, meta, &dbWorker)
		expected := entities.DevData{Meta: meta, Data: dataMap}
		dbWorker.FlushAll()
		convey.So(actual, convey.ShouldResemble, expected)
	})
}
