package strategy

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"talib"
	"time"
)

//strategy包是策略模块集合 每一个go文件对应其中一种策略

//测试参数设定
const test_money_init = 100000 //初值账户资金

const Atr_PeriodTime = 20 //atr公式 周期取值
const K_buy = 0.7         //阀值
const K_sell = 0.7

const block_len = 24 * 3 //轨道区间的长度
//const block_len = 18 //轨道区间的长度
//const block_len = 18 //轨道区间的长度
const Open_time = 4 //开仓次数
//这里乘以90，表示第四个月开始测试。测试的时候 测试用的第一根一分钟k线在整个一分钟k线数组中的位置，即起步k线的位置

const K_start_Index = 30000 //2013/5/31/21:23
//const K_end_Index = 100000  //
//const K_start_Index = 45362 //2013/7/15/ 00:01
//const K_start_Index = 50000 //2013/8/2  6:07:00
//const K_start_Index = 449810 //2017/1/1/ 00:01
//const K_start_Index = 315790 //2016/7/20

//const K_start_Index = 329537 //2016/8/1
const K_end_Index = 277843 //2014-8-21

//const K_end_Index = 1048573 //2018/3/7  18:33:00

//const K_start_Index = 292138 //2016/7/19 0:00
//const K_end_Index = 530000 //2017-3-8

//const K_start_Index = 217200 //这个区间是初期测试里，亏损的最多的一段
//const K_end_Index = 292143   //

const leverage = 1 //杠杆倍数

var tarde_id = 1 //单子序号的初始值
//var slippage float64 = 0.0005         //滑点
var slippage float64 = 0              //滑点
var one_order_cut float64 = 0         //一张单，最大只能损失总资金的2%
var open_buy_time int = 0             //单一方向开单 最多开4次
var open_sell_time int = 0            //单一方向开单 最多开4次
var open_level float64 = 0.1          //单次开仓比例
var service_charge float64 = 0.001    //单次开仓手续费
var bservice_charge_sell bool = false //做空是否需要交手续费

var Sell_Order_Cut float64 = 0.01
var Buy_Order_Cut float64 = 0.01

var buy_stop_positon float64 = 0.05  //多单止损阀值
var Sell_stop_positon float64 = 0.05 //空单止损阀值

///////////////////////////////////参数设置 end///////
var bSell_Open = false
var bBuy_Open = false

var One_unit float64

//整个总k线的数据
var K_line_array_Sum_OneMin []K_line //所有k线的集合 1分钟的
var k_len_sum_min int                //所有一分钟k线的长度

var K_line_array_Sum []K_line //所有k线的集合 60分钟的
var K_len_Sum int             //k线总个数

var K_line_array_Sum_day []K_line //所有日k线的集合
var k_len_day int

var Open_array_Sum []float64 //所有k线的开盘价集合 下同 60分钟的
var High_array_Sum []float64
var Low_array_Sum []float64
var Close_array_Sum []float64

var k_60min_flag int //保存最后一次开单时，60分钟k线的位置

var open_time_sum int = 0      //开单总次数
var win_time_sum int = 0       //盈利次数
var win_money_sum float64 = 0  //总盈利
var lost_money_sum float64 = 0 //总亏损

var every_Money_Sum []Money_sum //记录每天的账户总资产

/////////////////////////////////////////
var Rang float64
var setting_value_buy float64
var setting_value_sell float64

var open_buy_sum int
var open_sell_sum int

var cost_sell float64 //最后一次开的卖单的成本价
var cost_buy float64  //同理

type K_line struct {
	date  string
	time  string
	open  float64
	high  float64
	low   float64
	close float64
	vol   float64

	last_in_day bool //所在这根k线是否是 当天的最后一根
}

var test_money_now float64  //账户实时总额（单子的价值加上可用保证金余额）
var test_money_left float64 //账上可用保证金

type trading_detail struct {
	id   int    //单子序号
	date string //开仓日期 （年月日加上时间几点几分）
	time string //只是时间 几点几分

	price_cost  float64 //开仓成本价
	price_close float64 //平仓价
	lots        float64 //开仓了多少
	cost_money  float64 //开仓花了多少保证金
	//get_money_close float64 //平仓得到了多少钱
	//left_money          float64 //开仓完毕之后剩下多少保证金
	//left_money_close    float64 //平仓完毕后账上的保证金数额
	//sum_money_now_close float64 //平仓完毕后账上的总资金
	trade_flag    int     //开仓方向 开多是0， 多头平仓是1，开空是2  空头平仓是3
	profit        float64 //平仓盈亏 只有在平仓的时候才计算
	sum_money_now float64 //开仓完毕后的总资金
}

type Money_sum struct {
	date      string  //当天的日期
	money_sum float64 //当天快到零点时候，账户总资产 快照
}

var trading_detail_arrary []trading_detail //交易明细集合 记录每一笔交易明细，多空都在里面

var Money_sum_array []Money_sum //记录每次平仓之后的账户总资金 用来统计资金曲线的

