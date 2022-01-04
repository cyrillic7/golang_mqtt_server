package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
)

//序列号
type serialcode struct {
	Device_serial string `json:"device_serial"`
}

//var client mqtt.Client
var g_dnest string

//序列号
type Jsonserialcode struct {
	Code  int        `json:"code"`
	Datas serialcode `json:"data"`
}

type data struct {
	Reserve string `json:"reserve"`
}

type JsonMsg struct {
	Code  int  `json:"code"`
	Datas data `json:"data"`
}

//后台向中控推送操作 K3 各模块的消息格式
type k3MsgData struct {
	Sid int `json:"sid"`
	Rid int `json:"rid"`
	Cmd int `json:"cmd"`
	P1  int `json:"p1"`
	P2  int `json:"p2"`
	P3  int `json:"p3"`
	P4  int `json:"p4"`
	P5  int `json:"p5"`
}

//k3状态
type k3Msg struct {
	Code  int       `json:"code"`
	Datas k3MsgData `json:"data"`
}

type devMsgData struct {
	Dev    int `json:"dev"`
	Status int `json:"status"`
}

//设备信息
type devMsg struct {
	Code    int        `json:"code"`
	DevData devMsgData `json:"data"`
}

type devStateMsgData struct {
	Dev int `json:"dev"`
}

//设备状态
type devStateMsg struct {
	Code    int             `json:"code"`
	DevData devStateMsgData `json:"data"`
}

//空调温度状态
type devTempMsgData struct {
	Dev int `json:"temptype"`
}

type devTempMsg struct {
	Code     int            `json:"code"`
	TempData devTempMsgData `json:"data"`
}

//空调反馈
type devTempResultMsgData struct {
	Dev     int     `json:"temptype"`
	DevTemp float64 `json:"temp"`
}
type devTempResultMsg struct {
	Code     int                  `json:"code"`
	TempData devTempResultMsgData `json:"data"`
}

const broker = "mqtt://121.40.120.163:1883"
const username = "client"
const password = "UJqmsq"
const ClientID = "AH_Machine_nest_ccy"

var topiclinkresponse string
var topicuplinkheartbeat string
var topicuplinkwarning string

var topocAHheartBeat string = "/heisha/airport/heartbeat/"
var topocAHresponse string = "/heisha/airport/mission/response/"
var topocAHrequest string = "/heisha/airport/mission/request/"

//机库心跳
type HeartBeatData struct {
	Canopy int     `json:"canopy"` //防雨盖状态
	Posbar int     `json:"posbar"` //归中杆位置状态
	Sensor int     `json:"sensor"` //传感器故障状态码
	Motor  int     `json:"motor"`  //电机故障状态码
	Charge int     `json:"charge"` //充电及电池相关状态
	Vol    float64 `json:"vol"`    //充电电压,单位mV
	Cur    float64 `json:"cur"`    //充电电流,单位mA
}

type HeartBeatMsg struct {
	Code int           `json:"code"`
	Data HeartBeatData `json:"data"`
}

//机库警告
type WarningData struct {
	Power        int `json:"power"`        //UPS 的电源连接状态码，0 为正常，1为异常，即电源断开
	Bat_capacity int `json:"bat_capacity"` // UPS 电池余量百分比，值为 0~100
	Bat_remain   int `json:"bat_remain"`   // UPS电池保持时间，单位为分钟
}

type WarningMsg struct {
	Code int         `json:"code"`
	Data WarningData `json:"data"`
}

//机库心跳包
type HGHeartMsg struct {
	HGSerialcode string  `json:"device_id"`    //设备序列号
	Canopy       int     `json:"canopy"`       //防雨盖状态
	Posbar       int     `json:"posbar"`       //归中杆位置状态
	Sensor       int     `json:"sensor"`       //传感器故障状态码
	Motor        int     `json:"motor"`        //电机故障状态码
	Charge       int     `json:"charge"`       //充电及电池相关状态
	Vol          float64 `json:"vol"`          //充电电压,单位mV
	Cur          float64 `json:"cur"`          //充电电流,单位mA
	Power        int     `json:"power"`        //UPS 的电源连接状态码，0 为正常，1为异常，即电源断开
	Bat_capacity int     `json:"bat_capacity"` // UPS 电池余量百分比，值为 0~100
	Bat_remain   int     `json:"bat_remain"`   // UPS电池保持时间，单位为分钟
	FlowState    int     `json:"flowstate"`    //保留，流状态
	HeartTime    string  `json:"time"`         //时间
}

type MissionDataReq struct {
	Weather int `json:"weather"` //气象站          //0正常，1风速过大，2雨量过大，3温度（预留），4湿度（预留）
	Canpoy  int `json:"canpoy"`  //防雨盖状态        //0 正常 ，非0故障（返回错误码）
	Posbar  int `json:"posbar"`  //推拉杆状态         //0 正常,    非0故障（返回错误码）
	Remote  int `json:"remote"`  //遥控器电源         //0 打开    1关闭
	Motor   int `json:"motor"`   //电机状态          //0 正常  非0故障（返回错误码）
	Drone   int `json:"drone"`   //无人机开关机      //0 执行成功，1执行失败
	Ups     int `json:"ups"`     // ups电源状态        //0 正常，1异常
}

//任务request
type HGMission struct {
	RequestID   string      `json:"request_id"` //请求ID
	Platform    string      `json:"platform"`   //平台
	MissionID   string      `json:"mission_id"` //任务ID
	DeviceID    string      `json:"device_id"`  //设备ID
	SendSDK     string      `json:"send_sdk"`   //M-O接收
	MType       string      `json:"type"`       //类型
	MData       interface{} `json:"data"`       //返回数据
	State       int         `json:"status"`     //状态码
	MissionTime string      `json:"time"`       //时间
}

type HGWeather struct {
	AmbientTemperature float64 `json:"ambientTemperature"` //温度
	AmbientHumidity    float64 `json:"ambientHumidity"`    //湿度
	Pressure           float64 `json:"pressure"`           //气压
	Noise              float64 `json:"noise"`              //干扰
	WindSpeed          float64 `json:"windSpeed"`          //风速
	WindScale          float64 `json:"windScale"`          //风力等级
	WindDirection      float64 `json:"windDirection"`      //风向
	Rainfall           float64 `json:"rainfall"`           //雨量
}

type HGWeatherMsg struct {
	Code int       `json:"code"`
	Data HGWeather `json:"data"`
}

//机库状态
type StatusType int

const (
	STATUS_CHECK                 StatusType = 0  //机库自检
	STATUS_MISSION_START         StatusType = 1  //任务开始
	STATUS_HG_OPENING            StatusType = 2  //开库中
	STATUS_HG_OPEN               StatusType = 3  //机库打开
	STATUS_CLOSEING              StatusType = 4  //关库中
	STATUS_CLOSEED               StatusType = 5  //机库关闭
	STATUS_UAV_RETURN            StatusType = 6  //无人机返航
	STATUS_UAV_CHARGING          StatusType = 7  //无人机充电中
	STATUS_UAV_CHARGING_DONE     StatusType = 8  //无人机充电完成
	STATUS_UAV_CHARGING_NULL     StatusType = 9  //未检测到电池
	STATUS_UAV_CHARGING_ABNORMAL StatusType = 10 //充电异常
)

