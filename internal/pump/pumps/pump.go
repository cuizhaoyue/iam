// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package pumps

import (
	"context"
	"errors"

	"github.com/marmotedu/iam/internal/pump/analytics"
)

// Pump defines the interface for all analytics back-end.
type Pump interface {
	GetName() string                                //
	New() Pump                                      // 创建一个pump
	Init(interface{}) error                         // 初始化一个pump，例如，可以在init中创建下游系统的网络连接。
	WriteData(context.Context, []interface{}) error // 往下游系统写入数据。为了提高性能，最好支持批量写入
	SetFilters(analytics.AnalyticsFilters)          // 设置是否过滤某条数据，这也是一个非常常见的需求，因为不是所有的数据都是需要的
	GetFilters() analytics.AnalyticsFilters         //
	SetTimeout(timeout int)                         // 设置超时时间，通过超时处理，可以保证整个采集框架正常运行.
	GetTimeout() int
	SetOmitDetailedRecording(bool)  // 过滤掉详细数据，防止上传数据过于巨大导致占用大量磁盘
	GetOmitDetailedRecording() bool //
}

// GetPumpByName returns the pump instance by given name.
func GetPumpByName(name string) (Pump, error) {
	if pump, ok := availablePumps[name]; ok && pump != nil {
		return pump, nil
	}

	return nil, errors.New(name + " Not found")
}