var trading_detail_buy_array []trading_detail  // 记录买入开仓的所有单子
var trading_detail_sell_array []trading_detail //记录卖出开仓的所有单子

func Get_Atr_Unit(High_array []float64, Low_array []float64, Close_array []float64, Atr_PeriodTime int) float64 {

	var N []float64
	N = talib.Atr(High_array, Low_array, Close_array, Atr_PeriodTime)

	return N[Atr_PeriodTime]

}

func Get_One_Unit_now(N float64) float64 {

	//var order_unit float64

	//var unit float64

	//unit = (test_account_money / 100) / N //获得单位
	//order_unit = unit * 1

	if test_money_now > 0 {

		return (test_money_now / 1000) / N //获取当前账户每一次开仓 所能够开的单位数量
	} else {

		return -1
	}

}

//计算某个一分钟k线 所在60分钟k的位置
func Calc_MinK_In_60Min_position(k_min string) int {

	var position int
	var strdateTarget_date string
	var iTarget_time int

	var pos_temp_space int
	var pos_temp_colon int

	var strResorce_date string

	pos_temp_space = strings.Index(k_min, " ")
	pos_temp_colon = strings.Index(k_min, ":")
	//得到 目标字符串  例如：”2013/1/24  4“
	strdateTarget_date = string([]byte(k_min)[:pos_temp_space])
	var sttemp string
	sttemp = string([]byte(k_min)[pos_temp_space:pos_temp_colon])
	sttemp = strings.Replace(sttemp, " ", "", -1)

	iTarget_time, err := strconv.Atoi(sttemp)
	if err != nil {
		//log.Fatal(err)
	}

	//查找目标字符串在 小时k的位置
	var k60_len int
	k60_len = len(K_line_array_Sum)
	for i := 0; i < k60_len; i++ {

		pos_temp_space = strings.Index(K_line_array_Sum[i].date, " ")
		pos_temp_colon = strings.Index(K_line_array_Sum[i].date, ":")

		strResorce_date = string([]byte(K_line_array_Sum[i].date)[:pos_temp_space])
		sttemp = string([]byte(K_line_array_Sum[i].date)[pos_temp_space:pos_temp_colon])
		sttemp = strings.Replace(sttemp, " ", "", -1)
		iResorce_time, _ := strconv.Atoi(sttemp)

		if 0 == strings.Compare(strdateTarget_date, strResorce_date) {

			if iTarget_time == iResorce_time {

				position = i
				break

			}

		}

	}

	return position
}

func Calc_OneUnit(k60_pos int) float64 {

	var unit float64

	var High_array []float64
	var Low_array []float64
	var Close_array []float64

	var N float64

	High_array = High_array_Sum[k60_pos-Atr_PeriodTime-1 : k60_pos]
	Low_array = Low_array_Sum[k60_pos-Atr_PeriodTime-1 : k60_pos]
	Close_array = Close_array_Sum[k60_pos-Atr_PeriodTime-1 : k60_pos]

	N = Get_Atr_Unit(High_array, Low_array, Close_array, Atr_PeriodTime)

	if test_money_left <= 0 {

		return -1

	} else {

		//unit = test_money_left / 100 / 10 / N / 2
		unit = test_money_left / 100 / 10 / N / 1

	}

	return unit
}

func TestKlineIsLast(K_line_array_Sum_OneMin K_line) {

}