//机库状态map
var g_HGStatusMap = make(map[string]StatusType)

//var g_HGheart HGHeartMsg
var g_HGheartMap = make(map[string]*HGHeartMsg)

//任务
//var g_HGMissionMap = make(map[string]*HGMission)
var g_HGMissionList = list.New()

//气象
var g_HGWeather = make(map[string]*HGWeather)

//机库报警
var g_CanpoyWarning bool = false

// 将十进制数字转化为二进制字符串
func convertToBin(num int) string {
	s := ""

	if num == 0 {
		return "0"
	}

	// num /= 2 每次循环的时候 都将num除以2  再把结果赋值给 num
	for ; num > 0; num /= 2 {
		lsb := num % 2
		// strconv.Itoa() 将数字强制性转化为字符串
		s = strconv.Itoa(lsb) + s
	}
	return s
}

//message的回调
var onMessage mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("[%s] -> %s\n", msg.Topic(), msg.Payload())

	if strings.Compare(msg.Topic(), "DNEST/broadcast") == 0 {
		msgdevice := Jsonserialcode{}
		if err := json.Unmarshal(msg.Payload(), &msgdevice); err != nil {
			fmt.Printf("RegisterMqttHandler Unmarshal fail")
		}
		//fmt.Println(msgdevice)
		g_dnest = msgdevice.Datas.Device_serial
		//fmt.Println(g_dnest)

		if msgdevice.Datas.Device_serial == "" {

		} else {
			value, ok := g_HGheartMap[msgdevice.Datas.Device_serial]
			if ok {
				//处理找到的value
				fmt.Println("value DNEST/broadcast : ", value)
			} else {
				g_HGheartMap[msgdevice.Datas.Device_serial] = &HGHeartMsg{msgdevice.Datas.Device_serial, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, ""}
				g_HGStatusMap[msgdevice.Datas.Device_serial] = STATUS_UAV_CHARGING_NULL
			}
		}

		//g_HGheartMap[msgdevice.Datas.Device_serial] = HGHeartMsg{msgdevice.Datas.Device_serial, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, ""}

	}

	for key, value := range g_HGheartMap {
		topiclinkresponses := key + "/downlink/response"
		if strings.Compare(msg.Topic(), topiclinkresponses) == 0 {
			fmt.Println(topiclinkresponses)

			// {
			// 	code := gjson.Get(msg.Payload(), "code")
			// }
			var i map[string]interface{}
			if err := json.Unmarshal(msg.Payload(), &i); err != nil {
				log.Fatal(err)
			}
			fmt.Println("code", i["code"].(float64))
			switch codeid := i["code"].(float64); codeid {
			case 14:
				{
					fmt.Println("data", i["data"])
					idata := i["data"].(map[string]interface{})
					g_HGWeather[key] = &HGWeather{idata["ambientTemperature"].(float64), idata["ambientHumidity"].(float64), idata["pressure"].(float64), idata["noise"].(float64), idata["windSpeed"].(float64), idata["windScale"].(float64), idata["windDirection"].(float64), idata["rainfall"].(float64)}

					for key1, value1 := range g_HGWeather {
						fmt.Println("key1", key1, "value1", value1)
					}
				}
			default:
				{
					fmt.Println("codeid", i["code"])
				}
			}
			// if i["code"] == 14 {
			// 	fmt.Println("data", i["data"])
			// }

			// ndata := i.(map[string]interface{})
			// for k, v := range ndata {
			// 	switch v := v.(type) {
			// 	case string:
			// 		fmt.Println(k, v, "(string)")

			// 	case int:
			// 		fmt.Println(k, v, "(int)")

			// 	case []interface{}:
			// 		for i, u := range v {
			// 			fmt.Println(i, u, "  ")
			// 		}
			// 	default:
			// 		fmt.Println(k, v, "(unkonw)")
			// 	}

			// }
		}

		topicheartbeats := key + "/uplink/heartbeat"
		if strings.Compare(msg.Topic(), topicheartbeats) == 0 {
			//fmt.Println(topicheartbeats)
			heartmsg := HeartBeatMsg{}
			if err := json.Unmarshal(msg.Payload(), &heartmsg); err != nil {
				fmt.Printf("heartmsg Unmarshal fail")
			}
			switch g_CanpoyWarning {
			case false:
				{
					value.Canopy = heartmsg.Data.Canopy
				}
			case true:
				{
				}

			}

			value.Posbar = heartmsg.Data.Posbar
			value.Sensor = heartmsg.Data.Sensor
			value.Motor = heartmsg.Data.Motor
			value.Charge = heartmsg.Data.Charge
			value.Vol = heartmsg.Data.Vol
			value.Cur = heartmsg.Data.Cur
		}

		topicwarnings := key + "/uplink/warning"
		if strings.Compare(msg.Topic(), topicwarnings) == 0 {
			//fmt.Println(topicwarnings)
			warningmsg := WarningMsg{}
			if err := json.Unmarshal(msg.Payload(), &warningmsg); err != nil {
				fmt.Printf("heartmsg Unmarshal fail")
			}
			if warningmsg.Code == 17 {
				value.Power = warningmsg.Data.Power
				value.Bat_capacity = warningmsg.Data.Bat_capacity
				value.Bat_remain = warningmsg.Data.Bat_remain
			} else if warningmsg.Code == 18 {
				var iW map[string]interface{}
				if err := json.Unmarshal(msg.Payload(), &iW); err != nil {
					log.Fatal(err)
				}
				fmt.Println("iW[data]", iW["data"])
				iWdata := iW["data"].(map[string]interface{})
				if iWdata["canopy_abnormal"].(float64) != 0 {
					//g_HGheartMap[key].Canopy = (int)(iWdata["canopy_abnormal"].(float64))
					g_CanpoyWarning = true
				} else {
					g_CanpoyWarning = false
				}
			}

		}

		fmt.Println("value --------------------- === ", value)
	}

	//topiclinkresponse := g_dnest + "/downlink/request"
	// if strings.Compare(msg.Topic(), topiclinkresponse) == 0 {
	// 	fmt.Println(topiclinkresponse)

	// }

	// if strings.Compare(msg.Topic(), topicuplinkheartbeat) == 0 {
	// 	fmt.Println(topicuplinkheartbeat)
	// 	heartmsg := HeartBeatMsg{}
	// 	if err := json.Unmarshal(msg.Payload(), &heartmsg); err != nil {
	// 		fmt.Printf("heartmsg Unmarshal fail")
	// 	}
	// 	g_HGheart.Canopy = heartmsg.Data.Canopy
	// 	g_HGheart.Posbar = heartmsg.Data.Posbar
	// 	g_HGheart.Sensor = heartmsg.Data.Sensor
	// 	g_HGheart.Motor = heartmsg.Data.Motor
	// 	g_HGheart.Charge = heartmsg.Data.Charge
	// 	g_HGheart.Vol = heartmsg.Data.Vol
	// 	g_HGheart.Cur = heartmsg.Data.Cur
	// }

	// if strings.Compare(msg.Topic(), topicuplinkwarning) == 0 {
	// 	fmt.Println(topicuplinkwarning)
	// 	warningmsg := WarningMsg{}
	// 	if err := json.Unmarshal(msg.Payload(), &warningmsg); err != nil {
	// 		fmt.Printf("heartmsg Unmarshal fail")
	// 	}
	// 	g_HGheart.Power = warningmsg.Data.Power
	// 	g_HGheart.Bat_capacity = warningmsg.Data.Bat_capacity
	// 	g_HGheart.Bat_remain = warningmsg.Data.Bat_remain

	// }

	//艾航无人机任务主题
	if strings.Compare(msg.Topic(), topocAHrequest) == 0 {
		//fmt.Println(topocAHrequest)
		msgMission := HGMission{}
		if err := json.Unmarshal(msg.Payload(), &msgMission); err != nil {
			fmt.Printf("RegisterMqttHandler Unmarshal fail")
		}

		value, ok := g_HGheartMap[msgMission.DeviceID]
		if ok {
			//处理找到的value
			fmt.Println(value)
			//g_HGMissionMap[msgMission.DeviceID] = &msgMission
			fmt.Println("qqqqqqqqqqqqqqqqqqqqqqqqqqqqqmsgMission.MType : ", msgMission.MType)
			g_HGMissionList.PushBack(msgMission)
			switch {
			case msgMission.MType == "airport_check": //自检
				{
					fmt.Println("casetype 1  : ", "airport_check")
					go AH_HG_Inspectionself(msgMission.DeviceID)
					g_HGStatusMap[msgMission.DeviceID] = STATUS_CHECK
					return
				}
			case msgMission.MType == "airport_open": //开库
				{
					fmt.Println("casetype 2  : ", "airport_open")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 2, 41, 0, 0, 0, 0, 0, true)
					//go pushOperationModuleMsg(msgMission.DeviceID, 1, 2, 82, 0, 0, 0, 0, 0, false)
					g_HGStatusMap[msgMission.DeviceID] = STATUS_MISSION_START
					return
				}

			case msgMission.MType == "airport_close": //关库
				{
					fmt.Println("casetype 3  : ", "airport_close")
					//go pushOperationModuleMsg(msgMission.DeviceID, 1, 2, 125, 0, 0, 0, 0, 0, true)
					//time.Sleep(time.Duration(20) * time.Second)
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 2, 42, 0, 0, 0, 0, 0, true)
					g_HGStatusMap[msgMission.DeviceID] = STATUS_CLOSEING
					return
				}

			case msgMission.MType == "airport_shout": //喊话
				{
					shoutID := msgMission.MData.(float64)
					// int5, err := strconv.Atoi(shoutID)
					// if err != nil {
					// 	fmt.Println(err)
					// }
					go pushVoiceAlam(msgMission.DeviceID, int(shoutID))
					return
				}
			case msgMission.MType == "chargebar_open": //充电棒打开
				{
					fmt.Println("casetype 33  : ", "chargebar_open")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 3, 82, 0, 0, 0, 0, 0, true)
					return
				}
			case msgMission.MType == "chargebar_close": //充电棒收紧
				{
					fmt.Println("casetype 44  : ", "chargebar_close")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 3, 81, 0, 0, 0, 0, 0, true)
					return
				}
			// 	fallthrough
			case msgMission.MType == "airport_powerOn": //充电
				{
					fmt.Println("casetype 4  : ", "airport_powerOn")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 3, 81, 0, 0, 0, 0, 0, false)
					//time.Sleep(time.Duration(25) * time.Second)
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 4, 121, 0, 0, 0, 0, 0, true)
					return
				}

			case msgMission.MType == "airport_powerOff": //停止充电
				{
					fmt.Println("casetype 5  : ", "airport_powerOff")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 4, 123, 0, 0, 0, 0, 0, true)
					return
				}

			case msgMission.MType == "uav_retuen": //返航
				{
					fmt.Println("casetype 6  : ", "uav_retuen")
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 2, 41, 0, 0, 0, 0, 0, true)
					go pushOperationModuleMsg(msgMission.DeviceID, 1, 3, 82, 0, 0, 0, 0, 0, false)
					g_HGStatusMap[msgMission.DeviceID] = STATUS_UAV_RETURN
					return
				}

			case msgMission.MType == "uav_land_request": //降落请求
				{
					go AH_UavLandReq(msgMission.DeviceID)
					return
				}

			default:
				{

				}

			}
		} else {
			fmt.Println("can not find device ! ", msgMission.DeviceID)
		}
	}

}

