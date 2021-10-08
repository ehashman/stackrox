package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
)

const storeFile = `

package pg

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"database/sql"
)

var (
	log = logging.LoggerForModule()

	table = "{{.Table}}"
)

type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)
	GetIDs() ([]string, error)
	Get(id string) (*storage.{{.Type}}, bool, error)
	GetMany(ids []string) ([]*storage.{{.Type}}, []int, error)
	{{- if .NoKeyField}}
	UpsertWithID(id string, obj *storage.{{.Type}}) error
	UpsertManyWithIDs(ids []string, objs []*storage.{{.Type}}) error
	{{- else }}
	Upsert(obj *storage.{{.Type}}) error
	UpsertMany(objs []*storage.{{.Type}}) error
	{{- end}}
	Delete(id string) error
	DeleteMany(ids []string) error
	{{- if .NoKeyField}}
	WalkAllWithID(fn func(id string, obj *storage.{{.Type}}) error) error
	{{- else }}
	Walk(fn func(obj *storage.{{.Type}}) error) error
	{{- end}}
	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}

type storeImpl struct {
	db *sql.DB

	countStmt *sql.Stmt
	existsStmt *sql.Stmt
	getIDsStmt *sql.Stmt
	getStmt *sql.Stmt
	getManyStmt *sql.Stmt
	upsertWithIDStmt *sql.Stmt
	// UpsertManyWithIDsStmt
	upsertStmt *sql.Stmt
	// UpsertMany
	deleteStmt *sql.Stmt
	deleteManyStmt *sql.Stmt
}

func alloc() proto.Message {
	return &storage.{{.Type}}{}
}

{{- if not .NoKeyField}}

func keyFunc(msg proto.Message) string {
	return msg.(*storage.{{.Type}}).{{.KeyFunc}}
}
{{- end}}

{{- if .UniqKeyFunc}}

func uniqKeyFunc(msg proto.Message) string {
	return msg.(*storage.{{.Type}}).{{.UniqKeyFunc}}
}
{{- end}}

func compileStmtOrPanic(db *sql.DB, query string) *sql.Stmt {
	vulnStmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	return vulnStmt
}

// New returns a new Store instance using the provided sql instance.
func New(db *sql.DB) Store {
	globaldb.RegisterTable(table, "{{.Type}}")
//	{{- if .UniqKeyFunc}}
//	return &storeImpl{
//		crud: generic.NewUniqueKeyCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, allocCluster, uniqKeyFunc, {{.TrackIndex}}),
//	}
//	{{- else}}
	return &storeImpl{
		db: db,

		countStmt: compileStmtOrPanic(db, "select count(*) from {{.Table}}"),
		existsStmt: compileStmtOrPanic(db, "select exists(select 1 from {{.Table}} where id = $1)"),
		getIDsStmt: compileStmtOrPanic(db, "select id from {{.Table}}"),
		getStmt: compileStmtOrPanic(db, ""),
		getManyStmt: compileStmtOrPanic(db, ""),
		upsertWithIDStmt: compileStmtOrPanic(db, ""),
		// UpsertManyWithIDsStmt
		upsertStmt: compileStmtOrPanic(db, ""),
		// UpsertMany
		deleteStmt: compileStmtOrPanic(db, ""),
		deleteManyStmt: compileStmtOrPanic(db, ""),
	}
//	{{- end}}
}

// Count returns the number of objects in the store
func (s *storeImpl) Count() (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "{{.Type}}")

	row := s.countStmt.QueryRow()
	if err := row.Err(); err != nil {
		return 0, err
	}
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "{{.Type}}")

	row := s.existsStmt.QueryRow()
	if err := row.Err(); err != nil {
		return false, err
	}
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs() ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	rows, err := s.getIDsStmt.Query()
	if err != nil {
		return nil, err
	}
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

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(id string) (*storage.{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.Type}}")

	panic("unimplemented")

	msg, exists, err := b.crud.Get(id)
	if err != nil || !exists {
		return nil, false, err
	}
	return msg.(*storage.{{.Type}}), true, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice 
func (s *storeImpl) GetMany(ids []string) ([]*storage.{{.Type}}, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "{{.Type}}")

	panic("unimplemented")

	//msgs, missingIndices, err := b.crud.GetMany(ids)
	//if err != nil {
	//	return nil, nil, err
	//}
	//objs := make([]*storage.{{.Type}}, 0, len(msgs))
	//for _, m := range msgs {
	//	objs = append(objs, m.(*storage.{{.Type}}))
	//}
	//return objs, missingIndices, nil
}

{{- if .NoKeyField}}
// UpsertWithID inserts the object into the DB
func (s *storeImpl) UpsertWithID(id string, obj *storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")

	

	//
	//return b.crud.UpsertWithID(id, obj)
}

// UpsertManyWithIDs batches objects into the DB
func (s *storeImpl) UpsertManyWithIDs(ids []string, objs []*storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")

	//msgs := make([]proto.Message, 0, len(objs))
	//for _, o := range objs {
	//	msgs = append(msgs, o)
    //}
	//
	//return b.crud.UpsertManyWithIDs(ids, msgs)
}
{{- else}}

// Upsert inserts the object into the DB
func (s *storeImpl) Upsert(obj *storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")

	

	panic("unimplemented")

	//return b.crud.Upsert(obj)
}

// UpsertMany batches objects into the DB
func (s *storeImpl) UpsertMany(objs []*storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")
	panic("unimplemented")
	//msgs := make([]proto.Message, 0, len(objs))
	//for _, o := range objs {
	//	msgs = append(msgs, o)
    //}
	//
	//return b.crud.UpsertMany(msgs)
}
{{- end}}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.Type}}")
	panic("unimplemented")
	//return b.crud.Delete(id)
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.Type}}")
	panic("unimplemented")
		
	//
	//return b.crud.DeleteMany(ids)
}

{{- if .NoKeyField}}
// WalkAllWithID iterates over all of the objects in the store and applies the closure
func (s *storeImpl) WalkAllWithID(fn func(id string, obj *storage.{{.Type}}) error) error {
	panic("unimplemented")	
//return b.crud.WalkAllWithID(func(id []byte, msg proto.Message) error {
	//	return fn(string(id), msg.(*storage.{{.Type}}))
	//})
}
{{- else}}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(fn func(obj *storage.{{.Type}}) error) error {
	panic("unimplemented")
	//return b.crud.Walk(func(msg proto.Message) error {
	//	return fn(msg.(*storage.{{.Type}}))
	//})
}
{{- end}}

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex() ([]string, error) {
	return nil, nil
}
`