//测试主函数
func Du_gg_test() {

	var HH float64
	var LC float64
	var HC float64
	var LL float64

	var strLog string

	var High_array []float64
	var Low_array []float64
	var Close_array []float64
	var price_now float64

	var Price_close_day float64
	var money_sum_day float64
	var orders_sum_day float64
	var money_sum_today Money_sum
	//var N float64
	//K_start_Index 从一分钟k的第1000根开始的
	//for K_Index := K_start_Index; K_Index < k_len_sum_min-1; K_Index++ {
	for K_Index := K_start_Index; K_Index < K_end_Index; K_Index++ {

		High_array = High_array[:0:0]
		Low_array = Low_array[:0:0]
		Close_array = Close_array[:0:0]

		var k60_pos int //当前1分钟k线在60分钟k的位置
		//////////////计算ATR 得到单位头寸
		var Lots_buy float64
		var Lots_sell float64

		//log
		if 1000100 == K_Index || 1010100 == K_Index || 1020100 == K_Index || 1030100 == K_Index || 1040100 == K_Index || 1048575 == K_Index {

			strLog = "---Calc_MinK_In_60Min_position  begin---60min_Index: " + strconv.Itoa(K_Index)
			fmt.Println(strLog)

		}
		//end

		k60_pos = Calc_MinK_In_60Min_position(K_line_array_Sum_OneMin[K_Index].date)
		//一分钟对应的60分钟k线不存在就说明数据有问题
		if k60_pos <= 0 {

			fmt.Println("---k60_pos 1 return---")
			var strRe string
			strRe = fmt.Sprintf("%d", K_Index)
			fmt.Println(strRe)
			fmt.Println(K_line_array_Sum_OneMin[K_Index].date)
			fmt.Println("---k60_pos 2 return---")
			//return
			continue
		}

		//////////////end

		//从当前k线向过去数 block_len长度,用来计算上下轨道
		/*
				以K_Index下标的k线当做现在时刻，从K_Index-2开始往前推block_len个k线来计算rang，
			K_Index-1下标的k的开盘价做open配合rang构成上下轨道。再以K_Index的开盘价跟上下轨道做
			比较

		*/

		High_array = High_array[:0:0]
		Low_array = Low_array[:0:0]
		Close_array = Close_array[:0:0]

		High_array = High_array_Sum[k60_pos-2-block_len : k60_pos-2]
		Low_array = Low_array_Sum[k60_pos-2-block_len : k60_pos-2]
		Close_array = Close_array_Sum[k60_pos-2-block_len : k60_pos-2]

		HH = ExtremumInArray_max(High_array[0:])
		HC = ExtremumInArray_max(Close_array[0:])
		LC = ExtremumInArray_min(Close_array[0:])
		LL = ExtremumInArray_min(Low_array[0:])

		if HH == 0.0 ||
			LC == 0.0 ||
			HC == 0.0 ||
			LL == 0.0 {

			fmt.Println("-HH == 0.0 || return---")
			return
		}

		Rang = math.Max((HH - LC), (HC - LL))

		var Price_Open float64

		Price_Open = K_line_array_Sum[k60_pos-1].open
		//K_line_array_Sum_OneMin[K_Index-1]

		setting_value_buy = (Price_Open + Rang*K_buy)
		setting_value_sell = (Price_Open - Rang*K_sell)

		//用60分钟k做区间计算，用一分钟k线模拟当成TICK数据， 一分钟的开盘价比作是盘中最细价格

		price_now = K_line_array_Sum_OneMin[K_Index].close

		//fmt.Println(" ---开始判断 多和空----！！！！")

		//开多
		if (price_now > setting_value_buy) && (open_buy_sum > 0) {

			//有空单 先平空单再开多
			if bSell_Open {

				fmt.Println(" 开多单被 激活,先平空单")
				CloseAllSell(K_Index, price_now)
				bSell_Open = false
				fmt.Println("空单全部平完")

			}
			//第一次开多 记录下开多的时间 只记录小时
			if false == bBuy_Open {

				//开多,开之前 计算一下单位头寸的规模
				Lots_buy = Calc_OneUnit(k60_pos)
				if -1 == Lots_buy {

					fmt.Println(" wocao, 资金为负数了！！")
					return
				}
				OpenBuy(K_Index, price_now+slippage*price_now, Lots_buy)
				fmt.Println(" 第一次开多成功！")
				k_60min_flag = k60_pos //保存开单时候，60分k线位置
				bBuy_Open = true
			} else if k60_pos > k_60min_flag {

				Lots_buy = Calc_OneUnit(k60_pos)
				if -1 == Lots_buy {

					fmt.Println(" wocao, 资金为负数了！！")
					return
				}
				OpenBuy(K_Index, price_now+slippage*price_now, Lots_buy)
				fmt.Println(" 再次开多成功！")
				k_60min_flag = k60_pos //保存开单时候，60分k线位置

			}

		} else if (price_now < setting_value_sell) && (open_sell_sum > 0) {

			////满足开空条件
			if bBuy_Open {

				fmt.Println(" 开空单被 激活,先平多单")
				//如果有多单 先平多单 再开空
				CloseAllBuy(K_Index, price_now)
				bBuy_Open = false
				fmt.Println("---多单全部平仓完毕---")
			}
			//第一次开空记录下开空的时间 只记录小时
			if false == bSell_Open {

				//开空
				Lots_sell = Calc_OneUnit(k60_pos)
				if -1 == Lots_sell {

					fmt.Println(" wocao, 资金为负数了！！")
					return
				}
				OpenSell(K_Index, price_now-slippage*price_now, Lots_sell)
				fmt.Println(" 第一次开空成功！")
				k_60min_flag = k60_pos //保存开单时候，60分k线位置
				bSell_Open = true

			} else if k60_pos > k_60min_flag {

				Lots_sell = Calc_OneUnit(k60_pos)
				if -1 == Lots_sell {

					fmt.Println(" wocao, 资金为负数了！！")
					return
				}
				OpenSell(K_Index, price_now-slippage*price_now, Lots_sell)
				fmt.Println(" 再次开空成功！")
				k_60min_flag = k60_pos //保存开单时候，60分k线位置

			}

		}

		//检查 绝对止损

		///////////////////
		//起码间隔1个小时再去判断是否需要止损

		if k60_pos >= k_60min_flag+1 {

			if true == bSell_Open {

				if 1 == Normal_StopSellJudge(cost_sell, price_now, Lots_sell) {

					CloseAllSell(K_Index, price_now)
					bSell_Open = false
					cost_sell = 0
					fmt.Println(" ---空单全部止损止盈---AAAA")

				}
			} else if true == bBuy_Open {

				if 1 == Normal_StopBuyJudge(cost_buy, price_now, Lots_buy) {

					CloseAllBuy(K_Index, price_now)
					bBuy_Open = false
					cost_buy = 0
					fmt.Println("****多单全部止损止盈*****BBB")

				}

			}

		}

		//检查k线是不是当天的最后一根， 是就要存当天的总资产 ,买和卖只可能一个有记录
		if true == K_line_array_Sum_OneMin[K_Index].last_in_day {

			Price_close_day = K_line_array_Sum_OneMin[K_Index].close
			var loop_buy int
			var loop_sell int
			loop_buy = len(trading_detail_buy_array)
			loop_sell = len(trading_detail_sell_array)
			orders_sum_day = 0
			if loop_buy > 0 {

				for i := 0; i < loop_buy; i++ {

					orders_sum_day += trading_detail_buy_array[i].lots * Price_close_day

				}
				//剩余可用资金 加上订单总价值 等于当天账户总资产
				money_sum_day = test_money_left + orders_sum_day

			} else if loop_sell > 0 {

				for i := 0; i < loop_sell; i++ {

					orders_sum_day = orders_sum_day + trading_detail_sell_array[i].lots*trading_detail_sell_array[i].price_cost + trading_detail_sell_array[i].lots*(trading_detail_sell_array[i].price_cost-Price_close_day)

				}

			} else { //什么单子都没有，就是空仓的

				orders_sum_day = 0
			}
			//剩余可用资金 加上订单总价值 等于当天账户总资产
			money_sum_day = test_money_left + orders_sum_day

			money_sum_today.date = K_line_array_Sum_OneMin[K_Index].date
			money_sum_today.money_sum = money_sum_day
			every_Money_Sum = append(every_Money_Sum, money_sum_today)

			money_sum_today.date = ""
			money_sum_today.money_sum = 0

		}

	}

	//保存交易记录
	fmt.Println("--------开始 保存交易记录------")
	WriteDetalToCsv()
	fmt.Println("--------交易记录 保存完毕------")

	//保存每日总资产
	fmt.Println("--------开始 保存每日总资产------")
	writeEverydayToCsv()
	fmt.Println("----每日总资产保存完毕------")

}

