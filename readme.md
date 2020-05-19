# Lambda function

Filters out 

# Requirements

Lambda expected to have parameters:
 * FlagFileName - filename or path where application tags are returned in yaml format (e.g. tags.yaml)
 * IncludedHostsFilter - comma separated filters, online EC2 instances mathcing pattern in name will be returned only (e.g. *front*,*back*,*api*)
 * TargetPort - port, target is listening on (eg. 80)

Target node:
 * expected return yaml at https://ec2_instance_ip:[TargetPort]/[FlagFileName]

# Compiling application code
1. Machine, compiling code must contain GO installed 
2. Include dependencies:

Compile on windows:
set GOOS=linux
go build -o main main.go
%USERPROFILE%\Go\bin\build-lambda-zip.exe -output main.zip main

Compile on linux:
GOOS=linux
go build main.go
zip lambda_payload.zip main


# Running code
Upload lambda payload and trigger lambda

# Expected output example:
[
    {
        "InstanceID": "i-fffffffffffffffff",
        "InstanceIP": "x.x.x.x",
        "InstanceName": "frontend-srv1",
        "custom_flag1": "v1",
        "custom_flag2": "12",
    },
]

where
    InstanceID - always part of response and is equal AWS instance ID
    InstanceIP - always part of response and is equal AWS instance IP
    InstanceName - always part of response and is equal AWS instance tag 'Name'
    Rest of flags - is content returned by https://ec2_instance_ip:[TargetPort]/[FlagFileName] uri
