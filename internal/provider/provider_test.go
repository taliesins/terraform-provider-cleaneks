// package provider

// import (
// 	"encoding/json"
// 	"fmt"
// 	"testing"

// 	"github.com/hashicorp/terraform-plugin-framework/provider"
// 	"github.com/hashicorp/terraform-plugin-framework/providerserver"
// 	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
// 	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
// 	"github.com/hashicorp/terraform-plugin-testing/terraform"
// )

// // testAccProtoV6ProviderFactories are used to instantiate a provider during
// // acceptance testing. The factory function will be invoked for every Terraform
// // CLI command executed to create a provider server to which the CLI can
// // reattach.

// var cleanEksProvider provider.Provider
// var testAccProvider *provider.Provider
// var testAccProviders map[string]*provider.Provider
// var testAccProtoV6ProviderFactories map[string]func() (tfprotov6.ProviderServer, error)

// func init() {
// 	cleanEksProvider = New("0.0.1")()
// 	testAccProvider = &cleanEksProvider
// 	testAccProviders = map[string]*provider.Provider{
// 		"cleaneks": testAccProvider,
// 	}
// 	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 		"cleaneks": providerserver.NewProtocol6WithError(cleanEksProvider),
// 	}
// }

// func testAccPreCheck(t *testing.T) {
// 	// You can add code here to run prior to any test case execution, for example assertions
// 	// about the appropriate environment variables being set are common to see in a pre-check
// 	// function.
// }

// func TestAccProvider(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		IsUnitTest:               true,
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				//PreConfig:
// 				Config: testCaseValidProviderConfiguration(),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					checkState,
// 				),
// 			},
// 		},
// 	})
// }

// func checkState(state *terraform.State) error {
// 	return nil
// }

// func testCaseValidProviderConfiguration() string {
// 	// terraformString := testAccTerraformConfig(">= 1.1.7", ">= 0.0.1")
// 	providerString := testAccProviderConfig("127.0.0.1", "client.authentication.k8s.io/v1beta1", "aws", []string{"eks", "get-token", "--cluster-name", "cluster_name", "--role", "cluster_role"})
// 	resourceString := testAccResourceJobConfig(true, true, true, false)

// 	// return terraformString + providerString + resourceString
// 	return providerString + resourceString
// }

// /*
// func testAccTerraformConfig(requiredVersion string, cleanEksProviderVersion string) string {
// 	return fmt.Sprintf(`
// terraform {

//   # https://github.com/hashicorp/terraform/releases
//   required_version = "%[1]s"

//   required_providers {
//     cleaneks = {
//       source = "taliesins/cleaneks"
//       version = "%[2]s"
//     }
//   }
// }

// `, requiredVersion, cleanEksProviderVersion)
// }
// */

// func testAccProviderConfig(host string, apiVersion string, command string, args []string) string {
// 	argsJson, _ := json.Marshal(args)

// 	return fmt.Sprintf(`
// provider "cleaneks" {
//   host = "%[1]s"
//   exec {
//     api_version = "%[2]s"
//     command     = "%[3]s"
// 	args        = %[4]s
//   }
// }

// `, host, apiVersion, command, argsJson)
// }

// func testAccResourceJobConfig(removeAwsCni bool, removeKubeProxy bool, removeCoreDns bool, importCorednsToHelm bool) string {
// 	return fmt.Sprintf(`
// resource "cleaneks_job" "test" {
//   remove_aws_cni         = %[1]t
//   remove_kube_proxy      = %[2]t
//   remove_core_dns        = %[3]t
//   import_coredns_to_helm = %[4]t
// }

// `, removeAwsCni, removeKubeProxy, removeCoreDns, importCorednsToHelm)
// }
