// SPDX-License-Identifier: Apache-2.0

package ovsdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovnsb"
	ovsmodel "github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovs"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

var (
	sbLock  sync.Mutex
	sbConn  client.Client
	sbModel model.DatabaseModel
)

func discoverSBEndpoint(ctx context.Context) (string, error) {
	var vswitch ovsmodel.OpenvSwitch
	if err := Get(ctx, &vswitch); err != nil {
		return "", fmt.Errorf("reading Open_vSwitch for ovn-remote: %w", err)
	}
	remote, ok := vswitch.ExternalIDs["ovn-remote"]
	if !ok || remote == "" {
		return "", fmt.Errorf("ovn-remote not found in Open_vSwitch external_ids")
	}
	return remote, nil
}

func loadTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(config.OvnSBCertificate(), config.OvnSBPrivateKey())
	if err != nil {
		return nil, fmt.Errorf("loading OVN SB TLS keypair: %w", err)
	}
	caCert, err := os.ReadFile(config.OvnSBCACert())
	if err != nil {
		return nil, fmt.Errorf("reading OVN SB CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}

func sbConnect(ctx context.Context) (client.Client, error) {
	sbLock.Lock()
	defer sbLock.Unlock()

	if sbConn != nil {
		return sbConn, nil
	}

	endpoint := config.OvnSBConnection()
	if endpoint == "" {
		var err error
		endpoint, err = discoverSBEndpoint(ctx)
		if err != nil {
			return nil, err
		}
	}

	log.Debugf("connecting to OVN SB DB: %s", endpoint)

	schema, err := ovnsb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("OVN SB FullDatabaseModel: %w", err)
	}
	mod, errs := model.NewDatabaseModel(ovnsb.Schema(), schema)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errf("OVN SB model: %s", e)
		}
		return nil, errs[len(errs)-1]
	}

	opts := []client.Option{
		client.WithEndpoint(endpoint),
		client.WithLogger(log.OvsdbLogger()),
	}

	if strings.HasPrefix(endpoint, "ssl:") {
		tlsCfg, err := loadTLSConfig()
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithTLSConfig(tlsCfg))
	}

	db, err := client.NewOVSDBClient(schema, opts...)
	if err != nil {
		return nil, fmt.Errorf("NewOVSDBClient (SB): %w", err)
	}
	if err = db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("SB db.Connect: %w", err)
	}

	sbModel = mod
	sbConn = db
	return db, nil
}

func SBList[T model.Model](ctx context.Context, results *[]T) error {
	db, err := sbConnect(ctx)
	if err != nil {
		return err
	}

	var t T
	info, err := sbModel.NewModelInfo(&t)
	if err != nil {
		return fmt.Errorf("SB NewModelInfo: %w", err)
	}

	res, err := db.Transact(ctx, ovsdb.Operation{
		Op:    ovsdb.OperationSelect,
		Table: info.Metadata.TableName,
	})
	if err != nil {
		return fmt.Errorf("SB Transact: %w", err)
	}

	for _, r := range res {
		for _, row := range r.Rows {
			var value T
			info, _ = sbModel.NewModelInfo(&value)
			err = sbModel.Mapper.GetRowData(&row, info)
			if err != nil {
				log.Errf("SB Mapper.GetRowData: %s", err)
				return err
			}
			err = info.SetField("_uuid", row["_uuid"].(ovsdb.UUID).GoUUID)
			if err != nil {
				log.Errf("SB info.SetField: %s", err)
				return err
			}
			*results = append(*results, value)
		}
	}
	return nil
}
