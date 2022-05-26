// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package witchcraft

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/palantir/pkg/tlsconfig"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/v2/config"
)

func (s *Server) newServer(productName string, serverConfig config.Server, handler http.Handler) (rHTTPServer *http.Server, rStart func() error, rShutdown func(context.Context) error, rErr error) {
	return newServerStartShutdownFns(
		serverConfig,
		s.useSelfSignedServerCertificate,
		s.useInsecurePlaintextServer,
		s.clientAuth,
		productName,
		s.svcLogger,
		handler,
	)
}

func (s *Server) newMgmtServer(productName string, serverConfig config.Server, handler http.Handler) (rStart func() error, rShutdown func(context.Context) error, rErr error) {
	serverConfig.Port = serverConfig.ManagementPort
	_, start, shutdown, err := newServerStartShutdownFns(
		serverConfig,
		s.useSelfSignedServerCertificate,
		s.useInsecurePlaintextServer,
		tls.NoClientCert,
		fmt.Sprintf("%s-management", productName),
		s.svcLogger,
		handler,
	)
	return start, shutdown, err
}

func newServerStartShutdownFns(
	serverConfig config.Server,
	useSelfSignedServerCertificate bool,
	useInsecurePlaintextServer bool,
	clientAuthType tls.ClientAuthType,
	serverName string,
	svcLogger svc1log.Logger,
	handler http.Handler,
) (rHTTPServer *http.Server, start func() error, shutdown func(context.Context) error, rErr error) {
	if useInsecurePlaintextServer {
		return newInsecureServerStartShutdownFns(serverConfig, serverName, svcLogger, handler)
	}
	tlsConfig, err := newTLSConfig(serverConfig, useSelfSignedServerCertificate, clientAuthType)
	if err != nil {
		return nil, nil, nil, err
	}
	httpServer := &http.Server{
		Addr:      fmt.Sprintf("%v:%d", serverConfig.Address, serverConfig.Port),
		TLSConfig: tlsConfig,
		Handler:   handler,
	}

	return httpServer, startTLSFn(httpServer, serverName, svcLogger), httpServer.Shutdown, nil
}

func newInsecureServerStartShutdownFns(
	serverConfig config.Server,
	serverName string,
	svcLogger svc1log.Logger,
	handler http.Handler,
) (rHTTPServer *http.Server, start func() error, shutdown func(context.Context) error, rErr error) {
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%v:%d", serverConfig.Address, serverConfig.Port),
		Handler: handler,
	}
	if !isLoopback(serverConfig.Address, svcLogger) {
		return nil, nil, nil, werror.Error("refusing to run a plaintext server on non-loopback interface")
	}
	return httpServer, startFn(httpServer, serverName, svcLogger), httpServer.Shutdown, nil
}

func startTLSFn(
	httpServer *http.Server,
	serverName string,
	svcLogger svc1log.Logger,
) func() error {
	return func() error {
		svcLogger.Info("Listening to https", svc1log.SafeParam("address", httpServer.Addr), svc1log.SafeParam("server", serverName))

		// cert and key specified in TLS config so no need to pass in here
		if err := httpServer.ListenAndServeTLS("", ""); err != nil {
			if err == http.ErrServerClosed {
				svcLogger.Info(fmt.Sprintf("%s was closed", serverName))
				return nil
			}
			return werror.Wrap(err, "server failed", werror.SafeParam("serverName", serverName))
		}
		return nil
	}
}

func startFn(
	httpServer *http.Server,
	serverName string,
	svcLogger svc1log.Logger,
) func() error {
	return func() error {
		svcLogger.Info("Listening to http", svc1log.SafeParam("address", httpServer.Addr), svc1log.SafeParam("server", serverName))

		// cert and key specified in TLS config so no need to pass in here
		if err := httpServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				svcLogger.Info(fmt.Sprintf("%s was closed", serverName))
				return nil
			}
			return werror.Wrap(err, "server failed", werror.SafeParam("serverName", serverName))
		}
		return nil
	}
}

func newTLSConfig(serverConfig config.Server, useSelfSignedServerCertificate bool, clientAuthType tls.ClientAuthType) (*tls.Config, error) {
	tlsConfig, err := tlsconfig.NewServerConfig(
		newTLSCertProvider(useSelfSignedServerCertificate, serverConfig.CertFile, serverConfig.KeyFile),
		tlsconfig.ServerClientCAFiles(serverConfig.ClientCAFiles...),
		tlsconfig.ServerClientAuthType(clientAuthType),
		tlsconfig.ServerNextProtos("h2"),
	)
	if err != nil {
		return nil, werror.Wrap(err, "failed to initialize TLS configuration for server")
	}
	return tlsConfig, nil
}

func newTLSCertProvider(useSelfSignedServerCertificate bool, certFile, keyFile string) tlsconfig.TLSCertProvider {
	if useSelfSignedServerCertificate {
		return newSelfSignedCertificate
	}
	return newFileBasedCertificateProvider(certFile, keyFile)
}

// newSelfSignedCertificate creates a new self-signed certificate that can be used for TLS. The generated certificate is
// quite minimal: it has a hard-coded serial number, is valid for 1 year and does NOT have a common name or IP SANs
// set.
func newSelfSignedCertificate() (tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	privPKCS1DERBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privPKCS1PEMBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privPKCS1DERBytes})

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-30 * time.Second),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert, err := x509.ParseCertificate(certDERBytes)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return tls.X509KeyPair(certPEMBytes, privPKCS1PEMBytes)
}

func newFileBasedCertificateProvider(certFile, keyFile string) tlsconfig.TLSCertProvider {
	return func() (tls.Certificate, error) {
		var msg string
		if keyFile == "" && certFile == "" {
			msg = "key file and certificate file"
		} else if keyFile == "" {
			msg = "key file"
		} else if certFile == "" {
			msg = "certificate file"
		} else {
			return tlsconfig.TLSCertFromFiles(certFile, keyFile)()
		}
		return tls.Certificate{}, werror.Error(msg + " for server not specified in configuration")
	}
}

// isLoopback attempts to determine if the provided address is associated with a loopback interface. This is primarily
// done in two ways, (a) using the IP associated with the loopback interface or (b) using a hostname that locally
// resolves to the IP of the loopback interface.
func isLoopback(bindAddress string, svcLogger svc1log.Logger) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		svcLogger.Warn("Failed to list host interfaces, assuming address is not loopback", svc1log.Stacktrace(err))
		return false
	}

	bindIPs, err := net.LookupIP(bindAddress)
	if err != nil {
		svcLogger.Warn("Failed to lookup address IPs, assuming address is not loopback", svc1log.Stacktrace(err))
		return false
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 {
			continue
		}

		addresses, err := iface.Addrs()
		if err != nil {
			svcLogger.Warn("Failed to get addresses for interface while checking for loopback",
				svc1log.UnsafeParam("iface", iface.Name),
				svc1log.Stacktrace(err))
			continue
		}
		for _, addr := range addresses {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				svcLogger.Warn("Failed to parse CIDR for interface address",
					svc1log.UnsafeParam("iface", iface.Name),
					svc1log.UnsafeParam("address", addr.String()),
					svc1log.Stacktrace(err))
				continue
			}

			for _, bindIP := range bindIPs {
				if ip.Equal(bindIP) {
					return true
				}
			}
		}
	}

	svcLogger.Warn("No loopback interfaces matched the bind address, assuming address is not loopback")
	return false
}