func Normal_StopBuyJudge(cost_buy_price float64, price_now float64, Lots_buy float64) int {

	if cost_buy_price == 0 {

		return 0

	}
	var diff float64
	var result float64
	//var open_cost float64
	//var order_count int

	if price_now < cost_buy_price {

		diff = cost_buy_price - price_now
		result = diff / cost_buy_price
		if result > buy_stop_positon {

			fmt.Println("多单单笔止损被激活")
			return 1
		}
		//单笔的亏损额度止损
		if diff*Lots_buy > Buy_Order_Cut*test_money_now {

			fmt.Println("多单 总额度止损被激活")
			return 1

		}

	}

	return 0

}

//判断空单是否要止损，这时候传进来的cost_sell_price价格是空单里成本最好的一个，
//如果这一单的亏损都超过了总资金的1%,则其他也必然超过了，所以全部止损平掉
func Normal_StopSellJudge(cost_sell_price float64, price_now float64, lots_sell float64) int {

	if cost_sell_price == 0 {

		return 0

	}
	var diff float64
	var result float64
	//var open_cost float64
	//var order_count int

	if price_now > cost_sell_price {

		diff = price_now - cost_sell_price
		result = diff / cost_sell_price
		if result > Sell_stop_positon {

			fmt.Println("空单单笔止损被激活")
			return 1
		}

		//单笔的亏损额度止损
		if diff*lots_sell > Sell_Order_Cut*test_money_now {

			fmt.Println("空单 总额度止损被激活")
			return 1

		}

	}

	return 0

}

// 保存总资金
func writeEverydayToCsv() {

	var strFileTest_result string

	timestamp := time.Now().Unix()

	strFileTest_result = strconv.FormatInt(timestamp, 10)
	strFileTest_result += "_everyday.csv"
	strFileTest_result = "D:\\GoPath\\src\\re_doc\\" + strFileTest_result

	f, err := os.Create(strFileTest_result)
	if err != nil {
		panic(err)
	}

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f) //创建一个新的写入文件流
	data_title := [][]string{
		{"序号", "日期", "每日资金总额"},
	}
	w.WriteAll(data_title) //写入数据
	w.Flush()

	var strTemp string
	var data []string
	var loops int

	loops = len(every_Money_Sum)

	for i := 0; i < loops; i++ {

		strTemp = strconv.Itoa(i + 1)
		data = append(data, strTemp)

		strTemp = every_Money_Sum[i].date
		data = append(data, strTemp)

		strTemp = fmt.Sprintf("%.2f", every_Money_Sum[i].money_sum)
		data = append(data, strTemp)

		w.Write(data) //写入数据
		w.Flush()
		data = data[:0:0]

	}
}

