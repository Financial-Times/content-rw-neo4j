package main

import (
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"

	"time"

	"github.com/Financial-Times/base-ft-rw-app-go/baseftrwapp"
	"github.com/Financial-Times/content-rw-neo4j/v3/content"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jawher/mow.cli"
)

const (
	appName        = "content-rw-neo4j"
	appDescription = "A RESTful API for managing Content (bare bones representation as full content is served from MongoDB) in neo4j"
)

func main() {

	app := cli.App(appName, appDescription)
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

	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "LOG_LEVEL",
	})

	logger.InitLogger(appName, *logLevel)

	app.Action = func() {
		logger.Infof("Application started with args %s", os.Args)

		conf := neoutils.DefaultConnectionConfig()
		conf.BatchSize = *batchSize
		db, err := neoutils.Connect(*neoURL, conf)
		if err != nil {
			logger.Errorf("Could not connect to neo4j, error=[%s]\n", err)
		}

		contentDriver := content.NewCypherContentService(db)
		contentDriver.Initialise()

		services := map[string]baseftrwapp.Service{
			"content": contentDriver,
		}

		var checks []fthealth.Check
		for _, service := range services {
			checks = append(checks, makeCheck(service, db))
		}

		ymlBytes, err := ioutil.ReadFile(*apiYml)
		if err != nil {
			logger.WithField("api-yml", *apiYml).Warn("Failed to read OpenAPI yml file, please confirm the file exists and is not empty.")
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
		logger.Errorf("Application could not start, error=[%s]\n", err)
	}
}

func makeCheck(service baseftrwapp.Service, cr neoutils.CypherRunner) fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Cannot read/write content via this writer",
		Name:             "Check connectivity to Neo4j",
		PanicGuide:       "https://runbooks.in.ft.com/upp-content-rw-neo4j",
		Severity:         1,
		TechnicalSummary: fmt.Sprintf("Cannot connect to Neo4j instance %s with something written to it", cr),
		Checker:          func() (string, error) { return "", service.Check() },
	}
}
