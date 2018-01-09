package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
	"time"
)

// IModelMongo mongo model interface
type IModelMongo interface {
	GetCreatedTime() time.Time
	InitTime(t time.Time)
	SetUpdatedTime(t time.Time)
	SetId(id string)
	GetId() string
}

type ModelMongo struct {
	Id         string    `bson:"_id",json:"id"`
	Created_at time.Time `bson:"created_at,omitempty" json:"created_at"`
	Updated_at time.Time `bson:"updated_at,omitempty" json:"updated_at"`
}

func (m *ModelMongo) GetCreatedTime() time.Time {
	return m.Created_at
}

func (m *ModelMongo) InitTime(t time.Time) {
	m.Created_at = t
	m.Updated_at = t
}
func (m *ModelMongo) SetUpdatedTime(t time.Time) {
	m.Updated_at = t
}

func (m *ModelMongo) SetId(id string) {
	m.Id = id
}

func (m *ModelMongo) GetId() string {
	return m.Id
}

// Mongo mongo
type Mongo struct {
	DbName         string
	CollectionName string

	AutoIncrementId bool

	PrimaryKey string

	Mode    string
	Refresh bool
}

// MongoCounter 计数器
type MongoCounter struct {
	Id  string `bson:"_id,omitempty"`
	Seq int64  `bson:"seq,omitempty"`
}

// NewMongo
func NewMongo() *Mongo {

	return &Mongo{}
}

var (
	sessionMongo *mgo.Session
	configMongo  *config.Mongo
)

func init() {
	if config.FeatureMongo() {

		for _, c := range config.MongoGetAll() {
			configMongo := c.Conn
			if strings.Trim(configMongo.ReadOption, " ") == "" {
				configMongo.ReadOption = "nearest"
			}

			connectionString := fmt.Sprintf("mongodb://%s", configMongo.Servers)

			var err error
			sessionMongo, err = mgo.Dial(connectionString)

			if err != nil {

				log.Logf(log.LevelFatal, "connect to mongo server error:%s,%s", err.Error(), connectionString)
				panic("connect to mongo server")

			}
			sessionMongo.SetPoolLimit(configMongo.PoolLimit)
		}

	}
}

// SetMode 设置模式
func (p *Mongo) SetMode(session *mgo.Session, dft string) {
	var mode mgo.Mode
	modeStr := p.Mode
	if modeStr == "" {
		modeStr = dft
	}

	switch strings.ToUpper(modeStr) {
	case "EVENTUAL":
		mode = mgo.Eventual
	case "MONOTONIC":
		mode = mgo.Monotonic
	case "PRIMARYPREFERRED":
		mode = mgo.PrimaryPreferred
	case "SECONDARY":
		mode = mgo.Secondary
	case "SECONDARYPREFERRED":
		mode = mgo.SecondaryPreferred
	case "NEAREST":
		mode = mgo.Nearest
	default:
		mode = mgo.Strong
	}

	if session.Mode() != mode {
		session.SetMode(mode, p.Refresh)
	}
}

// GetId 获取id
func (p *Mongo) GetId(ctx context.Context) (string, error) {
	return p.GetNextSequence(ctx)
}

func (p *Mongo) getDbName() string {
	if p.DbName != "" {
		return p.DbName
	}
	for _, conf := range config.MongoGetAll() {
		if conf.Db != "" {
			return conf.Db
		}
	}

	return ""
}

