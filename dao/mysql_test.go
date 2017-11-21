package dao

import (
	"context"
	"testing"
)

type testModelMysql struct {
	Mysql
}

type m1 struct {
	ModelMysql `sql:",inline"`
	Name       string
	Value      int64
}

func testInit() *testModelMysql {

	dao := &testModelMysql{Mysql{DbName: "tgo1", TableName: "test"}}
	return dao
}
func TestMysql_GetWriteOrm(t *testing.T) {
	m := testInit()

	db, err := m.GetWriteOrm(context.TODO())

	if err != nil {
		t.Error(err)
	} else if db == nil {
		t.Error("db is null")
	}
}

func TestMysql_GetReadOrm(t *testing.T) {
	m := testInit()

	db, err := m.GetReadOrm(context.TODO())

	if err != nil {
		t.Error(err)
	} else if db == nil {
		t.Error("db is null")
	}
}

func TestMysql_Insert(t *testing.T) {
	m := testInit()

	s := m1{
		Name:  "test1",
		Value: 10,
	}
	err := m.Insert(context.Background(), nil, &s)

	if err != nil {
		t.Error(err)
	}
}

func TestMysql_Select(t *testing.T) {
	m := testInit()

	condition := "name = 'test1'"

	var s []m1

	err := m.Select(context.TODO(), nil, condition, &s, 0, 0, []string{}, "")

	if err != nil {
		t.Error(err)
	} else if len(s) == 0 {
		t.Error("data is empty")
	}
}

func BenchmarkMysql_Select(b *testing.B) {

	b.RunParallel(func(pb *testing.PB) {

		for pb.Next() {
			m := testInit()

			condition := "name = 'test1'"

			var s []m1

			err := m.Select(context.TODO(), nil, condition, &s, 0, 0, []string{}, "")

			if err != nil {
				b.Error(err)
			} else if len(s) == 0 {
				b.Error("data is empty")
			}
		}
	})
}

func TestMysql_Count(t *testing.T) {
	m := testInit()

	condition := "name = 'test1'"

	count, err := m.Count(context.Background(), nil, condition)

	if err != nil {
		t.Error(err)
	} else if count == 0 {
		t.Error("data is empty")
	}
}

func TestMysql_First(t *testing.T) {
	m := testInit()

	condition := "name = 'test1'"

	var s m1

	err := m.First(context.Background(), nil, condition, &s, "name desc")

	if err != nil {
		t.Error(err)
	} else if s.Value == 0 {
		t.Error("data is empty")
	}
}

func TestMysql_Trans(t *testing.T) {
	m := testInit()

	ctx := context.TODO()

	o, err := m.GetWriteOrm(ctx)

	if err != nil {
		t.Error(err)
	}

	db := o.Begin()

	s := m1{
		Name:  "testtrans",
		Value: 10,
	}

	err = m.Insert(ctx, db, &s)

	if err != nil {
		t.Error(err)
	}

	condition := "name = 'testtrans'"

	set := make(map[string]interface{})

	set["value"] = 25

	err = m.Update(ctx, db, condition, set)

	if err != nil {
		t.Error(err)
	}

	db.Rollback()
}

func TestMysql_Update(t *testing.T) {
	m := testInit()

	condition := "name = 'test1'"

	set := make(map[string]interface{})

	set["value"] = 20

	err := m.Update(context.Background(), nil, condition, set)

	if err != nil {
		t.Error(err)
	}
}

/*
func TestMysql_Delete(t *testing.T) {
	m:= testInit()

	condition := "name = 'test1'"

	err := m.Delete(context.Background(),nil,condition)

	if err!=nil{
		t.Error(err)
	}
}*/
