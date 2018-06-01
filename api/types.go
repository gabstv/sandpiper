package api

type NewRoute struct {
	Domain   string `json:"domain,omitempty"`
	Autocert bool   `json:"autocert,omitempty"`
	OutType  string `json:"out_type,omitempty"`
	OutPath  string `json:"out_path,omitempty"`
}
