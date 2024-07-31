package apitypes

type GetTokenInput struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name     string `json:"name"`
					Password string `json:"password"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
		Scope struct {
			Project struct {
				ID string `json:"id"`
			} `json:"project"`
		} `json:"scope"`
	} `json:"auth"`
}

type GetTokenOutput struct {
	Token struct {
		ExpiresAt string `json:"expires_at"`
	} `json:"token"`
}

type BlockDeviceMapping struct {
	UUID string `json:"uuid"`
}

type CreateServerInput struct {
	Server struct {
		FlavorID string `json:"flavorRef"`
		UserData string `json:"user_data"`
		MetaData struct {
			InstanceNameTag string `json:"instance_name_tag"`
		} `json:"metadata"`
		SecurityGroups []struct {
			Name string `json:"name"`
		} `json:"security_groups"`
		BlockDevices []BlockDeviceMapping `json:"block_device_mapping_v2"`
	} `json:"server"`
}

type RenameVolumeInput struct {
	Volume struct {
		Name string `json:"name"`
	} `json:"volume"`
}
