// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package ovs

import _ "github.com/ovn-org/libovsdb/modelgen"

//go:generate go run github.com/ovn-org/libovsdb/cmd/modelgen -o . -p ovs schema.json
