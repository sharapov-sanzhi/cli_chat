package room

func DBConn(conf *Config) string {
	return "postgres://" + conf.DBUserName + ":" + conf.DBPassword + "@" + conf.DBHost + "/" + conf.DBName + "?sslmode=disable"
}
