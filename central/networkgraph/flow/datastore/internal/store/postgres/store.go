package postgres

import (
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
)

// This Flow is custom to match the existing interface and how the functionality works through the system.
// Basically for draft #1 we are trying to minimize the blast radius.
// There are many places to improve as we go, for instance, utilizing the DB for queries.  Currently the service
// layer will pass in functions that loop through all the results and filter them out.  Many of these types of things
// can be handled via query and become much more efficient.  In order to really see the benefits of Postgres for
// this store, we will need to refactor how it is used.
const (
	baseTable  = "networkflow"
	countStmt  = "SELECT COUNT(*) FROM networkflow"
	existsStmt = "SELECT EXISTS(SELECT 1 FROM networkflow WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7)"

	getStmt    = "SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId FROM networkflow WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7"
	deleteStmt = "DELETE FROM networkflow WHERE Props_SrcEntity_Type = $1 AND Props_SrcEntity_Id = $2 AND Props_DstEntity_Type = $3 AND Props_DstEntity_Id = $4 AND Props_DstPort = $5 AND Props_L4Protocol = $6 AND ClusterId = $7"
	walkStmt   = "SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId FROM networkflow"

	// These mimic how the RocksDB version of the flow store work
	getSinceStmt = "SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId FROM networkflow WHERE (LastSeenTimestamp >= $1 OR LastSeenTimestamp IS NULL) AND ClusterId = $2"
	//deleteSrcDeploymentStmt = "DELETE FROM networkflow WHERE ClusterId = $1 AND Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2"
	//deleteDstDeploymentStmt = "DELETE FROM networkflow WHERE ClusterId = $1 AND Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2"
	deleteSrcDeploymentStmt = "DELETE FROM networkflow nf USING (SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId FROM networkflow WHERE Props_SrcEntity_Type = 1 AND Props_SrcEntity_Id = $2 AND ClusterId = $2 ORDER BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId FOR UPDATE) del WHERE nf.Props_SrcEntity_Type = del.Props_SrcEntity_Type AND nf.Props_SrcEntity_Id = del.Props_SrcEntity_Id AND nf.Props_DstEntity_Type = del.Props_DstEntity_Type AND nf.Props_DstEntity_Id = del.Props_DstEntity_Id AND nf.Props_DstPort = del.Props_DstPort AND nf.Props_L4Protocol = del.Props_L4Protocol AND nf.ClusterId = del.ClusterId"

	deleteDstDeploymentStmt = "DELETE FROM networkflow nf USING (SELECT Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId FROM networkflow WHERE Props_DstEntity_Type = 1 AND Props_DstEntity_Id = $2 AND ClusterId = $2 ORDER BY Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId FOR UPDATE) del WHERE nf.Props_SrcEntity_Type = del.Props_SrcEntity_Type AND nf.Props_SrcEntity_Id = del.Props_SrcEntity_Id AND nf.Props_DstEntity_Type = del.Props_DstEntity_Type AND nf.Props_DstEntity_Id = del.Props_DstEntity_Id AND nf.Props_DstPort = del.Props_DstPort AND nf.Props_L4Protocol = del.Props_L4Protocol AND nf.ClusterId = del.ClusterId"

	deleteOrphanByTimeStmt = "DELETE FROM networkflow WHERE ClusterId = $1 AND LastSeenTimestamp IS NOT NULL AND LastSeenTimestamp < $2"
)

var (
	log = logging.LoggerForModule()

	schema = walker.Walk(reflect.TypeOf((*storage.NetworkFlow)(nil)), baseTable)

	// We begin to process in batches after this number of records
	batchAfter = 10000000

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000

	// orphanWindow, With RocksDB, all flows are retrieved and this operation is performed via
	// a function passed in.  With Postgres we can easily do this via where clause and should.
	orphanWindow = -30 * time.Minute
)

func init() {
	globaldb.RegisterTable(schema)
}

