// Code generated by pg-bindings generator. DO NOT EDIT.

package postgres

import (
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

const (
	baseTable  = "alerts"
	countStmt  = "SELECT COUNT(*) FROM alerts"
	existsStmt = "SELECT EXISTS(SELECT 1 FROM alerts WHERE Id = $1)"

	getStmt     = "SELECT serialized FROM alerts WHERE Id = $1"
	deleteStmt  = "DELETE FROM alerts WHERE Id = $1"
	walkStmt    = "SELECT serialized FROM alerts"
	getIDsStmt  = "SELECT Id FROM alerts"
	getManyStmt = "SELECT serialized FROM alerts WHERE Id = ANY($1::text[])"

	deleteManyStmt = "DELETE FROM alerts WHERE Id = ANY($1::text[])"

	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

var (
	schema = walker.Walk(reflect.TypeOf((*storage.Alert)(nil)), baseTable)
	log    = logging.LoggerForModule()
)

func init() {
	globaldb.RegisterTable(schema)
}

type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.Alert, bool, error)
	Upsert(ctx context.Context, obj *storage.Alert) error
	UpsertMany(ctx context.Context, objs []*storage.Alert) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error)
	DeleteMany(ctx context.Context, ids []string) error

	Walk(ctx context.Context, fn func(obj *storage.Alert) error) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
	db *pgxpool.Pool
}

func createTableAlerts(ctx context.Context, db *pgxpool.Pool) {
	table := `
create table if not exists alerts (
    Id varchar,
    Policy_Id varchar,
    Policy_Name varchar,
    Policy_Description varchar,
    Policy_Disabled bool,
    Policy_Categories text[],
    Policy_LifecycleStages int[],
    Policy_Severity integer,
    Policy_EnforcementActions int[],
    Policy_LastUpdated timestamp,
    Policy_SORTName varchar,
    Policy_SORTLifecycleStage varchar,
    Policy_SORTEnforcement bool,
    LifecycleStage integer,
    Deployment_Id varchar,
    Deployment_Name varchar,
    Deployment_Inactive bool,
    Image_Id varchar,
    Image_Name_Registry varchar,
    Image_Name_Remote varchar,
    Image_Name_Tag varchar,
    Image_Name_FullName varchar,
    Time timestamp,
    State integer,
    serialized bytea,
    PRIMARY KEY(Id)
)
`

	_, err := db.Exec(ctx, table)
	if err != nil {
		log.Panicf("Error creating table %s: %v", table, err)
	}

	indexes := []string{

		"create index if not exists alerts_LifecycleStage on alerts using btree(LifecycleStage)",
	}
	for _, index := range indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Panicf("Error creating index %s: %v", index, err)
		}
	}

}

func insertIntoAlerts(ctx context.Context, tx pgx.Tx, obj *storage.Alert) error {

	serialized, marshalErr := obj.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetPolicy().GetId(),
		obj.GetPolicy().GetName(),
		obj.GetPolicy().GetDescription(),
		obj.GetPolicy().GetDisabled(),
		obj.GetPolicy().GetCategories(),
		obj.GetPolicy().GetLifecycleStages(),
		obj.GetPolicy().GetSeverity(),
		obj.GetPolicy().GetEnforcementActions(),
		pgutils.NilOrTime(obj.GetPolicy().GetLastUpdated()),
		obj.GetPolicy().GetSORTName(),
		obj.GetPolicy().GetSORTLifecycleStage(),
		obj.GetPolicy().GetSORTEnforcement(),
		obj.GetLifecycleStage(),
		obj.GetDeployment().GetId(),
		obj.GetDeployment().GetName(),
		obj.GetDeployment().GetInactive(),
		obj.GetImage().GetId(),
		obj.GetImage().GetName().GetRegistry(),
		obj.GetImage().GetName().GetRemote(),
		obj.GetImage().GetName().GetTag(),
		obj.GetImage().GetName().GetFullName(),
		pgutils.NilOrTime(obj.GetTime()),
		obj.GetState(),
		serialized,
	}

	finalStr := "INSERT INTO alerts (Id, Policy_Id, Policy_Name, Policy_Description, Policy_Disabled, Policy_Categories, Policy_LifecycleStages, Policy_Severity, Policy_EnforcementActions, Policy_LastUpdated, Policy_SORTName, Policy_SORTLifecycleStage, Policy_SORTEnforcement, LifecycleStage, Deployment_Id, Deployment_Name, Deployment_Inactive, Image_Id, Image_Name_Registry, Image_Name_Remote, Image_Name_Tag, Image_Name_FullName, Time, State, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Policy_Id = EXCLUDED.Policy_Id, Policy_Name = EXCLUDED.Policy_Name, Policy_Description = EXCLUDED.Policy_Description, Policy_Disabled = EXCLUDED.Policy_Disabled, Policy_Categories = EXCLUDED.Policy_Categories, Policy_LifecycleStages = EXCLUDED.Policy_LifecycleStages, Policy_Severity = EXCLUDED.Policy_Severity, Policy_EnforcementActions = EXCLUDED.Policy_EnforcementActions, Policy_LastUpdated = EXCLUDED.Policy_LastUpdated, Policy_SORTName = EXCLUDED.Policy_SORTName, Policy_SORTLifecycleStage = EXCLUDED.Policy_SORTLifecycleStage, Policy_SORTEnforcement = EXCLUDED.Policy_SORTEnforcement, LifecycleStage = EXCLUDED.LifecycleStage, Deployment_Id = EXCLUDED.Deployment_Id, Deployment_Name = EXCLUDED.Deployment_Name, Deployment_Inactive = EXCLUDED.Deployment_Inactive, Image_Id = EXCLUDED.Image_Id, Image_Name_Registry = EXCLUDED.Image_Name_Registry, Image_Name_Remote = EXCLUDED.Image_Name_Remote, Image_Name_Tag = EXCLUDED.Image_Name_Tag, Image_Name_FullName = EXCLUDED.Image_Name_FullName, Time = EXCLUDED.Time, State = EXCLUDED.State, serialized = EXCLUDED.serialized"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

