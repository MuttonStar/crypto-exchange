package orderbook

import (
	"fmt"
	"time"
	"sort"
	"math/rand"
)

type Match struct {
	Ask 		*Order
	Bid 		*Order
	SizeFilled	float64
	Price		float64
}

//订单
type Order struct {
	ID			int64
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
		ID:			int64(rand.Intn(10000000)),
		Size:		size,
		Bid:		bid,
		Timestamp:	time.Now().UnixNano(),
	}
}

func (o *Order) String() string{
	return fmt.Sprintf("[size:%.2f]",o.Size)
}

func (o *Order) IsFilled() bool {
	return o.Size == 0.0
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

func (l *Limit) Fill(o *Order) []Match{
	var(
		matches  		[]Match
		ordersToDelete	[]*Order
	)

	for _,order := range l.Orders {
		match := l.fillOrder(order,o)
		matches = append(matches,match)

		l.TotalVolume -= match.SizeFilled

		if order.IsFilled() {
			ordersToDelete = append(ordersToDelete,order)
		}

		if o.IsFilled() {
			break
		}
	}

	for _,order := range ordersToDelete {
		l.DeleteOrder(order)
	}

	return matches
}

func (l *Limit) fillOrder(a,b *Order) Match {
	var (
		bid 		*Order
		ask  		*Order
		sizeFilled	float64
	)

	if a.Bid {
		bid = a
		ask = b
	} else {
		bid = b
		ask = a
	}

	if a.Size >= b.Size {
		a.Size -= b.Size
		sizeFilled = b.Size
		b.Size = 0
	} else {
		b.Size -= a.Size
		sizeFilled = a.Size
		a.Size = 0
	}

	return Match {
		Bid: 		bid,
		Ask:		ask,
		SizeFilled:	sizeFilled,
		Price:		l.Price,
	}
}

//订单薄
type Orderbook struct {
	 asks []*Limit	//卖价
	 bids []*Limit	//买价

	 AskLimits map[float64]*Limit
	 BidLimits map[float64]*Limit
	 Orders    map[int64]*Order
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		asks:		[]*Limit{},
		bids:		[]*Limit{},
		AskLimits:make(map[float64]*Limit),
		BidLimits:make(map[float64]*Limit),
		Orders:	  make(map[int64]*Order),
	}
}

func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {
	matches := []Match{}

	if o.Bid {

		if o.Size > ob.AskTotalVolume() {
			panic(fmt.Errorf("not enough volume [szie:%.2f] for market order [size:%.2f]",ob.AskTotalVolume(),o.Size))
		}

		for _, limit := range ob.Asks() {
			limitMatches := limit.Fill(o)
			matches = append(matches,limitMatches...)

			if len(limit.Orders) == 0{
				ob.clearLimit(true,limit)
			}
		}
	} else {
		if o.Size > ob.BidTotalVolume() {
			panic(fmt.Errorf("not enough volume [szie:%.2f] for market order [size:%.2f]",ob.BidTotalVolume(),o.Size))
		}

		for _, limit := range ob.Bids() {
			limitMatches := limit.Fill(o)
			matches = append(matches,limitMatches...)

			if len(limit.Orders) == 0{
				ob.clearLimit(true,limit)
			}
		}
	}

	return matches
}



func (ob *Orderbook) PlaceLimitOrder(price float64,o *Order) {
	var limit *Limit

	if o.Bid {
		limit = ob.BidLimits[price]
	} else {
		limit = ob.AskLimits[price]
	}

	if limit == nil {
		limit = NewLimit(price)

		if o.Bid {
			ob.bids = append(ob.bids,limit)
			ob.BidLimits[price] = limit
		} else {
			ob.asks = append(ob.asks,limit)
			ob.AskLimits[price] = limit
		}
	}

	ob.Orders[o.ID] = o
	limit.AddOrder(o)
}

func (ob *Orderbook) clearLimit(bid bool,l *Limit) {
	if bid {
		delete(ob.BidLimits,l.Price)
		for i := 0 ; i < len(ob.bids) ; i++ {
			if ob.bids[i] == l {
				ob.bids[i] = ob.bids[len(ob.bids)-1]
				ob.bids = ob.bids[:len(ob.bids)-1]
			}
		}
	} else {
		delete(ob.AskLimits,l.Price)
		for i := 0 ; i < len(ob.asks) ; i++ {
			if ob.asks[i] == l {
				ob.asks[i] = ob.asks[len(ob.asks)-1]
				ob.asks = ob.asks[:len(ob.asks)-1]
			}
		}
	}
}

func (ob *Orderbook) CancelOrder(o *Order) {
	limit := o.Limit
	limit.DeleteOrder(o)
	//如果取消订单，需要删除对应订单的id
	delete(ob.Orders,o.ID)
}

func (ob *Orderbook) BidTotalVolume() float64{
	totalVolume := 0.0

	for i := 0 ; i < len(ob.bids) ; i++ {
		totalVolume += ob.bids[i].TotalVolume
	}

	return totalVolume
}

func (ob *Orderbook) AskTotalVolume() float64{
	totalVolume := 0.0

	for i := 0 ; i < len(ob.asks) ; i++ {
		totalVolume += ob.asks[i].TotalVolume
	}

	return totalVolume
}

func (ob *Orderbook) Asks() []*Limit {
	sort.Sort(ByBestAsk{ob.asks})
	return ob.asks
}

func (ob *Orderbook) Bids() []*Limit {
	sort.Sort(ByBestBid{ob.bids})
	return ob.bids
}


