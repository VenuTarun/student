package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var db *sql.DB
var tmpl *template.Template

// ================== TABLE CREATION ==================
func createTable() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS students (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		phone VARCHAR(15) NOT NULL UNIQUE,
		password VARCHAR(100) NOT NULL,
		branch VARCHAR(100),
		college VARCHAR(100),
		year VARCHAR(10),
		address VARCHAR(255)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS admins (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(100) NOT NULL UNIQUE,
		phone VARCHAR(15),
		password VARCHAR(100) NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`INSERT IGNORE INTO admins (username, phone, password)
		VALUES ('admin', '9876543210', 'admin123')`)
	if err != nil {
		log.Fatal(err)
	}
}

// ================== STUDENT HANDLERS ==================
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "register.html", nil)
		return
	}

	if r.Method == "POST" {
		name := r.FormValue("name")
		phone := r.FormValue("phone")
		password := r.FormValue("password")

		_, err := db.Exec("INSERT INTO students (name, phone, password) VALUES (?, ?, ?)", name, phone, password)
		if err != nil {
			http.Error(w, "Error registering student", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "login.html", nil)
		return
	}

	if r.Method == "POST" {
		phone := r.FormValue("phone")
		password := r.FormValue("password")

		var id int
		err := db.QueryRow("SELECT id FROM students WHERE phone=? AND password=?", phone, password).Scan(&id)
		if err != nil {
			tmpl.ExecuteTemplate(w, "login.html", "Invalid credentials")
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/student/%d", id), http.StatusSeeOther)
	}
}

func studentProfile(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	var student struct {
		ID       int
		Name     string
		Phone    string
		Password string
		Branch   string
		College  string
		Year     string
		Address  string
	}

	err := db.QueryRow(`SELECT id, name, phone, password, branch, college, year, address FROM students WHERE id=?`, id).
		Scan(&student.ID, &student.Name, &student.Phone, &student.Password, &student.Branch, &student.College, &student.Year, &student.Address)

	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	tmpl.ExecuteTemplate(w, "profile.html", student)
}

// ================== ADMIN HANDLERS ==================
func registerAdminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "admin_register.html", nil)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		phone := r.FormValue("phone")
		password := r.FormValue("password")

		_, err := db.Exec(`INSERT INTO admins (username, phone, password) VALUES (?, ?, ?)`, username, phone, password)
		if err != nil {
			http.Error(w, "Error registering admin", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
	}
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "admin_login.html", nil)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		var id int
		err := db.QueryRow("SELECT id FROM admins WHERE username=? AND password=?", username, password).Scan(&id)
		if err != nil {
			tmpl.ExecuteTemplate(w, "admin_login.html", "Invalid Admin credentials")
			return
		}

		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	}
}

func adminDashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, phone, branch, college, year, address FROM students")
	if err != nil {
		http.Error(w, "Error fetching students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Student struct {
		ID      int
		Name    string
		Phone   string
		Branch  string
		College string
		Year    string
		Address string
	}

	studentCh := make(chan Student)
	errCh := make(chan error)
	doneCh := make(chan bool)

	go func() {
		for rows.Next() {
			var s Student
			err := rows.Scan(&s.ID, &s.Name, &s.Phone, &s.Branch, &s.College, &s.Year, &s.Address)
			if err != nil {
				errCh <- err
				return
			}
			studentCh <- s
		}
		doneCh <- true
	}()

	var students []Student
	for {
		select {
		case s := <-studentCh:
			students = append(students, s)
		case err := <-errCh:
			http.Error(w, "Error scanning student: "+err.Error(), http.StatusInternalServerError)
			return
		case <-doneCh:
			tmpl.ExecuteTemplate(w, "admin.html", students)
			return
		}
	}
}

func addStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.ExecuteTemplate(w, "add_student.html", nil)
		return
	}

	if r.Method == "POST" {
		name := r.FormValue("name")
		phone := r.FormValue("phone")
		password := r.FormValue("password")
		branch := r.FormValue("branch")
		college := r.FormValue("college")
		year := r.FormValue("year")
		address := r.FormValue("address")

		_, err := db.Exec(`INSERT INTO students (name, phone, password, branch, college, year, address) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			name, phone, password, branch, college, year, address)

		if err != nil {
			http.Error(w, "Error adding student", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	}
}

func editStudent(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	if r.Method == "GET" {
		var s struct {
			ID      int
			Name    string
			Phone   string
			Branch  string
			College string
			Year    string
			Address string
		}

		err := db.QueryRow(`SELECT id, name, phone, branch, college, year, address FROM students WHERE id=?`, id).
			Scan(&s.ID, &s.Name, &s.Phone, &s.Branch, &s.College, &s.Year, &s.Address)

		if err != nil {
			http.Error(w, "Student not found", http.StatusNotFound)
			return
		}

		tmpl.ExecuteTemplate(w, "edit_student.html", s)
		return
	}

	if r.Method == "POST" {
		idStr := r.FormValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return
		}

		branch := r.FormValue("branch")
		college := r.FormValue("college")
		year := r.FormValue("year")
		address := r.FormValue("address")

		_, err = db.Exec(`UPDATE students SET branch=?, college=?, year=?, address=? WHERE id=?`,
			branch, college, year, address, id)

		if err != nil {
			http.Error(w, "Error updating student", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	}
}

func deleteStudent(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	_, err := db.Exec("DELETE FROM students WHERE id=?", id)
	if err != nil {
		http.Error(w, "Error deleting student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// ================== MAIN FUNCTION ==================
func main() {
	var err error
	tmpl, err = template.ParseGlob("templates/*.html") // ✅ Correct glob pattern
	if err != nil {
		log.Fatalf("Template parsing error: %v", err)
	}

	db, err = sql.Open("mysql", "root:1234@tcp(127.0.0.1:3306)/college")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	r := mux.NewRouter()

	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/student/{id}", studentProfile).Methods("GET")

	r.HandleFunc("/admin/register", registerAdminHandler).Methods("GET", "POST")
	r.HandleFunc("/admin/login", adminLoginHandler).Methods("GET", "POST")
	r.HandleFunc("/admin/dashboard", adminDashboard).Methods("GET")
	r.HandleFunc("/admin/add", addStudent).Methods("GET", "POST")
	r.HandleFunc("/admin/edit/{id}", editStudent).Methods("GET", "POST")
	r.HandleFunc("/admin/delete/{id}", deleteStudent).Methods("GET")

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
