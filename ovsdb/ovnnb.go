// SPDX-License-Identifier: Apache-2.0

package ovsdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovnnb"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

var (
	nbLock  sync.Mutex
	nbConn  client.Client
	nbModel model.DatabaseModel
)

func loadNBTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(config.OvnNBCertificate(), config.OvnNBPrivateKey())
	if err != nil {
		return nil, fmt.Errorf("loading OVN NB TLS keypair: %w", err)
	}
	caCert, err := os.ReadFile(config.OvnNBCACert())
	if err != nil {
		return nil, fmt.Errorf("reading OVN NB CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}, nil
}

func nbConnect(ctx context.Context) (client.Client, error) {
	nbLock.Lock()
	defer nbLock.Unlock()

	if nbConn != nil {
		return nbConn, nil
	}

	endpoint := config.OvnNBConnection()
	if endpoint == "" {
		return nil, fmt.Errorf("ovn-nb-connection not configured")
	}

	log.Debugf("connecting to OVN NB DB: %s", endpoint)

	schema, err := ovnnb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("OVN NB FullDatabaseModel: %w", err)
	}
	mod, errs := model.NewDatabaseModel(ovnnb.Schema(), schema)
	if len(errs) > 0 {
		for _, e := range errs {
			log.Errf("OVN NB model: %s", e)
		}
		return nil, errs[len(errs)-1]
	}

	opts := []client.Option{
		client.WithEndpoint(endpoint),
		client.WithLogger(log.OvsdbLogger()),
	}

	if strings.HasPrefix(endpoint, "ssl:") {
		tlsCfg, err := loadNBTLSConfig()
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithTLSConfig(tlsCfg))
	}

	db, err := client.NewOVSDBClient(schema, opts...)
	if err != nil {
		return nil, fmt.Errorf("NewOVSDBClient (NB): %w", err)
	}
	if err = db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("NB db.Connect: %w", err)
	}

	nbModel = mod
	nbConn = db
	go nbForgetOnDisconnect(db)
	return db, nil
}

func nbForgetOnDisconnect(db client.Client) {
	<-db.DisconnectNotify()
	nbForget(db)
}

func nbForget(db client.Client) {
	nbLock.Lock()
	defer nbLock.Unlock()
	if nbConn != db {
		return
	}
	log.Warningf("lost connection to the OVN NB DB, reconnecting on the next call")
	db.Close()
	nbConn = nil
}

func NBList[T model.Model](ctx context.Context, results *[]T) error {
	db, err := nbConnect(ctx)
	if err != nil {
		return err
	}

	var t T
	info, err := nbModel.NewModelInfo(&t)
	if err != nil {
		return fmt.Errorf("NB NewModelInfo: %w", err)
	}

	res, err := db.Transact(ctx, ovsdb.Operation{
		Op:    ovsdb.OperationSelect,
		Table: info.Metadata.TableName,
	})
	if err != nil {
		if errors.Is(err, client.ErrNotConnected) {
			nbForget(db)
		}
		return fmt.Errorf("NB Transact: %w", err)
	}

	for _, r := range res {
		for _, row := range r.Rows {
			var value T
			info, _ = nbModel.NewModelInfo(&value)
			err = nbModel.Mapper.GetRowData(&row, info)
			if err != nil {
				log.Errf("NB Mapper.GetRowData: %s", err)
				return err
			}
			err = info.SetField("_uuid", row["_uuid"].(ovsdb.UUID).GoUUID)
			if err != nil {
				log.Errf("NB info.SetField: %s", err)
				return err
			}
			*results = append(*results, value)
		}
	}
	return nil
}