// GetSession 获取session
func (p *Mongo) GetSession(span opentracing.Span) (*mgo.Session, string, error) {

	dbName := p.getDbName()

	configMongo := config.MongoGet(dbName).Conn

	if sessionMongo != nil {
		clone := sessionMongo.Clone()
		p.SetMode(clone, configMongo.ReadOption)
		return clone, dbName, nil
	}
	msg := "session mongo is nul"

	err := p.processError(span, errors.New("Mongo Error"), pconst.ERROR_MONGO_SESSION, msg)

	if span != nil {
		span.SetTag("session_err", err)
	}
	return nil, dbName, err
}
func (p *Mongo) GetNextSequence(ctx context.Context) (string, error) {

	span, ctx := p.ZipkinNewSpan(ctx, "getid")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return "0", err
	}
	defer session.Close()

	c := session.DB(dbName).C("counters")

	condition := bson.M{"_id": p.CollectionName}

	//_, errUpsert := c.Upsert(condition, bson.M{"$inc": bson.M{"seq": 1}})

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		Upsert:    true,
		ReturnNew: true,
	}
	result := bson.M{}

	_, errApply := c.Find(condition).Apply(change, &result)

	if errApply != nil {
		errApply = p.processError(span, errApply, pconst.ERROR_MONGO_FIND, "mongo findAndModify counter %s failed:%s", p.CollectionName, errApply.Error())
		terr := terror.NewFromError(errApply)
		terr.Code = pconst.ERROR_MONGO_SEQUENCE
		return "0", terr
	}

	setInt, resultNext := result["seq"].(int)

	var seq int64
	if !resultNext {
		seq, resultNext = result["seq"].(int64)

		if !resultNext {
			log.Errorf("mongo findAndModify get counter %s failed", p.CollectionName)
		}
	} else {
		seq = int64(setInt)
	}

	return strconv.FormatInt(seq, 10), nil
}

func (p *Mongo) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("mongo:%s", name))
	} else {
		return nil, ctx
	}
}

// Find
func (p *Mongo) Find(ctx context.Context, condition interface{}, limit int, skip int, data interface{}, sortFields ...string) error {

	span, ctx := p.ZipkinNewSpan(ctx, "find")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	s := session.DB(dbName).C(p.CollectionName).Find(condition)

	if len(sortFields) == 0 {
		sortFields = append(sortFields, "-_id")
	}
	s = s.Sort(sortFields...)

	if skip > 0 {
		s = s.Skip(skip)
	}

	if limit > 0 {
		s = s.Limit(limit)
	}

	errSelect := s.All(data)

	if errSelect != nil {
		errSelect = p.processError(span, errSelect, pconst.ERROR_MONGO_ALL, "mongo %s find failed:%v", p.CollectionName, errSelect.Error())

		if span != nil {
			span.SetTag("mongo_err", errSelect)
		}
	}

	return errSelect
}

// FindById
func (p *Mongo) FindById(ctx context.Context, id int64, data interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "findbyid")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	errFind := session.DB(dbName).C(p.CollectionName).Find(bson.M{"_id": id}).One(data)

	if errFind != nil {
		e := p.processError(span, errFind, pconst.ERROR_MONGO_FIND, "mongo %s get id failed:%v", p.CollectionName, errFind.Error())

		return e
	}

	return err
}

// Insert
func (p *Mongo) Insert(ctx context.Context, data IModelMongo) error {

	span, ctx := p.ZipkinNewSpan(ctx, "insert")

	if span != nil {
		defer span.Finish()
	}

	if p.AutoIncrementId {

		bson.NewObjectId()
		id, err := p.GetNextSequence(ctx)

		if err != nil {
			return err
		}
		data.SetId(id)
	}

	// 是否初始化时间
	created_at := data.GetCreatedTime()
	if created_at.Equal(time.Time{}) {
		data.InitTime(time.Now())
	}

	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	errInsert := coll.Insert(data)

	if errInsert != nil {

		errInsert = p.processError(span, errInsert, pconst.ERROR_MONGO_INSERT, "mongo %s insert failed:%v", p.CollectionName, errInsert.Error())

		return errInsert
	}
	return nil
}

// InsertM
func (p *Mongo) InsertM(ctx context.Context, data []IModelMongo) error {

	span, ctx := p.ZipkinNewSpan(ctx, "insertm")

	if span != nil {
		defer span.Finish()
	}
	for _, item := range data {
		if p.AutoIncrementId {

			id, err := p.GetNextSequence(ctx)

			if err != nil {
				return err
			}
			item.SetId(id)
		}

		// 是否初始化时间
		created_at := item.GetCreatedTime()
		if created_at.Equal(time.Time{}) {
			item.InitTime(time.Now())
		}
	}

	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	var idata []interface{}

	for i := 0; i < len(data); i++ {
		idata = append(idata, data[i])
	}
	errInsert := coll.Insert(idata...)

	if errInsert != nil {

		errInsert = p.processError(span, errInsert, pconst.ERROR_MONGO_INSERT, "mongo %s insertM failed:%v", p.CollectionName, errInsert.Error())

		return errInsert
	}
	return nil
}

