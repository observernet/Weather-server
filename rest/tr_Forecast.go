package rest

import (
	"time"
	"context"

	"fmt"
	"reflect"
	"strconv"

	"Weather-server/global"
	"Weather-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

type _Forecast_DustData struct {
	Date		int64
	Time		int64
	DG			string
}

func TR_Forecast(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout)
	defer cancel()

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if retcode := _Forecast_CheckInput(reqBody); retcode != 0 {
		return retcode
	}

	// 해당 위경도의 주소를 가져온다
	addr, err := common.GetAddressFromSAPI(reqBody["lng"].(float64), reqBody["lat"].(float64))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(addr)

	// 해당 위경도의 가장 가까운 격자점을 가져온다
	grid, err := common.GetGridInfo(ctx, db, reqBody["lng"].(float64), reqBody["lat"].(float64))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(grid)

	// 현재 예보값을 가져온다
	curr, err := _Forecast_GetCurr(ctx, db, grid.X, grid.Y)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(curr)

	// 24시간 예보값을 가져온다
	fcst_time, err := _Forecast_GetFcstTime(ctx, db, grid.X, grid.Y, curr["date"].(int64), curr["time"].(int64))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(fcst_time)

	// +1, +2 시간 예보값을 가져온다
	fcst_time_2, err := _Forecast_GetFcstTime2(ctx, db, grid.X, grid.Y, curr["date"].(int64))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(fcst_time_2)

	// 일 예보값을 가져온다
	fcst_day, err := _Forecast_GetFcstDay(ctx, db, grid.X, grid.Y, curr["date"].(int64))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	//global.FLog.Println(fcst_day)

	// 응답값을 세팅한다
	resBody["addr"] = addr
	resBody["curr"] = curr
	resBody["fcst_time"] = fcst_time
	resBody["fcst_time_2"] = fcst_time_2
	resBody["fcst_day"] = fcst_day
	
	return 0
}

func _Forecast_GetCurr(ctx context.Context, db *sql.DB, x int64, y int64) (map[string]interface{}, error) {

	var base_date, base_time int64;
	var data_type string
	var value float64

	var ODAM_PTY, ODAM_T1H, ODAM_WSD float64
	var VSRT_SKY, VSRT_T1H, VSRT_WSD float64
	var SHRT_TMX, SHRT_TMN float64
	var DUST_PM25 float64

	// 값 초기화
	ODAM_PTY = -1; ODAM_WSD = -1; VSRT_WSD = -1; VSRT_SKY = -1
	ODAM_T1H = -50; VSRT_T1H = -50; SHRT_TMX = -50; SHRT_TMN = -50
	DUST_PM25 = -1

	// 실황데이타를 가져온다
	query := "SELECT DATA_TYPE, GRID_VAL, BASE_DATE, BASE_TIME FROM DFS_ODAM_CURR " +
			 "WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			 "  and GRID_Y = '" + fmt.Sprintf("%d", y) + "' " +
			 "  and DATA_TYPE in ('PTY', 'T1H', 'WSD') "
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return nil, err }

	for rows.Next() {	
		err = rows.Scan(&data_type, &value, &base_date, &base_time)
		if err != nil { rows.Close(); return nil, err }

		switch data_type {
			case "PTY": ODAM_PTY = value
			case "T1H": ODAM_T1H = value
			case "WSD": ODAM_WSD = value
		}
	}
	rows.Close()

	// 초단기 데이타를 가져온다
	query = "SELECT GRID_VAL FROM DFS_VSRT_CURR " +
			"WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			"  and GRID_Y = '" + fmt.Sprintf("%d", y) + "' " +
			"  and DATA_TYPE = :1 " +
			"  and FCST_DATE * 10000 + FCST_TIME >= '" + fmt.Sprintf("%08d%04d", base_date, base_time) + "' " +
			"ORDER BY FCST_DATE, FCST_TIME"
	stmt, err := db.PrepareContext(ctx, query)

	err = stmt.QueryRow("SKY").Scan(&VSRT_SKY)
	if err != nil && err != sql.ErrNoRows { stmt.Close(); return nil, err }

	err = stmt.QueryRow("T1H").Scan(&VSRT_T1H)
	if err != nil && err != sql.ErrNoRows { stmt.Close(); return nil, err }

	err = stmt.QueryRow("WSD").Scan(&VSRT_WSD)
	if err != nil && err != sql.ErrNoRows { stmt.Close(); return nil, err }
	stmt.Close()

	// 단기 데이타를 가져온다
	query = "SELECT GRID_VAL FROM DFS_SHRT_CURR " +
			"WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			"  and GRID_Y = '" + fmt.Sprintf("%d", y) + "' " +
			"  and DATA_TYPE = :1 " +
			"  and FCST_DATE = '" + fmt.Sprintf("%08d", base_date) + "' " +
			"ORDER BY FCST_DATE"
	stmt, err = db.PrepareContext(ctx, query)

	err = stmt.QueryRow("TMX").Scan(&SHRT_TMX)
	if err != nil { stmt.Close(); return nil, err }

	err = stmt.QueryRow("TMN").Scan(&SHRT_TMN)
	if err != nil { stmt.Close(); return nil, err }
	stmt.Close()

	// 미세먼지 데이타를 가져온다
	query = "SELECT PM25 FROM OBSERVER_CURR_DUST " +
			"WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			"  and GRID_Y = '" + fmt.Sprintf("%d", y) + "' "
	err = db.QueryRowContext(ctx, query).Scan(&DUST_PM25)
	if err != nil { return nil, err }


	// 하늘상태를 계산한다
	var WS string = "?"
	if ODAM_PTY > 0 {
		if ODAM_PTY == 1 || ODAM_PTY == 2 || ODAM_PTY == 4 {
			WS = "R"
		} else if ODAM_PTY == 3 {
			WS = "N"
		}
	} else {
		if VSRT_SKY > 0 {
			if VSRT_SKY == 1 {
				WS = "S"
			} else if VSRT_SKY == 3 {
				WS = "P"
			} else if VSRT_SKY == 4 {
				WS = "C"
			}
		} else {
			// 과거 예보자료 그대로 사용
		}
	}

	// 기온값을 계산한다
	var TMP float64 = -50
	if ODAM_T1H > -50 {
		TMP = ODAM_T1H
	} else {
		if VSRT_T1H > -50 {
			TMP = VSRT_T1H
		} else {
			// 과거 예보자료 그대로 사용
		}
	}

	// 풍속을 계산한다
	var WSD float64 = -50
	if ODAM_WSD > -1 {
		WSD = ODAM_WSD
	} else {
		WSD = VSRT_WSD
	}

	list := make(map[string]interface{})
	list["date"] = base_date
	list["time"] = base_time
	list["WS"] = WS
	list["TMP"] = TMP
	list["TMX"] = SHRT_TMX
	list["TMN"] = SHRT_TMN
	list["WSD"] = WSD
	list["DG"] = common.GetDustGradeFromPM25(DUST_PM25)

	return list, nil
}

