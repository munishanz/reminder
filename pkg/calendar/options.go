package calendar

type Options struct {
	CredentialFile string `json:"credential_file" yaml:"credential_file" mapstructure:"credential_file"`
	TokenFile      string `json:"token_file" yaml:"token_file" mapstructure:"token_file"`
	DryMode        bool   `json:"dry_mode" yaml:"dry_mode" mapstructure:"dry_mode"`
}

func DefaultOptions() *Options {
	return &Options{
		CredentialFile: "~/calendar_credentials.json",
		TokenFile:      "~/calendar_token.json",
		DryMode:        false,
	}
}
