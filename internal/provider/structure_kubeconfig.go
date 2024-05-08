package provider

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/go-homedir"
	"k8s.io/apimachinery/pkg/api/meta"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memcached "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeConfig is a RESTClientGetter interface implementation
type KubeConfig struct {
	ClientConfig clientcmd.ClientConfig

	Burst int

	sync.Mutex
}

// ToRESTConfig implemented interface method
func (k *KubeConfig) ToRESTConfig() (*rest.Config, error) {
	config, err := k.ToRawKubeConfigLoader().ClientConfig()
	return config, err
}

// ToDiscoveryClient implemented interface method
func (k *KubeConfig) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := k.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	// The more groups you have, the more discovery requests you need to make.
	// given 25 groups (our groups + a few custom resources) with one-ish version each, discovery needs to make 50 requests
	// double it just so we don't end up here again for a while.  This config is only used for discovery.
	config.Burst = k.Burst

	return memcached.NewMemCacheClient(discovery.NewDiscoveryClientForConfigOrDie(config)), nil
}

// ToRESTMapper implemented interface method
func (k *KubeConfig) ToRESTMapper(ctx context.Context) (meta.RESTMapper, error) {
	discoveryClient, err := k.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient, func(warning string) {
		tflog.Warn(ctx, warning)
	})
	return expander, nil
}

// ToRawKubeConfigLoader implemented interface method
func (k *KubeConfig) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return k.ClientConfig
}

func getKubeConfig(ctx context.Context, resp *provider.ConfigureResponse, configPaths []string, configContext string, configContextAuthInfo string, configContextCluster string, insecure bool, tlsServerName string, clusterCaCertificate string, clientCertificate string, host string, username string, password string, clientKey string, token string, proxyUrl string, namespace string, burstLimit int64) (bool, *KubeConfig) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error expanding paths",
					fmt.Sprintf("Error expanding paths: %s", err),
				)
				return true, nil
			}

			tflog.Debug(ctx, "Using kubeconfig", map[string]interface{}{
				"kubeconfig": fmt.Sprintf("%+v", path),
			})

			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		if configContext != "" || configContextAuthInfo != "" || configContextCluster != "" {
			if configContext != "" {
				overrides.CurrentContext = configContext
				tflog.Debug(ctx, "Using custom current context", map[string]interface{}{
					"currentContext": fmt.Sprintf("%+v", overrides.CurrentContext),
				})
			}

			overrides.Context = clientcmdapi.Context{}
			if configContextAuthInfo != "" {
				overrides.Context.AuthInfo = configContextAuthInfo
			}

			if configContextCluster != "" {
				overrides.Context.Cluster = configContextCluster
			}
			tflog.Debug(ctx, "Using overidden context", map[string]interface{}{
				"context": fmt.Sprintf("%+v", overrides.Context),
			})
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = insecure
	if len(tlsServerName) > 0 {
		overrides.ClusterInfo.TLSServerName = tlsServerName
	}
	if len(clusterCaCertificate) > 0 {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(clusterCaCertificate).Bytes()
	}
	if len(clientCertificate) > 0 {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(clientCertificate).Bytes()
	}
	if len(host) > 0 {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		hostUrl, _, err := rest.DefaultServerURL(host, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error generating host url",
				fmt.Sprintf("Error generating host url: %s", err),
			)
			return true, nil
		}

		overrides.ClusterInfo.Server = hostUrl.String()
	}

	if len(username) > 0 {
		overrides.AuthInfo.Username = username
	}
	if len(password) > 0 {
		overrides.AuthInfo.Password = password
	}
	if len(clientKey) > 0 {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(clientKey).Bytes()
	}
	if len(token) > 0 {
		overrides.AuthInfo.Token = token
	}
	if len(proxyUrl) > 0 {
		overrides.ClusterDefaults.ProxyURL = proxyUrl
	}

	//EXEC

	overrides.Context.Namespace = "default"

	if namespace != "" {
		overrides.Context.Namespace = namespace
	}

	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		resp.Diagnostics.AddError(
			"Failed to initialize kubernetes config",
			fmt.Sprintf("Failed to initialize kubernetes config"),
		)
		return true, nil
	}
	tflog.Debug(ctx, "Successfully initialized kubernetes config")

	kubeConfig := &KubeConfig{ClientConfig: client, Burst: int(burstLimit)}
	return false, kubeConfig
}
