// Copyright (c) 2024 Palantir Technologies. All rights reserved.
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

package refreshingclient

import (
	"context"
	"crypto/tls"

	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/pkg/tlsconfig"
	werror "github.com/palantir/witchcraft-go-error"
)

// TLSParams contains the parameters needed to build a *tls.Config.
// Its fields must all be compatible with reflect.DeepEqual.
type TLSParams struct {
	CABytes            [][]byte
	CertFile           string
	KeyFile            string
	InsecureSkipVerify bool
}

// NewRefreshableTLSConfig evaluates the provided TLSParams and returns a RefreshableTLSConfig that will update the
// underlying *tls.Config when the TLSParams change.
// IF the initial TLSParams are invalid, NewRefreshableTLSConfig will return an error.
// If the updated TLSParams are invalid, the RefreshableTLSConfig will continue to use the previous value and log the error.
//
// N.B. This subscription only fires when the paths are updated, not when the contents of the files are updated.
// We could consider adding a file refreshable to watch the key and cert files.
func NewRefreshableTLSConfig(ctx context.Context, params refreshable.Refreshable[TLSParams]) (refreshable.Validated[*tls.Config], error) {
	r, _, err := refreshable.MapWithError(params, func(p TLSParams) (*tls.Config, error) {
		return NewTLSConfig(ctx, p)
	})
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "failed to build RefreshableTLSConfig")
	}
	return r, nil
}

// NewTLSConfig returns a *tls.Config built from the provided TLSParams.
func NewTLSConfig(ctx context.Context, p TLSParams) (*tls.Config, error) {
	var tlsParams []tlsconfig.ClientParam
	if len(p.CABytes) > 0 {
		var certPoolOptions []tlsconfig.CertPoolOption
		for _, ca := range p.CABytes {
			certPoolOptions = append(certPoolOptions, tlsconfig.CertPoolOptionCABytes(ca))
		}
		tlsParams = append(tlsParams, tlsconfig.ClientRootCAs(tlsconfig.CertPoolFromCertPoolOptions(certPoolOptions)))
	}
	if p.CertFile != "" && p.KeyFile != "" {
		tlsParams = append(tlsParams, tlsconfig.ClientKeyPairFiles(p.CertFile, p.KeyFile))
	}
	if p.InsecureSkipVerify {
		tlsParams = append(tlsParams, tlsconfig.ClientInsecureSkipVerify())
	}
	tlsConfig, err := tlsconfig.NewClientConfig(tlsParams...)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "failed to build tlsConfig")
	}
	return tlsConfig, nil
}
