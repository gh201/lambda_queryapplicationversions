package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/yaml.v2"
)

func getApplicationTags(targetAddress string, targetPort string, targetFlagFile string) (map[string]string, error) {
	url := "https://" + targetAddress + ":" + targetPort + "/" + targetFlagFile

	fmt.Println("Reading:", url)

	httpTransportSettings := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: httpTransportSettings, Timeout: time.Second * 5}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("Failed opening url, reason:", err)
		return nil, err
	}

	defer resp.Body.Close()

	htmlResponse, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Failed rading tags file:", err)
		return nil, err
	}

	htmlString := string(htmlResponse)

	if strings.Contains(htmlString, "not found") || strings.Contains(htmlString, "HTTP Status 404") {
		fmt.Println("Accessing", url, "returned 'Page not found'")
		return nil, nil
	}

	yamlMap := make(map[string]string)

	err = yaml.Unmarshal([]byte(htmlResponse), &yamlMap)
	if err != nil {
		log.Fatalf("Failed yaml parsing: %v", err)
	}

	tagmap := make(map[string]string)

	for k, v := range yamlMap {
		tagmap[k] = v
	}

	return tagmap, nil
}

func getEC2InstanceReservations(hostFilter string) ([]*ec2.Reservation, error) {
	fmt.Println("Listing ec2 instances")
	fmt.Println("Filtering hosts with status 'running' AND matching filter:", hostFilter, "in tag 'name")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ec2Client := ec2.New(sess)

	hostFilterCollection := strings.Split(hostFilter, ",")
	awsFilter := make([]*string, 0, 5)

	for _, value := range hostFilterCollection {
		awsFilter = append(awsFilter, aws.String(value))
	}

	ec2Filter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
			{
				Name:   aws.String("tag:Name"),
				Values: awsFilter,
			},
		},
	}

	result, err := ec2Client.DescribeInstances(ec2Filter)

	if err != nil {
		fmt.Println("Failed query ec2 instances:", err)
		return nil, err
	}

	fmt.Println("Criteria matching ec2 instance(-s) found:", len(result.Reservations))
	fmt.Println("Listing ec2 complete")

	return result.Reservations, nil
}

func getNodeInformation(ec2 *ec2.Instance, targetFlagFileName string, targetPort string, output chan map[string]string) {

	nodeInfo := make(map[string]string)

	nodeInfo["InstanceID"] = *ec2.InstanceId
	nodeInfo["InstanceIP"] = *ec2.PrivateIpAddress
	nodeInfo["InstanceLaunchTime"] = ec2.LaunchTime.String()

	for _, tag := range ec2.Tags {
		if *tag.Key == "Name" {
			nodeInfo["InstanceName"] = *tag.Value
			break
		}
	}

	flagFileContent, err := getApplicationTags(*ec2.PrivateIpAddress, targetPort, targetFlagFileName)

	if err != nil {
		fmt.Println("Failed reading target flag file:", err)
	}

	for key, value := range flagFileContent {
		nodeInfo[key] = value
	}

	output <- nodeInfo
}

func queryNodesConcurrently(ec2Reservations []*ec2.Reservation, targetFlagFileName string, targetPort string) *[]map[string]string {

	fmt.Println("Querying nodes")

	channelSize := len(ec2Reservations)

	appServersQueryResults := make(chan map[string]string, channelSize)
	appServersInfo := make([]map[string]string, 0, 15)

	for _, reservation := range ec2Reservations {
		for _, ec2Instance := range reservation.Instances {
			go getNodeInformation(ec2Instance, targetFlagFileName, targetPort, appServersQueryResults)
		}
	}

	for response := range appServersQueryResults {
		channelSize--

		if channelSize > 0 {
			appServersInfo = append(appServersInfo, response)
		} else {
			close(appServersQueryResults)
		}
	}

	fmt.Println("Querying nodes - complete")

	return &appServersInfo
}

type lambdaConfig struct {
	FlagFileName        string
	IncludedHostsFilter string
	TargetPort          string
}

func (c *lambdaConfig) readLambdaConfig() error {

	if len(os.Getenv("FlagFileName")) == 0 {
		return errors.New("Missing lambda parameter 'FlagFileName'")
	}

	if len(os.Getenv("IncludedHostsFilter")) == 0 {
		return errors.New("Missing lambda parameter 'IncludedHostsFilter'")
	}

	if len(os.Getenv("TargetPort")) == 0 {
		return errors.New("Missing lambda parameter 'TargetPort'")
	}

	c.FlagFileName = os.Getenv("FlagFileName")
	c.IncludedHostsFilter = os.Getenv("IncludedHostsFilter")
	c.TargetPort = os.Getenv("TargetPort")

	return nil
}

func getAppServerInfo() (string, error) {

	conf := lambdaConfig{}
	err := conf.readLambdaConfig()

	if err != nil {
		return "", err
	}

	ec2Instances, err := getEC2InstanceReservations(conf.IncludedHostsFilter)

	if err != nil {
		return "", err
	}

	appServersInfo := queryNodesConcurrently(ec2Instances, conf.FlagFileName, conf.TargetPort)

	jsonData, err := json.Marshal(*appServersInfo)

	if err != nil {
		fmt.Println("Can't convert to JSON:", err)
		return "", err
	}

	return string(jsonData), nil
}

// HandleRequest - Lambda Handler
func HandleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Println("Enter Lambda handler")

	jsonData, err := getAppServerInfo()

	if err != nil {

		fmt.Println(err)

		apiResponse := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}

		return apiResponse, nil
	}

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/json; charset=utf-8"},
		Body:       fmt.Sprintln(jsonData),
	}

	fmt.Println("Exit Lambda handler")

	return res, nil
}

func main() {
	lambda.Start(HandleRequest)
}