//将交易记录写入csv文件
func WriteDetalToCsv() {

	var strFileDetail string

	timestamp := time.Now().Unix()
	strFileDetail = strconv.FormatInt(timestamp, 10)
	strFileDetail += "_tradedetail.csv"
	strFileDetail = "D:\\GoPath\\src\\re_doc\\" + strFileDetail

	///////保存交易明细
	f, err := os.Create(strFileDetail)
	if err != nil {
		panic(err)
	}

	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	w := csv.NewWriter(f) //创建一个新的写入文件流
	data_title := [][]string{
		{"序号", "时间", "类型", "订单号", "数量", "价格", "盈亏", "余额"},
	}
	w.WriteAll(data_title) //写入数据
	w.Flush()

	var strTemp string
	var data []string
	var loops int
	var trade_flag int
	loops = len(trading_detail_arrary)

	for i := 0; i < loops; i++ {

		strTemp = strconv.Itoa(i + 1)
		data = append(data, strTemp)

		strTemp = trading_detail_arrary[i].date
		data = append(data, strTemp)

		trade_flag = trading_detail_arrary[i].trade_flag
		//////
		if 0 == trade_flag {

			data = append(data, "多开")

		} else if 1 == trade_flag {

			data = append(data, "多平")

		} else if 2 == trade_flag {

			data = append(data, "空开")

		} else if 3 == trade_flag {

			data = append(data, "空平")

		}
		////////

		strTemp = strconv.Itoa(trading_detail_arrary[i].id)
		data = append(data, strTemp)

		///////////
		strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].lots)
		data = append(data, strTemp)

		if 0 == trade_flag || 2 == trade_flag {

			//strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].lots)
			//data = append(data, strTemp)

			strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].price_cost)
			data = append(data, strTemp)

		}
		if 1 == trade_flag || 3 == trade_flag {

			//strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].lots)
			//data = append(data, strTemp)

			strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].price_close)
			data = append(data, strTemp)

		}

		strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].profit)
		data = append(data, strTemp)

		strTemp = fmt.Sprintf("%.2f", trading_detail_arrary[i].sum_money_now)
		data = append(data, strTemp)

		w.Write(data) //写入数据
		w.Flush()
		data = data[:0:0]

	}

	strTemp = fmt.Sprintf("开仓总次数:%d, ", open_time_sum)
	//fmt.Println(strTemp)
	//strTemp = ""

	strTemp = strTemp + fmt.Sprintf("盈利总次数:%d, ", win_time_sum)
	//fmt.Println(strTemp)
	//strTemp = ""

	strTemp = strTemp + fmt.Sprintf("胜率: %.2f, ", float64(win_time_sum)/float64(open_time_sum))
	//fmt.Println(strTemp)
	//strTemp = ""

	var lost_time int
	lost_time = open_time_sum - win_time_sum
	strTemp = strTemp + fmt.Sprintf("盈亏比是: %.2f\n", (win_money_sum/float64(win_time_sum))/(math.Abs(lost_money_sum)/float64(lost_time)))
	fmt.Println(strTemp)
	//strTemp = ""

	//strTemp
	data = append(data, strTemp)

	w.Write(data) //写入数据
	w.Flush()
	data = data[:0:0]

}

//初始化函数
func Du_Init() {

	////////---初始化测试参数----////////////////////////////
	open_buy_sum = Open_time
	open_sell_sum = Open_time

	test_money_now = test_money_init
	test_money_left = test_money_init

	/////////////////////end

	ReadTickData()

	ReadDayData()

	////-----读取csv，准备回测用的k线数据---//////////////////////////////
	//my_Kline := []K_line{}

	fileName := "D:\\good_doc\\btc_usd_hour_mh.csv"
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	var K_line_temp K_line
	var strTemp string
	defer file.Close()
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//读取 每一行数据 然后分离

		K_line_temp.date = record[0]
		//K_line_temp.time = string.Split(record[1], " ")
		strTemp = record[1]
		K_line_temp.open, err = strconv.ParseFloat(strTemp, 64)
		Open_array_Sum = append(Open_array_Sum, K_line_temp.open)

		strTemp = record[2]
		K_line_temp.high, err = strconv.ParseFloat(strTemp, 64)
		High_array_Sum = append(High_array_Sum, K_line_temp.high)

		strTemp = record[3]
		K_line_temp.low, err = strconv.ParseFloat(strTemp, 64)
		Low_array_Sum = append(Low_array_Sum, K_line_temp.low)

		strTemp = record[4]
		K_line_temp.close, err = strconv.ParseFloat(strTemp, 64)
		Close_array_Sum = append(Close_array_Sum, K_line_temp.close)

		strTemp = record[5]
		K_line_temp.vol, err = strconv.ParseFloat(strTemp, 64)

		K_line_array_Sum = append(K_line_array_Sum, K_line_temp)
		K_len_Sum++

		K_line_temp.date = ""
		K_line_temp.time = ""
		K_line_temp.open = 0
		K_line_temp.high = 0
		K_line_temp.low = 0
		K_line_temp.close = 0
		K_line_temp.vol = 0

		//f, err := strconv.ParseFloat("3.1415", 64)
		//bb, err = strconv.ParseFloat("33.444", 64)

		//fmt.Println(aa)
		//split_line := strings.Split(record, ",")
		//fmt.Println(record) // record has the type []string
	}

	////test

	//Calc_MinK_In_60Min_position("2013/1/21  18:33:00")
	///end

}

