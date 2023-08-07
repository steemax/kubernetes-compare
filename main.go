package main

import (
	"compareapp/handlers"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

var (
	gitlabServer    string
	clientID        string
	clientSecret    string
	redirectURL     string
	gitlabTokenTime int
	gitAllowedGroup string
	store           = sessions.NewCookieStore([]byte("90a541ecfa8ec6629c9")) // Replace with your secret
)

type Config struct {
	AppPort            int    `json:"server_port"`
	GitLabAuth         bool   `json:"gitlab_auth"`
	GitlabServer       string `json:"gitlab_server"`
	GitlabSkipTls      bool   `json:"gitlab_skip_tls_verify"`
	GitlabClientId     string `json:"client_id"`
	GitlabClientSecret string `json:"client_secret"`
	GitlabCallBackUrl  string `json:"callback_url"`
	GitlabTokenLife    int    `json:"max_age_session_token"`
	GitlabAllowedGroup string `json:"auth_group_name_allowed"`
}

func checkAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//callback := "auth/callback"
		//url := r.URL.Path
		session, _ := store.Get(r, "compare-app") // Replace with your session name
		token := session.Values["token"]

		if r.URL.Path != "/auth/login" && r.URL.Path != "/auth/callback" && (token == nil || token == "") {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isGroupAllowed(groups []string) bool {
	allowedGroups := []string{gitAllowedGroup}
	for _, group := range groups {
		for _, allowedGroup := range allowedGroups {
			if group == allowedGroup {
				return true
			}
		}
	}
	return false
}

func main() {

	// Read the configuration file
	configApp, err := os.ReadFile("./conf/config.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the configuration data
	var config Config
	if err := json.Unmarshal(configApp, &config); err != nil {
		panic(err)
	}

	gitlabServer = config.GitlabServer
	clientID = config.GitlabClientId
	clientSecret = config.GitlabClientSecret
	redirectURL = config.GitlabCallBackUrl
	gitlabTokenTime = config.GitlabTokenLife
	gitAllowedGroup = config.GitlabAllowedGroup
	gitlabAuth := config.GitLabAuth

	if gitlabAuth == true {
		// Create a custom HTTP client to ignore SSL verification
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.GitlabSkipTls},
		}
		httpClient := &http.Client{Transport: tr}
		ctx := context.Background()

		provider, err := oidc.NewProvider(oidc.ClientContext(ctx, httpClient), gitlabServer)
		if err != nil {
			fmt.Printf("failed to get provider: %v\n", err)
			return
		}

		conf := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		r := mux.NewRouter()
		r.Use(checkAuthentication)

		r.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, conf.AuthCodeURL("state"), http.StatusFound)
		})

		r.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
			var ctx = context.Background()

			if r.URL.Query().Get("state") != "state" {
				http.Error(w, "state did not match", http.StatusBadRequest)
				return
			}

			oauth2Token, err := conf.Exchange(oidc.ClientContext(ctx, httpClient), r.URL.Query().Get("code"))
			if err != nil {
				http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			rawIDToken, ok := oauth2Token.Extra("id_token").(string)
			if !ok {
				http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
				return
			}

			idToken, err := provider.Verifier(&oidc.Config{ClientID: conf.ClientID}).Verify(ctx, rawIDToken)
			if err != nil {
				http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			var profile map[string]interface{}
			if err := idToken.Claims(&profile); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			//fmt.Printf("Claims: %+v\n", profile)

			// check that user group is allowed to access
			groupsDirect, ok := profile["groups_direct"].([]interface{})
			if !ok {
				http.Error(w, "Groups not found", http.StatusForbidden)
				return
			}

			// convert []interface{} to []string
			var groups []string
			for _, group := range groupsDirect {
				groupStr, ok := group.(string)
				if ok {
					groups = append(groups, groupStr)
				}
			}

			if !isGroupAllowed(groups) {
				http.Error(w, "Access denied, your group in gitlab not allowed access for this tools, sorry", http.StatusForbidden)
				fmt.Print(groups)
				return
			}

			// Set the "token" in the session
			session, _ := store.Get(r, "compare-app") // Replace with your session name
			store.Options = &sessions.Options{
				Path:     "/",                  // cookie path
				MaxAge:   gitlabTokenTime * 60, // 15 minutes in seconds for session expiration
				HttpOnly: true,                 // HttpOnly to prevent XSS attacks
			}
			session.Values["token"] = rawIDToken
			session.Save(r, w)

			http.Redirect(w, r, "/", http.StatusSeeOther)
		})

		r.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			session, err := store.Get(r, "compare-app") // Replace with your session name
			if err != nil {
				http.Error(w, "Failed to retrieve session", http.StatusInternalServerError)
				return
			}

			session.Values["token"] = nil // delete session token, when logout action
			session.Save(r, w)

			http.Redirect(w, r, "/", http.StatusSeeOther) // redirect to login page
		})

		r.HandleFunc("/", handlers.IndexHandler)
		r.HandleFunc("/select_config", handlers.SelectConfigHandler)
		r.HandleFunc("/clear/selected/cluster/config/connections", handlers.ClearSelectHandler)
		r.HandleFunc("/namespaces", handlers.NamespaceHandler)
		r.HandleFunc("/resources", handlers.ResourceHandler)
		r.HandleFunc("/compare_cluster", handlers.CompareClusterHandler)
		r.HandleFunc("/compare_cluster/canary_json", handlers.DisplayCanaryJSONHandler)
		r.HandleFunc("/compare_cluster/mettempl", handlers.CompareClusterCMTHandler)
		r.HandleFunc("/compare_cluster/deploy_json", handlers.DisplayDeployJSONHandler)
		r.HandleFunc("/compare_cluster/dmnset_json", handlers.DisplayDmnSetJSONHandler)
		r.HandleFunc("/compare_cluster/services_json", handlers.DisplaySvsJSONHandler)
		r.HandleFunc("/compare_cluster/tingress_json", handlers.DisplayTingJSONHandler)
		r.HandleFunc("/compare_cluster/helmvalues_json", handlers.DisplayHelmJSONHandler)

		fmt.Println("Listening on port", config.AppPort)
		port := ":" + strconv.Itoa(config.AppPort)
		err = http.ListenAndServe(port, r)
		if err != nil {
			fmt.Println("Cannot open port", config.AppPort, err)
		}

	} else {
		r := mux.NewRouter()
		r.HandleFunc("/", handlers.IndexHandler)
		r.HandleFunc("/select_config", handlers.SelectConfigHandler)
		r.HandleFunc("/clear/selected/cluster/config/connections", handlers.ClearSelectHandler)
		r.HandleFunc("/namespaces", handlers.NamespaceHandler)
		r.HandleFunc("/resources", handlers.ResourceHandler)
		r.HandleFunc("/compare_cluster", handlers.CompareClusterHandler)
		r.HandleFunc("/compare_cluster/canary_json", handlers.DisplayCanaryJSONHandler)
		r.HandleFunc("/compare_cluster/mettempl", handlers.CompareClusterCMTHandler)
		r.HandleFunc("/compare_cluster/deploy_json", handlers.DisplayDeployJSONHandler)
		r.HandleFunc("/compare_cluster/dmnset_json", handlers.DisplayDmnSetJSONHandler)
		r.HandleFunc("/compare_cluster/services_json", handlers.DisplaySvsJSONHandler)
		r.HandleFunc("/compare_cluster/tingress_json", handlers.DisplayTingJSONHandler)
		r.HandleFunc("/compare_cluster/helmvalues_json", handlers.DisplayHelmJSONHandler)

		fmt.Println("Listening on port", config.AppPort)
		port := ":" + strconv.Itoa(config.AppPort)
		err = http.ListenAndServe(port, r)
		if err != nil {
			fmt.Println("Cannot open port", config.AppPort, err)
		}

	}
}
