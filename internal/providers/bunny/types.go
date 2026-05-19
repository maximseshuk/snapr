package bunny

type ListItem struct {
	Path        string `json:"Path"`
	ObjectName  string `json:"ObjectName"`
	Length      int64  `json:"Length"`
	LastChanged string `json:"LastChanged"`
	IsDirectory bool   `json:"IsDirectory"`
}