// FlowStore stores all of the flows for a single cluster.
type FlowStore interface {
	// These were autogenerated when I ran that to get started.
	// They are not currently used within the store.
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (bool, error)
	Get(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (*storage.NetworkFlow, bool, error)
	Upsert(ctx context.Context, obj *storage.NetworkFlow) error
	UpsertMany(ctx context.Context, objs []*storage.NetworkFlow) error
	Delete(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) error
	Walk(ctx context.Context, fn func(obj *storage.NetworkFlow) error) error
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)

	// GetAllFlows The methods below are the ones that match the flow interface which is what we probably have to match.
	GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)
	GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error)

	// UpsertFlows Same as other Upserts but it takes in a time
	UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error
	// RemoveFlow Same as Delete except it takes in the object vs the IDs.  Keep an eye on it.
	RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error
	// RemoveFlowsForDeployment
	RemoveFlowsForDeployment(ctx context.Context, id string) error

	// RemoveMatchingFlows We can probably phase out the functions
	// valueMatchFn checks to see if time difference vs now is greater than orphanWindow i.e. 30 minutes
	// keyMatchFn checks to see if either the source or destination are orphaned.  Orphaned means it is type deployment and the id does not exist in deployments.
	// Though that appears to be dackbox so that is gross.  May have to keep the keyMatchFn for now and replace with a join when deployments are moved to a table?
	RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error
}

type flowStoreImpl struct {
	db        *pgxpool.Pool
	clusterID string
}

func createTableNetworkflow(ctx context.Context, db *pgxpool.Pool) {
	// The flow store only deals with the identifying information, so this table has been shrunk accordingly
	// The rest of the data is looked up as the graph is built from other sources.
	table := `
create table if not exists networkflow (
    Props_SrcEntity_Type integer,
    Props_SrcEntity_Id varchar,
    Props_DstEntity_Type integer,
    Props_DstEntity_Id varchar,
    Props_DstPort integer,
    Props_L4Protocol integer,
    LastSeenTimestamp timestamp,
    ClusterId varchar,
    PRIMARY KEY(Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, ClusterId)
) 
`

	_, err := db.Exec(ctx, table)
	if err != nil {
		log.Info(err)
		panic("error creating table: " + table)
	}

	indexes := []string{
		"create index if not exists networkflow_LastSeenTimestamp on networkflow using brin(LastSeenTimestamp)  WITH (pages_per_range = 32)",
	}
	for _, index := range indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			panic(err)
		}
	}

}

func insertIntoNetworkflow(ctx context.Context, tx pgx.Tx, clusterID string, obj *storage.NetworkFlow) error {

	values := []interface{}{
		// parent primary keys start
		obj.GetProps().GetSrcEntity().GetType(),
		obj.GetProps().GetSrcEntity().GetId(),
		obj.GetProps().GetDstEntity().GetType(),
		obj.GetProps().GetDstEntity().GetId(),
		obj.GetProps().GetDstPort(),
		obj.GetProps().GetL4Protocol(),
		pgutils.NilOrTime(obj.GetLastSeenTimestamp()),
		clusterID,
	}

	finalStr := "INSERT INTO networkflow (Props_SrcEntity_Type, Props_SrcEntity_Id, Props_DstEntity_Type, Props_DstEntity_Id, Props_DstPort, Props_L4Protocol, LastSeenTimestamp, ClusterId) VALUES($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT ON CONSTRAINT networkflow_pkey DO UPDATE SET LastSeenTimestamp = EXCLUDED.LastSeenTimestamp"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		log.Info("Insert error")
		log.Info(obj)
		return err
	}

	return nil
}

