package mesosUtils

import (
	"github.com/golang/protobuf/proto"
	mesos "github.com/mesos/mesos-go/api/v0/mesosproto"
)

var (
	DefaultFilter = &mesos.Filters{RefuseSeconds: proto.Float64(1)}
	LongFilter    = &mesos.Filters{RefuseSeconds: proto.Float64(1000)}
)