func (s *storeImpl) copyFromAlerts(ctx context.Context, tx pgx.Tx, objs ...*storage.Alert) error {

	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"policy_id",

		"policy_name",

		"policy_description",

		"policy_disabled",

		"policy_categories",

		"policy_lifecyclestages",

		"policy_severity",

		"policy_enforcementactions",

		"policy_lastupdated",

		"policy_sortname",

		"policy_sortlifecyclestage",

		"policy_sortenforcement",

		"lifecyclestage",

		"deployment_id",

		"deployment_name",

		"deployment_inactive",

		"image_id",

		"image_name_registry",

		"image_name_remote",

		"image_name_tag",

		"image_name_fullname",

		"time",

		"state",

		"serialized",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj in the loop is not used as it only consists of the parent id and the idx.  Putting this here as a stop gap to simply use the object.  %s", obj)

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{

			obj.GetId(),

			obj.GetPolicy().GetId(),

			obj.GetPolicy().GetName(),

			obj.GetPolicy().GetDescription(),

			obj.GetPolicy().GetDisabled(),

			obj.GetPolicy().GetCategories(),

			obj.GetPolicy().GetLifecycleStages(),

			obj.GetPolicy().GetSeverity(),

			obj.GetPolicy().GetEnforcementActions(),

			pgutils.NilOrTime(obj.GetPolicy().GetLastUpdated()),

			obj.GetPolicy().GetSORTName(),

			obj.GetPolicy().GetSORTLifecycleStage(),

			obj.GetPolicy().GetSORTEnforcement(),

			obj.GetLifecycleStage(),

			obj.GetDeployment().GetId(),

			obj.GetDeployment().GetName(),

			obj.GetDeployment().GetInactive(),

			obj.GetImage().GetId(),

			obj.GetImage().GetName().GetRegistry(),

			obj.GetImage().GetName().GetRemote(),

			obj.GetImage().GetName().GetTag(),

			obj.GetImage().GetName().GetFullName(),

			pgutils.NilOrTime(obj.GetTime()),

			obj.GetState(),

			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}
			// clear the inserts and vals for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"alerts"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

// New returns a new Store instance using the provided sql instance.
func New(ctx context.Context, db *pgxpool.Pool) Store {
	createTableAlerts(ctx, db)

	return &storeImpl{
		db: db,
	}
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.Alert) error {
	conn, release := s.acquireConn(ctx, ops.Get, "Alert")
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromAlerts(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.Alert) error {
	conn, release := s.acquireConn(ctx, ops.Get, "Alert")
	defer release()

	for _, obj := range objs {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		if err := insertIntoAlerts(ctx, tx, obj); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Alert) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "Alert")

	return s.upsert(ctx, obj)
}

func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.Alert) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "Alert")

	if len(objs) < batchAfter {
		return s.upsert(ctx, objs...)
	} else {
		return s.copyFrom(ctx, objs...)
	}
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "Alert")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "Alert")

	row := s.db.QueryRow(ctx, existsStmt, id)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Alert, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "Alert")

	conn, release := s.acquireConn(ctx, ops.Get, "Alert")
	defer release()

	row := conn.QueryRow(ctx, getStmt, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.Alert
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func()) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	return conn, conn.Release
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Alert")

	conn, release := s.acquireConn(ctx, ops.Remove, "Alert")
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, id); err != nil {
		return err
	}
	return nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "storage.AlertIDs")

	rows, err := s.db.Query(ctx, getIDsStmt)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.Alert, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Alert")

	conn, release := s.acquireConn(ctx, ops.GetMany, "Alert")
	defer release()

	rows, err := conn.Query(ctx, getManyStmt, ids)
	if err != nil {
		if err == pgx.ErrNoRows {
			missingIndices := make([]int, 0, len(ids))
			for i := range ids {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	defer rows.Close()
	resultsByID := make(map[string]*storage.Alert)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		msg := &storage.Alert{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, nil, err
		}
		resultsByID[msg.GetId()] = msg
	}
	missingIndices := make([]int, 0, len(ids)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input ids
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*storage.Alert, 0, len(resultsByID))
	for i, id := range ids {
		if result, ok := resultsByID[id]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ctx context.Context, ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "Alert")

	conn, release := s.acquireConn(ctx, ops.RemoveMany, "Alert")
	defer release()
	if _, err := conn.Exec(ctx, deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *storage.Alert) error) error {
	rows, err := s.db.Query(ctx, walkStmt)
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		var msg storage.Alert
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		if err := fn(&msg); err != nil {
			return err
		}
	}
	return nil
}

//// Used for testing

func dropTableAlerts(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS alerts CASCADE")

}

func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableAlerts(ctx, db)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
