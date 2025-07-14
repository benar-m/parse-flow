package internal

import (
	ip2 "github.com/ip2location/ip2location-go"
)

//Here Define a function to read from ParseLogChan and then serve an api
//Define rules to classify what is a slow request and flag/visualize them
//Dashboard of GET Requests,POSTs, Success rates of the requests
//Maybe also determine if something the host dyno is okay or not based on failing requests

//UI will be a TUI option or a local server(boring js!

func (a *App) fingerPrintIp(ip string) ip2.IP2Locationrecord {

	result, err := a.GeoDb.Get_all(ip)
	if err != nil {
		return ip2.IP2Locationrecord{}
	}
	return result
}