var wg sync.WaitGroup
var client mqtt.Client

//艾航机库心跳
func AH_HeartbeatPublish() {

	timer := time.NewTicker(1 * time.Second)
	for range timer.C {
		for key, value := range g_HGheartMap {
			fmt.Println("Key:", key, "Value:", value)
			// charges := convertToBin(value.Charge)
			// fmt.Println("charges ---------------------  :", charges)
			value.FlowState = 0
			value.HeartTime = time.Now().Format("2006-01-02 15:04:05")
			jsonStr, err := json.Marshal(value)
			if err != nil {
				log.Fatal()
			}
			//topicdownlinkrequest := g_dnest + "/downlink/request"
			//fmt.Println("jsonStr == ", jsonStr)
			client.Publish(topocAHheartBeat, 0, false, jsonStr)

			// _tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[key].Charge))
			// fmt.Println("cmdheartbeat:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
		}
	}
}

//机库自检
var bInSelf bool = true

func AH_HG_Inspectionself(HGDev string) {
	//var bInSelf bool = true
	valueH, ok := g_HGheartMap[HGDev]
	missionreqs := MissionDataReq{0, 0, 0, 0, 0, 0, 0}
	if ok {
		fmt.Println("AH_HG_Inspectionself Value:", valueH)
		valueW, ok := g_HGWeather[HGDev]
		if ok {
			//气象
			if valueW.AmbientTemperature <= -30 || valueW.AmbientTemperature >= 60 {
				missionreqs.Weather = 3
				bInSelf = false
			} else if valueW.WindSpeed >= 7 {
				missionreqs.Weather = 1
				bInSelf = false
			} else if valueW.Rainfall >= 10 {
				missionreqs.Weather = 2
				bInSelf = false
			}

			//防雨棚
			if valueH.Canopy == 0 || valueH.Canopy == 3 {
				missionreqs.Canpoy = -1
				bInSelf = false
			}

			//推拉杆电机
			if valueH.Posbar == 6 {
				missionreqs.Posbar = -1
				bInSelf = false
			}

			//电机状态
			if valueH.Motor != 0 {
				missionreqs.Motor = valueH.Motor
				bInSelf = false
			}

			//ups
			// if valueH.Power != 0 {
			// 	missionreqs.Ups = valueH.Power
			// 	bInSelf = false
			// }
		}
		newInterReq := interface{}(missionreqs).(MissionDataReq)
		//newInterface1 = missionreqs

		//g_HGMissionMap[HGDev].MData = newInterReq
		for i := g_HGMissionList.Front(); i != nil; i = i.Next() {
			fmt.Println(i.Value)
			p, ok := (i.Value).(HGMission)
			if ok {
				if strings.Compare("airport_check", p.MType) == 0 {
					p.MData = newInterReq
					switch bInSelf {
					case false:
						{
							p.State = 400
						}
					case true:
						{
							p.State = 200
						}
					}

					p.MissionTime = time.Now().Format("2006-01-02 15:04:05")

					jsonMissionStr, err := json.Marshal(p)
					if err != nil {
						log.Fatal()
					}

					client.Publish(topocAHresponse, 2, false, jsonMissionStr)

					g_HGMissionList.Remove(i)

					return
				}
			} else {
				fmt.Println("not HGMission struct")
			}
		}
	}
}

