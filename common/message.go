package common

func GetErrorMessage(lang string, code int) string {

	var mesg string

	if lang == "K" {
		mesg = GetErrorMessageKOR(code)
	} else {
		mesg = GetErrorMessageENG(code)
	}

	return mesg
}

func GetErrorMessageKOR(code int) string {

	switch code {
		case 0: 	return "정상처리"

		case 9001:	return "검증 오류"
		case 9002:	return "요청이 만료되었습니다"
		case 9003:	return "요청 데이타 오류"
		case 9004:	return "정의되지 않은 요청"
		case 9005:	return "요청 데이타 타입 오류"

		case 9901:	return "시스템 오류"
		case 9902:	return "잘못된 접근입니다"
	}

	return "정의되지 않은 메세지"
}

func GetErrorMessageENG(code int) string {

	switch code {
		case 0:		return "Processed"

		case 9001:	return "Validation Error"
		case 9002:	return "Your request has expired"
		case 9003:	return "Request data error"
		case 9004:	return "undefined request"
		case 9005:	return "Request data type error"

		case 9901:	return "System Error"
		case 9902:	return "Incorrect Access"
	}

	return "undefined message"
}
