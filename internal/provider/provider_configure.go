package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (p *CleanEksProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring provider")

	// Load configuration into the model
	var model CleanEksProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := ""
	if !(model.Host.IsNull() || model.Host.IsUnknown()) {
		host = model.Host.ValueString()
	}

	burstLimit := int64(0)
	if !(model.BurstLimit.IsNull() || model.BurstLimit.IsUnknown()) {
		burstLimit = model.BurstLimit.ValueInt64()
	}

	p.model = model
	p.Host = host
	p.BurstLimit = burstLimit

	if p.clientSet == nil {
		clientSet, err := p.GetClientSet(ctx)
		if err != nil {
			if errors.Is(err, clientcmd.ErrEmptyConfig) {
				// We don't want to throw error here as we EKS cluster might not exist yet
				resp.Diagnostics.Append(diag.NewWarningDiagnostic("Invalid provider configuration was supplied. Provider operations likely to fail. Failed to initialize Kubernetes client configuration, this could be because credentials are not available during provider initialization", err.Error()))
			} else {
				resp.Diagnostics.AddError(
					"Error getting Kubernetes client during JobResource.Configure",
					fmt.Sprintf("Error getting Kubernetes client during JobResource.Configure: %s", err),
				)
				return
			}
		} else {
			p.clientSet = clientSet
		}
	}

	// Since the provider instance is being passed, ensure these response
	// values are always set before early returns, etc.
	resp.DataSourceData = p
	resp.ResourceData = p

	tflog.Debug(ctx, "Loaded provider configuration during Provider.Configuration", map[string]interface{}{
		"providerConfig": fmt.Sprintf("%+v", p),
	})
}

func newKubernetesClientConfig(ctx context.Context, data CleanEksProviderModel) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if v := data.ConfigPath.ValueString(); v != "" {
		tflog.Debug(ctx, "Using provided config path")
		configPaths = []string{v}
	} else if len(data.ConfigPaths) > 0 {
		tflog.Debug(ctx, "Using provided config paths")
		configPaths = append(configPaths, data.ConfigPaths...)
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		tflog.Debug(ctx, "Using KUBE_CONFIG_PATHS env variable")
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}

			tflog.Debug(ctx, "Using kubeconfig file", map[string]interface{}{"path": path})
			expandedPaths = append(expandedPaths, path)
		}
		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		// ctxSuffix := "; default context"

		kubectx := data.ConfigContext.ValueString()
		authInfo := data.ConfigContextAuthInfo.ValueString()
		cluster := data.ConfigContextCluster.ValueString()
		if kubectx != "" || authInfo != "" || cluster != "" {
			// ctxSuffix = "; overridden context"
			if kubectx != "" {
				overrides.CurrentContext = kubectx
				// ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
				tflog.Debug(ctx, "Using custom current context", map[string]interface{}{"context": overrides.CurrentContext})
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfo != "" {
				overrides.Context.AuthInfo = authInfo
				// ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if cluster != "" {
				overrides.Context.Cluster = cluster
				// ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
			tflog.Debug(ctx, "Using overridden context", map[string]interface{}{"context": overrides.Context})
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = data.Insecure.ValueBool()
	overrides.ClusterInfo.TLSServerName = data.TLSServerName.ValueString()
	overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(data.ClusterCACertificate.ValueString()).Bytes()
	overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(data.ClientCertificate.ValueString()).Bytes()
	overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(data.ClientKey.ValueString()).Bytes()
	if len(overrides.AuthInfo.ClientCertificateData) > 0 || len(overrides.AuthInfo.ClientKeyData) > 0 {
		tflog.Debug(ctx, "Using certificate for authentication")
	}

	if v := data.Host.ValueString(); v != "" {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(v, "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}

	overrides.AuthInfo.Username = data.Username.ValueString()
	overrides.AuthInfo.Password = data.Password.ValueString()
	if overrides.AuthInfo.Username != "" || overrides.AuthInfo.Password != "" {
		tflog.Debug(ctx, "Using username/password for authentication")
	}

	overrides.AuthInfo.Token = data.Token.ValueString()
	if overrides.AuthInfo.Token != "" {
		tflog.Debug(ctx, "Using token for authentication")
	}

	overrides.ClusterDefaults.ProxyURL = data.ProxyURL.ValueString()

	if len(data.Exec) > 0 {
		execData := data.Exec[0]

		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = execData.APIVersion.ValueString()
		exec.Command = execData.Command.ValueString()

		if execData.Args != nil {
			exec.Args = append(exec.Args, execData.Args...)
		}
		if execData.Env != nil {
			for environmentVariableName, environmentVariableValue := range execData.Env {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: environmentVariableName, Value: environmentVariableValue})
			}
		}

		overrides.AuthInfo.Exec = exec
		if overrides.AuthInfo.Exec != nil {
			tflog.Debug(ctx, "Using exec for authentication: %s", map[string]interface{}{
				"APIVersion": overrides.AuthInfo.Exec.APIVersion,
				"Command":    overrides.AuthInfo.Exec.Command,
			})
		}
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()

	if err != nil {
		return nil, err
	}
	return cfg, nil
}
