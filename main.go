package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
	"github.com/Financial-Times/content-rw-neo4j/v3/content"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger/v2"
	cli "github.com/jawher/mow.cli"
)

const (
	appDescription = "A RESTful API for managing content (barebone representation as full content is served from MongoDB) in Neo4j"
)

func main() {
	app := cli.App("content-rw-neo4j", appDescription)

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "content-rw-neo4j",
		Desc:   "Name of the application",
		EnvVar: "APP_NAME",
	})

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "upp-content-rw-neo4j",
		Desc:   "System code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	neoURL := app.String(cli.StringOpt{
		Name:   "neo-url",
		Value:  "http://localhost:7474/db/data",
		Desc:   "neo4j endpoint URL",
		EnvVar: "NEO_URL",
	})

	apiYml := app.String(cli.StringOpt{
		Name:   "api-yml",
		Value:  "./api.yml",
		Desc:   "Location of the API Swagger YML file.",
		EnvVar: "API_YML",
	})

	port := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Port to listen on",
		EnvVar: "APP_PORT",
	})

	batchSize := app.Int(cli.IntOpt{
		Name:   "batchSize",
		Value:  1024,
		Desc:   "Maximum number of statements to execute per batch",
		EnvVar: "BATCH_SIZE",
	})

	log := logger.NewUPPInfoLogger(*appName)
	log.WithFields(map[string]interface{}{
		"appName":       *appName,
		"appSystemCode": *appSystemCode,
		"neoURL":        *neoURL,
		"port":          *port,
		"batchSize":     *batchSize,
	}).Info("Application starting...")

	app.Action = func() {
		log.Infof("Application started with args %s", os.Args)

		driver, err := cmneo4j.NewDefaultDriver(*neoURL, log)
		if err != nil {
			log.WithError(err).Fatal("Could not create a new instance of cmneo4j driver")
		}
		defer driver.Close()

		contentDriver := content.NewContentService(driver)
		contentDriver.Initialise()

		services := map[string]baseftrwapp.Service{
			"content": contentDriver,
		}

		var checks []fthealth.Check
		for _, service := range services {
			checks = append(checks, makeCheck(service, driver))
		}

		ymlBytes, err := ioutil.ReadFile(*apiYml)
		if err != nil {
			//logger.WithField("api-yml", *apiYml).Warn("Failed to read OpenAPI yml file, please confirm the file exists and is not empty.")
			ymlBytes = nil // the base-ft-rw-app-go lib will not add the /__api endpoint if OpenAPIData is nil
		}

		hc := fthealth.TimedHealthCheck{
			HealthCheck: fthealth.HealthCheck{
				SystemCode:  "upp-content-rw-neo4j",
				Name:        "ft-content_rw_neo4j ServiceModule",
				Description: "Writes 'content' to Neo4j, usually as part of a bulk upload done on a schedule",
				Checks:      checks,
			},
			Timeout: 10 * time.Second,
		}

		baseftrwapp.RunServerWithConf(baseftrwapp.RWConf{
			Services:      services,
			HealthHandler: fthealth.Handler(hc),
			Port:          *port,
			ServiceName:   "content-rw-neo4j",
			Env:           "local",
			EnableReqLog:  true,
			OpenAPIData:   ymlBytes,
		})
	}
	err := app.Run(os.Args)
	if err != nil {
		log.WithError(err).Fatalf("Application could not start, error=[%s]\n", err)
	}
}

func makeCheck(service baseftrwapp.Service, cd *cmneo4j.Driver) fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Cannot read/write content via this writer",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://runbooks.in.ft.com/upp-content-rw-neo4j",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with something written to it", cd),
		Checker:          func() (string, error) { return "", service.Check() },
	}
}