//降落请求
func AH_UavLandReq(dnest string) {
	valueH, ok := g_HGheartMap[dnest]
	if ok {
		fmt.Println("AH_HG_Inspectionself Value:", valueH)
		if valueH.Canopy == 2 && valueH.Posbar == 1 {
			AH_mission_response(dnest, "allow", 200, "uav_land_request")
		} else {
			if valueH.Canopy == 1 {
				AH_mission_response(dnest, "防雨棚未开启", 400, "uav_land_request")
			} else if valueH.Posbar == 2 {
				AH_mission_response(dnest, "充电杆未放开", 400, "uav_land_request")
			} else {
				AH_mission_response(dnest, "未知错误", 400, "uav_land_request")
			}
		}
	}
}

func Cors(ctx iris.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Credentials", "true")
	if ctx.Request().Method == "OPTIONS" {
		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		ctx.StatusCode(204)
		return
	}
	ctx.Next()
}

// func Cors(ctx iris.Context) {
// 	ctx.Header("Access-Control-Allow-Origin", "*")
// 	if ctx.Request().Method == "OPTIONS" {
// 		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
// 		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Api, Accept, Authorization, Version, Token")
// 		ctx.StatusCode(204)
// 		return
// 	}
// 	ctx.Next()
// }

// func cors(ctx iris.Context) {
// 	middleware.Cors(ctx)
// }

// // InitRouter 路由初始化
// func InitRouter() *iris.Application {
// 	r := iris.New()
// 	r.Use(recover.New())

// 	// cors
// 	r.Use(cors)

// 	// common
// 	common := r.Party("/")
// 	{
// 		common.Options("*", func(ctx iris.Context) {
// 			ctx.Next()
// 		})
// 	}

// 	return r
// }

// func Cors(ctx iris.Context) {
// 	ctx.Header("Access-Control-Allow-Origin", "http://localhost:10901")
// 	ctx.Header("Access-Control-Allow-Credentials", "true")
// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
// 	ctx.Next()
// }
// func CRS(ctx iris.Context) { //<!-- -->
// 	ctx.Header("Access-Control-Allow-Origin", "*")
// 	ctx.Header("Access-Control-Allow-Credentials", "true")
// 	if ctx.Method() == iris.MethodOptions { //<!-- -->
// 		ctx.Header("Access-Control-Methods", "POST, PUT, PATCH, DELETE")
// 		ctx.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,X-API-CHANNEL,Token")
// 		ctx.Header("Access-Control-Max-Age", "86400")
// 		ctx.StatusCode(iris.StatusNoContent)
// 		return
// 	}
// 	ctx.Next()
// }

