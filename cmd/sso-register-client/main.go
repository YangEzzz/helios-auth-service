// sso-register-client registers a first-party application without granting roles.
package main

import (
	"errors"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/session"
	"helios-auth-service/internal/sso"
	"log"
	"os"
)

func main() {
	cfg := config.LoadConfig()
	id, secret, redirect, projectID := os.Getenv("SSO_CLIENT_ID"), os.Getenv("SSO_CLIENT_SECRET"), os.Getenv("SSO_REDIRECT_URI"), os.Getenv("SSO_PROJECT_ID")
	if id == "" || len(secret) < 32 || redirect == "" || projectID == "" {
		log.Fatal("Set SSO_CLIENT_ID, SSO_CLIENT_SECRET (32+), SSO_REDIRECT_URI and SSO_PROJECT_ID")
	}
	db, err := database.NewConnection(cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)
	if err != nil {
		log.Fatal(err)
	}
	var p models.Project
	if err = db.First(&p, "project_id_string = ?", projectID).Error; err != nil {
		log.Fatal("Project must already exist; registration never creates a project or grants membership")
	}
	var cl sso.Client
	err = db.First(&cl, "id = ?", id).Error
	if err == nil {
		if cl.ProjectID != p.ID || !session.Equal(cl.SecretHash, session.Hash(secret)) {
			log.Fatal("Client already exists with different settings; use admin UI for changes")
		}
		log.Println("SSO client already registered; no changes")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatal(err)
	}
	cl = sso.Client{ID: id, ProjectID: p.ID, Name: p.ProjectName, SecretHash: session.Hash(secret), Enabled: true, RedirectURIs: pq.StringArray{redirect}}
	if err = db.Create(&cl).Error; err != nil {
		log.Fatal(err)
	}
	log.Println("SSO client registered for existing project; account memberships unchanged")
}
