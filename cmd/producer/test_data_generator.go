// Описание: Генерация тестовых данных для заказов в формате JSON
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"l0_test_self/models/orders"

	"github.com/brianvoe/gofakeit/v6"
)

func GenerateTestOrder() orders.Order {
	gofakeit.Seed(time.Now().UnixNano())

	// Генерируем основной заказ
	order := orders.Order{
		OrderUid:          gofakeit.UUID(),
		TrackNumber:       fmt.Sprintf("WB%s", gofakeit.LetterN(10)),
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerId:        gofakeit.LetterN(4),
		DeliveryService:   "meest",
		Shardkey:          fmt.Sprintf("%d", gofakeit.Number(1, 10)),
		SmId:              gofakeit.Number(1, 100),
		DateCreated:       time.Now(),
		OofShard:          fmt.Sprintf("%d", gofakeit.Number(1, 10)),
	}

	// Генерируем данные доставки
	order.Delivery = orders.Delivery{
		Name:    gofakeit.Name(),
		Phone:   gofakeit.Phone(),
		Zip:     gofakeit.Zip(),
		City:    gofakeit.City(),
		Address: gofakeit.Address().Address,
		Region:  gofakeit.State(),
		Email:   gofakeit.Email(),
	}

	// Генерируем данные оплаты
	order.Payment = orders.Payment{
		Transaction:  order.OrderUid,
		RequestId:    "",
		Currency:     "USD",
		Provider:     "wbpay",
		Amount:       gofakeit.Number(100, 5000),
		PaymentDt:    int(time.Now().Unix()),
		Bank:         "alpha",
		DeliveryCost: gofakeit.Number(10, 200),
		GoodsTotal:   gofakeit.Number(100, 4800),
		CustomFee:    0,
	}

	// Генерируем товары
	itemsCount := gofakeit.Number(1, 5)
	for i := 0; i < itemsCount; i++ {
		item := orders.Item{
			ChrtId:      gofakeit.Number(1000000, 9999999),
			TrackNumber: order.TrackNumber,
			Price:       gofakeit.Number(100, 1000),
			Rid:         gofakeit.UUID(),
			Name:        gofakeit.ProductName(),
			Sale:        gofakeit.Number(0, 50),
			Size:        gofakeit.RandomString([]string{"S", "M", "L", "XL", "0"}),
			TotalPrice:  gofakeit.Number(100, 1000),
			NmId:        gofakeit.Number(1000000, 9999999),
			Brand:       gofakeit.Company(),
			Status:      gofakeit.Number(200, 202),
		}
		order.Items = append(order.Items, item)
	}

	return order
}

func GenerateTestOrderJSON() ([]byte, error) {
	order := GenerateTestOrder()
	return json.MarshalIndent(order, "", "  ")
}
