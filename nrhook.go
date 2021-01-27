package logrusnrhook

import (
	"bytes"
	"compress/gzip"
	"log"
	"net/http"

	"github.com/sethgrid/pester"
	"github.com/sirupsen/logrus"
)

const (
	stdEndpoint = "https://log-api.newrelic.com/log/v1"
	euEndpoint  = "https://log-api.eu.newrelic.com/log/v1"
)

type NrHook struct {
	client      *pester.Client
	application string
	licenseKey  string
	endpoint    string
}

func NewNrHook(appName string, license string, eu bool) *NrHook {
	nrHook := &NrHook{
		client:      pester.New(),
		application: appName,
		licenseKey:  license,
		endpoint:    stdEndpoint,
	}
	if eu {
		nrHook.endpoint = euEndpoint
	}

	return nrHook
}

func (h *NrHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		log.Printf("NrHook failed to fire. Unable to read entry, %v", err)
		return err
	}

	// fire and forget
	go func(line string) {
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		if _, err := writer.Write([]byte(line)); err != nil {
			log.Printf("failed to gzip message: %v", err)
		}
		if err := writer.Flush(); err != nil {
			log.Printf("error flushing gzip writer, %v", err)
		}
		if err := writer.Close(); err != nil {
			log.Printf("error flushing gzip writer, %v", err)
		}

		request, err := http.NewRequest("POST", h.endpoint, &buffer)
		if err != nil {
			log.Printf("error creating log request to NR: %v", err)
		}

		request.Header.Add("Content-Type", "application/gzip")
		request.Header.Add("Content-Encoding", "gzip")
		request.Header.Add("Accept", "*/*")
		request.Header.Add("X-License-Key", h.licenseKey)

		_, err = h.client.Do(request)
		if err != nil {
			log.Printf("error sending log request to NR: %v", err)
		}

	}(line)

	return nil
}

func (h *NrHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