type properties struct {
	Type        string
	Table      string
	NoKeyField  bool
	KeyFunc     string
	UniqKeyFunc string
	Cache       bool
	TrackIndex  bool
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects")
	utils.Must(c.MarkFlagRequired("table"))

	c.Flags().BoolVar(&props.NoKeyField, "no-key-field", false, "whether or not object contains key field. If no, then to key function is not applied on object")
	c.Flags().StringVar(&props.KeyFunc, "key-func", "GetId()", "the function on the object to retrieve the key")
	c.Flags().StringVar(&props.UniqKeyFunc, "uniq-key-func", "", "when set, unique key constraint is added on the object field retrieved by the function")
	c.Flags().BoolVar(&props.Cache, "cache", false, "whether or not to add a fully inmem cache")
	c.Flags().BoolVar(&props.TrackIndex, "track-index", false, "whether or not to track the index updates and wait for them to be acknowledged")

	c.RunE = func(*cobra.Command, []string) error {
		templateMap := map[string]interface{}{
			"Type":        props.Type,
			"Bucket":      props.Table,
			"NoKeyField":  props.NoKeyField,
			"KeyFunc":     props.KeyFunc,
			"UniqKeyFunc": props.UniqKeyFunc,
			"Cache":       props.Cache,
			"TrackIndex":  props.TrackIndex,
			"Table":       props.Table,
		}

		t := template.Must(template.New("gen").Parse(autogenerated + storeFile))
		buf := bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := ioutil.WriteFile("store.go", buf.Bytes(), 0644); err != nil {
			return err
		}
		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
