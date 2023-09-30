package main

import "log"

func main() {
	ostack, err := NewOstack(TenantCredentials("tenantId", "user", "password"), Token("xxxxxxxxxx"))
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(ostack.Start())
}
