// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package analytics

// AnalyticsFilters defines the analytics options.
type AnalyticsFilters struct {
	Usernames        []string `json:"usernames"`
	SkippedUsernames []string `json:"skip_usernames"`
}

// ShouldFilter determine whether a record should to be filtered out.
// 定义消息是否被过滤的条件
func (filters AnalyticsFilters) ShouldFilter(record AnalyticsRecord) bool {
	switch {
	case len(filters.SkippedUsernames) > 0 && stringInSlice(record.Username, filters.SkippedUsernames):
		return true
	case len(filters.Usernames) > 0 && !stringInSlice(record.Username, filters.Usernames):
		return true
	}

	return false
}

// HasFilter determine whether a record has a filter.
// 判断一条消息是否有过滤器
func (filters AnalyticsFilters) HasFilter() bool {
	if len(filters.SkippedUsernames) == 0 && len(filters.Usernames) == 0 {
		return false
	}

	return true
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}
