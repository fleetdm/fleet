package main

func ExampleGetUserRoles() {
	app := createApp()
	app.Run([]string{"get", "user_roles"})
	// Output:
	// hello
}
