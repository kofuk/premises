package main

import (
	"log"
	"os"
	"time"

	"chronoscoper.com/premises/cloudflare"
	"chronoscoper.com/premises/config"
	"chronoscoper.com/premises/conoha"
)

func BuildVM(gameConfig []byte, cfg *config.Config) error {
	token, err := conoha.GetToken(cfg)
	if err != nil {
		return err
	}

	flavors, err := conoha.GetFlavors(cfg, token)
	if err != nil {
		return err
	}
	flavorID := flavors.GetIDByCondition(2, 1, 100)

	imageID, err := conoha.GetImageID(cfg, token, "mc-premises")
	if err != nil {
		return err
	}

	startupScript, err := conoha.GenerateStartupScript(gameConfig, cfg)
	if err != nil {
		return err
	}

	if _, err := conoha.CreateVM(cfg, token, imageID, flavorID, startupScript); err != nil {
		return err
	}

	if err := conoha.DeleteImage(cfg, token, imageID); err != nil {
		return err
	}

	return nil
}

func DestroyVM(cfg *config.Config) error {
	token, err := conoha.GetToken(cfg)
	if err != nil {
		return err
	}

	detail, err := conoha.GetVMDetail(cfg, token, "mc-premises")
	if err != nil {
		return err
	}


	if err := conoha.StopVM(cfg, token, detail.ID); err != nil {
		return err
	}

	// Wait for VM to stop
	for {
		detail, err := conoha.GetVMDetail(cfg, token, "mc-premises")
		if err != nil {
			return err
		}
		if detail.Status == "SHUTOFF" {
			break
		}
		time.Sleep(30 * time.Second)
	}

	if err := conoha.CreateImage(cfg, token, detail.ID, "mc-premises"); err != nil {
		return err
	}

	// Wait for image to be saved
	for {
		if _, err := conoha.GetImageID(cfg, token, "mc-premises"); err == nil {
			break
		}
		time.Sleep(30 * time.Second)		
	}

	if err := conoha.DeleteVM(cfg, token, detail.ID); err != nil {
		return err
	}

	return nil
}

func main() {
	prefix := ""
	if len(os.Args) > 1 {
		prefix = os.Args[1]
	}

	cfg, err := config.LoadConfig(prefix)
	if err != nil {
		log.Fatal(err)
	}
	cfg.Prefix = prefix

	zoneID, err := cloudflare.GetZoneID(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := cloudflare.UpdateDNS(cfg, zoneID, "2001:db8::2", 6); err != nil {
		log.Fatal(err)
	}

	// if err := monitor.GenerateTLSKey(cfg); err != nil {
	// 	log.Fatal(err)
	// }

	// ss, err := conoha.GenerateStartupScript([]byte("hoge"), cfg)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println(ss)

	// if err := BuildVM(cfg); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := DestroyVM(cfg); err != nil {
	// 	log.Fatal(err)
	// }
}
