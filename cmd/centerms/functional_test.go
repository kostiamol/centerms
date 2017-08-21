package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"

	"menteslibres.net/gosexy/redis"

	"fmt"
	"github.com/giperboloid/centerms/db"
	"github.com/giperboloid/centerms/entities"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
	"time"
)

var timeForSleep time.Duration = 1000 * time.Millisecond

func deleteAllInBase(dbClient db.Client) {
	defer treatmentPanic("Recovered in TestCheckJSONToServer")
	err := dbClient.FlushAll()
	if err != nil {
		errors.Wrap(err, "FlushAll()")
	}
}

func treatmentPanic(message string) {
	if r := recover(); r != nil {
		fmt.Println(message, r)
	}
}

func TestSendJSONToServer(t *testing.T) {
	defer treatmentPanic("Recovered in TestSendJSONToServer")

	var coonNotNil bool = false
	buffer := make([]byte, 1024)

	conn, err := net.Dial("tcp", localhost+":"+fmt.Sprint(tcpDataPort))
	if conn != nil {
		coonNotNil = true
		defer conn.Close()
	} else {
		errors.Wrap(err, "conn is nil")
	}
	//Create redis client------------------------------------------------------------
	defer treatmentPanic("Recovered in TestCheckJSONToServer")
	var dbCli db.Client = &db.RedisClient{DbServer: dbServer}
	dbCli.Connect()
	defer dbCli.Close()
	//--------------------------------------------------------------------------------

	res := "\"status\":200,\"descr\":\"Data has been delivered successfully\""
	req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge",
		Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
	message, _ := json.Marshal(req)

	if coonNotNil {
		conn.Write(message)
		time.Sleep(timeForSleep)

		for i := 0; i == 0; {
			i, _ = conn.Read(buffer)
		}
	}
	response := bytes.NewBuffer(buffer).String()

	if !strings.Contains(response, res) {
		t.Error("Bad JSON", response, res)
	}
	deleteAllInBase(dbCli)
}

func TestCheckJSONToServer(t *testing.T) {
	defer treatmentPanic("Recovered in TestCheckJSONToServer")
	var coonNotNil bool = false

	conn, err := net.Dial("tcp", localhost+":"+fmt.Sprint(tcpDataPort))
	if conn != nil {
		coonNotNil = true
		defer conn.Close()
	} else {
		errors.Wrap(err, "conn is nil")
	}

	res := "\"status\":200,\"descr\":\"Data has been delivered successfully\""

	//Create redis client------------------------------------------------------------
	defer treatmentPanic("Recovered in TestCheckJSONToServer")
	var dbCli db.Client = &db.RedisClient{DbServer: dbServer}
	dbCli.Connect()
	defer dbCli.Close()
	//--------------------------------------------------------------------------------

	convey.Convey("Send Correct JSON to server", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		//Check on error
		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}

		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! Uncorrect JSON was sent to server", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("JSON was sent to server. Action of fridge should be update", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server with uncorrect action value", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "nil", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("JSON was sent to server. Action of washer should be update", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "washer", Name: "bosh0e31", MAC: "00-15-E9-2B-99-3B"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server with uncorrect type value", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "nil", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server without MAC value", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: ""}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server without type value", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server without name value", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge", Name: "", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})

	convey.Convey("Warning! JSON was sent to server without time value ", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		req := entities.Request{Action: "update", Time: 0, Meta: entities.DevMeta{Type: "fridge", Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
		message, _ := json.Marshal(req)
		buffer := make([]byte, 1024)

		if coonNotNil {
			conn.Write(message)
			time.Sleep(timeForSleep)
			for i := 0; i == 0; {
				i, _ = conn.Read(buffer)
			}
		}
		response := bytes.NewBuffer(buffer).String()

		convey.So(response, convey.ShouldContainSubstring, res)
		deleteAllInBase(dbCli)
	})
}
func TestRedisConnection(t *testing.T) {
	client := redis.New()
	convey.Convey("Check redis client connection"+dbServer.Host+":"+string(dbServer.Port)+". Should be without error ", t, func() {
		err := client.Connect(dbServer.Host, dbServer.Port)
		defer client.Close()
		convey.So(err, convey.ShouldBeNil)
	})
}
func TestHTTPConnection(t *testing.T) {
	var httpClient = &http.Client{}

	//Create redis client------------------------------------------------------------
	defer treatmentPanic("Recovered in TestCheckJSONToServer")
	var dbCli db.Client = &db.RedisClient{DbServer: dbServer}
	dbCli.Connect()
	defer dbCli.Close()
	//--------------------------------------------------------------------------------

	convey.Convey("Check http://"+localhost+":"+fmt.Sprint(httpPort)+"/devices/{id}/data. Should be without error ", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/id/data?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C")
		convey.So(res, convey.ShouldNotBeNil)
		deleteAllInBase(dbCli)
	})
	convey.Convey("Check http://"+localhost+":"+fmt.Sprint(httpPort)+"/devices. Should be without error ", t, func() {
		defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		convey.So(res, convey.ShouldNotBeNil)
	})
}

