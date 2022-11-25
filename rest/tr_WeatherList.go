package rest

import (
	"context"

	"fmt"
	"reflect"

	"Weather-server/global"
	"Weather-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func TR_WeatherList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout)
	defer cancel()

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if retcode := _WeatherList_CheckInput(reqBody); retcode != 0 {
		return retcode
	}

	var bdate, btime int64
	var ODAM_PTY, VSRT_SKY, PM25 float64
	var WS string = "?"
	var DG string = "?"

	var list []map[string]interface{}
	for _, lst := range reqBody["list"].([]interface{}) {
		
		// 해당 위경도의 가장 가까운 격자점을 가져온다
		grid, err := common.GetGridInfo(ctx, db, lst.(map[string]interface{})["lng"].(float64), lst.(map[string]interface{})["lat"].(float64))
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 실황데이타를 가져온다
		query := "SELECT BASE_DATE, BASE_TIME, GRID_VAL FROM DFS_ODAM_CURR " +
				 "WHERE GRID_X = '" + fmt.Sprintf("%d", grid.X) + "' and GRID_Y = '" + fmt.Sprintf("%d", grid.Y) + "' and DATA_TYPE = 'PTY'"
		err = db.QueryRowContext(ctx, query).Scan(&bdate, &btime, &ODAM_PTY)
		if err != nil { 
			global.FLog.Println(err)
			return 9901
		}

		if ODAM_PTY > 0 {
			if ODAM_PTY == 1 || ODAM_PTY == 2 || ODAM_PTY == 4 {
				WS = "R"
			} else if ODAM_PTY == 3 {
				WS = "N"
			}
		} else {
			// 초단기 데이타를 가져온다
			query = "SELECT GRID_VAL FROM DFS_VSRT_CURR " +
					"WHERE GRID_X = '" + fmt.Sprintf("%d", grid.X) + "' " +
					"  and GRID_Y = '" + fmt.Sprintf("%d", grid.Y) + "' " +
					"  and DATA_TYPE = 'SKY' " +
					"  and FCST_DATE * 10000 + FCST_TIME >= '" + fmt.Sprintf("%08d%04d", bdate, btime) + "' " +
					"ORDER BY FCST_DATE, FCST_TIME"
			err = db.QueryRowContext(ctx, query).Scan(&VSRT_SKY)
			if err != nil { 
				global.FLog.Println(err)
				return 9901
			}

			if VSRT_SKY == 1 {
				WS = "S"
			} else if VSRT_SKY == 3 {
				WS = "P"
			} else if VSRT_SKY == 4 {
				WS = "C"
			}
		}

		// 미세먼지 데이타를 가져온다
		query = "SELECT PM25, GRADE FROM OBSERVER_CURR_DUST WHERE GRID_X = '" + fmt.Sprintf("%d", grid.X) + "' and GRID_Y = '" + fmt.Sprintf("%d", grid.Y) + "'"
		err = db.QueryRowContext(ctx, query).Scan(&PM25, &DG)
		if err != nil { 
			global.FLog.Println(err)
			return 9901
		}

		data := make(map[string]interface{})
		data["key"] = lst.(map[string]interface{})["key"].(string)
		data["WS"] = WS
		data["DG"] = DG
		data["PM25"] = PM25
		list = append(list, data)
	}
	
	resBody["list"] = list

	return 0
}


func _WeatherList_CheckInput(reqBody map[string]interface{}) (int) {

	if reqBody["list"] == nil { return 9003 }
	if reflect.TypeOf(reqBody["list"]).Kind() != reflect.Slice { return 9005 }

	for _, lst := range reqBody["list"].([]interface{}) {
		if lst.(map[string]interface{})["key"] == nil { return 9003 }
		if lst.(map[string]interface{})["lat"] == nil { return 9003 }
		if lst.(map[string]interface{})["lng"] == nil { return 9003 }

		if reflect.TypeOf(lst.(map[string]interface{})["key"]).Kind() != reflect.String { return 9005 }
		if reflect.TypeOf(lst.(map[string]interface{})["lat"]).Kind() != reflect.Float64 { return 9005 }
		if reflect.TypeOf(lst.(map[string]interface{})["lng"]).Kind() != reflect.Float64 { return 9005 }
	}

	return 0
}
