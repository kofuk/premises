package entity

type GetTokenReq struct {
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

type GetTokenResp struct {
	Token struct {
		ExpiresAt string `json:"expires_at"`
	} `json:"token"`
}

type ServerDetailAddress struct {
	Addr    string `json:"addr"`
	Version int    `json:"version"`
}

type ServerDetailMetadata struct {
	InstanceNameTag string `json:"instance_name_tag"`
}

type Volume struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ServerDetail struct {
	ID        string                           `json:"id"`
	Name      string                           `json:"name"`
	Status    string                           `json:"status"`
	Addresses map[string][]ServerDetailAddress `json:"addresses"`
	Metadata  ServerDetailMetadata             `json:"metadata"`
	Volumes   []Volume                         `json:"os-extended-volumes:volumes_attached"`
}

type ListServerDetailsResp struct {
	Servers []ServerDetail `json:"servers"`
}

type GetServerDetailResp struct {
	Server ServerDetail `json:"server"`
}

type ListVolumesResp struct {
	Volumes []Volume `json:"volumes"`
}

type Server struct {
	FlavorRef string `json:"flavorRef"`
	UserData  string `json:"user_data"`
	MetaData  struct {
		InstanceNameTag string `json:"instance_name_tag"`
	} `json:"metadata"`
	SecurityGroups []struct {
		Name string `json:"name"`
	} `json:"security_groups"`
	BlockDevices []struct {
		UUID string `json:"uuid"`
	} `json:"block_device_mapping_v2"`
}

type LaunchServerReq struct {
	Server Server `json:"server"`
}

type LaunchServerResp struct {
	Server struct {
		ID string `json:"id"`
	} `json:"server"`
}

type VolumeActionReq struct {
	UploadImage struct {
		ImageName string `json:"image_name"`
	} `json:"os-volume_upload_image"`
}

type UpdateVolumeReq struct {
	Volume struct {
		Name string `json:"name"`
	} `json:"volume"`
}

type ListFlavorsResp struct {
	Flavors []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		RAM        int    `json:"ram"`
		Disk       int    `json:"disk"`
		Swap       string `json:"swap"`
		VCPUS      int    `json:"vcpus"`
		RxTxFactor int    `json:"rxtx_factor"`
		Disabled   bool   `json:"OS-FLV-DISABLED:disabled"`
		Public     bool   `json:"os-flavor-access:is_public"`
	} `json:"flavors"`
}
