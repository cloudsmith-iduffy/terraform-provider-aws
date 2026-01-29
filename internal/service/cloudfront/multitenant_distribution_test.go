// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package cloudfront_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	awstypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfcloudfront "github.com/hashicorp/terraform-provider-aws/internal/service/cloudfront"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccCloudFrontMultiTenantDistribution_basic(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "tenant_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tenant_config.0.parameter_definition.0.definition.0.string_schema.0.required", acctest.CtTrue),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfcloudfront.ResourceMultiTenantDistribution, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_comprehensive(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_comprehensive(rName, "Comprehensive multi-tenant distribution test", "index.html", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, names.AttrComment, "Comprehensive multi-tenant distribution test"),
					resource.TestCheckResourceAttr(resourceName, "default_root_object", "index.html"),
					resource.TestCheckResourceAttr(resourceName, "http_version", "http2"),

					// Check connection_mode is computed
					resource.TestCheckResourceAttrSet(resourceName, "connection_mode"),

					// Check multiple origins
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.id", "custom-origin"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.domain_name", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.origin_path", "/api"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_header.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_header.0.header_name", "X-Custom-Header"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_header.0.header_value", "test-value"),

					// Check cache behaviors
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.path_pattern", "/api/*"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.target_origin_id", "custom-origin"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.compress", acctest.CtFalse),

					// Check custom error responses
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.0.error_code", "404"),
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.0.response_code", "200"),
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.0.response_page_path", "/404.html"),

					// Check tags
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.Environment", "test"),
					resource.TestCheckResourceAttr(resourceName, "tags.Name", rName),

					// Check tenant config with single parameter
					resource.TestCheckResourceAttr(resourceName, "tenant_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "tenant_config.0.parameter_definition.#", "1"),
				),
			},
			{
				Config: testAccMultiTenantDistributionConfig_comprehensive(rName, "Updated comprehensive test", "updated.html", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, names.AttrComment, "Updated comprehensive test"),
					resource.TestCheckResourceAttr(resourceName, "default_root_object", "updated.html"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.compress", acctest.CtTrue),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_s3OriginWithOAC(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_s3OriginWithOAC(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.origin_access_control_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_tags(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_tags(map[string]string{acctest.CtKey1: acctest.CtValue1}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey1, acctest.CtValue1),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
			{
				Config: testAccMultiTenantDistributionConfig_tags(map[string]string{acctest.CtKey1: acctest.CtValue1Updated, acctest.CtKey2: acctest.CtValue2}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "2"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey1, acctest.CtValue1Updated),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
				),
			},
			{
				Config: testAccMultiTenantDistributionConfig_tags(map[string]string{acctest.CtKey2: acctest.CtValue2}),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
				),
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_update(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_update("Initial comment", "http1.1", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, names.AttrComment, "Initial comment"),
					resource.TestCheckResourceAttr(resourceName, "http_version", "http1.1"),
				),
			},
			{
				Config: testAccMultiTenantDistributionConfig_update("Updated comment", "http2", "updated.html"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, names.AttrComment, "Updated comment"),
					resource.TestCheckResourceAttr(resourceName, "http_version", "http2"),
					resource.TestCheckResourceAttr(resourceName, "default_root_object", "updated.html"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_optionalTenantConfig(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_noTenantConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "tenant_config.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_enabled(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_enabled(false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
			{
				Config: testAccMultiTenantDistributionConfig_enabled(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtTrue),
				),
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_originGroups(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_originGroups(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin_group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin_group.0.member.#", "2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_originShield(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_originShield(true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.origin_shield.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.origin_shield.0.enabled", acctest.CtTrue),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_httpVersion(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_httpVersion("http2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "http_version", "http2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
			{
				Config: testAccMultiTenantDistributionConfig_httpVersion("http3"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "http_version", "http3"),
				),
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_responseCompletionTimeout(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_responseCompletionTimeout(10),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.response_completion_timeout", "10"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
			{
				Config: testAccMultiTenantDistributionConfig_responseCompletionTimeout(60),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.response_completion_timeout", "60"),
				),
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_multipleCustomHeaders(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_multipleCustomHeaders(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					// Verify we have 3 custom headers (order doesn't matter with sets)
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_header.#", "3"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_customOrigin(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_customOrigin(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_origin_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_origin_config.0.http_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_origin_config.0.https_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.custom_origin_config.0.origin_protocol_policy", "http-only"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_orderedCacheBehavior(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_orderedCacheBehavior(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.path_pattern", "images1/*.jpg"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.1.path_pattern", "images2/*.jpg"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_orderedCacheBehaviorCachePolicy(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_orderedCacheBehaviorCachePolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.path_pattern", "images/*.jpg"),
					resource.TestCheckResourceAttrSet(resourceName, "cache_behavior.0.cache_policy_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_orderedCacheBehaviorResponseHeadersPolicy(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_orderedCacheBehaviorResponseHeadersPolicy(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.0.path_pattern", "images/*.jpg"),
					resource.TestCheckResourceAttrSet(resourceName, "cache_behavior.0.response_headers_policy_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_s3Origin(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_s3Origin(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.domain_name"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_Origin_connectionAttempts(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_originConnectionAttempts(2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.connection_attempts", "2"),
				),
			},
			{
				Config: testAccMultiTenantDistributionConfig_originConnectionAttempts(1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.connection_attempts", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_Origin_connectionTimeout(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_originConnectionTimeout(5),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.connection_timeout", "5"),
				),
			},
			{
				Config: testAccMultiTenantDistributionConfig_originConnectionTimeout(15),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.0.connection_timeout", "15"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_Origin_originAccessControl(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_originAccessControl(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.origin_access_control_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_vpcOriginConfig(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_vpcOriginConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.vpc_origin_config.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.vpc_origin_config.0.vpc_origin_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_vpcOriginConfigOwnerAccountID(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_vpcOriginConfigOwnerAccountID(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "origin.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "origin.0.vpc_origin_config.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.vpc_origin_config.0.vpc_origin_id"),
					resource.TestCheckResourceAttrSet(resourceName, "origin.0.vpc_origin_config.0.owner_account_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_DefaultCacheBehavior_realtimeLogARN(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	realtimeLogConfigResourceName := "aws_cloudfront_realtime_log_config.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_defaultCacheBehaviorRealtimeLogARN(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "default_cache_behavior.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "default_cache_behavior.0.realtime_log_config_arn", realtimeLogConfigResourceName, names.AttrARN),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_OrderedCacheBehavior_realtimeLogARN(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	realtimeLogConfigResourceName := "aws_cloudfront_realtime_log_config.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_orderedCacheBehaviorRealtimeLogARN(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "1"),
					resource.TestCheckResourceAttrPair(resourceName, "cache_behavior.0.realtime_log_config_arn", realtimeLogConfigResourceName, names.AttrARN),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_DefaultCacheBehavior_trustedKeyGroups(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_defaultCacheBehaviorTrustedKeyGroups(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "default_cache_behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "default_cache_behavior.0.trusted_key_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "default_cache_behavior.0.trusted_key_groups.0.enabled", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "default_cache_behavior.0.trusted_key_groups.0.items.#", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_ViewerCertificate_acmCertificateARN(t *testing.T) {
	// Note: Multitenant distributions reject self-signed certificates with InvalidViewerCertificate error.
	// Testing the default CloudFront certificate instead, which is the only viable option without real DNS/CA.
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_viewerCertificateCloudFrontDefault(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "viewer_certificate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "viewer_certificate.0.cloudfront_default_certificate", acctest.CtTrue),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_noCustomErrorResponse(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_noCustomErrorResponse(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_noOptionalItems(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_noOptionalItems(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "cache_behavior.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tenant_config.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func testAccCheckMultiTenantDistributionDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).CloudFrontClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_cloudfront_multitenant_distribution" {
				continue
			}

			_, err := tfcloudfront.FindDistributionByID(ctx, conn, rs.Primary.ID)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("CloudFront Multi-tenant Distribution %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckMultiTenantDistributionExists(ctx context.Context, n string, v *awstypes.Distribution) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).CloudFrontClient(ctx)

		output, err := tfcloudfront.FindDistributionByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output.Distribution

		return nil
	}
}

func testAccMultiTenantDistributionConfig_basic() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test multi-tenant distribution"

  origin {
    domain_name = "example.com"
    id          = "example"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "example"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # AWS Managed CachingDisabled policy

    allowed_methods {
      items          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
      cached_methods = ["GET", "HEAD", "OPTIONS"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter for tenants"
        }
      }
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_comprehensive(rName, comment, defaultRootObject string, compress bool) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket        = %[1]q
  force_destroy = true
}

resource "aws_cloudfront_origin_access_control" "test" {
  name                              = %[1]q
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled             = false
  comment             = %[2]q
  default_root_object = %[3]q

  origin {
    domain_name = "example.com"
    id          = "custom-origin"
    origin_path = "/api"

    custom_header {
      header_name  = "X-Custom-Header"
      header_value = "test-value"
    }

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "custom-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # AWS Managed CachingDisabled policy

    allowed_methods {
      items          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  # Single cache behavior
  cache_behavior {
    path_pattern           = "/api/*"
    target_origin_id       = "custom-origin"
    viewer_protocol_policy = "https-only"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    compress               = %[4]t

    allowed_methods {
      items          = ["GET", "HEAD", "OPTIONS"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  # Custom error response
  custom_error_response {
    error_code         = 404
    response_code      = "200"
    response_page_path = "/404.html"
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # Simplified tenant config
  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter for tenants"
        }
      }
    }
  }

  tags = {
    Environment = "test"
    Name        = %[1]q
  }
}
`, rName, comment, defaultRootObject, compress)
}

func testAccMultiTenantDistributionConfig_tags(tags map[string]string) string {
	var tagLines []string
	for key, value := range tags {
		tagLines = append(tagLines, fmt.Sprintf("    %q = %q", key, value))
	}
	tagConfig := fmt.Sprintf("  tags = {\n%s\n  }", strings.Join(tagLines, "\n"))

	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test multi-tenant distribution for tags"

  origin {
    domain_name = "example.com"
    id          = "example"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "example"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # AWS Managed CachingDisabled policy

    allowed_methods {
      items          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
      cached_methods = ["GET", "HEAD", "OPTIONS"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter for tenants"
        }
      }
    }
  }

%s
}
`, tagConfig)
}

func testAccMultiTenantDistributionConfig_update(comment, httpVersion, defaultRootObject string) string {
	defaultRootObjectConfig := ""
	if defaultRootObject != "" {
		defaultRootObjectConfig = fmt.Sprintf("default_root_object = %q", defaultRootObject)
	}

	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled      = false
  comment      = %[1]q
  http_version = %[2]q
  %[3]s

  origin {
    domain_name = "example.com"
    id          = "example"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "example"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" # AWS Managed CachingDisabled policy

    allowed_methods {
      items          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Updated origin domain parameter for tenants"
        }
      }
    }
  }
}
`, comment, httpVersion, defaultRootObjectConfig)
}

func testAccMultiTenantDistributionConfig_s3OriginWithOAC(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket        = %[1]q
  force_destroy = true
}

resource "aws_cloudfront_origin_access_control" "test" {
  name                              = %[1]q
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = true
  comment = "Test Distribution with S3 OAC"

  tenant_config {}

  origin {
    id                       = aws_s3_bucket.test.bucket_regional_domain_name
    domain_name              = aws_s3_bucket.test.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.test.id

    connection_attempts         = 3
    connection_timeout          = 10
    response_completion_timeout = 30
  }

  default_cache_behavior {
    target_origin_id       = aws_s3_bucket.test.bucket_regional_domain_name
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    compress               = true

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_multipleCustomHeaders() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with multiple custom headers"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    # Multiple custom headers to test set ordering doesn't cause issues
    custom_header {
      header_name  = "X-Custom-Header-1"
      header_value = "value1"
    }

    custom_header {
      header_name  = "X-Custom-Header-2"
      header_value = "value2"
    }

    custom_header {
      header_name  = "X-Custom-Header-3"
      header_value = "value3"
    }

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`
}

func TestAccCloudFrontMultiTenantDistribution_multipleOrigins(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_multipleOrigins(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					// Verify we have 3 origins (order doesn't matter with sets)
					resource.TestCheckResourceAttr(resourceName, "origin.#", "3"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccCloudFrontMultiTenantDistribution_multipleCustomErrorResponses(t *testing.T) {
	t.Parallel()

	ctx := acctest.Context(t)
	var distribution awstypes.Distribution
	resourceName := "aws_cloudfront_multitenant_distribution.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckPartitionHasService(t, names.CloudFrontEndpointID) },
		ErrorCheck:               acctest.ErrorCheck(t, names.CloudFrontServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckMultiTenantDistributionDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccMultiTenantDistributionConfig_multipleCustomErrorResponses(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMultiTenantDistributionExists(ctx, resourceName, &distribution),
					resource.TestCheckResourceAttr(resourceName, names.AttrEnabled, acctest.CtFalse),
					// Verify we have 3 custom error responses (order doesn't matter with sets)
					resource.TestCheckResourceAttr(resourceName, "custom_error_response.#", "3"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func testAccMultiTenantDistributionConfig_multipleOrigins() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with multiple origins"

  # Multiple origins to test set ordering doesn't cause issues
  origin {
    domain_name = "example1.com"
    id          = "origin-1"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  origin {
    domain_name = "example2.com"
    id          = "origin-2"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  origin {
    domain_name = "example3.com"
    id          = "origin-3"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "origin-1"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_multipleCustomErrorResponses() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with multiple custom error responses"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  # Multiple custom error responses to test set ordering doesn't cause issues
  custom_error_response {
    error_code         = 404
    response_code      = "200"
    response_page_path = "/404.html"
  }

  custom_error_response {
    error_code         = 403
    response_code      = "200"
    response_page_path = "/403.html"
  }

  custom_error_response {
    error_code         = 500
    response_code      = "200"
    response_page_path = "/500.html"
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_noTenantConfig() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with no tenant config"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # No tenant_config block at all - testing it's truly optional
}
`
}

func testAccMultiTenantDistributionConfig_enabled(enabled bool) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = %[1]t
  comment = "Test distribution enabled toggle"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`, enabled)
}

func testAccMultiTenantDistributionConfig_originGroups(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test1" {
  bucket        = "%[1]s-1"
  force_destroy = true
}

resource "aws_s3_bucket" "test2" {
  bucket        = "%[1]s-2"
  force_destroy = true
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with origin groups"

  origin {
    domain_name = aws_s3_bucket.test1.bucket_regional_domain_name
    id          = "primary"
  }

  origin {
    domain_name = aws_s3_bucket.test2.bucket_regional_domain_name
    id          = "secondary"
  }

  origin_group {
    id = "group1"

    failover_criteria {
      status_codes = [403, 404, 500, 502, 503, 504]
    }

    member {
      origin_id = "primary"
    }

    member {
      origin_id = "secondary"
    }
  }

  default_cache_behavior {
    target_origin_id       = "group1"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_originShield(enabled bool) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with origin shield"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    origin_shield {
      enabled              = %[1]t
      origin_shield_region = "us-east-1"
    }

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`, enabled)
}

func testAccMultiTenantDistributionConfig_httpVersion(version string) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled      = false
  comment      = "Test distribution with HTTP version"
  http_version = %[1]q

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`, version)
}

func testAccMultiTenantDistributionConfig_responseCompletionTimeout(timeout int) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with response completion timeout"

  origin {
    domain_name                = "example.com"
    id                         = "test-origin"
    response_completion_timeout = %[1]d

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  tenant_config {
    parameter_definition {
      name = "origin_domain"
      definition {
        string_schema {
          required = true
          comment  = "Origin domain parameter"
        }
      }
    }
  }
}
`, timeout)
}

func testAccMultiTenantDistributionConfig_customOrigin() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with custom origin"

  origin {
    domain_name = "www.example.com"
    id          = "customOrigin"

    custom_origin_config {
      http_port                = 80
      https_port               = 443
      origin_protocol_policy   = "http-only"
      origin_ssl_protocols     = ["TLSv1.2"]
      origin_read_timeout      = 30
      origin_keepalive_timeout = 5
    }
  }

  default_cache_behavior {
    target_origin_id       = "customOrigin"
    viewer_protocol_policy = "allow-all"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD", "OPTIONS"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_orderedCacheBehavior() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with ordered cache behaviors"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  cache_behavior {
    path_pattern           = "images1/*.jpg"
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  cache_behavior {
    path_pattern           = "images2/*.jpg"
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_orderedCacheBehaviorCachePolicy(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_cache_policy" "test" {
  name        = %[1]q
  min_ttl     = 1
  default_ttl = 50
  max_ttl     = 100

  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }

    headers_config {
      header_behavior = "none"
    }

    query_strings_config {
      query_string_behavior = "none"
    }
  }
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with cache policy on ordered behavior"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  cache_behavior {
    path_pattern           = "images/*.jpg"
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = aws_cloudfront_cache_policy.test.id

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_orderedCacheBehaviorResponseHeadersPolicy(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_response_headers_policy" "test" {
  name = %[1]q

  cors_config {
    access_control_allow_credentials = true

    access_control_allow_headers {
      items = ["X-Example-Header"]
    }

    access_control_allow_methods {
      items = ["GET", "POST"]
    }

    access_control_allow_origins {
      items = ["https://example.com"]
    }

    origin_override = false
  }
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with response headers policy on ordered behavior"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  cache_behavior {
    path_pattern              = "images/*.jpg"
    target_origin_id          = "test-origin"
    viewer_protocol_policy    = "redirect-to-https"
    cache_policy_id           = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    response_headers_policy_id = aws_cloudfront_response_headers_policy.test.id

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_s3Origin(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket = %[1]q
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with S3 origin"

  origin {
    domain_name = aws_s3_bucket.test.bucket_regional_domain_name
    id          = "s3Origin"
  }

  default_cache_behavior {
    target_origin_id       = "s3Origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_originConnectionAttempts(attempts int) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with connection attempts"

  origin {
    domain_name         = "example.com"
    id                  = "test-origin"
    connection_attempts = %[1]d

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, attempts)
}

func testAccMultiTenantDistributionConfig_originConnectionTimeout(timeout int) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with connection timeout"

  origin {
    domain_name        = "example.com"
    id                 = "test-origin"
    connection_timeout = %[1]d

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, timeout)
}

func testAccMultiTenantDistributionConfig_originAccessControl(rName string) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket = %[1]q
}

resource "aws_cloudfront_origin_access_control" "test" {
  name                              = %[1]q
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with origin access control"

  origin {
    domain_name              = aws_s3_bucket.test.bucket_regional_domain_name
    id                       = "s3Origin"
    origin_access_control_id = aws_cloudfront_origin_access_control.test.id
  }

  default_cache_behavior {
    target_origin_id       = "s3Origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccVPCOriginConfig_basicForMultiTenant(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigVPCWithSubnets(rName, 2), fmt.Sprintf(`
resource "aws_security_group" "test" {
  name   = %[1]q
  vpc_id = aws_vpc.test.id

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = %[1]q
  }
}

resource "aws_internet_gateway" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_lb" "test" {
  name            = %[1]q
  security_groups = [aws_security_group.test.id]
  subnets         = aws_subnet.test[*].id

  idle_timeout               = 30
  enable_deletion_protection = false

  tags = {
    Name = %[1]q
  }

  depends_on = [aws_internet_gateway.test]
}

resource "aws_cloudfront_vpc_origin" "test" {
  vpc_origin_endpoint_config {
    name                   = %[1]q
    arn                    = aws_lb.test.arn
    http_port              = 80
    https_port             = 443
    origin_protocol_policy = "https-only"

    origin_ssl_protocols {
      items    = ["TLSv1.2"]
      quantity = 1
    }
  }

  tags = {
    Name = %[1]q
  }
}
`, rName))
}

func testAccMultiTenantDistributionConfig_vpcOriginConfig(rName string) string {
	return acctest.ConfigCompose(testAccVPCOriginConfig_basicForMultiTenant(rName), `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with VPC origin"

  origin {
    domain_name = "www.example.com"
    id          = "test"

    vpc_origin_config {
      vpc_origin_id = aws_cloudfront_vpc_origin.test.id
    }
  }

  default_cache_behavior {
    target_origin_id       = "test"
    viewer_protocol_policy = "allow-all"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`)
}

func testAccMultiTenantDistributionConfig_vpcOriginConfigOwnerAccountID(rName string) string {
	return acctest.ConfigCompose(testAccVPCOriginConfig_basicForMultiTenant(rName), `
data "aws_caller_identity" "current" {}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with VPC origin and owner account ID"

  origin {
    domain_name = "www.example.com"
    id          = "test"

    vpc_origin_config {
      vpc_origin_id    = aws_cloudfront_vpc_origin.test.id
      owner_account_id = data.aws_caller_identity.current.account_id
    }
  }

  default_cache_behavior {
    target_origin_id       = "test"
    viewer_protocol_policy = "allow-all"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`)
}

func testAccRealtimeLogConfigBase(rName string) string {
	return fmt.Sprintf(`
data "aws_iam_policy_document" "test_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "test_policy" {
  statement {
    actions = [
      "kinesis:DescribeStreamSummary",
      "kinesis:DescribeStream",
      "kinesis:PutRecord",
      "kinesis:PutRecords",
    ]
    resources = [aws_kinesis_stream.test.arn]
  }
}

resource "aws_iam_role" "test" {
  name               = %[1]q
  assume_role_policy = data.aws_iam_policy_document.test_assume.json
}

resource "aws_iam_role_policy" "test" {
  name   = %[1]q
  role   = aws_iam_role.test.id
  policy = data.aws_iam_policy_document.test_policy.json
}

resource "aws_kinesis_stream" "test" {
  name        = %[1]q
  shard_count = 2
}

resource "aws_cloudfront_realtime_log_config" "test" {
  name          = %[1]q
  sampling_rate = 50

  fields = [
    "timestamp",
    "c-ip",
  ]

  endpoint {
    stream_type = "Kinesis"

    kinesis_stream_config {
      role_arn   = aws_iam_role.test.arn
      stream_arn = aws_kinesis_stream.test.arn
    }
  }

  depends_on = [aws_iam_role_policy.test]
}
`, rName)
}

func testAccMultiTenantDistributionConfig_defaultCacheBehaviorRealtimeLogARN(rName string) string {
	return acctest.ConfigCompose(testAccRealtimeLogConfigBase(rName), `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with realtime log config on default cache behavior"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id        = "test-origin"
    viewer_protocol_policy  = "redirect-to-https"
    cache_policy_id         = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    realtime_log_config_arn = aws_cloudfront_realtime_log_config.test.arn

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`)
}

func testAccMultiTenantDistributionConfig_orderedCacheBehaviorRealtimeLogARN(rName string) string {
	return acctest.ConfigCompose(testAccRealtimeLogConfigBase(rName), `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with realtime log config on ordered cache behavior"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  cache_behavior {
    path_pattern            = "images/*.jpg"
    target_origin_id        = "test-origin"
    viewer_protocol_policy  = "redirect-to-https"
    cache_policy_id         = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
    realtime_log_config_arn = aws_cloudfront_realtime_log_config.test.arn

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`)
}

func testAccMultiTenantDistributionConfig_defaultCacheBehaviorTrustedKeyGroups(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_public_key" "test" {
  comment     = "test key"
  encoded_key = file("test-fixtures/cloudfront-public-key.pem")
  name        = %[1]q
}

resource "aws_cloudfront_key_group" "test" {
  comment = "test key group"
  items   = [aws_cloudfront_public_key.test.id]
  name    = %[1]q
}

resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution with trusted key groups"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }

    trusted_key_groups {
      enabled = true
      items   = [aws_cloudfront_key_group.test.id]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_viewerCertificateCloudFrontDefault(rName string) string {
	return fmt.Sprintf(`
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = %[1]q

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "allow-all"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`, rName)
}

func testAccMultiTenantDistributionConfig_noCustomErrorResponse() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false
  comment = "Test distribution without custom error responses"

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`
}

func testAccMultiTenantDistributionConfig_noOptionalItems() string {
	return `
resource "aws_cloudfront_multitenant_distribution" "test" {
  enabled = false

  origin {
    domain_name = "example.com"
    id          = "test-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "test-origin"
    viewer_protocol_policy = "redirect-to-https"
    cache_policy_id        = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"

    allowed_methods {
      items          = ["GET", "HEAD"]
      cached_methods = ["GET", "HEAD"]
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }
}
`
}
