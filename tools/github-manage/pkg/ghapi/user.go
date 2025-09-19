package ghapi

import (
	"encoding/json"
	"fmt"
)

var username_mapping = map[string]string{}

// ParseJSONtoUser converts JSON data to a slice of Issue structs.
func ParseJSONtoUser(jsonData []byte) (User, error) {
	var user User
	err := json.Unmarshal(jsonData, &user)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

// GetIssues fetches issues from GitHub using optional search criteria.
func GetUserName(userLogin string) (User, error) {
	var user User

	if fullName, ok := username_mapping[userLogin]; ok {
		user.Login = userLogin
		user.Name = fullName
		return user, nil
	}

	command := fmt.Sprintf("gh api /users/%s", userLogin)

	results, err := RunCommandAndReturnOutput(command)
	if err != nil {
		return User{}, err
	}
	user, err = ParseJSONtoUser(results)
	if err != nil {
		return User{}, err
	}
	username_mapping[user.Login] = user.Name
	return user, nil
}

func init() {
	// Ensure any package-level initialization here if needed
	username_mapping = make(map[string]string)
}
