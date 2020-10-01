package main

import (
        "testing"

        "github.com/sensu-community/sensu-plugin-sdk/sensu"
        corev2 "github.com/sensu/sensu-go/api/core/v2"
        "github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
}

func TestValidateArgs(t *testing.T) {
	event := corev2.FixtureEvent("entity1", "check")
	assert := assert.New(t)

	ret, err := validateArgs(event)
	assert.Error(err)
	assert.Equal(sensu.CheckStateCritical, ret)
	config.sensuAccessToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDE1NTUzOTEsImp0aSI6ImMyNjJhMTNkYzY0ZWRkNmZhMzZlNTRjYjA4YjY4MDdiIiwiaXNzIjoiaHR0cHM6Ly9iYWNrZW5kOjgwODAiLCJzdWIiOiJhZG1pbiIsImdyb3VwcyI6WyJjbHVzdGVyLWFkbWlucyIsInN5c3RlbTp1c2VycyJdLCJwcm92aWRlciI6eyJwcm92aWRlcl9pZCI6ImJhc2ljIiwicHJvdmlkZXJfdHlwZSI6IiIsInVzZXJfaWQiOiJhZG1pbiJ9LCJhcGlfa2V5IjpmYWxzZX0.QG81zM4GnmZwf_gvGW_NNIoY84czK-0bIaK_a9AZoO0"
	ret, err = validateArgs(event)
	assert.Error(err)
	assert.Equal(sensu.CheckStateCritical, ret)
	config.ec2InstanceRegions = "us-west-2,us-east-1"
	ret, err = validateArgs(event)
	assert.NoError(err)
	assert.Equal(sensu.CheckStateOK, ret)
}

