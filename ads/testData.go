package main

import (
	"sync"
	"time"
)

// AdsRepository отвечает за доступ к данным
type AdsRepository struct {
	sync.RWMutex
	data []Ad // список объявлений
}

var repo = &AdsRepository{
	data: []Ad{
		{
			ID:          1,
			Title:       "Продам гараж",
			Description: "Очень ухоженный",
			Price:       1_000_000,
			Photos: []string{
				"/static/img/garage_1.png",
				"/static/img/garage_2.png",
			},
			Tags:      []string{"недвижимость", "гараж"},
			SellerID:  1,
			CreatedAt: time.Now(),
			Views:     12,
		},
	},
}
