package insightsclient

import (
	"context"
	"crypto/tls"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	configclientset "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/openshift/library-go/pkg/crypto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// GetTLSConfigFromAPIServer fetches the TLS profile from the API Server configuration.
// This is the default source for most components.
func GetTLSConfigFromAPIServer(configClient configclientset.Interface) (*tls.Config, error) {
	apiserver, err := configClient.ConfigV1().APIServers().Get(context.Background(), "cluster", metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get APIServer config: %v", err)
		return nil, fmt.Errorf("failed to get APIServer config: %w", err)
	}

	klog.Infof("apiServer: %v", apiserver)

	profile := apiserver.Spec.TLSSecurityProfile
	if profile == nil {
		profile = &configv1.TLSSecurityProfile{
			Type: configv1.TLSProfileIntermediateType,
		}
	}

	klog.Infof("profile: %v", profile)

	return buildTLSConfigFromProfile(profile)
}

func buildTLSConfigFromProfile(profile *configv1.TLSSecurityProfile) (*tls.Config, error) {
	profileSpec, err := getTLSProfileSpec(profile)
	if err != nil {
		klog.Errorf("failed to getTLSProfileSpec: %v", err)
		return nil, err
	}

	minVersion, err := parseTLSVersion(string(profileSpec.MinTLSVersion))
	if err != nil {
		klog.Errorf("failed to parseTLSVersion: %v", err)
		return nil, fmt.Errorf("invalid MinTLSVersion: %w", err)
	}

	klog.Infof("minVersion: %v", minVersion)

	config := &tls.Config{
		MinVersion: minVersion,
	}

	if minVersion == tls.VersionTLS13 {
		config.MaxVersion = tls.VersionTLS13
	} else {
		// library-go/pkg/crypto/crypto.go
		// cipherSuites := parseCipherSuites(profileSpec.Ciphers)
		cipherSuites := crypto.OpenSSLToIANACipherSuites(profileSpec.Ciphers)
		if len(cipherSuites) == 0 {
			klog.Errorf("failed to get valid cipher suites: %v", err)
			return nil, fmt.Errorf("no valid cipher suites found")
		}

		config.CipherSuites = crypto.CipherSuitesOrDie(cipherSuites)
	}

	klog.Infof("config: %v", config)

	return config, nil
}

func getTLSProfileSpec(profile *configv1.TLSSecurityProfile) (*configv1.TLSProfileSpec, error) {
	switch profile.Type {
	case configv1.TLSProfileOldType,
		configv1.TLSProfileIntermediateType,
		configv1.TLSProfileModernType:
		return configv1.TLSProfiles[profile.Type], nil
	case configv1.TLSProfileCustomType:
		if profile.Custom == nil {
			return nil, fmt.Errorf("custom TLS profile specified but Custom field is nil")
		}
		return &profile.Custom.TLSProfileSpec, nil
	default:
		return configv1.TLSProfiles[configv1.TLSProfileIntermediateType], nil
	}
}

func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "VersionTLS10", "TLSv1.0":
		return tls.VersionTLS10, nil
	case "VersionTLS11", "TLSv1.1":
		return tls.VersionTLS11, nil
	case "VersionTLS12", "TLSv1.2":
		return tls.VersionTLS12, nil
	case "VersionTLS13", "TLSv1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unknown TLS version: %s", version)
	}
}
