# Sensu EC2 Discovery

## Table of Contents
- [Overview](#overview)
- [Installation and usage](#installation-and-usage)
- [AWS credentials](#aws-credentials)
- [Sensu credentials](#sensu-credentials)

## Overview

Sensu EC2 Discovery is a `sensuctl` command plugin that collects AWS EC2
instance data from the EC2 API and registers a Sensu Proxy Entity for each EC2
instance.

## Installation and usage

1. Install the plugin using `sensuctl`:

   ```shell
   sensuctl command install ec2-discovery calebhailey/sensu-ec2-discovery
   ```

2. Verify installation using `sensuctl`:

   ```shell
   sensuctl command exec ec2-discovery -h

   ```

3. Discover EC2 instances using `sensuctl`:

   ```shell
   sensuctl command exec ec2-discovery -- --ec2-instance-regions us-west-2,us-east-1
   ```

##  AWS credentials

This plugin makes use of the AWS SDK for Go.  The SDK uses the [default credential provider chain][1]
to find AWS credentials.  The SDK uses the first provider in the chain that returns credentials
without an error. The default provider chain looks for credentials in the following order:

1. Environment variables (AWS_SECRET_ACCESS_KEY, AWS_ACCESS_KEY_ID, and AWS_REGION).

2. Shared credentials file (typically ~/.aws/credentials).

3. If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

4. If your application uses an ECS task definition or RunTask API operation, IAM role for tasks.

The SDK detects and uses the built-in providers automatically, without requiring manual configurations.
For example, if you use IAM roles for Amazon EC2 instances, your applications automatically use the
instance’s credentials. You don’t need to manually configure credentials in your application.

Source: [Configuring the AWS SDK for Go][2]

If you go the route of using environment variables, it is highly suggested you use them via the
[Env secrets provider][3].

## Sensu credentials

Since this is `sensuctl` command plugin, it will use the sensuctl credentials to connect to
the Sensu API.  If you need to override any of those provided settings they are available as
command-line arguments.

However, if you choose to run the included binary outside of `sensuctl`, possibly as a scheduled
check, it does suport the use of an [API key][4] for accessing the Sensu backend.  And, while
the API key can be provided on the command line, it is advised to use it as an environment
variable (SENSU_API_KEY) that is surfaced by the use of [secrets management][5].

[1]: https://docs.aws.amazon.com/sdk-for-go/api/aws/defaults/#CredChain
[2]: https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
[3]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/#use-env-for-secrets-management
[4]: https://docs.sensu.io/sensu-go/latest/operations/control-access/use-apikeys/#api-key-authentication
[5]: https://docs.sensu.io/sensu-go/latest/guides/secrets-management/

