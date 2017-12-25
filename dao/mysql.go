package dao

import (
	"context"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tonyjt/tgo_v2/config"
	"github.com/tonyjt/tgo_v2/log"
	"github.com/tonyjt/tgo_v2/pconst"
	"github.com/tonyjt/tgo_v2/terror"
	"math/rand"
	"time"
)

var (
	dbMysqlWrite map[string]*gorm.DB
	dbMysqlReads map[string][]*gorm.DB
)

type IModelMysql interface {
	GetCreatedTime() time.Time
	InitTime(t time.Time)
	SetUpdatedTime(t time.Time)
}

type ModelMysql struct {
	Id         int       `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	TimeCustom bool      `sql:"-"`
}

func (m *ModelMysql) GetCreatedTime() time.Time {
	return m.CreatedAt
}

func (m *ModelMysql) InitTime(t time.Time) {
	if !m.TimeCustom {
		m.CreatedAt = t
		m.UpdatedAt = t
	}
}
func (m *ModelMysql) SetUpdatedTime(t time.Time) {
	m.UpdatedAt = t
}

func init() {
	if config.FeatureMysql() {

		dbMysqlWrite = make(map[string]*gorm.DB)
		dbMysqlReads = make(map[string][]*gorm.DB)

		for _, conf := range config.MysqlGetAll() {

			var err error
			var dbWrite *gorm.DB
			dbWrite, err = initDb(conf.Conn.DbName, conf.Conn.Write, conf.Conn.Pool)

			if err != nil {
				panic("connect to mysql write server failed")
			}

			dbMysqlWrite[conf.Db] = dbWrite

			for _, c := range conf.Conn.Reads {
				d, err := initDb(conf.Conn.DbName, c, conf.Conn.Pool)

				if err == nil {
					dbMysqlReads[conf.Db] = append(dbMysqlReads[conf.Db], d)
				} else {
					log.Errorf("mysql read init failed:%+v", err)
				}
			}

			if len(dbMysqlReads) == 0 {
				dbMysqlReads[conf.Db] = append(dbMysqlReads[conf.Db], dbWrite)
			}
		}
	}
}

func initDb(dbName string, configMysql config.MysqlBase, configPool config.MysqlPool) (*gorm.DB, error) {
	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local", configMysql.User,
		configMysql.Password, configMysql.Address, configMysql.Port, dbName)

	resultDb, err := gorm.Open("mysql", addr)

	if err != nil {
		log.Errorf("connect mysql error: %s", err.Error())
		return resultDb, err
	}
	resultDb.DB().SetMaxOpenConns(configPool.Max)
	resultDb.DB().SetMaxIdleConns(configPool.IdleMax)
	//resultDb.DB().SetConnMaxLifetime(time.Duration(configPool.LifeTimeSeconds) * time.Second)

	resultDb.DB().Ping()

	if config.AppEnvIsDev() {
		resultDb.LogMode(true)
	}

	return resultDb, nil
}

type Mysql struct {
	DbName    string
	TableName string
}

// NewMysql NewMysql
func NewMysql(tableName string) *Mysql {
	return &Mysql{TableName: tableName}
}

func (p *Mysql) getDbName() string {
	if p.DbName != "" {
		return p.DbName
	}
	for _, conf := range config.MysqlGetAll() {
		if conf.Db != "" {
			return conf.Db
		}
	}

	return ""
}

// GetWriteOrm
func (p *Mysql) GetWriteOrm(ctx context.Context) (*gorm.DB, error) {

	dbName := p.getDbName()

	span, ctx := p.ZipkinNewSpan(ctx, dbName+":getWriteOrm")

	if span != nil {
		defer span.Finish()
	}
	if dbMysqlWrite == nil {
		err := terror.New(pconst.ERROR_MYSQL_WRITE_EMPTY)
		span.SetTag("err:getorm", err)
		return nil, err
	}
	return dbMysqlWrite[dbName], nil
}

// GetReadOrm
func (p *Mysql) GetReadOrm(ctx context.Context) (*gorm.DB, error) {

	span, ctx := p.ZipkinNewSpan(ctx, "getReadOrm")

	if span != nil {
		defer span.Finish()
	}
	dbName := p.getDbName()
	conf := dbMysqlReads[dbName]
	if len(conf) == 0 {
		err := terror.New(pconst.ERROR_MYSQL_READ_EMPTY)
		span.SetTag("err:getorm", err)
		return nil, err
	}

	var index int
	if len(conf) > 1 {
		rand.Seed(time.Now().UnixNano())

		index = rand.Intn(len(conf) - 1)

	} else {
		index = 0
	}

	return conf[index], nil
}

func (p *Mysql) ZipkinNewSpan(ctx context.Context, name string) (opentracing.Span, context.Context) {
	if config.FeatureZipkin() {
		return opentracing.StartSpanFromContext(ctx, fmt.Sprintf("mysql:%s", name))
	} else {
		return nil, ctx
	}
}

// Insert
func (p *Mysql) Insert(ctx context.Context, db *gorm.DB, model IModelMysql) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "insert")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetWriteOrm(ctx)

		if err != nil {
			return err
		}

	}
	model.InitTime(time.Now())

	errInsert := db.Table(p.TableName).Create(model).Error

	if errInsert != nil {
		err = p.processError(span, errInsert, pconst.ERROR_MYSQL_INSERT, "insert data error")
	}

	return err
}

// Select
func (p *Mysql) Select(ctx context.Context, db *gorm.DB, condition string, data interface{}, skip int, limit int, fields []string, sort string) (err error) {
	span, ctx := p.ZipkinNewSpan(ctx, "select")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetReadOrm(ctx)

		if err != nil {
			return err
		}

		//defer db.Close()
	}

	db = db.Table(p.TableName).Where(condition)

	var errFind error
	if len(fields) > 0 {
		db = db.Select(fields)
	}
	if skip > 0 {
		db = db.Offset(skip)
	}
	if limit > 0 {
		db = db.Limit(limit)
	}
	if sort != "" {
		db = db.Order(sort)
	}

	errFind = db.Find(data).Error

	if errFind != nil {
		if errFind.Error() == "record not found" {
			err = p.processError(span, errFind, pconst.ERROR_MYSQL_NOT_FOUND, "select data is empty")
		} else {
			err = p.processError(span, errFind, pconst.ERROR_MYSQL_SELECT, "select data error")
		}
	}
	return err
}

// Update
func (p *Mysql) Update(ctx context.Context, db *gorm.DB, condition string, sets map[string]interface{}) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "update")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetWriteOrm(ctx)

		if err != nil {
			return err
		}

		//defer db.Close()
	}

	errUpdate := db.Table(p.TableName).Where(condition).Updates(sets).Error
	if errUpdate != nil {
		err = p.processError(span, errUpdate, pconst.ERROR_MYSQL_UPDATE, "update data error")
	}
	return err
}

// Delete
func (p *Mysql) Delete(ctx context.Context, db *gorm.DB, condition string) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "delete")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetWriteOrm(ctx)

		if err != nil {
			return err
		}

		//defer db.Close()
	}

	errDel := db.Table(p.TableName).Where(condition).Delete(nil).Error
	if errDel != nil {
		err = p.processError(span, errDel, pconst.ERROR_MYSQL_DELETE, "delete data error")

	}
	return err
}

// First
func (p *Mysql) First(ctx context.Context, db *gorm.DB, condition string, data IModelMysql, sort string) (err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "first")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetReadOrm(ctx)

		if err != nil {
			return err
		}

		//defer db.Close()
	}

	db = db.Table(p.TableName).Where(condition)

	var errFind error

	if sort != "" {
		db = db.Order(sort)
	}

	errFirst := db.First(data).Error

	if errFirst != nil {
		err = p.processError(span, errFind, pconst.ERROR_MYSQL_FIRST, "first data error")
	}
	return err
}

// Count
func (p *Mysql) Count(ctx context.Context, db *gorm.DB, condition string) (count int, err error) {

	span, ctx := p.ZipkinNewSpan(ctx, "count")
	if span != nil {
		defer span.Finish()
	}

	if db == nil {
		db, err = p.GetReadOrm(ctx)

		if err != nil {
			return
		}

		//defer db.Close()
	}

	errCount := db.Table(p.TableName).Where(condition).Count(&count).Error

	if errCount != nil {
		err = p.processError(span, errCount, pconst.ERROR_MYSQL_COUNT, "count data error")
	}
	return
}

func (p *Mysql) processError(span opentracing.Span, err error, code int, formatter string, a ...interface{}) error {

	if err == nil {
		return err
	}

	log.Errorf("table :%s, %s", p.TableName, fmt.Sprintf(formatter, a...))

	if span != nil {
		ext.Error.Set(span, true)
		span.SetTag("err", err)
	}

	terr := terror.New(code)

	return terr
}
