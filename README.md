# TFC / TFE Plan Output

Simple tool to generate a json plan file from Terraform Cloud or Terraform Enterprise
that can be used with Conftest.

## Usage

Set the following envrionment variables that corresponds to the Terraform clour or Terraform
enterprise environment.

TFC_ORG: Name of your TFC organization
TFC_WORKSPACE: Name of the workspce which you would like to run the plan for
TFC_TOKEN: Valid API token enabling the ability to (run plans)[https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions]

**Note:** `tfc-plan` does not push the current version of your terraform configuration to TFC or TFE, if you are running locally you will need to run a 
`terraform plan` before this command.

To download a plan you can use the following command to run a plan in TFC and download it as a json file.

```shell
tfc-plan --out ./plan.json
```

Given you have the following rego policy that tests for properties of a Kubernetes deployment created
with Terraform.

**policy/test.rego**
```rego
package main

deployments := [deploy |
  deploy := input.planned_values.root_module.resources[_]
  deploy.type == "kubernetes_deployment"
  deploy.name == "minecraft"
]

deny[msg] {
  count(deployments) != 1
  msg := sprintf("there should be a deployment called minecraft: %v",[deployments])
}

deny[msg] {
  images := [image | 
    image := deployments[_].values.spec[_].template[_].spec[_].container[_]
    image.image == "hashicraft/minecraft:v1.20.1-fabric"
  ]

  count(images) != 1

  msg := sprintf("the deployment should have a container using the minecraft image: %v",[images])
}
```

You can run conftest using the following command

```shell
conftest -p ./policy ./plan.json
```

## Example Github Action

```yaml
name: "Terraform Test"

on:
  push:
    branches:
      - prod
      - test
      - dev

env:
  TF_CLOUD_ORGANIZATION: "${{ secrets.TF_CLOUD_ORG }}"
  TF_API_TOKEN: "${{ secrets.TF_API_TOKEN }}"
  CONFIG_DIRECTORY: "./terraform/gcp/app"

  test_policy:
    name: "Test Terraform Configuration For Deployment"
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Upload Configuration
        uses: hashicorp/tfc-workflows-github/actions/upload-configuration@v1.0.0
        id: apply-upload
        with:
          workspace: app-${{ github.ref_name }}
          directory: ${{ env.CONFIG_DIRECTORY }}
      
      - name: Download TFC Plan
        run: |
          wget  https://github.com/nicholasjackson/tfc-plan/releases/download/v0.0.3/binary-linux-amd64
          mv ./binary-linux-amd64 /usr/local/bin/tfc-plan
          chmod +x /usr/local/bin/tfc-plan
        
      - name: Download Conftest
        run: |
          LATEST_VERSION=$(wget -O - "https://api.github.com/repos/open-policy-agent/conftest/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)
          wget "https://github.com/open-policy-agent/conftest/releases/download/v${LATEST_VERSION}/conftest_${LATEST_VERSION}_Linux_x86_64.tar.gz"
          tar xzf conftest_${LATEST_VERSION}_Linux_x86_64.tar.gz
          sudo mv conftest /usr/local/bin

      - name: Get Plan And Output JSON
        id: plan-run
        run: |
          tfc-plan --out ${CONFIG_DIRECTORY}/app-plan.json
        env:
          TFC_ORG: ${{ secrets.TF_CLOUD_ORG }} 
          TFC_WORKSPACE: app-${{ github.ref_name }}
          TFC_TOKEN: ${{ secrets.TF_API_TOKEN }} 

      - name: Run Conftest
        run: |
          conftest test -p ${CONFIG_DIRECTORY}/policy ${CONFIG_DIRECTORY}/app-plan.json
```
