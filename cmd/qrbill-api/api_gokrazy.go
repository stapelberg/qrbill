// +build gokrazy

package main

func init() {
	// Open up listener from localhost to all IP addresses, assuming that
	// running on gokrazy means running as an appliance.
	defaultListenAddress = ":9933"
}