// Count
func (p *Mongo) Count(ctx context.Context, condition interface{}) (int, error) {

	span, ctx := p.ZipkinNewSpan(ctx, "count")

	if span != nil {
		defer span.Finish()
	}

	session, dbName, err := p.GetSession(span)

	if err != nil {
		return 0, err
	}

	defer session.Close()

	count, errCount := session.DB(dbName).C(p.CollectionName).Find(condition).Count()

	if errCount != nil {

		errCount = p.processError(span, errCount, pconst.ERROR_MONGO_COUNT, "mongo %s count failed:%v", p.CollectionName, errCount.Error())

	}
	return count, errCount
}

// Distinct
func (p *Mongo) Distinct(ctx context.Context, condition interface{}, field string, data interface{}) error {

	span, ctx := p.ZipkinNewSpan(ctx, "distinct")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	errDistinct := session.DB(dbName).C(p.CollectionName).Find(condition).Distinct(field, data)

	if errDistinct != nil {

		errDistinct = p.processError(span, errDistinct, pconst.ERROR_MONGO_DISTINCT, "mongo %s distinct failed:%s", p.CollectionName, errDistinct.Error())

	}

	return errDistinct
}

// DistinctWithPage
func (p *Mongo) DistinctWithPage(ctx context.Context, condition interface{}, field string, limit int, skip int, data interface{}, sortFields map[string]bool) error {
	span, ctx := p.ZipkinNewSpan(ctx, "distinctwithpage")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()
	/*
		s := session.DB(dbName).C(p.CollectionName).Find(condition)

		if len(sortFields) > 0 {
			s = s.Sort(sortFields...)
		}

		if skip > 0 {
			s = s.Skip(skip)
		}

		if limit > 0 {
			s = s.Limit(limit)
		}

		errDistinct := s.Distinct(field, data)
	*/

	var pipeSlice []bson.M

	pipeSlice = append(pipeSlice, bson.M{"$match": condition})

	if sortFields != nil && len(sortFields) > 0 {
		bmSort := bson.M{}

		for k, v := range sortFields {
			var vInt int
			if v {
				vInt = 1
			} else {
				vInt = -1
			}
			bmSort[k] = vInt
		}

		pipeSlice = append(pipeSlice, bson.M{"$sort": bmSort})
	}

	if skip > 0 {
		pipeSlice = append(pipeSlice, bson.M{"$skip": skip})
	}

	if limit > 0 {
		pipeSlice = append(pipeSlice, bson.M{"$limit": limit})
	}

	pipeSlice = append(pipeSlice, bson.M{"$group": bson.M{"_id": fmt.Sprintf("$%s", field)}})

	pipeSlice = append(pipeSlice, bson.M{"$project": bson.M{field: "$_id"}})

	coll := session.DB(dbName).C(p.CollectionName)

	pipe := coll.Pipe(pipeSlice)

	errPipe := pipe.All(data)

	if errPipe != nil {
		errPipe = p.processError(span, errPipe, pconst.ERROR_MONGO_PIPE_ALL, "mongo %s distinct page failed: %s", p.CollectionName, errPipe.Error())
	}

	return nil
}

func (p *Mongo) Sum(ctx context.Context, condition interface{}, sumField string) (int, error) {
	span, ctx := p.ZipkinNewSpan(ctx, "sum")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return 0, err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	sumValue := bson.M{"$sum": sumField}

	pipe := coll.Pipe([]bson.M{{"$match": condition}, {"$group": bson.M{"_id": 1, "sum": sumValue}}})

	type SumStruct struct {
		_id int
		Sum int
	}

	var result SumStruct

	errPipe := pipe.One(&result)

	if errPipe != nil {
		errPipe = p.processError(span, errPipe, pconst.ERROR_MONGO_PIPE_ALL, "mongo %s sum failed: %s", p.CollectionName, errPipe.Error())

		return 0, errPipe
	}

	return result.Sum, nil
}

