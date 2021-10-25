package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/wajuabolarin/uptime/pkg"
)

func main() {
	godotenv.Load(".env")
	var DB_USER string = os.Getenv("DB_USER")
	var DB_PASSWORD string = os.Getenv("DB_PASSWORD")
	var DB_NAME string = os.Getenv("DB_NAME")

	var dsn string = fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s?charset=utf8&parseTime=True&loc=Local", DB_USER, DB_PASSWORD, DB_NAME)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("There was a problem connecting to the DB - %s", err)
	}
	log.Println("Database connected ...")
	db.AutoMigrate(&pkg.Target{})
	run(db)
}

var wg sync.WaitGroup

func run(db *gorm.DB) {
	repo := pkg.NewRepo(db)

	// fetch all targets with their configs
	targets := repo.FetchAll()
	totalChecks := len(targets)
	log.Printf("Running %d checks ....\n", totalChecks)

	// launch goroutines to perform the checks for each site
	for _, target := range targets {
		wg.Add(1)
		go func(t *pkg.Target) {
			log.Printf("Checking %s \n", t.URL)
			checkSite(t)
			wg.Done()
		}(target)
	}
	wg.Wait()
	log.Printf("finished running %d checks ....\n", totalChecks)

}

//check the site 3 times and deduce it's health status
// a site is healthy if out of 3 checks.
// the latest is healthy and.
// at least 2/3 of the checks are healthy.
func checkSite(config *pkg.Target) {
	type Check struct {
		StatusCode int
		Content    string
	}

	checks := make([]Check, 3)

	for i := 2; i >= 0; i-- {
		resp, err := config.MakeRequest()

		if err != nil {
			log.Printf("There was an error checking %s, %s\n", config.URL, err.Error())
			continue
		}
		content := config.ParseContent(resp)
		log.Printf("%s [%d]: %v", config.URL, i, resp.Status)

		checks[i] = Check{StatusCode: resp.StatusCode, Content: content}
	}

	healthyCount := 0
resolution:
	for i := 0; i <= 2; i++ {
		check := checks[i]
		isHealthy, err := config.IsHealthyCheck(pkg.Checker{
			StatusCode: check.StatusCode,
			Content:    check.Content,
		})
		switch {
		case i == 0:

			if !isHealthy {
				log.Println(err)
				break resolution
			}
			healthyCount++
		default:
			if isHealthy {
				healthyCount++
			}
			if err != nil {
				log.Println(err)
			}
		}
	}
	log.Printf("finished checking %s, total checks %d,  total healthy %d, total unhealthy %d \n", config.URL, len(checks), healthyCount, len(checks)-healthyCount)

}
