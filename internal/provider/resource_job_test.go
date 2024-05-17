// package provider

// import (
// 	"testing"

// 	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
// )

// func TestAccExampleResource(t *testing.T) {
// 	resource.Test(t, resource.TestCase{
// 		IsUnitTest:               true,
// 		PreCheck:                 func() { testAccPreCheck(t) },
// 		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				//PreConfig:
// 				Config: testAccResourceJobConfig(true, true, true, false),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("cleaneks_job.test", "remove_aws_cni", "true"),
// 					resource.TestCheckResourceAttr("cleaneks_job.test", "remove_kube_proxy", "true"),
// 					resource.TestCheckResourceAttr("cleaneks_job.test", "remove_core_dns", "true"),
// 					resource.TestCheckResourceAttr("cleaneks_job.test", "import_coredns_to_helm", "false"),
// 				),
// 			},
// 		},
// 	})
// }