func TestWorkingServerAfterSendingJSON(t *testing.T) {

	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	conn, _ := net.Dial("tcp", localhost+":"+fmt.Sprint(tcpDataPort))
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	defer conn.Close()
	var httpClient = &http.Client{}

	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	connForDAta, _ := net.Dial("tcp", localhost+":"+fmt.Sprint(tcpDataPort))
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	defer conn.Close()

	//Create redis client------------------------------------------------------------
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	var dbCli db.Client = &db.RedisClient{DbServer: dbServer}
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	dbCli.Connect()
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	defer dbCli.Close()
	//--------------------------------------------------------------------------------

	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Should be return all ok ", t, func() {
		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"10\":10.5}}}"
		reqConfig := "{\"action\":\"config\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"10\":10.5}}}"
		mustHave := "[{\"site\":\"\",\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\",\"mac\":\"00-15-E9-2B-99-3C\"," +
			"\"ip\":\"\"},\"data\":{\"TempCam1\":[\"10:10.5\"],\"TempCam2\":[\"10:10.5\"]}}]"
		conn.Write([]byte(reqConfig))
		connForDAta.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)
		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send JSON where action = wrongValue. Should not be return data about our fridge", t, func() {
		reqMessage := "{\"action\":\"wrongValue\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName2\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"TempCam1\":[\"10:10.5\"]," +
			"\"TempCam2\":[\"1500:15.5\"]}}"

		mustNotHave := "testName2"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldNotContainSubstring, mustNotHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send JSON where type = wrongValue. Should not to return data about our fridge", t, func() {
		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"wrongValue\",\"name\":\"testName3\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustNotHave := "testName3"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldNotContainSubstring, mustNotHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send JSON without name. Should not to return data about our fridge", t, func() {
		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustNotHave := "TestMACFridge3"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldNotContainSubstring, mustNotHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send JSON without mac. Should not to return data about our fridge", t, func() {

		reqMessage := "{\"action\":\"config\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"fridge4\"" +
			",\"mac\":\"\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustNotHave := "fridge4"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldNotContainSubstring, mustNotHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send JSON with wrong data. Should not to return data about our fridge", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"fridge5\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"qwe\":qwe},\"tempCam2\":{\"" +
			"qwe\":qwe}}}"

		mustNotHave := "fridge5"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices")
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldNotBeNil)
		convey.So(bodyString, convey.ShouldNotContainSubstring, mustNotHave)
		deleteAllInBase(dbCli)
	})
	//	// my part
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Initialize turned on as false ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"turnedOn\":false"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C")

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)
		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Initialize CollectFreq as 0 ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"collectFreq\":0"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C")

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)
		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Initialize SendFreq as 0 ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"sendFreq\":0"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C")

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)
		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Initialize StreamOn as false ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"streamOn\":false"
		conn.Write([]byte(reqMessage))
		time.Sleep(timeForSleep)
		res, _ := httpClient.Get("http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C")

		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)
		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)

	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Patch device data: turned on as true ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"turnedOn\":false"
		conn.Write([]byte(reqMessage))
		url := "http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C"
		r, _ := http.NewRequest("PATCH", url, bytes.NewBuffer([]byte("{\"turnedOn\":true}")))
		httpClient.Do(r)
		res, _ := httpClient.Get(url)
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
	defer treatmentPanic("Recovered in TestWorkingServerAfterSendingJSON")
	convey.Convey("Send correct JSON. Patch device data: stream on as true ", t, func() {

		reqMessage := "{\"action\":\"update\",\"time\":20,\"meta\":{\"type\":\"fridge\",\"name\":\"testName1\"" +
			",\"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":{\"tempCam1\":{\"10\":10.5},\"tempCam2\":{\"" +
			"1500\":15.5}}}"

		mustHave := "\"streamOn\":false"
		conn.Write([]byte(reqMessage))
		url := "http://" + localhost + ":" + fmt.Sprint(httpPort) + "/devices/00-15-E9-2B-99-3C/config?type=fridge&name=testName1&mac=00-15-E9-2B-99-3C"
		r, _ := http.NewRequest("PATCH", url, bytes.NewBuffer([]byte("{\"streamOn\":true}")))
		time.Sleep(timeForSleep)
		httpClient.Do(r)

		res, _ := httpClient.Get(url)
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		bodyString := string(bodyBytes)

		convey.So(bodyString, convey.ShouldContainSubstring, mustHave)
		deleteAllInBase(dbCli)
	})
}

func TestWSConnection(t *testing.T) {
	defer treatmentPanic("Recovered in TestWSConnection")

	req := entities.Request{Action: "update", Time: 1496741392463499334, Meta: entities.DevMeta{Type: "fridge",
		Name: "hladik0e31", MAC: "00-15-E9-2B-99-3C"}}
	mustBe := "{\"action\":\"update\",\"time\":1496741392463499334,\"meta\":{\"type\":\"fridge\",\"name\":\"hladik0e31\",\"" +
		"mac\":\"00-15-E9-2B-99-3C\",\"ip\":\"\"},\"data\":null}"

	//Create redis client------------------------------------------------------------
	defer treatmentPanic("Recovered in TestWSConnection")
	var dbCli db.Client = &db.RedisClient{DbServer: dbServer}
	dbCli.Connect()
	defer dbCli.Close()
	//--------------------------------------------------------------------------------

	convey.Convey("Checking how to work ws connection. Should be true", t, func() {
		//Create Web Socket connection from the client side--------------------------------
		url := "ws://" + localhost + ":" + fmt.Sprint(wsPort) + "/devices/00-15-E9-2B-99-3C"
		var dialer *websocket.Dialer
		conn, _, err := dialer.Dial(url, nil)
		if err != nil {
			errors.Wrap(err, "conn is nil")
		}
		//---------------------------------------------------------------------------------

		defer treatmentPanic("Recovered in TestWSConnection")
		db.PublishWS(req, "devWS", dbCli)

		_, message, _ := conn.ReadMessage()
		convey.So(bytes.NewBuffer(message).String(), convey.ShouldEqual, mustBe)
	})
}
