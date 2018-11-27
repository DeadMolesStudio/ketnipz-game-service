package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	PrometheusNamespace = "game_service"
)

var (
	TotalRooms = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: PrometheusNamespace,
		Name:      "total_rooms",
		Help:      "Count of alive game rooms",
	})
)

func AddRoomToCounter() {
	TotalRooms.Inc()
}

func SubtractRoomFromCounter() {
	TotalRooms.Dec()
}