func (s *flowStoreImpl) copyFromNetworkflow(ctx context.Context, tx pgx.Tx, objs ...*storage.NetworkFlow) error {

	log.Infof("copyFromNetworkFlow => %d", len(objs))

	inputRows := [][]interface{}{}
	var err error

	copyCols := []string{
		"props_srcentity_type",
		"props_srcentity_id",
		"props_dstentity_type",
		"props_dstentity_id",
		"props_dstport",
		"props_l4protocol",
		"lastseentimestamp",
		"clusterid",
	}

	for idx, obj := range objs {
		inputRows = append(inputRows, []interface{}{
			obj.GetProps().GetSrcEntity().GetType(),
			obj.GetProps().GetSrcEntity().GetId(),
			obj.GetProps().GetDstEntity().GetType(),
			obj.GetProps().GetDstEntity().GetId(),
			obj.GetProps().GetDstPort(),
			obj.GetProps().GetL4Protocol(),
			pgutils.NilOrTime(obj.GetLastSeenTimestamp()),
			s.clusterID,
		})

		_, err = tx.Exec(ctx, deleteStmt, obj.GetProps().GetSrcEntity().GetType(), obj.GetProps().GetSrcEntity().GetId(), obj.GetProps().GetDstEntity().GetType(), obj.GetProps().GetDstEntity().GetId(), obj.GetProps().GetDstPort(), obj.GetProps().GetL4Protocol(), s.clusterID)
		if err != nil {
			return err
		}

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			_, err = tx.CopyFrom(ctx, pgx.Identifier{baseTable}, copyCols, pgx.CopyFromRows(inputRows))

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
// Todo:  Another one to minimize the blast radius as all the upstream calls expect there to be a
// store per cluster and thus the cluster ID is not passed or stored with the flows.  Obviously with
// PG we will store the cluster id with the flow and pass it along, but this is how we get it in the first place.
func New(ctx context.Context, db *pgxpool.Pool, clusterID string) FlowStore {
	createTableNetworkflow(ctx, db)

	return &flowStoreImpl{
		db:        db,
		clusterID: clusterID,
	}
}

func (s *flowStoreImpl) copyFrom(ctx context.Context, objs ...*storage.NetworkFlow) error {
	conn, release := s.acquireConn(ctx, ops.Get, "NetworkFlow")
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromNetworkflow(ctx, tx, objs...); err != nil {
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

func (s *flowStoreImpl) upsert(ctx context.Context, objs ...*storage.NetworkFlow) error {
	log.Infof("upsert => %d", len(objs))
	conn, release := s.acquireConn(ctx, ops.Get, "NetworkFlow")
	defer release()

	// Moved the transaction outside the loop which greatly improved the performance of these individual inserts.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		//tx, err := conn.Begin(ctx)
		//if err != nil {
		//	return err
		//}

		if err := insertIntoNetworkflow(ctx, tx, s.clusterID, obj); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		//if err := tx.Commit(ctx); err != nil {
		//	return err
		//}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *flowStoreImpl) Upsert(ctx context.Context, obj *storage.NetworkFlow) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "NetworkFlow")

	return s.upsert(ctx, obj)
}

func (s *flowStoreImpl) UpsertMany(ctx context.Context, objs []*storage.NetworkFlow) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

	// for small batches, simply write them 1 at a time.
	if len(objs) < batchAfter {
		return s.upsert(ctx, objs...)
	}

	return s.copyFrom(ctx, objs...)
}

func (s *flowStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "NetworkFlow")

	// RocksDB implementation was adding the lastUpdatedTS to a key.  That is not necessary in PG world so that
	// parameter is not being passed forward and should be removed from the interface once RocksDB is removed.
	if len(flows) < batchAfter {
		return s.upsert(ctx, flows...)
	}

	return s.copyFrom(ctx, flows...)
}

// Count returns the number of objects in the store
func (s *flowStoreImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "NetworkFlow")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *flowStoreImpl) Exists(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "NetworkFlow")

	row := s.db.QueryRow(ctx, existsStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store
func (s *flowStoreImpl) Get(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) (*storage.NetworkFlow, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	conn, release := s.acquireConn(ctx, ops.Get, "NetworkFlow")
	defer release()

	// We can discuss this a bit, but this statement should only ever return 1 row.  Doing it this way allows
	// us to use the readRows function
	rows, err := conn.Query(ctx, getStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID)
	if err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil || flows == nil {
		return nil, false, err
	}

	return flows[0], true, nil
}

func (s *flowStoreImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func()) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	return conn, conn.Release
}

func (s *flowStoreImpl) readRows(rows pgx.Rows, pred func(*storage.NetworkFlowProperties) bool) ([]*storage.NetworkFlow, error) {
	var flows []*storage.NetworkFlow

	for rows.Next() {
		var srcType storage.NetworkEntityInfo_Type
		var srcID string
		var destType storage.NetworkEntityInfo_Type
		var destID string
		var port uint32
		var protocol storage.L4Protocol
		var lastTime *time.Time
		var clusterID string

		if err := rows.Scan(&srcType, &srcID, &destType, &destID, &port, &protocol, &lastTime, &clusterID); err != nil {
			log.Info(err)
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		var ts *types.Timestamp
		if lastTime != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*lastTime)
		}

		flow := &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: srcType,
					Id:   srcID,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: destType,
					Id:   destID,
				},
				DstPort:    port,
				L4Protocol: protocol,
			},
			LastSeenTimestamp: ts,
			ClusterId:         clusterID,
		}

		// Apply the predicate function.  Will phase out as we move away form Rocks to where clause
		if pred == nil || pred(flow.Props) {
			flows = append(flows, flow)
		}
	}

	log.Debugf("Read returned %d flows", len(flows))
	return flows, nil
}

