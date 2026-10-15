// SPDX-License-Identifier: Apache-2.0

package ovnnb

import _ "github.com/ovn-kubernetes/libovsdb/modelgen"

//go:generate go run github.com/ovn-kubernetes/libovsdb/cmd/modelgen -o . -p ovnnb schema.json
