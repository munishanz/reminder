package model

import (
	"fmt"
)

/*
A User represents the user of the reminder app.

The app doesn't have multi-tenant setting, but it is supposed
to be used only locally and by the user who is owner of the OS account.
*/
type User struct {
	Name    string `json:"name"`
	EmailId string `json:"email_id"`
}

// String provides basic string representation of a user.
func (u User) String() string {
	return fmt.Sprintf("{Name: %v, EmailId: %v}", u.Name, u.EmailId)
}