func _Forecast_GetFcstTime(ctx context.Context, db *sql.DB, x int64, y int64, bdate int64, btime int64) ([]map[string]interface{}, error) {

	var fcst_date, fcst_time int64
	var WS, DUST_GRADE string
	var TMP float64

	// 예보 데이타를 가져온다
	query := "SELECT A.FCST_DATE, A.FCST_TIME, A.WS, A.TMP, NVL(B.GRADE, '?') " +
			 "FROM OBSERVER_FCST_TIME A, OBSERVER_FCST_DUST B " +
			 "WHERE A.GRID_X = B.GRID_X  " +
			 "  and A.GRID_Y = B.GRID_Y  " +
			 "  and A.FCST_DATE = B.FCST_DATE (+) " +
			 "  and A.FCST_TIME = B.FCST_TIME (+) " +
			 "  and A.GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			 "  and A.GRID_Y = '" + fmt.Sprintf("%d", y) + "'  " +
			 "  and A.FCST_DATE * 10000 + A.FCST_TIME >= '" + fmt.Sprintf("%08d%04d", bdate, btime) + "' " +
			 "ORDER BY A.FCST_DATE, A.FCST_TIME "
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return nil, err }

	var count int = 0
	var list []map[string]interface{}
	for rows.Next() {	
		err = rows.Scan(&fcst_date, &fcst_time, &WS, &TMP, &DUST_GRADE)
		if err != nil { rows.Close(); return nil, err }

		data := make(map[string]interface{})
		data["date"] = fcst_date
		data["time"] = fcst_time
		data["WS"] = WS
		data["TMP"] = TMP
		data["DG"] = DUST_GRADE
		list = append(list, data)

		count = count + 1
		if count == 24 { break }
	}
	rows.Close()

	return list, nil
}