func main() {
	app := iris.New() //iris.Default()
	opts := cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Content-Type"},
		AllowedMethods: []string{"GET", "POST", "PUT", "HEAD"},
		ExposedHeaders: []string{"X-Header"},
		MaxAge:         int((24 * time.Hour).Seconds()),
		// Debug:          true,
	}
	app.UseRouter(cors.New(opts))
	mqttConnect()
	defer client.Disconnect(250) //注册销毁

	wg.Add(1)
	go mqttSubScribe("DNEST/broadcast", 1)
	wg.Add(1)
	go DNESTPublish()
	time.Sleep(time.Duration(3) * time.Second)

	for key, value := range g_HGheartMap {
		fmt.Println("Key:", key, "Value:", value)
		topiclinkresponses := key + "/downlink/response"
		go mqttSubScribe(topiclinkresponses, 1)

		topicuplinkheartbeats := key + "/uplink/heartbeat"
		go mqttSubScribe(topicuplinkheartbeats, 1)

		topicuplinkwarnings := key + "/uplink/warning"
		go mqttSubScribe(topicuplinkwarnings, 1)

		go AH_weatherHeart()
	}
	// topiclinkresponse = g_dnest + "/downlink/response"
	// go mqttSubScribe(topiclinkresponse)

	// topicuplinkheartbeat = g_dnest + "/uplink/heartbeat"
	// go mqttSubScribe(topicuplinkheartbeat)

	// topicuplinkwarning = g_dnest + "/uplink/warning"
	// go mqttSubScribe(topicuplinkwarning)

	//艾航无人机任务主题
	go mqttSubScribe(topocAHrequest, 2)

	//艾航机库心跳
	go AH_HeartbeatPublish()

	//go pullWeatherStations(1)
	// go pullWeatherStations(2)
	// go pullWeatherStations(3)
	// go pullWeatherStations(4)
	//v1 := app.Party("/", crs).AllowMethods(iris.MethodOptions) // <- 对于预检很重要。
	//v1 := app.Party("/", crs).AllowMethods(iris.MethodOptions)
	{
		//打开防雨棚
		app.Get("MN_ACTION/CanopyOpen", func(ctx iris.Context) {
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()
			fmt.Print("/MN_ACTION/CanopyOpen/============= \n")

			dnest := ctx.URLParam("dnest")
			// inttemp, err := strconv.Atoi(TempType)
			// if err != nil {
			// 	panic(err)
			// }

			fmt.Print(dnest + "\n")

			pushOperationModuleMsg(dnest, 1, 2, 41, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
			ctx.JSON(iris.Map{
				"msg":    "CanopyOpen",
				"data":   "执行成功",
				"status": 200,
			})
		})

		//关闭防雨棚
		app.Get("MN_ACTION/CanopyClose", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/CanopyClose/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 2, 42, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
			ctx.JSON(iris.Map{
				"msg":    "CanopyClose",
				"data":   "执行成功",
				"status": 200,
			})
		})

		//无人机开机
		app.Get("MN_ACTION/UavStartUp", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/UavStartUp/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()
			dnest := ctx.URLParam("dnest")

			pushOperationModuleMsg(dnest, 1, 4, 124, 0, 0, 0, 0, 0, false)

			ctx.JSON(iris.Map{
				"msg":    "UavStartUp",
				"data":   "执行成功",
				"status": 200,
			})

			//ctx.JSON(Missionlives)
		})

		//无人机关机
		app.Get("MN_ACTION/UavShutDown", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/UavShutDown/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 4, 125, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
			ctx.JSON(iris.Map{
				"msg":    "UavShutDown",
				"data":   "执行成功",
				"status": 200,
			})
		})

		//归中杆收
		app.Get("MN_ACTION/PowerRodIn", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/PowerRodIn/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 3, 81, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
		})

		//归中杆放
		app.Get("MN_ACTION/PowerRodOut", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/PowerRodOut/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 3, 82, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
		})

		//开始充电
		app.Get("MN_ACTION/PowerOn", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/PowerOn/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 3, 121, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
		})

		//停止充电
		app.Get("MN_ACTION/PowerOff", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/PowerOn/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushOperationModuleMsg(dnest, 1, 3, 123, 0, 0, 0, 0, 0, false)

			//ctx.JSON(Missionlives)
		})

		//关闭安卓电源
		app.Get("MN_ACTION/AndroidPoweroff", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweroff/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 2, 0)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//开启安卓电源
		app.Get("MN_ACTION/AndroidPoweron", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweron/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 2, 1)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//关闭遥控器电源
		app.Get("MN_ACTION/RemoteCtlPowerooff", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweron/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 4, 0)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//打开遥控器电源
		app.Get("MN_ACTION/RemoteCtlPoweron", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweron/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 4, 1)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//关闭照明灯电源
		app.Get("MN_ACTION/FloodlightPoweroff", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweron/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 5, 0)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//打开照明灯电源
		app.Get("MN_ACTION/FloodlightPoweron", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPoweron/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			pushDevOperationModuleMsg(dnest, 5, 1)
			//pushOperationModuleMsg(1, 3, 82, 0, 0, 0, 0, 0)

			//ctx.JSON(Missionlives)
		})

		//Android 电源状态获取
		app.Get("MN_STATE/AndroidPowerStatus", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPowerStatus/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")

			devStateMsgs := devStateMsg{2, devStateMsgData{2}}

			fmt.Println(devStateMsgs)
			jsonStr, err := json.Marshal(devStateMsgs)
			if err != nil {
				log.Fatal()
			}
			topicdownlinkrequest := dnest + "/downlink/request"
			fmt.Println(jsonStr)
			client.Publish(topicdownlinkrequest, 0, false, jsonStr)
			time.Sleep(time.Duration(3) * time.Second)

			//ctx.JSON(Missionlives)
		})

		//照明灯电源状态获取
		app.Get("MN_STATE/FloodlightPowerStatus", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPowerStatus/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			devStateMsgs := devStateMsg{2, devStateMsgData{5}}

			fmt.Println(devStateMsgs)
			jsonStr, err := json.Marshal(devStateMsgs)
			if err != nil {
				log.Fatal()
			}
			topicdownlinkrequest := dnest + "/downlink/request"
			fmt.Println(jsonStr)
			client.Publish(topicdownlinkrequest, 0, false, jsonStr)
			time.Sleep(time.Duration(3) * time.Second)

			//ctx.JSON(Missionlives)
		})

		//获取空调温度
		app.Get("MN_STATE/AirCdtTemp", func(ctx iris.Context) {
			fmt.Print("/MN_ACTION/AndroidPowerStatus/============= \n")
			// ctx.Header("Access-Control-Allow-Origin", "*")
			// if ctx.Request().Method == "OPTIONS" {
			// 	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
			// 	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			// 	ctx.StatusCode(204)
			// 	return
			// }
			// ctx.Next()

			dnest := ctx.URLParam("dnest")
			TempType := ctx.Params().Get("temptype")
			inttemp, err := strconv.Atoi(TempType)
			if err != nil {
				panic(err)
			}

			devStateMsgs := devTempMsg{3, devTempMsgData{inttemp}}

			fmt.Println(devStateMsgs)
			jsonStr, err := json.Marshal(devStateMsgs)
			if err != nil {
				log.Fatal()
			}
			topicdownlinkrequest := dnest + "/downlink/request"
			fmt.Println(jsonStr)
			client.Publish(topicdownlinkrequest, 0, false, jsonStr)
			time.Sleep(time.Duration(3) * time.Second)

			//ctx.JSON(Missionlives)
		})
	}
	app.Run(iris.Addr(":10901"), iris.WithoutPathCorrectionRedirection)
}

//控制空调开关
type devAirCdtMsgData struct {
	Dev int `json:"status"`
}

type devAirCdtMsg struct {
	Code       int              `json:"code"`
	AirCdtData devAirCdtMsgData `json:"data"`
}

//控制空调开关，0关机，1开机
func AirCdtControl(action int) {

	devAirctlMsgs := devAirCdtMsg{4, devAirCdtMsgData{action}}

	fmt.Println(devAirctlMsgs)
	jsonStr, err := json.Marshal(devAirctlMsgs)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)

}

