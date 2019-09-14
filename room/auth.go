package room

func (rm *RoomManager) LoginOrRegister() (err error) {
	err = rm.db.QueryRow(`
	INSERT INTO room (name, online) VALUES ($1, 1)
	ON CONFLICT (name) DO UPDATE SET online = 1
	RETURNING id
	`, rm.userName).Scan(&rm.userID)
	if err != nil {
		return err
	}
	return nil
}

func (rm *RoomManager) Logout() (err error) {
	_, err = rm.db.Exec(`UPDATE room SET online = 0 WHERE id = $1`, rm.userID)
	return err
}
