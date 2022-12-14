package rest

import (
	"bytes"
	"database/sql"
	"time"
	"strconv"
	"strings"
	"net/http"

    "encoding/base64"
	"encoding/json"

	"Weather-server/global"
	"Weather-server/common"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// Receive Request & Send Response
func ProcRestWeather(c *gin.Context, db *sql.DB, rdp *redis.Pool) {

	var err error
	var header global.HeaderParameter

	// Read Header & Body
	if err = c.ShouldBindHeader(&header); err != nil {
		global.FLog.Println("Header JSON Parse Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9001, "lang": "E", "message": common.GetErrorMessage("E", 9001)})
		return
	}
	buf := new(bytes.Buffer); buf.ReadFrom(c.Request.Body)
	hbody := buf.String()

	// Check Valid Header & Data
	if ret := CheckHeader(header, hbody); ret > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": ret, "lang": "E", "message": common.GetErrorMessage("E", ret)})
		return
	}

	// write log
	tr, _ := base64.StdEncoding.DecodeString(hbody)
	global.FLog.Println("Request:", string(tr))

	// request body json to Map
	reqData := make(map[string]interface{})	
	if err = json.Unmarshal(tr, &reqData); err != nil {
		global.FLog.Println("Body JSON Parse Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9003, "lang": "E", "message": common.GetErrorMessage("E", 9003)})
		return
	}

	// check tr struct
	if reqData["trid"] == nil || reqData["key"] == nil || reqData["lang"] == nil || reqData["body"] == nil {
		global.FLog.Println("Body Struct Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9003, "lang": "E", "message": common.GetErrorMessage("E", 9003)})
		return
	}

	var res_code int
	var lang string = reqData["lang"].(string)

	// Change Lang
	if strings.EqualFold(lang, "kr") { lang = "K" }
	if strings.EqualFold(lang, "en") { lang = "E" }

	// Redis Connect
	rds := rdp.Get();
	defer rds.Close();

	// switch request
	resBody := make(map[string]interface{})
	switch reqData["trid"] {
		case "forecast": res_code = TR_Forecast(c, db, rds, lang, reqData, resBody)
		case "weather_list": res_code = TR_WeatherList(c, db, rds, lang, reqData, resBody)
		default:
			global.FLog.Println("???????????? ?????? TR:", reqData["trid"])
			res_code = 9004
	}

	// Error Occurred
	//if ( res_code > 0 ) {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": res_code, "lang": lang, "message": common.GetErrorMessage(lang, res_code)})
	//	return
	//}

	// Fill Response TR
	responseTR := make(map[string]interface{})
	responseTR["code"] = res_code
	if res_code > 0 { responseTR["message"] = common.GetErrorMessage(lang, res_code) }
	responseTR["lang"] = lang
	responseTR["trid"] = reqData["trid"]
	if len(resBody) > 0 { responseTR["body"] = resBody }

	// Print Response
	meJson, _ := json.Marshal(responseTR)
	strJson := string(meJson)
	c.String(http.StatusOK, strJson)

	// write log
	global.FLog.Println("Response:", strJson)
}

func CheckHeader(header global.HeaderParameter, reqdata string) int {

	// Check Access Key
	if header.XAccess != global.Config.Service.APIKey {
		return 9001
	}

	// Check Timeout
	timestamp := time.Now().Unix() * 1000;
	if timestamp - header.XNonce > 5000 {
		global.FLog.Println("Header Time Error", header)
		return 9002
	}

	// Check Signatue
	endata := strconv.FormatInt(header.XNonce, 10) + header.XAccess + reqdata
	sign := common.EncyptData(endata, global.Config.Service.APISecret)
	if sign != header.XSignature {
		global.FLog.Println("Header Sign Error", header, sign)
		return 9001
	}

	return 0
}