//这里把一分钟k线当成是 tikc数据
func ReadTickData() {

	////////-----读一分钟的k线数据文件csv，当成是 TICK数据来用---//////////////////////////////////////

	//K_line_array_Sum_OneMin
	var lenth int
	var pos_prev int
	var pos_next int
	//var strspace string
	var strDate_prev string
	var strDate_next string
	fileName := "D:\\good_doc\\btc_usd.csv"
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	var K_line_temp K_line
	var strTemp string
	defer file.Close()
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//读取 每一行数据 然后分离

		K_line_temp.date = record[0]
		//K_line_temp.time = string.Split(record[1], " ")
		strTemp = record[1]
		K_line_temp.open, err = strconv.ParseFloat(strTemp, 64)
		//Open_array_Sum = append(Open_array_Sum, K_line_temp.open)

		strTemp = record[2]
		K_line_temp.high, err = strconv.ParseFloat(strTemp, 64)
		//High_array_Sum = append(High_array_Sum, K_line_temp.high)

		strTemp = record[3]
		K_line_temp.low, err = strconv.ParseFloat(strTemp, 64)
		//Low_array_Sum = append(Low_array_Sum, K_line_temp.low)

		strTemp = record[4]
		K_line_temp.close, err = strconv.ParseFloat(strTemp, 64)
		//Close_array_Sum = append(Close_array_Sum, K_line_temp.close)

		strTemp = record[5]
		K_line_temp.vol, err = strconv.ParseFloat(strTemp, 64)

		K_line_array_Sum_OneMin = append(K_line_array_Sum_OneMin, K_line_temp)
		k_len_sum_min++

		K_line_temp.date = ""
		K_line_temp.time = ""
		K_line_temp.open = 0
		K_line_temp.high = 0
		K_line_temp.low = 0
		K_line_temp.close = 0
		K_line_temp.vol = 0

		//判断这根一分钟k线是否是当天最后一根k线，在K_line_array_Sum_OneMin里 比较相邻的二根k线， last_in_day

		lenth = len(K_line_array_Sum_OneMin)
		if lenth >= 2 {

			pos_prev = strings.Index(K_line_array_Sum_OneMin[lenth-2].date, " ")
			pos_next = strings.Index(K_line_array_Sum_OneMin[lenth-1].date, " ")
			if pos_prev > 0 && pos_next > 0 {
				strDate_next = K_line_array_Sum_OneMin[lenth-1].date[0:pos_next]
				strDate_prev = K_line_array_Sum_OneMin[lenth-2].date[0:pos_prev]
				if strDate_prev != strDate_next {

					K_line_array_Sum_OneMin[lenth-2].last_in_day = true

				}

			}

		}

	}

	///////////////////
}

//读每天的日线数据，只读取 日期 和 当天收盘价
func ReadDayData() {

	////////-----读一分钟的k线数据文件csv，当成是 TICK数据来用---//////////////////////////////////////

	//K_line_array_Sum_OneMin

	fileName := "D:\\good_doc\\btcusd_day_day_2.csv"
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	var K_line_temp K_line
	var strTemp string
	defer file.Close()
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//读取 每一行数据 然后分离

		K_line_temp.date = record[0]

		strTemp = record[4]
		K_line_temp.close, err = strconv.ParseFloat(strTemp, 64)

		K_line_array_Sum_day = append(K_line_array_Sum_day, K_line_temp)
		k_len_day++

		K_line_temp.date = ""
		K_line_temp.time = ""
		K_line_temp.open = 0
		K_line_temp.high = 0
		K_line_temp.low = 0
		K_line_temp.close = 0
		K_line_temp.vol = 0

	}

}

func ExtremumInArray_min(array []float64) float64 {
	if len(array) < 1 {
		return 0
	}
	min := array[0]
	max := array[0]
	for _, v := range array {
		if v < min {
			min = v
		} else if v > max {
			max = v
		}
	}
	return min
}

func ExtremumInArray_max(array []float64) float64 {
	if len(array) < 1 {
		return 0
	}
	min := array[0]
	max := array[0]
	for _, v := range array {
		if v < min {
			min = v
		} else if v > max {
			max = v
		}
	}
	return max
}

