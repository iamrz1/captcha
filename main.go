package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/mojocn/base64Captcha"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//configJsonBody json request body.
type configJsonBody struct {
	Id          string
	VerifyValue string
}

var store = base64Captcha.DefaultMemStore

// base64Captcha create http handler
func generateCaptchaHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var driver base64Captcha.Driver

	//create base64 encoding captcha
	//driver = base64Captcha.NewDriverString(80, 240, 20, 100, 2, 5, nil, fontsAll)
	driver = base64Captcha.DefaultDriverDigit
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, err := c.Generate()

	b64s = strings.TrimPrefix(b64s, "data:")

	body := map[string]interface{}{"success": true, "data": b64s, "captcha_id": id, "message": "Captcha generated successfully"}
	if err != nil {
		log.Println(err.Error())
		body = map[string]interface{}{"success": false, "data": b64s, "captcha_id": id, "message": "Something went wrong"}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(body)
}

// base64Captcha verify http handler
func captchaVerifyHandler(w http.ResponseWriter, r *http.Request) {

	//parse request json body
	decoder := json.NewDecoder(r.Body)
	var param configJsonBody
	err := decoder.Decode(&param)
	if err != nil {
		log.Println(err)
	}
	defer r.Body.Close()
	//verify the captcha
	body := map[string]interface{}{"success": false, "message": "Captcha generated successfully"}
	if store.Verify(param.Id, param.VerifyValue, true) {
		body = map[string]interface{}{"success": true, "message": "Captcha verified"}
	}

	//set json response
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	json.NewEncoder(w).Encode(body)
}

//start a net/http server
func main() {
	godotenv.Load()
	//api for create captcha
	http.HandleFunc("/api/v1/get-captcha", generateCaptchaHandler)

	//api for verify captcha
	http.HandleFunc("/api/v1/verify-captcha", captchaVerifyHandler)

	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "8080"
	}

	server, err := start(port)
	if err != nil {
		log.Println("err:", err)
		return
	}

	err = stopServer(server)
	if err != nil {
		log.Println("err:", err)
		return
	}

	return
}

func V1Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/get-captcha", generateCaptchaHandler)
	r.Post("/verify-captcha", captchaVerifyHandler)

	return r
}

func start(port string) (*http.Server, error) {

	addr := fmt.Sprintf(":%s", port)

	handler, err := setupRouter()
	if err != nil {
		log.Println("cant setup router:", err)
		return nil, err
	}

	srv := &http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		Handler:      handler,
	}

	go func() {
		log.Println("Staring server with address ", addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println("Stopping server:", err)
			os.Exit(-1)
		}
	}()

	return srv, nil
}

func setupRouter() (*chi.Mux, error) {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// enforce cors
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		//AllowedOrigins: []string{"*"},
		AllowOriginFunc:  verifyOrigin,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Mount("/api/v1", V1Router())

	return r, nil
}

func verifyOrigin(r *http.Request, origin string) bool {
	log.Println("cors from ", origin)
	// todo: write a function to allow only valid origins
	return true
}

func stopServer(server *http.Server) error {
	var err error
	graceful := func() error {
		log.Println("Shutting down server gracefully")
		return nil
	}

	forced := func() error {
		log.Println("Shutting down server forcefully")
		return nil
	}

	sigs := []os.Signal{syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM}
	errCh := make(chan error)
	go func() {
		errCh <- HandleSignals(sigs, graceful, forced)
	}()
	if err = <-errCh; err != nil {
		log.Println(err)
		return err
	}

	err = stop(server)
	if err != nil {
		log.Println("server stop err:", err)
		return err
	}

	return nil
}

// HandleSignals listen on the registered signals and fires the gracefulHandler for the
// first signal and the forceHandler (if any) for the next this function blocks and
// return any error that returned by any of the api first
func HandleSignals(sigs []os.Signal, gracefulHandler, forceHandler func() error) error {
	sigCh := make(chan os.Signal)
	errCh := make(chan error, 1)

	signal.Notify(sigCh, sigs...)
	defer signal.Stop(sigCh)

	grace := true
	for {
		select {
		case err := <-errCh:
			return err
		case <-sigCh:
			if grace {
				grace = false
				go func() {
					errCh <- gracefulHandler()
				}()
			} else if forceHandler != nil {
				err := forceHandler()
				errCh <- err
			}
		}
	}
}


func stop(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Http server couldn't shutdown gracefully", err)
		return err
	}

	log.Println("shutting down")
	return nil
}
