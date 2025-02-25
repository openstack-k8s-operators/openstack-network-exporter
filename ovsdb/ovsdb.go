// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package ovsdb

import (
	"context"
	"fmt"
	"sync"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovs"
	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
)

var (
	ovsdbLock  sync.Mutex
	ovsdbConn  client.Client
	ovsdbModel model.DatabaseModel
)

func connect(ctx context.Context) (client.Client, error) {
	ovsdbLock.Lock()
	defer ovsdbLock.Unlock()

	if ovsdbConn != nil {
		return ovsdbConn, nil
	}

	endpoint := fmt.Sprintf("unix:%s/db.sock", config.OvsRundir())

	log.Debugf("connecting to ovsdb: %s", endpoint)

	schema, err := ovs.FullDatabaseModel()
	if err != nil {
		log.Errf("NewOVSDBClient: %s", err)
		return nil, err
	}
	mod, errs := model.NewDatabaseModel(ovs.Schema(), schema)
	if len(errs) > 0 {
		for _, err = range errs {
			log.Errf("model.NewDatabaseModel: %s", err)
		}
		return nil, err
	}

	db, err := client.NewOVSDBClient(
		schema,
		client.WithEndpoint(endpoint),
		client.WithLogger(log.OvsdbLogger()),
	)
	if err != nil {
		log.Errf("NewOVSDBClient: %s", err)
		return nil, err
	}
	if err = db.Connect(ctx); err != nil {
		log.Errf("db.Connect: %s", err)
		return nil, err
	}

	ovsdbModel = mod
	ovsdbConn = db

	return db, nil
}

func Get(ctx context.Context, result model.Model) error {
	db, err := connect(ctx)
	if err != nil {
		log.Errf("connect: %s", err)
		return err
	}

	info, err := ovsdbModel.NewModelInfo(result)
	if err != nil {
		log.Errf("NewModelInfo: %s", err)
		return err
	}
	res, err := db.Transact(ctx, ovsdb.Operation{
		Op:    ovsdb.OperationSelect,
		Table: info.Metadata.TableName,
	})
	if err != nil {
		log.Errf("Transact: %s", err)
		return err
	}
	for _, r := range res {
		for _, row := range r.Rows {
			err = info.SetField("_uuid", row["_uuid"].(ovsdb.UUID).GoUUID)
			if err != nil {
				log.Errf("info.SetField: %s", err)
				return err
			}
			return ovsdbModel.Mapper.GetRowData(&row, info) //nolint: staticcheck // the surrounding loop is unconditionally terminated
		}
	}
	return client.ErrNotFound
}

func List[T model.Model](ctx context.Context, results *[]T) error {
	db, err := connect(ctx)
	if err != nil {
		log.Errf("connect: %s", err)
		return err
	}

	var t T

	info, err := ovsdbModel.NewModelInfo(&t)
	if err != nil {
		log.Errf("NewModelInfo: %s", err)
		return err
	}

	res, err := db.Transact(ctx, ovsdb.Operation{
		Op:    ovsdb.OperationSelect,
		Table: info.Metadata.TableName,
	})
	if err != nil {
		log.Errf("Transact: %s", err)
		return err
	}

	for _, r := range res {
		for _, row := range r.Rows {
			var value T
			info, _ = ovsdbModel.NewModelInfo(&value)
			err = ovsdbModel.Mapper.GetRowData(&row, info)
			if err != nil {
				log.Errf("Mapper.GetRowData: %s", err)
				return err
			}
			err = info.SetField("_uuid", row["_uuid"].(ovsdb.UUID).GoUUID)
			if err != nil {
				log.Errf("info.SetField: %s", err)
				return err
			}
			*results = append(*results, value)
		}
	}
	return nil
}