// Delete removes the specified ID from the store
func (s *flowStoreImpl) Delete(ctx context.Context, propsSrcEntityType storage.NetworkEntityInfo_Type, propsSrcEntityID string, propsDstEntityType storage.NetworkEntityInfo_Type, propsDstEntityID string, propsDstPort uint32, propsL4Protocol storage.L4Protocol) error {
	log.Info("Delete")
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, propsSrcEntityType, propsSrcEntityID, propsDstEntityType, propsDstEntityID, propsDstPort, propsL4Protocol, s.clusterID); err != nil {
		return err
	}
	return nil
}

// Walk iterates over all of the objects in the store and applies the closure
// Todo: investigate this method to see if it is doing what it should
func (s *flowStoreImpl) Walk(ctx context.Context, fn func(obj *storage.NetworkFlow) error) error {
	rows, err := s.db.Query(ctx, walkStmt)
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var srcType storage.NetworkEntityInfo_Type
		var srcID string
		var destType storage.NetworkEntityInfo_Type
		var destID string
		var port uint32
		var protocol storage.L4Protocol
		var lastTime *time.Time
		var clusterID string

		if err := rows.Scan(&srcType, &srcID, &destType, &destID, &port, &protocol, &lastTime, &clusterID); err != nil {
			log.Info(err)
			return nil
		}

		var ts *types.Timestamp
		if lastTime != nil {
			ts = protoconv.MustConvertTimeToTimestamp(*lastTime)
		}

		flow := &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: srcType,
					Id:   srcID,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: destType,
					Id:   destID,
				},
				DstPort:    port,
				L4Protocol: protocol,
			},
			LastSeenTimestamp: ts,
			ClusterId:         clusterID,
		}

		if err := fn(flow); err != nil {
			return err
		}
	}
	return nil
}

// RemoveFlowsForDeployment removes all flows where the source OR destination match the deployment id
func (s *flowStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	log.Infof("RemoveFlowsForDeployment => %s", id)
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteSrcDeploymentStmt, id, s.clusterID); err != nil {
		log.Info(err)
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	tx, err = conn.Begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteDstDeploymentStmt, id, s.clusterID); err != nil {
		log.Info(err)
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

// GetAllFlows returns the object, if it exists from the store, timestamp and error
func (s *flowStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NetworkFlow")

	var rows pgx.Rows
	var err error
	// Default to Now as that is when we are reading them
	lastUpdateTS := *types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}
	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, nil)
	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}

	return flows, lastUpdateTS, nil
}

