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
