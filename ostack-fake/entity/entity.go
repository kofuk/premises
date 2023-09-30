package entity

type GetTokenReq struct {
	Auth struct {
		PasswordCredentials struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
		TenantID string `json:"tenantId"`
	} `json:"auth"`
}

type GetTokenResp struct {
	Access struct {
		Token struct {
			Id      string `json:"id"`
			Expires string `json:"expires"`
		} `json:"token"`
	} `json:"access"`
}

type ServerDetailAddress struct {
	Addr    string `json:"addr"`
	Version int    `json:"version"`
}

type ServerDetailMetadata struct {
	InstanceNameTag string `json:"instance_name_tag"`
}

type ServerDetail struct {
	ID        string                           `json:"id"`
	Name      string                           `json:"name"`
	Status    string                           `json:"status"`
	Addresses map[string][]ServerDetailAddress `json:"addresses"`
	Metadata  ServerDetailMetadata             `json:"metadata"`
}

type ServerDetailResp struct {
	Servers []ServerDetail `json:"servers"`
}

type Flavor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FlavorsResp struct {
	Flavors []Flavor `json:"flavors"`
}

type ImageReq struct {
	Name string
}

type Image struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ImageResp struct {
	Images []Image `json:"images"`
}

type Server struct {
	ImageRef  string `json:"imageRef"`
	FlavorRef string `json:"flavorRef"`
	UserData  string `json:"user_data"`
	MetaData  struct {
		InstanceNameTag string `json:"instance_name_tag"`
	} `json:"metadata"`
	SecurityGroups []struct {
		Name string `json:"name"`
	} `json:"security_groups"`
}

type LaunchServerReq struct {
	Server Server `json:"server"`
}

type LaunchServerResp struct {
	Server struct {
		ID string `json:"id"`
	} `json:"server"`
}

type ServerActionReq struct {
	CreateImage *struct {
		Name string `json:"name"`
	} `json:"createImage"`
}