//设置空调制冷质量热温度,2制冷，3制热，temp温度值
func AirCdtTempControl(temptype int, temp float64) {
	airtempMsg := devTempResultMsg{5, devTempResultMsgData{temptype, temp}}

	fmt.Println(airtempMsg)
	jsonStr, err := json.Marshal(airtempMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//控制空调开关
type devtrackMsgData struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Alt   float64 `json:"alt"`
	Stars int     `json:"starts"`
}

type devtrackMsg struct {
	Code      int             `json:"code"`
	TrackData devtrackMsgData `json:"data"`
}

//向中控推送定向天线跟踪目标位置信息
func pushtrackingPos(lat float64, lon float64, alt float64, stars int) {

	lat = lat * 10000000
	lon = lon * 10000000
	trackMsg := devtrackMsg{6, devtrackMsgData{lat, lon, alt, stars}}

	fmt.Println(trackMsg)
	jsonStr, err := json.Marshal(trackMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//中控推送访问天线跟踪目标home位置
func pulltrackingHomePos() {
	trackHomeMsg := JsonMsg{7, data{"null"}}

	fmt.Println(trackHomeMsg)
	jsonStr, err := json.Marshal(trackHomeMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//控制空调开关
type paramMsgData struct {
	Param int `json:"param"`
}

type ParamMsg struct {
	Code      int          `json:"code"`
	ParamData paramMsgData `json:"data"`
}

func AH_weatherHeart() {

	timer := time.NewTicker(10 * time.Second)
	for range timer.C {
		for key, value := range g_HGheartMap {
			fmt.Println(key, value)
			go pullWeatherStations(key, 1)
		}

	}
}

//获取气象站信息 1 表示请求的是风速，2 是环境湿度，3 是环境温度，4 是累计雨量值
func pullWeatherStations(dnest string, att int) {
	weatherMsgs := ParamMsg{13, paramMsgData{att}}

	fmt.Println(weatherMsgs)
	jsonStr, err := json.Marshal(weatherMsgs)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//声光报警
type VoiceWarningMsgData struct {
	Param int `json:"speak_msg"`
}

type VoiceWarningMsg struct {
	Code             int                 `json:"code"`
	VoiceWarningData VoiceWarningMsgData `json:"data"`
}

//推送声光报警器信息 1 代表飞机备飞完成，2 表示飞机即将起飞，3 表示飞机自动返航
func pushVoiceAlam(dnest string, att int) {
	WarningMsgs := VoiceWarningMsg{19, VoiceWarningMsgData{att}}

	fmt.Println(WarningMsgs)
	jsonStr, err := json.Marshal(WarningMsgs)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)

	AH_mission_response(dnest, "喊话执行成功", 200, "airport_shout")
}

//中控推送访问K100-CD参数配置
/* 1 为充电电池串数值，3 为 3S，4 为 4S；2 为充满电的最小电压值,单位为 mV；
3 为充满电的最大电流值,单位为 mA；4 为慢充充电的时间，单位为秒钟*/
func pullKCDParam(att int) {
	KCDParamMsg := ParamMsg{24, paramMsgData{att}}

	fmt.Println(KCDParamMsg)
	jsonStr, err := json.Marshal(KCDParamMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//控制空调开关
type paramValueMsgData struct {
	Param int `json:"param"`
	Value int `json:"value"`
}

type ParamValueMsg struct {
	Code      int               `json:"code"`
	ParamData paramValueMsgData `json:"data"`
}

//向中控推送K100-CD参数消息
func pushKCDParam(param int, value int) {
	ParamResultMsg := ParamValueMsg{27, paramValueMsgData{param, value}}

	fmt.Println(ParamResultMsg)
	jsonStr, err := json.Marshal(ParamResultMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := g_dnest + "/downlink/request"
	fmt.Println(jsonStr)
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)
}

//设备信息指令
func pushDevOperationModuleMsg(dnest string, dev int, status int) {
	devMsg := devMsg{1, devMsgData{dev, status}}

	fmt.Println(devMsg)
	jsonStr, err := json.Marshal(devMsg)
	if err != nil {
		log.Fatal()
	}
	topicdownlinkrequest := dnest + "/downlink/request"
	client.Publish(topicdownlinkrequest, 0, false, jsonStr)
	time.Sleep(time.Duration(3) * time.Second)

}

func MissionRequest(dnest string, cmd int, bMission bool) {
	switch bMission {
	case false:
		{
			return
		}
	case true:
		{
			switch cmd {
			case 41:
				{
					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						if g_HGheartMap[dnest].Canopy == 2 {

							if g_HGStatusMap[dnest] == STATUS_MISSION_START {
								//无人机开机
								go pushOperationModule(dnest, 1, 4, 124, 0, 0, 0, 0, 0, false)
								g_HGStatusMap[dnest] = STATUS_HG_OPENING
							} else if g_HGStatusMap[dnest] == STATUS_UAV_RETURN && g_HGheartMap[dnest].Posbar == 1 {
								//充电杆放开
								go pushOperationModule(dnest, 1, 3, 82, 0, 0, 0, 0, 0, false)
							}
							//无人机开机
							//go pushOperationModuleMsg(dnest, 1, 4, 124, 0, 0, 0, 0, 0, true)

							//time.Sleep(time.Duration(10) * time.Second)

							// _tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[dnest].Charge))
							// fmt.Println("cmd41:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
							// if _UavState == 1 {
							// 	AH_mission_response(dnest, "开库成功，无人机开机成功", 200, "airport_open")
							// } else {
							// 	AH_mission_response(dnest, "开库成功,无人机飞机开机失败", 400, "airport_open")
							// }
							fmt.Println("sssssssg_HGStatusMap[dnest] : ", g_HGStatusMap[dnest])
							if g_HGStatusMap[dnest] == STATUS_HG_OPENING {
								_tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[dnest].Charge))
								fmt.Println("cmd41:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
								AH_mission_response(dnest, "开库成功", 200, "airport_open")
								g_HGStatusMap[dnest] = STATUS_HG_OPEN
								return
							} else if g_HGStatusMap[dnest] == STATUS_UAV_RETURN {
								if g_HGheartMap[dnest].Posbar == 1 {
									AH_mission_response(dnest, "开库成功", 200, "airport_open")
								}
								return
							}
							//AH_mission_response(dnest, "开库成功，无人机开机成功", 200, "airport_open")
							//AH_mission_response(dnest, "开库成功", 200, "airport_open")
							//return
						} else {
							if timeOut >= 150 {
								AH_mission_response(dnest, "开库超时", 400, "airport_open")
								return
							}
						}
						timeOut++
					}
				}
				return
			case 42:
				{
					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						if g_HGheartMap[dnest].Canopy == 1 {
							AH_mission_response(dnest, "关库成功", 200, "airport_close")
							g_HGStatusMap[dnest] = STATUS_CLOSEED
							return
						} else {
							if timeOut >= 150 {
								AH_mission_response(dnest, "关库超时", 400, "airport_close")

								return
							}
						}
						timeOut++
					}
				}
				return
			case 81: //归中杆收
				{
					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						if g_HGheartMap[dnest].Posbar == 2 {
							go pushOperationModule(dnest, 1, 4, 125, 0, 0, 0, 0, 0, false)
							AH_mission_response(dnest, "充电杆收紧,无人机关机", 200, "chargebar_close")
							return
						} else {
							if timeOut >= 150 {
								AH_mission_response(dnest, "充电杆收紧超时", 400, "chargebar_close")
								return
							}
						}
						timeOut++
					}
				}
				return
			case 82: //归中杆放
				{
					//fmt.Println("ccccccccccccccccccccccccccccccccccccccccccccccccccccccccc82:")
					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						if g_HGheartMap[dnest].Posbar == 1 {

							AH_mission_response(dnest, "充电杆放开", 200, "chargebar_open")
							return
						} else {
							if timeOut >= 150 {
								AH_mission_response(dnest, "充电杆放开超时", 400, "chargebar_open")
								return
							}
						}
						timeOut++
					}
				}
				return
			case 121: //开始充电
				{
					if g_HGheartMap[dnest].Posbar != 2 {
						AH_mission_response(dnest, "充电杆未收紧", 400, "")
						return
					}

					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						_tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[dnest].Charge))
						fmt.Println("cmd121:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
						switch _tps {
						case UNCHARGED0:
							{
								//AH_mission_response(dnest, _chargeStr, 400, "")
								//return
							}
							//fallthrough
						case UNCHARGED1:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
							//fallthrough
						case UNCHARGED2:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
							//fallthrough
						case UNCHARGED3:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
							//fallthrough
						case UNCHARGED4:
							{
								AH_mission_response(dnest, _chargeStr, 200, "")
								g_HGStatusMap[dnest] = STATUS_UAV_CHARGING
								//return
							}
							//fallthrough
						case UNCHARGED5:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
							//fallthrough
						case UNCHARGED6:
							{
								AH_mission_response(dnest, _chargeStr, 200, "")
								g_HGStatusMap[dnest] = STATUS_UAV_CHARGING
								//return
							}
							//fallthrough
						case UNCHARGED7:
							{
								AH_mission_response(dnest, _chargeStr, 200, "")
								g_HGStatusMap[dnest] = STATUS_UAV_CHARGING_DONE
							}
							//fallthrough
						case UNCHARGED8:
							{
								AH_mission_response(dnest, _chargeStr, 400, "")
								g_HGStatusMap[dnest] = STATUS_UAV_CHARGING_ABNORMAL
							}
							//fallthrough
						default:
							{
								if timeOut >= 150 {
									AH_mission_response(dnest, "充电失败", 400, "")
									g_HGStatusMap[dnest] = STATUS_UAV_CHARGING_ABNORMAL
									return
								}
							}
						}
						// if g_HGheartMap[dnest].Charge == 64 {

						// 	AH_mission_response(dnest, "开始充电", 200, "")
						// 	return
						// } else {
						// 	if timeOut >= 150 {
						// 		AH_mission_response(dnest, "充电失败", 400, "")
						// 		return
						// 	}
						// }
						timeOut++
					}
				}
				return
			case 123: //停止充电
				{
					timeOut := 0
					timer := time.NewTicker(1 * time.Second)
					for range timer.C {
						_tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[dnest].Charge))
						fmt.Println("cmd123:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
						switch _tps {
						case UNCHARGED0:
							{
								AH_mission_response(dnest, _chargeStr, 200, "")
								return
							}
						case UNCHARGED1:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
						case UNCHARGED2:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
						case UNCHARGED3:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
						case UNCHARGED4:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
								//return
							}
						case UNCHARGED5:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
						case UNCHARGED6:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
								//return
							}
						case UNCHARGED7:
							{
								//AH_mission_response(dnest, _chargeStr, 200, "")
							}
						case UNCHARGED8:
							{
								AH_mission_response(dnest, _chargeStr, 400, "")
							}
						default:
							{
								if timeOut >= 150 {
									AH_mission_response(dnest, "停止充电失败", 400, "")
									return
								}
							}
						}
						// if g_HGheartMap[dnest].Charge == 64 {

						// 	AH_mission_response(dnest, "开始充电", 200, "")
						// 	return
						// } else {
						// 	if timeOut >= 150 {
						// 		AH_mission_response(dnest, "充电失败", 400, "")
						// 		return
						// 	}
						// }
						timeOut++
					}
				}
				return
			case 124: //无人机开机
				{

				}
				return
			case 125: //无人机关机
				{
					_tps, _chargeStr, _powerState, _UavState := convert_charge(uint8(g_HGheartMap[dnest].Charge))
					fmt.Println("cmd41:::::::::::::::::::::_tps:", _tps, "_chargeStr:", _chargeStr, "_powerState:", _powerState, "_UavState:", _UavState)
					// if _UavState == 2 {
					// 	//AH_mission_response(dnest, "开库成功，无人机开机成功", 200, "airport_open")
					// 	go pushOperationModuleMsg(dnest, 1, 2, 42, 0, 0, 0, 0, 0, true) //关库
					// } else {
					// 	AH_mission_response(dnest, "无人机关机失败", 400, "airport_close")
					// }
					AH_mission_response(dnest, "无人机关机成功", 200, "airport_close")
				}
			}

		}
	}
}

