// SPDX-License-Identifier: Apache-2.0

package ovnsb

import _ "github.com/ovn-kubernetes/libovsdb/modelgen"

//go:generate go run github.com/ovn-kubernetes/libovsdb/cmd/modelgen -o . -p ovnsb schema.json
