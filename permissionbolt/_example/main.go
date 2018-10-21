package main

func main() {

}

/*
See ../permissionbolt.go comments for why this is commented.

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/iris-contrib/middleware/permissionbolt"
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// New permissions middleware
	perm, handler, err := permissions.New("bolt.db")
	if err != nil {
		log.Fatalln(err)
	}

	// Blank slate, no default permissions
	//perm.Clear()

	// Enable the permissions middleware
	app.Use(handler)

	// Get the userstate, used in the handlers below
	userstate := perm.UserState()

	app.Get("/", func(ctx iris.Context) {
		w := ctx.ResponseWriter()
		fmt.Fprintf(w, "Has user bob: %v\n", userstate.HasUser("bob"))
		fmt.Fprintf(w, "Logged in on server: %v\n", userstate.IsLoggedIn("bob"))
		fmt.Fprintf(w, "Is confirmed: %v\n", userstate.IsConfirmed("bob"))
		fmt.Fprintf(w, "Username stored in cookies (or blank): %v\n", userstate.Username(ctx.Request()))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *user rights*: %v\n", userstate.UserRights(ctx.Request()))
		fmt.Fprintf(w, "Current user is logged in, has a valid cookie and *admin rights*: %v\n", userstate.AdminRights(ctx.Request()))
		fmt.Fprintf(w, "\nTry: /register, /confirm, /remove, /login, /logout, /makeadmin, /clear, /data and /admin")
	})

	app.Get("/register", func(ctx iris.Context) {
		userstate.AddUser("bob", "hunter1", "bob@zombo.com")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was created: %v\n", userstate.HasUser("bob"))
	})

	app.Get("/confirm", func(ctx iris.Context) {
		userstate.MarkConfirmed("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was confirmed: %v\n", userstate.IsConfirmed("bob"))
	})

	app.Get("/remove", func(ctx iris.Context) {
		userstate.RemoveUser("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "User bob was removed: %v\n", !userstate.HasUser("bob"))
	})

	app.Get("/login", func(ctx iris.Context) {
		userstate.Login(ctx.ResponseWriter(), "bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now logged in: %v\n", userstate.IsLoggedIn("bob"))
	})

	app.Get("/logout", func(ctx iris.Context) {
		userstate.Logout("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now logged out: %v\n", !userstate.IsLoggedIn("bob"))
	})

	app.Get("/makeadmin", func(ctx iris.Context) {
		userstate.SetAdminStatus("bob")
		fmt.Fprintf(ctx.ResponseWriter(), "bob is now administrator: %v\n", userstate.IsAdmin("bob"))
	})

	app.Get("/clear", func(ctx iris.Context) {
		userstate.ClearCookie(ctx.ResponseWriter())
		fmt.Fprintf(ctx.ResponseWriter(), "Clearing cookie")
	})

	app.Get("/data", func(ctx iris.Context) {
		fmt.Fprintf(ctx.ResponseWriter(), "Success!\n\nUser page that only logged in users must see.")
	})

	app.Get("/admin", func(ctx iris.Context) {
		fmt.Fprintf(ctx.ResponseWriter(), "Success!\n\nSuper secret information that only logged in administrators must see.\n\n")
		if usernames, err := userstate.AllUsernames(); err == nil {
			fmt.Fprintf(ctx.ResponseWriter(), "list of all users: "+strings.Join(usernames, ", "))
		}
	})

	// Custom handler for when permissions are denied
	perm.SetDenyFunction(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Permission denied!", http.StatusForbidden)
	})

	// Serve
	app.Run(iris.Addr(":3000"))
}
*/
