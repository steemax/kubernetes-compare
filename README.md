# Kubernetes Configuration Comparison Tool

## Overview

This application is designed to compare the configuration of selected resources (Deployments, DaemonSets, Services, Traefik Ingress Routes, Helm Values for installed releases) in two different Kubernetes clusters. It aims to streamline the process of identifying differences between configurations, which is particularly useful in complex environments.

## How It Works

1.  **Choose Kubernetes Config Source**: At the beginning, users need to select the source of the Kubernetes config file. Two options are available:
    
    -   Internal. Use the config file bundled with the application (the application will look for it in `./conf/kubernetes`, any files in this directiry with *.kubeconfig names).
    -   Use the config file from the user's directory (`~/.kube/config`).
    - External. (not available now)
2.  **Select Clusters**: After defining the source of the kubeconfig, users must select two clusters to compare.
    
3.  **Choose Namespaces**: On the next step, users have to select the namespaces they want to compare.
    
4.  **Select Resources**: Then, the choice of resources to be compared is available.
    
5.  **View Report**: The final report is generated as a table of differences, analyzing the specs of the selected resource types and highlighting the discrepancies. The differences are displayed in a clear and concise format.
    
6.  **Navigate to Full Specifications**: Users also have the option to navigate from the differences page to a page containing the full specifications, sorted by names, for both clusters.
    

## Features
- **GitLab OpenID Authorization**: The application integrates with GitLab using OpenID Connect (OIDC) for user authentication, ensuring secure access and alignment with existing identity management.
-   **Comparison of Various Resources**: Compare configurations of Deployments, DaemonSets, Services, Traefik Ingress Routes, and Helm Values for installed releases.
    
-   **Intuitive Selection Process**: Easily select the source of Kubernetes config, clusters, namespaces, and resources to compare.
    
-   **Detailed Difference Report**: View a tabular report showing the differences in configurations.
    
-   **Full Specifications View**: Access a page containing full specifications for both clusters, sorted by names.

## Why Use This Tool?

In today's complex Kubernetes environments, understanding and managing configurations across different clusters can be a challenging task. This tool simplifies the comparison process by providing an easy-to-use interface and detailed reporting capabilities. It enables DevOps, SREs, and Kubernetes administrators to quickly identify differences in configurations, reducing the risk of inconsistencies and aiding in troubleshooting and compliance verification.


## Installation & Deployment Options

You can use this application in various ways, depending on your preferences and requirements:

### Kubernetes Environment

Deploy the application within a Kubernetes environment, taking advantage of native orchestration and scaling capabilities.

### Standalone Container

Run it as a standalone container using containerization tools like Docker or Podman. It provides a contained environment for the application, ensuring compatibility and easy deployment.

### Executable File

You also have the option to run it as an executable file on your personal computer. It's a convenient way to use the application without any containerization or virtualization overhead.

### Configuration

You'll need to create a configuration file (./conf/config.json) in JSON format with the following content:

{
	"server_port": 8080,
	"gitlab_server": "https://your_gitlab_server_url",
	"gitlab_skip_tls_verify": true,
	"client_id": "id",
	"client_secret": "secret",
	"callback_url": "http://localhost:8080/auth/callback",
	"max_age_session_token": 15,
	"auth_group_name_allowed": "compare"
}

**Configuration Explanation:**

-   **server_port**: The port on which the application will run.
-   **gitlab_server**: The GitLab server for OIDC authorization integration.
-   **gitlab_skip_tls_verify**: Ignore SSL certificate verification on the GitLab server.
-   **client_id** and **client_secret**: Required for the authorization application created in GitLab.
-   **callback_url**: The URL to which the response should return after authorization.
-   **max_age_session_token**: The lifetime of the authorization token (in minutes).
-   **auth_group_name_allowed**: The GitLab group that users must belong to for successful authorization.

These diverse deployment options and configurable parameters provide flexibility, making it adaptable to various use cases and environments.