//平掉所有的空单
func CloseAllSell(k_index_min int, price float64) {

	//平掉所有的
	var array_len int
	var td trading_detail
	//var money_sum_temp float64
	//var cost_price float64
	var service_charge_money float64
	array_len = len(trading_detail_sell_array)
	if array_len > 0 {

		//有n张单，循环平，平一次记录一次
		for i := array_len; i > 0; i-- {

			td.date = K_line_array_Sum_OneMin[k_index_min].date
			td.time = K_line_array_Sum_OneMin[k_index_min].time
			td.id = trading_detail_sell_array[i-1].id
			td.price_close = price
			td.lots = trading_detail_sell_array[i-1].lots

			service_charge_money = td.price_close * td.lots * service_charge //计算平仓手续费

			td.trade_flag = 3

			td.profit = td.lots * (trading_detail_sell_array[i-1].price_cost - price)

			open_time_sum++
			if td.profit > 0 {

				win_money_sum += td.profit
				win_time_sum++
			} else {

				lost_money_sum += td.profit
			}

			//平仓后的总资金 = 剩余资金+投入的成本 - 平仓的盈亏- 手续费
			//money_sum_temp = test_money_left + cost_price*td.lots + td.profit - service_charge_money

			//有些规则，做空不需要手续费
			if true == bservice_charge_sell {

				test_money_now = test_money_now + td.profit - service_charge_money

			} else {

				test_money_now = test_money_now + td.profit
			}
			//test_money_now = test_money_now + td.profit - service_charge_money
			/////

			//Money_sum_array = append(Money_sum_array, test_money_now)
			td.sum_money_now = test_money_now
			trading_detail_arrary = append(trading_detail_arrary, td)

			///清零
			td.id = 0
			td.date = "" //开仓日期 （年月日加上时间几点几分）
			td.time = "" //只是时间 几点几分

			td.price_cost = 0  //开仓成本价
			td.price_close = 0 //平仓价
			td.lots = 0        //开仓了多少
			td.cost_money = 0  //开仓花了多少保证金

			td.trade_flag = 0    //开仓方向 开多是0， 多头平仓是1，开空是2  空头平仓是3
			td.profit = 0        //平仓盈亏 只有在平仓的时候才计算
			td.sum_money_now = 0 //开仓完毕后的总资金

		}

		//平仓完毕 切片全部清零
		trading_detail_sell_array = trading_detail_sell_array[:0:0]

		//这个时候账上总资金就等于总可用余额，因为没有单子了，全是现金
		test_money_left = test_money_now

		open_sell_sum = Open_time

	}

	/*
		//平掉所有的空单
		var array_len int
		var td trading_detail
		var money_sum_temp float64
		array_len = len(trading_detail_sell_array)
		if array_len > 0 {

			//有n张空单，循环平，平一次记录一次
			for i := array_len; i > 0; i++ {

				td.date = K_line_array_Sum[k_index].date
				td.time = K_line_array_Sum[k_index].time
				td.id = trading_detail_sell_array[i].id
				td.price_close = price
				td.lots = trading_detail_sell_array[i].lots
				td.get_money_close = td.price_close * td.lots
				money_sum_temp += td.get_money_close

				td.trade_flag = 3
				td.profit = td.lots * (trading_detail_sell_array[i].price_cost - price)

				//td.left_money_close = test_money_left + td.get_money_close
				//td.sum_money_now_close = td.left_money_close

				//test_money_left = td.left_money_close
				//test_money_now = td.left_money_close

				trading_detail_arrary = append(trading_detail_arrary, td)

			}

			//平仓完毕 空单切片全部清零
			trading_detail_sell_array = trading_detail_sell_array[:0:0]

			//剩下的保证金加上平仓得到的所有的资金 等于平仓之后的总可用余额
			test_money_left = test_money_left + money_sum_temp
			//这个时候账上总资金就等于总可用余额，因为没有单子了，全是现金
			test_money_now = test_money_left

			//平仓后 记录这时刻 账户总资金
			Money_sum_array = append(Money_sum_array, test_money_now)

		}
	*/

}

//平所有的多单
func CloseAllBuy(k_index_min int, price float64) {

	//平掉所有的
	var array_len int
	var td trading_detail
	//var money_sum_temp float64
	//var cost_price float64
	var service_charge_money float64
	array_len = len(trading_detail_buy_array)
	if array_len > 0 {

		//有n张多单，循环平，平一次记录一次
		for i := array_len; i > 0; i-- {

			td.date = K_line_array_Sum_OneMin[k_index_min].date
			td.time = K_line_array_Sum_OneMin[k_index_min].time
			td.id = trading_detail_buy_array[i-1].id
			td.price_close = price
			td.lots = trading_detail_buy_array[i-1].lots
			//td.get_money_close = td.price_close * td.lots
			service_charge_money = td.price_close * td.lots * service_charge //计算平仓手续费

			td.trade_flag = 1

			td.profit = td.lots * (price - trading_detail_buy_array[i-1].price_cost)

			open_time_sum++
			if td.profit > 0 {

				win_money_sum += td.profit
				win_time_sum++
			} else {

				lost_money_sum += td.profit
			}

			//平仓后的总资金 = 剩余资金+投入的成本 - 平仓的盈亏- 手续费
			//money_sum_temp = test_money_left + cost_price*td.lots + td.profit - service_charge_money
			test_money_now = test_money_now + td.profit - service_charge_money
			//Money_sum_array = append(Money_sum_array, test_money_now)
			td.sum_money_now = test_money_now
			trading_detail_arrary = append(trading_detail_arrary, td)

			///清零
			td.id = 0
			td.date = "" //开仓日期 （年月日加上时间几点几分）
			td.time = "" //只是时间 几点几分

			td.price_cost = 0  //开仓成本价
			td.price_close = 0 //平仓价
			td.lots = 0        //开仓了多少
			td.cost_money = 0  //开仓花了多少保证金

			td.trade_flag = 0    //开仓方向 开多是0， 多头平仓是1，开空是2  空头平仓是3
			td.profit = 0        //平仓盈亏 只有在平仓的时候才计算
			td.sum_money_now = 0 //开仓完毕后的总资金

		}

		//平仓完毕 切片全部清零
		trading_detail_buy_array = trading_detail_buy_array[:0:0]

		//这个时候账上总资金就等于总可用余额，因为没有单子了，全是现金
		test_money_left = test_money_now

	}
	open_buy_sum = Open_time

}

