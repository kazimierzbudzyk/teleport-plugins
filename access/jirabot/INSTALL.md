# Teleport Jira Plugin Setup Quickstart

If you're using Jira Cloud or Jira Server to manage your projects, you can also use it to monitor, approve, deny, or discuss Teleport permission requests. This quickstart will walk you through the setup.

For the purpose of this quickstart, we assume you've already setup an [Enterprise Teleport Cluster](https://gravitational.com/teleport/docs/enterprise/quickstart-enterprise/). 

Note: The Approval Workflow only works with Pro and Enterprise version of Teleport

## Prerequisites
- An Enterprise or Pro Teleport Cluster
- Admin Privileges with access and control of [`tctl`](https://gravitational.com/teleport/docs/cli-docs/#tctl)
- Jira Server or Jira Cloud installation with an owner privileges, specifically to setup webhooks, issue types, and workflows.

### Create an access-plugin role and user within Teleport 
First off, using an exsiting Teleport Cluster, we are going to create a new Teleport User and Role to access Teleport.

#### Create User and Role for access. 
Log into Teleport Authenticaiont Server, this is where you normally run `tctl`. Don't change the username and the role name, it should be `access-plugin` for the plugin to work correctly.

_Note: if you're using other plugins, you might want to create different users and roles for different plugins_.

```
$ cat > rscs.yaml <<EOF
kind: user
metadata:
  name: access-plugin
spec:
  roles: ['access-plugin']
version: v2
---
kind: role
metadata:
  name: access-plugin
spec:
  allow:
    rules:
      - resources: ['access_request']
        verbs: ['list','read','update']
    # teleport currently refuses to issue certs for a user with 0 logins,
    # this restriction may be lifted in future versions.
    logins: ['access-plugin']
version: v3
EOF

# ...
$ tctl create -f rscs.yaml
```

#### Export access-plugin Certificate
Teleport Plugin uses the `access-plugin`role and user to peform the approval. We export the identify files, using [`tctl auth sign`](https://gravitational.com/teleport/docs/cli-docs/#tctl-auth-sign).

```
$ tctl auth sign --format=tls --user=access-plugin --out=auth --ttl=8760h
# ...
```

The above sequence should result in three PEM encoded files being generated: auth.crt, auth.key, and auth.cas (certificate, private key, and CA certs respectively).  We'll reference these later when [configuring Teleport-Plugins](#configuration-file).

_Note: by default, tctl auth sign produces certificates with a relatively short lifetime. For production deployments, the --ttl flag can be used to ensure a more practical certificate lifetime. --ttl=8760h exports a 1 year token_

### Setting up your Jira Project

#### Creating the permission management project
All new permission requests are going to show up in a project you choose. We recommend that you create a separate project for permissions management, and a new board in said project.

You'll need the project Jira key to configure the plugin.

#### Setting up the status board
Create a new board for tasks in the permission management project. The board can have as many different columns (task statuses) as you want, but it must have at least these tho exactly matching: 
1. Approved
2. Denied

Teleport Jira Plugin will create a new issue for each new permission request in the first available column on the board. When you drag the request task to Approved column on Jira, the request will be appvored. Ditto for Denied column on Jira.

#### Setting up Request ID field on Jira
Teleport Jira Plugin requires a custom issue field to be created. 
Go to your Jira Project settings -> Issue Types -> Select type `Task` -> add a new Short Text field named `TeleportAccessRequestId`. 
Teleport uses this field to reference it's internal request ID. If anyone changes this field on Jira, or tries to forge the permission request, Teleport will validate it and ignore it.

#### Getting your Jira API token

If you're using Jira Cloud, navigate to [Account Settings -> Security -> API Tokens](https://id.atlassian.com/manage/api-tokens) and create a new app specific API token in your Jira installation.
You'll need this token later to configure the plugin.

For Jira Server, the URL of the API tokens page will be different depending on your installation.


#### Settings up Jira Webhooks

Go to Settings -> General -> System -> Webhooks and create a new Webhook for Jira to tell the Teleport Plugin about updates. 

For the webhook URL, use the URL that you'll run the plugin on. It needs to be a publicly accessible URL that we'll later setup.
Jira requires the webhook listener to run over HTTPS.

The Teleport Jira plugin webhook needs to be notified only about new issues being created, issues being updated, or deleted. You can leave all the other boxes empty.

_Note: by default, Jira Webhook will send updates about any issues in any projects in your Jira installation to the webhook. 
We suggest that you use JQL filters to limit which issues are being sent to the plugin._

_Note: by default, the Plugin's web server will run with TLS, but you can disable it with `--insecure-no-tls` to test things out in a dev environment._

In the webhook settings page, make sure that the webhook will only send Issue Updated updates. It's not critical if anything else gets sent, the plugin will just ignore everything else.

## Installing

To start using Teleport Plugins, you will need to Download the binaries and the license file from the customer portal. After downloading the binary tarball, run:

```bash
$ wget https://get.gravitational.com/teleport-jirabot-v0.0.1-linux-amd64-bin.tar.gz
$ tar -xzf teleport-jirabot-v0.0.1-linux-amd64-bin.tar.gz
$ cd teleport-jirabot
$ ./install
$ which teleport-jirabot
/usr/local/bin/teleport-jirabot
```

### Configuration file

You can now run `sudo teleport-jirabot configure > /etc/teleport-jirabot.toml`, or copy and paste the following template. 

By default, Jira Teleport Plugin will use a config in `/etc/teleport-jirabot.toml`, and you can override it with `-c config/file/path.toml` flag.

```toml
# example jirabot configuration TOML file
[teleport]
auth-server = "example.com:3025"  # Auth GRPC API address
client-key = "/var/lib/teleport/plugins/jirabot/auth.key" # Teleport GRPC client secret key
client-crt = "/var/lib/teleport/plugins/jirabot/auth.crt" # Teleport GRPC client certificate
root-cas = "/var/lib/teleport/plugins/jirabot/auth.cas"   # Teleport cluster CA certs

[jira]
url = "https://example.com/jira"    # JIRA URL. For JIRA Cloud, https://[my-jira].atlassian.net
username = "bot@example.com"        # JIRA username
api-token = "token"                 # JIRA API Basic Auth token
project = "MYPROJ"                  # JIRA Project key

[http]
listen = ":8081"          # JIRA webhook listener
# host = "example.com"    # Host name by which bot is accessible
# https-key-file = "/var/lib/teleport/plugins/jirabot/server.key"  # TLS private key
# https-cert-file = "/var/lib/teleport/plugins/jirabot/server.crt" # TLS certificate

[log]
output = "stderr" # Logger output. Could be "stdout", "stderr" or "/var/lib/teleport/jirabot.log"
severity = "INFO" # Logger severity. Could be "INFO", "ERROR", "DEBUG" or "WARN"`

```

The `[teleport]` section describes where is the teleport service running, and what keys should the plugin use to authenticate itself. Use the keys that you've generated [above in exporting your Certificate section](#Export access-plugin Certificate).

The `[jira]` section requires a few things: 
1. Your Jira Cloud or Jira Server URL. For Jira Cloud, it looks something like yourcompany.atlassian.net. 
2. Your username on Jira, i.e. ben@gravitational.com
3. Your Jira API token that you've created above. 
4. And the Jira Project key, available in Project settings. 

`[http]` setting block describes how the Plugin's HTTP server works. The HTTP server is responsible for listening for updates from Jira, and processing updates, like when someone drags a task from Inbox to Approved column. 

You must provide an address the server should listen on, and a certificate to use, unless you plan on running with `--insecure-no-tls`, which we don't recommend in production. 


## Testing

You should be able to run the Teleport plugin now! 

```bash
teleport-jirabot start
```

The log output should look familiar to what Teleport service logs. You should see that it connected to Teleport, and is listening for new Teleport requests and Jira webhooks. 

Go ahead and test it: 

```bash
tsh login --request-roles=admin
```

That should create a new permission request on Teleport (you can test if it did with `tctl request ls` ), and you should see a new task on your Jira project board.

## FAQ