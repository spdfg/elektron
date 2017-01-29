package mesosUtils

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	"github.com/golang/protobuf/proto"
)

var (
	DefaultFilter = &mesos.Filters{RefuseSeconds: proto.Float64(1)}
	LongFilter    = &mesos.Filters{RefuseSeconds: proto.Float64(1000)}
)