//开多 ，price 开仓价格，此价格已经包含了滑点
func OpenBuy(k_index_min int, price float64, Lots_buy float64) {

	//买多
	//println("AAA")
	var td_temp trading_detail
	var service_charge_money float64
	td_temp.date = K_line_array_Sum_OneMin[k_index_min].date
	td_temp.time = K_line_array_Sum_OneMin[k_index_min].time
	td_temp.price_cost = price
	td_temp.lots = Lots_buy
	td_temp.cost_money = td_temp.price_cost * td_temp.lots / leverage

	//开一笔单位头寸需要的资金比 总可用资金还要大，就放弃
	if td_temp.cost_money > test_money_left {

		return
	}

	service_charge_money = td_temp.cost_money * service_charge
	test_money_left = test_money_left - td_temp.cost_money - service_charge_money
	//td_temp.left_money = test_money_left
	//td_temp.sum_money_now = td_temp.cost_money + td_temp.left_money
	//test_money_now = test_money_left + td_temp.cost_money

	td_temp.trade_flag = 0
	td_temp.id = tarde_id
	tarde_id = tarde_id + 1

	trading_detail_buy_array = append(trading_detail_buy_array, td_temp)
	trading_detail_arrary = append(trading_detail_arrary, td_temp)

	///清零
	td_temp.id = 0
	td_temp.date = "" //开仓日期 （年月日加上时间几点几分）
	td_temp.time = "" //只是时间 几点几分

	td_temp.price_cost = 0  //开仓成本价
	td_temp.price_close = 0 //平仓价
	td_temp.lots = 0        //开仓了多少
	td_temp.cost_money = 0  //开仓花了多少保证金

	td_temp.trade_flag = 0    //开仓方向 开多是0， 多头平仓是1，开空是2  空头平仓是3
	td_temp.profit = 0        //平仓盈亏 只有在平仓的时候才计算
	td_temp.sum_money_now = 0 //开仓完毕后的总资金

	open_buy_sum--

	cost_buy = price

}

//开空 ，price 开仓价格，此价格已经包含了滑点
func OpenSell(k_index_min int, price float64, Lots_sell float64) {

	var td_temp trading_detail
	var service_charge_money float64

	td_temp.date = K_line_array_Sum_OneMin[k_index_min].date
	td_temp.time = K_line_array_Sum_OneMin[k_index_min].time
	td_temp.price_cost = price
	td_temp.lots = Lots_sell
	td_temp.cost_money = td_temp.price_cost * td_temp.lots / leverage

	//开一笔单位头寸需要的资金比 总可用资金还要大，就放弃
	if td_temp.cost_money > test_money_left {

		return
	}

	service_charge_money = td_temp.cost_money * service_charge

	//有些规则，做空不需要手续费
	if true == bservice_charge_sell {

		test_money_left = test_money_left - td_temp.cost_money - service_charge_money

	} else {

		test_money_left = test_money_left - td_temp.cost_money
	}

	//td_temp.left_money = test_money_left
	//td_temp.sum_money_now = td_temp.left_money + td_temp.cost_money
	//test_money_now = test_money_left + td_temp.cost_money

	td_temp.trade_flag = 2
	td_temp.id = tarde_id
	tarde_id = tarde_id + 1

	trading_detail_sell_array = append(trading_detail_sell_array, td_temp)
	trading_detail_arrary = append(trading_detail_arrary, td_temp)

	///清零
	td_temp.id = 0
	td_temp.date = "" //开仓日期 （年月日加上时间几点几分）
	td_temp.time = "" //只是时间 几点几分

	td_temp.price_cost = 0  //开仓成本价
	td_temp.price_close = 0 //平仓价
	td_temp.lots = 0        //开仓了多少
	td_temp.cost_money = 0  //开仓花了多少保证金

	td_temp.trade_flag = 0    //开仓方向 开多是0， 多头平仓是1，开空是2  空头平仓是3
	td_temp.profit = 0        //平仓盈亏 只有在平仓的时候才计算
	td_temp.sum_money_now = 0 //开仓完毕后的总资金

	open_sell_sum--

	cost_sell = price
}
