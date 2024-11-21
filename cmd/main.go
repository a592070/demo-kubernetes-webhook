package main

import (
	"demo-kubernetes-webhook/pkg/injection"
	"log"
)

func main() {
	injector, err := injection.NewDependenciesInjector()
	if err != nil {
		log.Fatal(err)
	}
	err = injector.Server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
