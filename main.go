package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Financial-Times/opa-client-go"

	"github.com/Financial-Times/content-rw-neo4j/v3/policy"

	cli "github.com/jawher/mow.cli"

	"github.com/Financial-Times/base-ft-rw-app-go/v2/baseftrwapp"
	cmneo4j "github.com/Financial-Times/cm-neo4j-driver"
	"github.com/Financial-Times/content-rw-neo4j/v3/content"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger/v2"
)

const (
	appDescription = "A RESTful API for managing content (bare-bones representation as full content is served from MongoDB) in Neo4j"
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
		Value:  "bolt://localhost:7687",
		Desc:   "neoURL must point to a leader node or use neo4j:// scheme, otherwise writes will fail",
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

	dbDriverLogLevel := app.String(cli.StringOpt{
		Name:   "dbDriverLogLevel",
		Value:  "WARN",
		Desc:   "Db's driver logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "DB_DRIVER_LOG_LEVEL",
	})

	opaURL := app.String(cli.StringOpt{
		Name:   "opaURL",
		Desc:   "URL of the policy agent.",
		EnvVar: "OPA_URL",
	})

	opaSpecialContentPolicyPath := app.String(cli.StringOpt{
		Name:   "opaSpecialContentPolicyPath",
		Desc:   "Query path for the special content policy.",
		EnvVar: "OPA_SPECIAL_CONTENT_POLICY_PATH",
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
		dbLog := logger.NewUPPLogger(*appName+"-cmneo4j-driver", *dbDriverLogLevel)

		driver, err := cmneo4j.NewDefaultDriver(context.Background(), *neoURL, dbLog)
		if err != nil {
			log.WithError(err).Fatal("Could not create a new instance of cmneo4j driver")
		}
		defer func(driver *cmneo4j.Driver) {
			err = driver.Close(context.Background())
			if err != nil {
				log.WithError(err).Error("could not close the cmneo4j driver instance.")
			}
		}(driver)

		paths := map[string]string{
			policy.SpecialContentKey: *opaSpecialContentPolicyPath,
		}
		opaClient := opa.NewOpenPolicyAgentClient(*opaURL, paths, opa.WithLogger(log))
		agent := policy.NewOpenPolicyAgent(opaClient, log)

		contentDriver := content.NewContentService(driver, agent, log)

		services := map[string]baseftrwapp.Service{
			"content": contentDriver,
		}

		var checks []fthealth.Check
		for _, service := range services {
			checks = append(checks, makeCheck(service, driver))
		}

		ymlBytes, err := os.ReadFile(*apiYml)
		if err != nil {
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
		BusinessImpact: "Cannot read/write content via this writer",
		Name:           "Check connectivity to Neo4j",
		PanicGuide:     "https://runbooks.in.ft.com/upp-content-rw-neo4j",
		Severity:       1,
		TechnicalSummary: fmt.Sprintf(
			"Cannot connect to Neo4j instance %s with something written to it",
			cd,
		),
		Checker: func() (string, error) { return "", service.Check() },
	}
}
