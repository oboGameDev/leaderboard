package app

type LeaderboardItem struct {
	UserID string  `json:"userId"`
	Points float64 `json:"points"`
	Rank   int64   `json:"rank"`
}

type League struct {
	ID    int
	Min   int
	Max   int
	Names map[string]string
}

func (l League) Name(lang string) string {
	if l.Names == nil {
		return ""
	}
	if v, ok := l.Names[lang]; ok && v != "" {
		return v
	}
	if v, ok := l.Names["en"]; ok && v != "" {
		return v
	}
	for _, v := range l.Names {
		return v
	}
	return ""
}