func pushOperationModule(dnest string, sid int, rid int, cmd int, p1 int, p2 int, p3 int, p4 int, p5 int, bMission bool) {

	//OMdata := k3MsgData{sid, rid, cmd, p1, p2, p3, p4, p5}
	OMMsg := k3Msg{8, k3MsgData{sid, rid, cmd, p1, p2, p3, p4, p5}}

	fmt.Println(OMMsg)
	jsonStr, err := json.Marshal(OMMsg)
	if err != nil {
		log.Fatal()
	}

	topiclinkrequest := dnest + "/downlink/request"
	fmt.Println(topiclinkrequest)
	client.Publish(topiclinkrequest, 0, false, jsonStr)
}

//设备信息指令
func pushOperationModuleMsg(dnest string, sid int, rid int, cmd int, p1 int, p2 int, p3 int, p4 int, p5 int, bMission bool) {

	//OMdata := k3MsgData{sid, rid, cmd, p1, p2, p3, p4, p5}
	OMMsg := k3Msg{8, k3MsgData{sid, rid, cmd, p1, p2, p3, p4, p5}}

	fmt.Println(OMMsg)
	jsonStr, err := json.Marshal(OMMsg)
	if err != nil {
		log.Fatal()
	}

	topiclinkrequest := dnest + "/downlink/request"
	fmt.Println(topiclinkrequest)
	client.Publish(topiclinkrequest, 0, false, jsonStr)
	//time.Sleep(time.Duration(3) * time.Second)

	if bMission == true {
		go MissionRequest(dnest, cmd, bMission)
	}
	//go MissionRequest(dnest, cmd, bMission)
	// switch bMission {
	// case false:
	// 	{

	// 	}
	// case true:
	// 	{
	// 		switch cmd {
	// 		case 41:
	// 			{
	// 				timeOut := 0
	// 				timer := time.NewTicker(1 * time.Second)
	// 				for range timer.C {
	// 					if g_HGheartMap[dnest].Canopy == 2 {

	// 						AH_mission_response(dnest, "开库成功", 200, "airport_open")
	// 						return
	// 					} else {
	// 						if timeOut >= 150 {
	// 							AH_mission_response(dnest, "开库超时", 400, "airport_open")
	// 							return
	// 						}
	// 					}
	// 					timeOut++
	// 				}
	// 			}
	// 		case 42:
	// 			{
	// 				timeOut := 0
	// 				timer := time.NewTicker(1 * time.Second)
	// 				for range timer.C {
	// 					if g_HGheartMap[dnest].Canopy == 1 {

	// 						AH_mission_response(dnest, "关库成功", 200, "airport_close")
	// 						return
	// 					} else {
	// 						if timeOut >= 150 {
	// 							AH_mission_response(dnest, "关库超时", 400, "airport_close")
	// 							return
	// 						}
	// 					}
	// 					timeOut++
	// 				}
	// 			}
	// 		case 81: //归中杆收
	// 			{
	// 				timeOut := 0
	// 				timer := time.NewTicker(1 * time.Second)
	// 				for range timer.C {
	// 					if g_HGheartMap[dnest].Posbar == 2 {

	// 						AH_mission_response(dnest, "充电杆收紧", 200, "")
	// 						return
	// 					} else {
	// 						if timeOut >= 150 {
	// 							AH_mission_response(dnest, "充电杆收紧超时", 400, "")
	// 							return
	// 						}
	// 					}
	// 					timeOut++
	// 				}
	// 			}
	// 		case 82: //归中杆放
	// 			{
	// 				timeOut := 0
	// 				timer := time.NewTicker(1 * time.Second)
	// 				for range timer.C {
	// 					if g_HGheartMap[dnest].Posbar == 1 {

	// 						AH_mission_response(dnest, "充电杆放开", 200, "")
	// 						return
	// 					} else {
	// 						if timeOut >= 150 {
	// 							AH_mission_response(dnest, "充电杆放开超时", 400, "")
	// 							return
	// 						}
	// 					}
	// 					timeOut++
	// 				}
	// 			}
	// 		case 121: //开始充电
	// 			{
	// 				// timeOut := 0
	// 				// timer := time.NewTicker(1 * time.Second)
	// 				// for range timer.C {
	// 				// 	if g_HGheartMap[dnest].Charge == 2 {

	// 				// 		AH_mission_response(dnest, "充电杆放开", 200, "")
	// 				// 		return
	// 				// 	} else {
	// 				// 		if timeOut >= 150 {
	// 				// 			AH_mission_response(dnest, "充电杆放开超时", 400, "")
	// 				// 			return
	// 				// 		}
	// 				// 	}
	// 				// 	timeOut++
	// 				// }
	// 			}
	// 		case 123: //停止充电
	// 			{

	// 			}
	// 		case 124: //无人机开机
	// 			{

	// 			}
	// 		case 125: //无人机关机
	// 			{

	// 			}
	// 		}

	// 	}
	// }

}

