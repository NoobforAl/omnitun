package main

import "github.com/NoobforAl/omnitun/server/manager"

func apiPatchEnabled(on bool) manager.APIPatch { return manager.APIPatch{Enabled: &on} }

func apiPatchToken() manager.APIPatch { return manager.APIPatch{NewToken: true} }

func apiPatchListen(host string, port *int) manager.APIPatch {
	return manager.APIPatch{Listen: &host, Port: port}
}
