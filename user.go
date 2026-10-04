package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	store "github.com/jsol/finance/store"
)

func (app *application) printUsers(w http.ResponseWriter, r *http.Request, template string) {
	app.executeTemplate(w, template, app)
}

func (app *application) userHandler(w http.ResponseWriter, r *http.Request) {
	app.printUsers(w, r, "users")
}

func (app *application) addUserHandler(w http.ResponseWriter, r *http.Request) {

	var factorInt int64
	name := r.PostFormValue("name")
	factor := r.PostFormValue("factor")

	factorInt, err := strconv.ParseInt(factor, 10, 64)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	if slices.Contains(app.Users, name) {
		fmt.Printf("User %s already exists", name)
		w.WriteHeader(400)
		return
	}

	args := store.AddUserParams{Name: name, Factor: sql.NullInt64{Valid: true, Int64: factorInt}}

	err = app.queries.AddUser(r.Context(), args)

	if err != nil {
		fmt.Printf("Error adding user %s: ", name)
		fmt.Println(err)
	}

	app.fetchUsers(r.Context())
	app.printUsers(w, r, "user_list")
}

func (app *application) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	err := app.queries.RemoveUser(r.Context(), id)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	app.fetchUsers(r.Context())
	app.printUsers(w, r, "user_list")
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {

	var factorInt int64
	name := r.PostFormValue("name")
	factor := r.PostFormValue("factor")

	factorInt, err := strconv.ParseInt(factor, 10, 64)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	args := store.UpdateUserParams{Factor: sql.NullInt64{Valid: true, Int64: factorInt}, Name: name}
	err = app.queries.UpdateUser(r.Context(), args)

	if err != nil {
		fmt.Printf("Error updating user %s: ", name)
		fmt.Println(err)
	}

	app.fetchUsers(r.Context())
	app.printUsers(w, r, "user_list")
}
