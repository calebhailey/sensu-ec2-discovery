package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type CheckConfig struct {
	sensu.PluginConfig
	ec2InstanceStates          string
	ec2InstanceRegions         string
	ec2InstanceTags            string
	ec2Filters                 []*ec2.Filter
	sensuNamespace             string
	sensuApiUrl                string
	sensuAccessToken           string
	sensuTrustedCaFile         string
	sensuInsecureSkipTlsVerify string
	sensuAPIKey                string
}

var (
	config = CheckConfig{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-ec2-discovery",
			Short:    "Auto-discover EC2 instances and update your Sensu Go registry.",
			Keyspace: "sensu.io/plugins/ec2-discovery",
		},
	}

	ec2DiscoveryConfigOptions = []*sensu.PluginConfigOption{
		{
			Path:      "ec2-instance-states",
			Env:       "EC2_INSTANCE_STATES",
			Argument:  "ec2-instance-states",
			Shorthand: "s",
			Usage:     "The AWS EC2 instance states to discover. Can also be set via the $EC2_INSTANCE_STATES environment variable.",
			Value:     &config.ec2InstanceStates,
			Default:   "pending,running,rebooting",
		},
		{
			Path:      "ec2-instance-regions",
			Env:       "EC2_INSTANCE_REGIONS",
			Argument:  "ec2-instance-regions",
			Shorthand: "r",
			Usage:     "The AWS EC2 region(s) to discover. Can also be set via the $EC2_INSTANCE_REGIONS environment variable. REQUIRED.",
			Value:     &config.ec2InstanceRegions,
			Default:   "",
		},
		{
			Path:      "ec2-instance-tags",
			Env:       "EC2_INSTANCE_TAGS",
			Argument:  "ec2-instance-tags",
			Shorthand: "t",
			Usage:     "The AWS EC2 instance tags to discover. Can also be set via the $EC2_INSTANCE_TAGS environment variable.",
			Value:     &config.ec2InstanceTags,
			Default:   "",
		},
		{
			Path:      "sensu-namespace",
			Env:       "SENSU_NAMESPACE",
			Argument:  "sensu-namespace",
			Shorthand: "n",
			Usage:     "The Sensu Go Namespace to register entities in. Can also be set via the $SENSU_NAMESPACE environment variable.",
			Value:     &config.sensuNamespace,
			Default:   "default",
		},
		{
			Path:      "sensu-api-url",
			Env:       "SENSU_API_URL",
			Argument:  "sensu-api-url",
			Shorthand: "u",
			Usage:     "The Sensu Go API URL. Can also be set via the $SENSU_API_URL environment variable.",
			Value:     &config.sensuApiUrl,
			Default:   "https://127.0.0.1:8080",
		},
		{
			Path:      "sensu-access-token",
			Env:       "SENSU_ACCESS_TOKEN",
			Argument:  "sensu-access-token",
			Shorthand: "T",
			Usage:     "The Sensu Go API access token. Can also be set via the $SENSU_ACCESS_TOKEN environment variable.",
			Value:     &config.sensuAccessToken,
			Secret:    true,
			Default:   "",
		},
		{
			Path:      "sensu-api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "sensu-api-key",
			Shorthand: "k",
			Usage:     "The Sensu Go API access key. Can also be set via the $SENSU_API_KEY environment variable.",
			Value:     &config.sensuAPIKey,
			Secret:    true,
			Default:   "",
		},
		{
			Path:      "sensu-trusted-ca-file",
			Env:       "SENSU_TRUSTED_CA_FILE",
			Argument:  "sensu-trusted-ca-file",
			Shorthand: "c",
			Usage:     "TLS CA certificate bundle in PEM format.",
			Value:     &config.sensuTrustedCaFile,
			Default:   "",
		},
		{
			Path:      "sensu-insecure-tls-skip-verify",
			Env:       "SENSU_INSECURE_SKIP_TLS_VERIFY",
			Argument:  "sensu-insecure-tls-skip-verify",
			Shorthand: "i",
			Usage:     "Skip TLS certificate verification (not recommended!)",
			Value:     &config.sensuInsecureSkipTlsVerify,
			Default:   "false",
		},
	}
)

func main() {
	check := sensu.NewGoCheck(
		&config.PluginConfig,
		ec2DiscoveryConfigOptions,
		validateArgs,
		discoverInstances,
		false)
	check.Execute()
}

func validateArgs(event *corev2.Event) (int, error) {
	if len(config.sensuAccessToken) == 0 && len(config.sensuAPIKey) == 0 {
		log.Fatalf("ERROR: no Sensu API access token or key provided. Exiting.")
		return sensu.CheckStateCritical, fmt.Errorf("No Sensu API access token or key provided. Exiting.")
	}

	if len(config.ec2InstanceRegions) == 0 {
		log.Fatalf("ERROR: no EC2 instance regions provided. Exiting.")
		return sensu.CheckStateCritical, fmt.Errorf("No EC2 instance regions provided. Exiting.")
	}

	err := createFilters()
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
		return sensu.CheckStateCritical, err
	}

	return sensu.CheckStateOK, nil
}