func (p *Mongo) DistinctCount(ctx context.Context, condition interface{}, field string) (int, error) {
	span, ctx := p.ZipkinNewSpan(ctx, "distinctCount")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return 0, err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	pipe := coll.Pipe([]bson.M{{"$match": condition}, {"$group": bson.M{"_id": fmt.Sprintf("$%s", field)}},
		{"$group": bson.M{"_id": "_id", "count": bson.M{"$sum": 1}}}})

	type CountStruct struct {
		_id   int
		Count int
	}

	var result CountStruct

	errPipe := pipe.One(&result)

	if errPipe != nil {
		errPipe = p.processError(span, errPipe, pconst.ERROR_MONGO_PIPE_ONE, "mongo %s distinct count failed: %s", p.CollectionName, errPipe.Error())

		return 0, errPipe
	}

	return result.Count, nil
}

func (p *Mongo) Update(ctx context.Context, condition interface{}, data map[string]interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "update")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	setBson := bson.M{}
	for key, value := range data {
		setBson[fmt.Sprintf("%s", key)] = value
	}

	updateData := bson.M{"$set": setBson, "$currentDate": bson.M{"updated_at": true}}

	errUpdate := coll.Update(condition, updateData)

	if errUpdate != nil {
		errUpdate = p.processError(span, errUpdate, pconst.ERROR_MONGO_UPDATE, "mongo %s update failed: %s", p.CollectionName, errUpdate.Error())
	}

	return errUpdate
}

func (p *Mongo) Upsert(ctx context.Context, condition interface{}, data map[string]interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "upsert")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	setBson := bson.M{}
	for key, value := range data {
		setBson[fmt.Sprintf("%s", key)] = value
	}

	updateData := bson.M{"$inc": setBson, "$currentDate": bson.M{"updated_at": true}}

	_, errUpsert := coll.Upsert(condition, updateData)

	if errUpsert != nil {
		errUpsert = p.processError(span, errUpsert, pconst.ERROR_MONGO_UPSERT, "mongo %s errUpsert failed: %s", p.CollectionName, errUpsert.Error())
	}

	return errUpsert
}

func (p *Mongo) RemoveId(ctx context.Context, id interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "removeid")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	errRemove := coll.RemoveId(id)

	if errRemove != nil {
		errRemove = p.processError(span, errRemove, pconst.ERROR_MONGO_REMOVEID, "mongo %s removeId failed: %s, id:%v", p.CollectionName, errRemove.Error(), id)
	}

	return errRemove
}

func (p *Mongo) RemoveAll(ctx context.Context, selector interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "removeAll")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	_, errRemove := coll.RemoveAll(selector)

	if errRemove != nil {
		errRemove = p.processError(span, errRemove, pconst.ERROR_MONGO_REMOVEALL, "mongo %s removeAll failed: %s, selector:%v", p.CollectionName, errRemove.Error(), selector)
	}

	return errRemove
}

func (p *Mongo) UpdateAllSupported(ctx context.Context, condition map[string]interface{}, update map[string]interface{}) error {
	span, ctx := p.ZipkinNewSpan(ctx, "updateAllSupported")

	if span != nil {
		defer span.Finish()
	}
	session, dbName, err := p.GetSession(span)

	if err != nil {
		return err
	}

	defer session.Close()

	coll := session.DB(dbName).C(p.CollectionName)

	update["$currentDate"] = bson.M{"updated_at": true}

	errUpdate := coll.Update(condition, update)

	if errUpdate != nil {
		errUpdate = p.processError(span, errUpdate, pconst.ERROR_MONGO_UPDATE, "mongo %s update failed: %s", p.CollectionName, errUpdate.Error())
	}

	return errUpdate
}

func (p *Mongo) processError(span opentracing.Span, err error, code int, formatter string, a ...interface{}) error {
	if err.Error() == "not found" {
		return nil
	}

	terr := terror.NewFromError(err)
	terr.Code = code

	log.Errorf("collection :%s, %s", p.CollectionName, fmt.Sprintf(formatter, a...))

	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag("err", terr)
	}

	return err
}
