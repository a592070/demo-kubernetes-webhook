package main

import (
	"demo-kubernetes-webhook/pkg/injection"
)

func main() {
	injector, err := injection.NewDependenciesInjector()
	if err != nil {
		panic(err)
	}
	err = injector.Server.Run()
	if err != nil {
		panic(err)
	}
}
