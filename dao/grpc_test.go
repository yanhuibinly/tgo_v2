package dao

import (
	"testing"
	"google.golang.org/grpc"
	"context"
	"fmt"
)

func TestDaoGRPC_GetConn(t *testing.T) {
	daoGrpc := &Grpc{}
	daoGrpc.DialOptions = append(daoGrpc.DialOptions, grpc.WithInsecure())
	daoGrpc.Service = "tgo"

	ctx := context.Background()
	conn, err := daoGrpc.GetConn(ctx)

	if err != nil {
		t.Errorf("get failed:%s", err.Error())
	}else if conn ==nil{
		t.Error("conn is null")
	} else {
		defer daoGrpc.CloseConn(ctx,conn)
	}


	daoGrpc2 := &Grpc{}
	daoGrpc2.DialOptions = append(daoGrpc.DialOptions, grpc.WithInsecure())
	daoGrpc2.Service = "tgo2"

	conn2, err2 := daoGrpc2.GetConn(ctx)

	if err != nil {
		t.Errorf("get failed:%s", err2.Error())
	} else if conn2==nil {
		t.Error("conn2 is null")
	} else{
		defer daoGrpc.CloseConn(ctx,conn2)
	}

}

func BenchmarkDaoGRPC_GetConn(b *testing.B) {
	ctx := context.Background()
	daoGrpc := &Grpc{}
	daoGrpc.DialOptions = append(daoGrpc.DialOptions, grpc.WithInsecure())
	daoGrpc.Service = "tgo"

	for i := 0; i < b.N; i++ {
		conn, err := daoGrpc.GetConn(ctx)

		if err != nil {
			b.Errorf("get failed:%s", err.Error())
		} else {
			fmt.Printf("conn:%v\n", conn)
			daoGrpc.CloseConn(ctx,conn)
		}
	}

}