// GetMatchingFlows iterates over all of the objects in the store and applies the closure
func (s *flowStoreImpl) GetMatchingFlows(ctx context.Context, pred func(*storage.NetworkFlowProperties) bool, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	var rows pgx.Rows
	var err error

	// Default to Now as that is when we are reading them
	lastUpdateTS := *types.TimestampNow()

	// handling case when since is nil.  Assumption is we want everything in that case vs when date is not null
	if since == nil {
		rows, err = s.db.Query(ctx, walkStmt)
	} else {
		rows, err = s.db.Query(ctx, getSinceStmt, pgutils.NilOrTime(since), s.clusterID)
	}

	if err != nil {
		return nil, types.Timestamp{}, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	flows, err := s.readRows(rows, pred)

	return flows, lastUpdateTS, err
}

func (s *flowStoreImpl) delete(ctx context.Context, objs ...*storage.NetworkFlowProperties) error {
	log.Info("delete")
	conn, release := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	defer release()

	// Moved the transaction outside the loop which greatly improved the performance of these individual inserts.
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		_, err := tx.Exec(ctx, deleteStmt, obj.GetSrcEntity().GetType(), obj.GetSrcEntity().GetId(), obj.GetDstEntity().GetType(), obj.GetDstEntity().GetId(), obj.GetDstPort(), obj.GetL4Protocol(), s.clusterID)

		if err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}

	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// RemoveFlow removes the specified flow from the store
func (s *flowStoreImpl) RemoveFlow(ctx context.Context, props *storage.NetworkFlowProperties) error {
	log.Info("RemoveFlow")
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	if err := s.delete(ctx, props); err != nil {
		return err
	}
	return nil
}

// RemoveMatchingFlows removes the specified flows from the store
// Todo: Figure out what to do with the functions.
func (s *flowStoreImpl) RemoveMatchingFlows(ctx context.Context, keyMatchFn func(props *storage.NetworkFlowProperties) bool, valueMatchFn func(flow *storage.NetworkFlow) bool) error {
	log.Info("RemoveMatchingFlows")
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "NetworkFlow")

	conn, release := s.acquireConn(ctx, ops.Remove, "NetworkFlow")
	defer release()

	// want to do all this as a single transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	// That operation is easy to do in SQL.  If the valueMatchFn is not null then based on the only place
	// it is set, we are deleting flows orphaned based on elapsed time.
	if valueMatchFn != nil {
		// Delete based on orphan time
		// Do this first because I can do this delete easily via SQL, but until there is a deployment table in PG,
		// I need to loop over all the IDs in the cluster if keyMatchFn is not null.
		deleteBefore := time.Now().Add(orphanWindow)

		if _, err := tx.Exec(ctx, deleteOrphanByTimeStmt, s.clusterID, deleteBefore); err != nil {
			return err
		}
	}

	// This operation matches if the either the dest or src deployment no longer exists.
	// We should be able to do that easily in SQL if we have access to the table with the active deployments.
	// Then we just have to make sure the functions that do this for Rocks don't change.
	// For now continue to pull EVERYTHING and compare it to the function which just checks to see if a deployment exists.
	// If the deployment does not exist, delete the row.
	if keyMatchFn != nil {
		// Run the query in the transaction to make sure I don't get stuff that may have been deleted by date.
		rows, err := tx.Query(ctx, walkStmt)
		if err != nil {
			return err
		}
		defer rows.Close()

		deleteFlows, err := s.readRows(rows, keyMatchFn)

		if err != nil {
			return nil
		}

		for _, flow := range deleteFlows {
			_, err := tx.Exec(ctx, deleteStmt, flow.GetProps().GetSrcEntity().GetType(), flow.GetProps().GetSrcEntity().GetId(), flow.GetProps().GetDstEntity().GetType(), flow.GetProps().GetDstEntity().GetId(), flow.GetProps().GetDstPort(), flow.GetProps().GetL4Protocol(), s.clusterID)

			if err != nil {
				if err := tx.Rollback(ctx); err != nil {
					return err
				}
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

//// Used for testing

func dropTableNetworkflow(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS networkflow CASCADE")
}

// Destroy destroys the tables
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableNetworkflow(ctx, db)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *flowStoreImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *flowStoreImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
