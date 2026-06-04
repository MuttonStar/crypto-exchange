package main

import (
	"fmt"
	"time"
	"sort"
)

type Match struct {
	Ask 		*Order
	Bid 		*Order
	SizeFilled	float64
	Price		float64
}

//订单
type Order struct {
	Size 		float64 //数量
	Bid 		bool	//区分买卖
	Limit		*Limit	//追踪是在哪个特价单中
	Timestamp 	int64	//时间戳
}

type Orders []*Order

func (o Orders) Len() int {
	return len(o)
}
func (o Orders) Swap(i,j int) {
	o[i] , o[j] = o[j] , o[i]
}
func (o Orders) Less(i,j int) bool{
	return o[i].Timestamp < o[j].Timestamp
}

func NewOrder (bid bool,size float64) *Order{
	return &Order{
		Size:		size,
		Bid:		bid,
		Timestamp:	time.Now().UnixNano(),
	}
}

func (o *Order) String() string{
	return fmt.Sprintf("[size:%.2f]",o.Size)
}

//特价单,一组订单，并且是有特价
type Limit struct {
	Price  		float64
	Orders 		Orders
	TotalVolume float64
}



type Limits []*Limit

type ByBestAsk struct {
	Limits
}

func (a ByBestAsk) Len() int {
	return len(a.Limits)
}
func (a ByBestAsk) Swap(i,j int) {
	a.Limits[i] , a.Limits[j] = a.Limits[j] , a.Limits[i]
}
func (a ByBestAsk) Less(i,j int) bool{
	return a.Limits[i].Price < a.Limits[j].Price
}

type ByBestBid struct {
	Limits
}

func (a ByBestBid) Len() int {
	return len(a.Limits)
}
func (a ByBestBid) Swap(i,j int) {
	a.Limits[i] , a.Limits[j] = a.Limits[j] , a.Limits[i]
}
func (a ByBestBid) Less(i,j int) bool{
	return a.Limits[i].Price > a.Limits[j].Price
}





func NewLimit(price float64) *Limit{
	return &Limit{
		Price:		price,
		Orders:		[]*Order{},
	}
}

//添加订单
func (l *Limit) AddOrder(o *Order){
	o.Limit = l
	l.Orders = append(l.Orders,o)
	l.TotalVolume += o.Size
}

//删除订单
func (l *Limit) DeleteOrder(o *Order){
	for i := 0 ; i < len(l.Orders) ; i++ {
		if l.Orders[i] == o {
			l.Orders[i] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size

	// TODO:对整个数组重新排序
	sort.Sort(l.Orders)
}

//订单薄
type Orderbook struct {
	 Asks []*Limit	//卖价
	 Bids []*Limit	//买价

	 AskLimits map[float64]*Limit
	 BidLimits map[float64]*Limit
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		Asks:		[]*Limit{},
		Bids:		[]*Limit{},
		AskLimits:make(map[float64]*Limit),
		BidLimits:make(map[float64]*Limit),
	}
}


func (ob *Orderbook) PlaceOrder(price float64,o *Order) []Match{
	//1.尝试去匹配订单

	//2.将剩余的订单放入订单薄中
	if o.Size > 0.0 {
		ob.add(price,o)
	}
	
	return []Match{}
}

func (ob *Orderbook) add(price float64,o *Order) {
	var limit *Limit

	if o.Bid {
		limit = ob.BidLimits[price]
	} else {
		limit = ob.AskLimits[price]
	}

	if limit == nil {
		limit = NewLimit(price)
		limit.AddOrder(o)

		if o.Bid {
			ob.Bids = append(ob.Bids,limit)
			ob.BidLimits[price] = limit
		} else {
			ob.Asks = append(ob.Asks,limit)
			ob.AskLimits[price] = limit
		}
	}
}