func _Forecast_GetFcstTime2(ctx context.Context, db *sql.DB, x int64, y int64, bdate int64) (map[string]interface{}, error) {

	var fcst_date, fcst_time int64
	var WS, DUST_GRADE string
	var TMP float64

	// T0, T1, T1 날짜를 가져온다
	tm, _ := time.Parse("20060102", fmt.Sprintf("%d", bdate))
	tm = tm.Add(time.Hour * 24); T1_date, _ := strconv.ParseInt(tm.Format("20060102"), 10, 64)
	tm = tm.Add(time.Hour * 24); T2_date, _ := strconv.ParseInt(tm.Format("20060102"), 10, 64)

	// 미세먼지 예보 데이타를 가져온다
	query := "SELECT FCST_DATE, FCST_TIME, GRADE " +
			 "FROM OBSERVER_FCST_DUST " +
			 "WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			 "  and GRID_Y = '" + fmt.Sprintf("%d", y) + "'  " +
			 "  and FCST_DATE in (" + fmt.Sprintf("%d", T1_date) + ", " + fmt.Sprintf("%d", T2_date) + ") " +
			 "  and FCST_TIME in (0, 300, 600, 900, 1200, 1500, 1800, 2100) " +
			 "ORDER BY FCST_DATE, FCST_TIME "
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return nil, err }

	dust := []_Forecast_DustData{}
	for rows.Next() {	
		err = rows.Scan(&fcst_date, &fcst_time, &DUST_GRADE)
		if err != nil { rows.Close(); return nil, err }

		data := _Forecast_DustData{fcst_date, fcst_time, DUST_GRADE}
		dust = append(dust, data)
	}
	rows.Close()

	// 날씨 예보 데이타를 가져온다
	query = "SELECT FCST_DATE, FCST_TIME, WS, TMP " +
			"FROM OBSERVER_FCST_TIME " +
			"WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			"  and GRID_Y = '" + fmt.Sprintf("%d", y) + "'  " +
			"  and FCST_DATE in (" + fmt.Sprintf("%d", T1_date) + ", " + fmt.Sprintf("%d", T2_date) + ") " +
			"  and FCST_TIME in (0, 300, 600, 900, 1200, 1500, 1800, 2100) " +
			"ORDER BY FCST_DATE, FCST_TIME "
	rows, err = db.QueryContext(ctx, query)
	if err != nil { return nil, err }

	var T1, T2 []map[string]interface{}
	for rows.Next() {	
		err = rows.Scan(&fcst_date, &fcst_time, &WS, &TMP)
		if err != nil { rows.Close(); return nil, err }

		data := make(map[string]interface{})
		data["date"] = fcst_date
		data["time"] = fcst_time
		data["WS"] = WS
		data["TMP"] = TMP

		// 미세먼지 데이타를 가져온다
		DUST_GRADE = "N"
		for _, d := range dust {
			if d.Date == fcst_date && d.Time == fcst_time {
				DUST_GRADE = d.DG
				break
			}
		}
		data["DG"] = DUST_GRADE

		if fcst_date == T1_date {
			T1 = append(T1, data)
		} else if fcst_date == T2_date {
			T2 = append(T2, data)
		} else {
			break
		}
	}
	rows.Close()

	list := make(map[string]interface{})
	list["T1"] = T1
	list["T2"] = T2

	return list, nil
}

func _Forecast_GetFcstDay(ctx context.Context, db *sql.DB, x int64, y int64, bdate int64) ([]map[string]interface{}, error) {

	var fcst_date int64
	var WS_AM, WS_PM string
	var TMN, TMX float64

	// 예보 데이타를 가져온다
	query := "SELECT FCST_DATE, WS_AM, WS_PM, TMN, TMX " +
			 "FROM OBSERVER_FCST_DAY " +
			 "WHERE GRID_X = '" + fmt.Sprintf("%d", x) + "' " +
			 "  and GRID_Y = '" + fmt.Sprintf("%d", y) + "'  " +
			 "  and FCST_DATE > '" + fmt.Sprintf("%08d", bdate) + "' " +
			 "ORDER BY FCST_DATE "
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return nil, err }

	var list []map[string]interface{}
	for rows.Next() {	
		err = rows.Scan(&fcst_date, &WS_AM, &WS_PM, &TMN, &TMX)
		if err != nil { rows.Close(); return nil, err }

		t, _ := time.Parse("20060102", fmt.Sprintf("%d", fcst_date))

		data := make(map[string]interface{})
		data["date"] = fcst_date
		data["week"] = int(t.Weekday())
		data["WS_AM"] = WS_AM
		data["WS_PM"] = WS_PM
		data["TMN"] = TMN
		data["TMX"] = TMX
		list = append(list, data)
	}
	rows.Close()

	return list, nil
}

func _Forecast_CheckInput(reqBody map[string]interface{}) (int) {

	if reqBody["lat"] == nil { return 9003 }
	if reqBody["lng"] == nil { return 9003 }

	if reflect.TypeOf(reqBody["lat"]).Kind() != reflect.Float64 { return 9005 }
	if reflect.TypeOf(reqBody["lng"]).Kind() != reflect.Float64 { return 9005 }

	return 0
}
