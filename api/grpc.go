package api

import "github.com/giperboloid/centerms/services"

type API struct {
	Config services.ConfigService
	Data services.DataService
}