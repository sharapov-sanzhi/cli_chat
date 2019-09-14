package room

import (
	"fmt"
	"time"
)

func (rm *RoomManager) monitorUsers() {
	var already = false
	for {
		select {
		case <-rm.exit:
			return
		default:
			newUsers := rm.getNewUsersListOnline()
			rm.rememberUsers(newUsers)

			exitedUsers := rm.getUsersListExited()
			rm.forgetUsers(exitedUsers)

			if len(rm.usersOnline) > 0 {
				already = false
			} else {
				if !already {
					fmt.Println("Room is empty")
					already = true
				}
			}
			time.Sleep(3 * time.Second)
		}
	}
}

func (rm *RoomManager) rememberUsers(list []string) {
	for _, u := range list {
		done := rm.startMonitoringUser(u)
		rm.usersOnline[u] = done
	}
}

func (rm *RoomManager) startMonitoringUser(name string) chan struct{} {
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			i++
			select {
			case <-done:
				rm.printMsg(fmt.Sprintf("[%s] exited\n", name), time.Now().Format("15:04:05"))
				return
			default:
				if i == 1 {
					rm.printMsg(fmt.Sprintf("[%s] is online\n", name), time.Now().Format("15:04:05"))
				}
			}

			time.Sleep(3 * time.Second)
		}
	}()

	return done
}

func (rm *RoomManager) forgetUsers(list []string) {
	for _, u := range list {
		rm.stopMonitoringUser(u)
	}
}

func (rm *RoomManager) stopMonitoringUser(u string) {
	close(rm.usersOnline[u])
	delete(rm.usersOnline, u)
}

func (rm *RoomManager) getUsersListOnline() (usersList []string, err error) {
	err = rm.db.Select(&usersList, `SELECT name FROM room WHERE id <> $1 AND online = 1`, rm.userID)
	return usersList, err
}

func (rm *RoomManager) getNewUsersListOnline() (newl []string) {
	ul, err := rm.getUsersListOnline()
	rm.failOnError(err, "Failed to get list of users online")
	var wl []string
	for wu := range rm.usersOnline {
		wl = append(wl, wu)
	}
	for _, u := range ul {
		if !contains(wl, u) {
			newl = append(newl, u)
		}
	}

	return newl
}

func (rm *RoomManager) getUsersListExited() (old []string) {
	ul, err := rm.getUsersListOnline()
	rm.failOnError(err, "Failed to get list of users online")
	for wu := range rm.usersOnline {
		if !contains(ul, wu) {
			old = append(old, wu)
		}
	}

	return old
}

func contains(list []string, s string) bool {
	for _, ss := range list {
		if ss == s {
			return true
		}
	}
	return false
}