func createFilters() error {
	var states []string
	var tags []string

	if len(config.ec2InstanceStates) > 0 {
		states = strings.Split(config.ec2InstanceStates, ",")
		config.ec2Filters = append(config.ec2Filters, &ec2.Filter{
			Name:   aws.String("instance-state-name"),
			Values: aws.StringSlice(states),
		})
	}

	if len(config.ec2InstanceTags) > 0 {
		tags = strings.Split(config.ec2InstanceTags, ",")
		for _, tag := range tags {
			tagPair := strings.Split(tag, "=")
			filter := &ec2.Filter{
				Name:   aws.String(strings.Join([]string{"tag", tagPair[0]}, ":")),
				Values: []*string{aws.String(tagPair[1])},
			}
			config.ec2Filters = append(config.ec2Filters, filter)
		}
	}

	return nil
}

func loadCACerts(path string) (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("ERROR: failed to load system cert pool: %s", err)
		return nil, err
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if path != "" {
		certs, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("ERROR: failed to read CA file (%s): %s", path, err)
			return nil, err
		} else {
			rootCAs.AppendCertsFromPEM(certs)
		}
	}
	return rootCAs, nil
}

func initHttpClient() *http.Client {
	client := &http.Client{
		Transport: http.DefaultTransport,
	}

	if len(config.sensuTrustedCaFile) > 0 {
		certs, err := loadCACerts(config.sensuTrustedCaFile)
		if err != nil {
			log.Fatalf("ERROR: %s\n", err)
		}
		tlsConfig := &tls.Config{
			RootCAs: certs,
		}
		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}
	// sensuInsecureSkipTlsVerify is a string as it comes in from the
	// sensuctl env
	skipVerify, _ := strconv.ParseBool(config.sensuInsecureSkipTlsVerify)
	if skipVerify {
		if transport, ok := client.Transport.(*http.Transport); ok {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = new(tls.Config)
			}
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}
	return client
}

func registerInstance(instance *ec2.Instance) {
	var entity corev2.Entity
	entity.Name = *instance.InstanceId
	entity.Namespace = config.sensuNamespace
	entity.EntityClass = "proxy"
	entity.Labels = make(map[string]string)
	for _, tag := range instance.Tags {
		entity.Labels[*tag.Key] = *tag.Value
	}

	// fmt.Printf("%s\n", entity.Name)

	postBody, err := json.Marshal(entity)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	body := bytes.NewReader(postBody)
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/core/v2/namespaces/%s/entities",
			config.sensuApiUrl,
			entity.Namespace,
		),
		body,
	)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	var httpClient *http.Client = initHttpClient()
	if len(config.sensuAccessToken) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.sensuAccessToken))
	} else if len(config.sensuAPIKey) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Key %s", config.sensuAPIKey))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("ERROR: %s\n", err)
	} else if resp.StatusCode == http.StatusNotFound {
		log.Fatalf("ERROR: %v %s (%s)\n", resp.StatusCode, http.StatusText(resp.StatusCode), req.URL)
	} else if resp.StatusCode == http.StatusConflict {
		log.Printf("INFO: entity \"%s\" already exists (%v: %s)\n", entity.Name, resp.StatusCode, http.StatusText(resp.StatusCode))
	} else if resp.StatusCode >= http.StatusMultipleChoices {
		log.Fatalf("ERROR: %v %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	} else if resp.StatusCode == http.StatusCreated {
		log.Printf("INFO: registered entity for EC2 instance \"%s\"", entity.Name)
	} else {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("ERROR: %s\n", err)
		} else {
			fmt.Printf("%s\n", string(b))
		}
	}

	return
}

// Usage: instancesByRegion -api <url> -state <value> [-state value...] [-region region...] [-tag key=value...]
func discoverInstances(event *corev2.Event) (int, error) {
	for _, region := range strings.Split(config.ec2InstanceRegions, ",") {
		aws_session := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		svc := ec2.New(aws_session)

		params := &ec2.DescribeInstancesInput{Filters: config.ec2Filters}
		result, err := svc.DescribeInstances(params)
		if err != nil {
			log.Fatalf("ERROR: %s\n", err)
			return sensu.CheckStateCritical, err
		} else {
			for _, reservation := range result.Reservations {
				for _, instance := range reservation.Instances {
					registerInstance(instance)
				}
			}
		}
	}
	return sensu.CheckStateOK, nil
}
