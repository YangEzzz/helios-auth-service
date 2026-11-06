package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/organization"
	"log"
)

func main() {
	check := flag.Bool("check-unassigned", false, "Report department assignment counts without changing data")
	flag.Parse()
	cfg := config.LoadConfig()
	db, err := database.NewConnection(cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)
	if err != nil {
		log.Fatal(err)
	}
	if *check {
		var counts struct {
			Departments               int64 `json:"departments"`
			ApprovedWithoutDepartment int64 `json:"approved_without_department"`
			Pending                   int64 `json:"pending"`
		}
		if err := db.Raw(`SELECT (SELECT count(*) FROM departments) departments,
		 (SELECT count(*) FROM users u WHERE u.status IN ('active','locked') AND NOT EXISTS (SELECT 1 FROM department_memberships m WHERE m.user_id=u.id)) approved_without_department,
		 (SELECT count(*) FROM users WHERE status='pending_approval') pending`).Scan(&counts).Error; err != nil {
			log.Fatal(err)
		}
		output, _ := json.Marshal(counts)
		fmt.Println(string(output))
		return
	}
	if err := organization.Migrate(db); err != nil {
		log.Fatal(err)
	}
	log.Println("Department schema ready; existing users and project permissions unchanged")
}
