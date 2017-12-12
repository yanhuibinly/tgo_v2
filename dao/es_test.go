package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic"
	"testing"
)

type loan struct {
	LoanSn int `json:"loan_sn"`
}

func TestEs_Invoke(t *testing.T) {
	es := &Es{Service: "tgo", Index: "xingxiangrong_loan", Type: "1"}

	ctx := context.Background()

	client, err := es.GetConn(ctx)

	if err != nil {
		t.Error(err)
		return
	}
	id := "214825152366055276"
	var res *elastic.GetResult
	//参考：https://olivere.github.io/elastic/
	fi := func(ctx2 context.Context) (err error) {
		res, err = client.Get().Index(es.Index).Id(id).Do(ctx)
		return
	}
	err = es.Invoke(ctx, client, "search", fi)

	if err != nil {
		t.Error(err)
		return
	}

	if res.Found {
		data := loan{}
		err = json.Unmarshal(*res.Source, &data)
		if err != nil {
			t.Error(err)
		} else {
			fmt.Println(data)
		}
	}

}
