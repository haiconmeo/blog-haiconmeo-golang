package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Article struct {
	gorm.Model
	Title   string        `json:"title"`
	Image   string        `json:"image"`
	Slug    string        `json:"slug"`
	Tags    string        `json:"tags"`
	Content template.HTML `json:"content"`
}

var tlp = template.Must(template.ParseFiles("index.html"))

func uploadImage(cld *cloudinary.Cloudinary, ctx context.Context, file interface{}) string {

	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       uuid.New().String(),
		UniqueFilename: api.Bool(false),
		Overwrite:      api.Bool(true)})
	if err != nil {
		fmt.Println("error")
	}
	return string(resp.SecureURL)
}
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	const MAX_UPLOAD_SIZE = 10 << 20 // Set the max upload size to 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, MAX_UPLOAD_SIZE)
	ctx := context.Background()
	if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
		http.Error(w, "The uploaded file is too big. Please choose a file that's less than 10MB in size", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	cloudinaryUrl := os.Getenv("CLOUDINARY_URL")
	cld, err := cloudinary.NewFromURL(cloudinaryUrl)
	a := uploadImage(cld, ctx, file)
	if err != nil {
		fmt.Fprintf(w, "lỗi")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	response, _ := json.Marshal(map[string]string{"location": a})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(response)
}
func getAll(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var result []Article
		db.Model(&Article{}).Limit(100).Find(&result)

		buf := &bytes.Buffer{}
		err := tlp.Execute(buf, result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		buf.WriteTo(w)
	}
}
func creatPost(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			t, _ := template.ParseFiles("create-blog.html")
			t.Execute(w, nil)
		} else {

			r.ParseMultipartForm(32 << 20)
			ctx := context.Background()
			fmt.Println("creating")
			file, _, errFile := r.FormFile("uploadfileImage")
			if errFile != nil {
				// Do your error handling here
				fmt.Fprintf(w, "lỗi file")
			}
			defer file.Close()
			cloudinaryUrl := os.Getenv("CLOUDINARY_URL")
			fmt.Println("uplaoding")
			cld, err := cloudinary.NewFromURL(cloudinaryUrl)
			a := uploadImage(cld, ctx, file)
			if err != nil {
				fmt.Fprintf(w, "lỗi")
			}
			title := r.FormValue("title")
			content := r.FormValue("content")
			post := Article{
				Title:   title,
				Image:   a,
				Slug:    r.FormValue("slug"),
				Content: template.HTML(content),
				Tags:    r.FormValue("tags"),
			}
			db.Create(&post)
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
}
func article(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)["slug"]
		var post Article
		db.Where("slug = ?", params).First(&post)
		t, _ := template.ParseFiles("blog.html")
		buf := &bytes.Buffer{}
		t.Execute(buf, post)
		buf.WriteTo(w)

	}
}
func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	godotenv.Load()
	port := os.Getenv("PORT")
	db.AutoMigrate(&Article{})
	mux := mux.NewRouter()
	mux.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	mux.HandleFunc("/", getAll(db))
	mux.HandleFunc("/blog/{slug}", article(db))
	mux.HandleFunc("/create-post", creatPost(db))
	mux.HandleFunc("/upload", UploadHandler)
	http.ListenAndServe(":"+port, mux)
}
