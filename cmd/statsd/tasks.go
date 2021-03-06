package main

import "github.com/deissh/osu-api-server/pkg"

func GetUsersOnline() (f float64, err error) {
	var count int
	err = pkg.Db.
		QueryRow("SELECT count('id') FROM users WHERE is_online = true").
		Scan(&count)

	return float64(count), err
}

func GetActiveUsers() (f float64, err error) {
	var count int
	err = pkg.Db.
		QueryRow("SELECT count('id') FROM users WHERE is_active = true").
		Scan(&count)

	return float64(count), err
}