func AH_mission_response(dnest string, reqdata string, reqstatus int, Mtype string) {
	fmt.Println("AH_mission_response -------------------- ", Mtype)
	reqData := reqdata
	newInterface := interface{}(reqData).(string)

	for i := g_HGMissionList.Front(); i != nil; i = i.Next() {
		fmt.Println("AH_mission_response=========================", i.Value)
		M, ok := (i.Value).(HGMission)
		if ok {
			fmt.Println("AH_mission_response=========================ok", i.Value)
			if strings.Compare(dnest, M.DeviceID) == 0 && strings.Compare(Mtype, M.MType) == 0 {
				fmt.Println("dnest -------------------- ", dnest, "M.DeviceID--------------------", M.DeviceID)
				M.MData = newInterface
				M.State = reqstatus

				M.MissionTime = time.Now().Format("2006-01-02 15:04:05")

				jsonMissionStr, err := json.Marshal(M)
				if err != nil {
					log.Fatal()
				}

				client.Publish(topocAHresponse, 2, false, jsonMissionStr)

				g_HGMissionList.Remove(i)
			}
		}
	}

	// g_HGMissionMap[dnest].MData = newInterface
	// g_HGMissionMap[dnest].State = reqstatus

	// if strings.Compare(Mtype, g_HGMissionMap[dnest].MType) == 0 {
	// 	jsonMissionStr, err := json.Marshal(g_HGMissionMap[dnest])
	// 	if err != nil {
	// 		log.Fatal()
	// 	}

	// 	client.Publish(topocAHresponse, 0, false, jsonMissionStr)
	// }

}

//mqtt连接
func mqttConnect() {
	//配置
	clinetOptions := mqtt.NewClientOptions().AddBroker(broker)
	clinetOptions.SetClientID(ClientID)
	clinetOptions.SetConnectTimeout(time.Duration(60) * time.Second)
	//连接
	client = mqtt.NewClient(clinetOptions)
	//客户端连接判断
	if token := client.Connect(); token.WaitTimeout(time.Duration(60)*time.Second) && token.Wait() && token.Error() != nil {
		//panic(token.Error())
		fmt.Println(token.Error())

	}
}

//mqtt订阅
func mqttSubScribe(topic string, qos byte) {
	defer wg.Done()
	for {
		token := client.Subscribe(topic, qos, onMessage)
		token.Wait()
	}
}

//获取序列码
func DNESTPublish() {
	//defer wg.Done()
	//for {
	//timer := time.NewTicker(20 * time.Second)
	//for range timer.C {
	dnest := JsonMsg{15, data{"null"}}
	jsonStr, err := json.Marshal(dnest)
	if err != nil {
		log.Fatal()
	}
	client.Publish("DNEST/broadcast", 1, false, jsonStr)
	//time.Sleep(time.Duration(3) * time.Second)
	//}
	//}
}

type ChargedType int

const (
	UNCHARGED0 ChargedType = 0
	UNCHARGED1 ChargedType = 1
	UNCHARGED2 ChargedType = 2
	UNCHARGED3 ChargedType = 3
	UNCHARGED4 ChargedType = 4
	UNCHARGED5 ChargedType = 5
	UNCHARGED6 ChargedType = 6
	UNCHARGED7 ChargedType = 7
	UNCHARGED8 ChargedType = 8

// 	0x00：未充电
// 0x01：快充启动
// 0x02：慢充启动
// 0x03：快充初始化
// 0x04：快充中
// 0x05：慢充初始化
// 0x06：慢充中
// 0x07：充电完成
// 0x08：故障
)

func convert_charge(charge uint8) (ChargedType, string, uint8, uint8) {
	//a := Int64ToBytes(charge)

	validValue := charge & 0x000000FF
	low4BitValue := validValue & 0x0F
	var chargestate string
	// if low4BitValue == 0x00 {
	// 	chargestate = "未充电"
	// } else if low4BitValue == 0x01 {

	// }
	var tp ChargedType
	switch low4BitValue {
	case 0x00:
		{
			chargestate = "未充电"
			tp = UNCHARGED0
		}
	case 0x01:
		{
			chargestate = "快充启动"
			tp = UNCHARGED1
		}
	case 0x02:
		{
			chargestate = "慢充启动"
			tp = UNCHARGED2
		}
	case 0x03:
		{
			chargestate = "快充初始化"
			tp = UNCHARGED3
		}
	case 0x04:
		{
			chargestate = "快充中"
			tp = UNCHARGED4
		}
	case 0x05:
		{
			chargestate = "慢充初始化"
			tp = UNCHARGED5
		}
	case 0x06:
		{
			chargestate = "慢充中"
			tp = UNCHARGED6
		}
	case 0x07:
		{
			chargestate = "充电完成"
			tp = UNCHARGED7
		}
	case 0x08:
		{
			chargestate = "故障"
			tp = UNCHARGED8
		}
	}

	valueBit4_5 := (validValue >> 4) & 0x03 //
	valueBit6_7 := (validValue >> 6)

	return tp, chargestate, valueBit4_5, valueBit6_7
}
