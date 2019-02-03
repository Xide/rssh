package agent

type ForwardedHost struct {
	Domain string `json:"domain" mapstructure:"domain"`
	Host   string `json:"host" mapstructure:"host"`
	Port   uint16 `json:"port" mapstructure:"ports"`
}

type Agent struct {
	Hosts []ForwardedHost `json:"hosts" mapstructure:"hosts"`
}
