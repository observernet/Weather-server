package common

import (
	"time"
	"context"
	"fmt"
	"errors"
	"strings"
    "math"
	"math/rand"
	"strconv"

	"io/ioutil"
    "net/http"

	//"encoding/json"

	"Weather-server/global"

	"database/sql"
)

func GetCodeKey(length int) string {

	var idx int
	var code string = ""
	var source string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i := 0; i < length ; i++ {
		rand.Seed(time.Now().UnixNano())
		idx = rand.Intn(len(source))
		code = code + string(source[idx])
	}

	return code
}

func GetInt64FromString(val string) int64 {

	ret, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}

	return ret
}

func GetFloat64FromString(val string) float64 {

	ret, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}

	return ret
}

func GetIntDate() (int64) {
	//loc, _ := time.LoadLocation(global.Config.Service.Timezone)
	//kst := time.Now().In(loc)
	curtime := time.Now().Format("20060102")
	return GetInt64FromString(curtime)
}

func GetIntTime() (int64) {
	//loc, _ := time.LoadLocation(global.Config.Service.Timezone)
	//kst := time.Now().In(loc)
	curtime := time.Now().Format("150405")
	return GetInt64FromString(curtime)
}

func GetRowsResult(rows *sql.Rows, limit int) ([]map[string]interface{}, error) {

	if rows == nil {
		return nil, errors.New("Rows is null")
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	data := make([]interface{}, len(cols))
	dataPtr := make([]interface{}, len(cols))
	for i, _ := range data {
		dataPtr[i] = &data[i]
	}

	var count int
	var results []map[string]interface{}
	for rows.Next() {	
		err = rows.Scan(dataPtr...)
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		for i, item := range dataPtr {
			val := item.(*interface{})
			result[cols[i]] = *val
		}
		results = append(results, result)

		count = count + 1
		if limit > 0 && count >= limit {
			break
		}
	}

	return results, nil
}

func GetDustGradeFromPM25(pm25 float64) (string) {

	if pm25 < 15 { return "1"; }
	if pm25 < 35 { return "2"; }
	if pm25 < 75 { return "3"; }
	
	return "4";
}

func GetGridInfo(ctx context.Context, db *sql.DB, lng float64, lat float64) (global.GridInfo, error) {

	var grid global.GridInfo
	
	query := "SELECT LONGITUDE, LATITUDE, KMA_X, KMA_Y, KMA_LAND, KMA_TEMP, " +
			 "       DISTANCE_WGS84(LATITUDE, LONGITUDE, " + fmt.Sprintf("%f", lat) + ", " + fmt.Sprintf("%f", lng) + ") D " +
			 "FROM WRT_GRID_INFO " +
			 "ORDER BY D "
	/*rows, err := db.Query(query)
	if err != nil { return grid, err }

	for rows.Next() {	
		err = rows.Scan(&grid.Lng, &grid.Lat, &grid.X, &grid.Y, &grid.Land, &grid.Temp, &distance)
		if err != nil { return grid, err }
		break
	}*/
	err := db.QueryRowContext(ctx, query).Scan(&grid.Lng, &grid.Lat, &grid.X, &grid.Y, &grid.Land, &grid.Temp, &grid.Distance)
	if err != nil { return grid, err }

	return grid, nil
}

func GetAddressFromSAPI(lng float64, lat float64) ([]string, error) {

	// 요청 URL을 만든다
	reqUri := "http://210.180.118.117:7885/get_address.php?key=f923hfo2n128y&lng=" + fmt.Sprintf("%f", lng) + "&lat=" + fmt.Sprintf("%f", lat)

	// 요청 객체를 생성한다
	req, err := http.NewRequest("GET", reqUri, nil)
    if err != nil {
        return nil, err
    }

	// Client객체에서 Request 실행
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

	// 결과 출력
    bytes, _ := ioutil.ReadAll(resp.Body)
	address := strings.Split(string(bytes), ";")

	return address, nil
}

// mode 0: 위경도 => 격자, mode 1: 격자 => 위경도
func ConvertXYLatLng(mode int, var1 float64, var2 float64) (float64, float64, error) {

	NX := (float64)(149)
	NY := (float64)(253)

	if mode == 0 {

		x, y := lamcproj(mode, var1, var2)
		x = math.Floor(x + 1.5)
		y = math.Floor(y + 1.5)
		return x, y, nil

	} else if mode == 1 {

		if var1 < 1 || var1 > NX || var2 < 1 || var2 > NY {
			return 0, 0, errors.New(fmt.Sprintf("X-grid range [1,%f] / Y-grid range [1,%f]", NX, NY))
		}

		lng, lat := lamcproj(mode, var1 - 1, var2 - 1)
		return lng, lat, nil
	}
	
	return 0, 0, errors.New("Not allow mode value")
}

// mode 0 => var1: lng, var2: lat, ret1: x, ret2: y
// mode 1 => var1: x, var2: y, ret1: lng, ret2: lat
func lamcproj(mode int, var1 float64, var2 float64) (float64, float64) {

	map_Re    := 6371.00877      // 지도반경
	map_grid  := 5.0             // 격자간격 (km)
    map_slat1 := 30.0            // 표준위도 1
    map_slat2 := 60.0            // 표준위도 2
	map_olng  := 126.0           // 기준점 경도
	map_olat  := 38.0            // 기준점 위도
	map_xo    := 210/map_grid    // 기준점 X좌표
	map_yo    := 675/map_grid    // 기준점 Y좌표

	var PI, DEGRAD, RADDEG float64
	var re, olng, olat, sn, sf, ro float64
	var slat1, slat2, alng, alat, xn, yn, ra, theta float64
	var ret1, ret2 float64

	PI = math.Asin(1.0) * 2.0
	DEGRAD = PI / 180.0
	RADDEG = 180.0 / PI

	re = map_Re / map_grid
	slat1 = map_slat1 * DEGRAD
	slat2 = map_slat2 * DEGRAD
	olng = map_olng * DEGRAD
	olat = map_olat * DEGRAD

	sn = math.Tan(PI * 0.25 + slat2 * 0.5) / math.Tan(PI * 0.25 + slat1 * 0.5)
	sn = math.Log(math.Cos(slat1) / math.Cos(slat2)) / math.Log(sn)
	sf = math.Tan(PI * 0.25 + slat1 * 0.5)
	sf = math.Pow(sf, sn) * math.Cos(slat1) / sn
	ro = math.Tan(PI * 0.25 + olat * 0.5)
	ro = re * sf / math.Pow(ro, sn)

	if mode == 0 {

		ra = math.Tan(PI * 0.25 + var2 * DEGRAD * 0.5)
		ra = re * sf / math.Pow(ra, sn)
		theta = var1 * DEGRAD - olng
		if theta >  PI { theta -= 2.0 * PI }
		if theta < -PI { theta += 2.0 * PI }
		theta = theta * sn
        ret1 = (float64)(ra * math.Sin(theta)) + map_xo
        ret2 = (float64)(ro - ra * math.Cos(theta)) + map_yo

	} else {
		
		xn = var1 - map_xo
		yn = ro - var2 + map_yo
		ra = math.Sqrt(xn * xn + yn * yn);
		if sn < 0.0 { ra = (-1.0) * ra }
		alat = math.Pow((re * sf / ra), (1.0 / sn))
		alat = 2.0 * math.Atan(alat) - PI * 0.5
		if math.Abs(xn) <= 0.0 {
			theta = 0.0
		} else {
			if math.Abs(yn) <= 0.0 {
				theta = PI * 0.5
				if xn < 0.0 { theta = (-1.0) * theta }
            } else {
				theta = math.Atan2(xn, yn)
			}
        }
        
		alng = theta / sn + olng
		ret1 = (float64)(alng * RADDEG);
        ret2 = (float64)(alat * RADDEG);
	}

	return ret1, ret2
}
