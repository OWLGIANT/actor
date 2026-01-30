// save 保存行情到文件 供回测系统使用
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"actor"

	"actor/helper"
)

func main() {
	// trademsg
	tradeMsg := helper.TradeMsg{}
	refMsg := helper.TradeMsg{}

	// 创建ws
	// p, _ := helper.StringPairToPair("btc_usdt")
	_, ws := beastquant.GetClient(beastquant.ClientTypeWs,
		"binance_usdt_swap",
		helper.BrokerConfig{},
		&tradeMsg,
		helper.CallbackFunc{OnTicker: func(ts int64) {}})
	ws.Run()
	// 创建refws
	_, refws := beastquant.GetClient(beastquant.ClientTypeWs,
		"coinex_usdt_swap",
		helper.BrokerConfig{},
		&refMsg,
		helper.CallbackFunc{OnTicker: func(ts int64) {
		}})
	refws.Run()

	// 创建writer
	file, _ := os.Create("history.csv")
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Write([]string{"ts", "refbp", "refbq", "refap", "refaq", "bp", "ap", "maxFill", "minFill", "buyNum", "sellNum", "buyQ", "sellQ", "buyV", "sellV"})

	isReady := false
	for {
		if isReady {
			time.Sleep(time.Millisecond * 100)
			//
			_s := make([]string, 0)
			_s = append(_s, fmt.Sprint(time.Now().UnixMilli()))
			_s = append(_s, fmt.Sprint(refMsg.Ticker.Bp))
			_s = append(_s, fmt.Sprint(refMsg.Ticker.Bq))
			_s = append(_s, fmt.Sprint(refMsg.Ticker.Ap))
			_s = append(_s, fmt.Sprint(refMsg.Ticker.Aq))
			_s = append(_s, fmt.Sprint(tradeMsg.Ticker.Bp))
			_s = append(_s, fmt.Sprint(tradeMsg.Ticker.Ap))
			maxFill, minFill, buyNum, sellNum, buyQ, sellQ, buyV, sellV := tradeMsg.Trade.Get()
			_s = append(_s, fmt.Sprint(maxFill))
			_s = append(_s, fmt.Sprint(minFill))
			_s = append(_s, fmt.Sprint(buyNum))
			_s = append(_s, fmt.Sprint(sellNum))
			_s = append(_s, fmt.Sprint(buyQ))
			_s = append(_s, fmt.Sprint(sellQ))
			_s = append(_s, fmt.Sprint(buyV))
			_s = append(_s, fmt.Sprint(sellV))
			writer.Write(_s)
			writer.Flush()
			fmt.Println(time.Now().String())
		} else {
			if tradeMsg.Ticker.Mp.Load() > 0 && refMsg.Ticker.Mp.Load() > 0 {
				isReady = true
				fmt.Println("开始记录行情")
			}
		}
	}
}
