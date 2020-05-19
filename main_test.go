package main

import (
	"testing"
)

func TestReadLambdaConfig(t *testing.T) {
	expectedResult, err := readLambdaConfig()
	if err == nil {
		t.Error("readLambdaConfig() expected return an error")
	}

	if expectedResult != nil {
		t.Error("readLambdaConfig() expected not to return any data, got", expectedResult)
	}
}

type getHostFlagsTestCaseInput struct {
	host     string
	port     string
	fileName string
}

func TestGetHostFlags(t *testing.T) {
	testCases := []getHostFlagsTestCaseInput{
		{"127.0.0.1", "80", "file.txt"},
		{"", "80", "file.txt"},
		{"127.0.0.1", "", "file.txt"},
		{"127.0.0.1", "80", ""},
		{"127", "80", ""},
		{"127.0.0.1", "abc", ""},
		{"", "", ""},
	}

	for _, testCase := range testCases {
		expectedResult, err := getHostFlags(testCase.host, testCase.port, testCase.fileName)

		if expectedResult != nil {
			t.Error("getHostFlags(", testCase.host, ",", testCase.port, ",", testCase.fileName, ") expected return no data, returned", expectedResult)
		}

		if err == nil {
			t.Error("getHostFlags(", testCase.host, ",", testCase.port, ",", testCase.fileName, ") expected return error, returned", err)
		}
	}

}

func TestGetEC2InstanceReservations(t *testing.T) {
	_, err := getEC2InstanceReservations("")

	if err == nil {
		t.Error("getEC2InstanceReservations() expected return error, returned nil")
	}

	_, err = getEC2InstanceReservations("*")

	if err == nil {
		t.Error("getEC2InstanceReservations(\"*\") expected return error, returned nil")
	}

}

func TestGetAppServerInformation(t *testing.T) {

	expectedResult := getAppServerInformation(nil, "tags.yaml", "80")

	if expectedResult == nil {
		t.Error("getAppServerInformation(nil,\"tags.yaml\",\"80\") expect to return empty map")
	}

}

func TestGetApplicationFlags(t *testing.T) {
	_, err := getApplicationFlags()

	if err == nil {
		t.Error("getApplicationFlags() expect to return error, returned nill")
	}
}
