package main

type Repository struct {
	URL      string
	Username string
	Password string
	Branch   string
	Commit   string
	Path     string
}

func FromMap(m map[string]string) Repository {
	return Repository{
		URL:      m["URL"],
		Username: m["USERNAME"],
		Password: m["PASSWORD"],
		Branch:   m["BRANCH"],
		Commit:   m["COMMIT"],
		Path:     m["PATH"],
	}
}
