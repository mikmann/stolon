// Copyright 2017 Sorint.lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/sorintlab/stolon/internal/cluster"
	"github.com/sorintlab/stolon/internal/common"
)

var curUID int

func TestParseSynchronousStandbyNames(t *testing.T) {
	tests := []struct {
		in  string
		out []string
		err error
	}{
		{
			in:  "2 (stolon_2c3870f3,stolon_c874a3cb)",
			out: []string{"stolon_2c3870f3", "stolon_c874a3cb"},
		},
		{
			in:  "2 ( stolon_2c3870f3 , stolon_c874a3cb )",
			out: []string{"stolon_2c3870f3", "stolon_c874a3cb"},
		},
		{
			in:  "21 (\" stolon_2c3870f3\",stolon_c874a3cb)",
			out: []string{"\" stolon_2c3870f3\"", "stolon_c874a3cb"},
		},
		{
			in:  "stolon_2c3870f3,stolon_c874a3cb",
			out: []string{"stolon_2c3870f3", "stolon_c874a3cb"},
		},
		{
			in:  "node1",
			out: []string{"node1"},
		},
		{
			in:  "2 (node1,",
			out: []string{"node1"},
			err: errors.New("synchronous standby string has number but lacks brackets"),
		},
	}

	for i, tt := range tests {
		out, err := parseSynchronousStandbyNames(tt.in)

		if tt.err != nil {
			if err == nil {
				t.Errorf("%d: got no error, wanted error: %v", i, tt.err)
			} else if tt.err.Error() != err.Error() {
				t.Errorf("%d: got error: %v, wanted error: %v", i, err, tt.err)
			}
		} else {
			if err != nil {
				t.Errorf("%d: unexpected error: %v", i, err)
			} else if !reflect.DeepEqual(out, tt.out) {
				t.Errorf("%d: wrong output: got:\n%s\nwant:\n%s", i, out, tt.out)
			}
		}
	}
}

func TestGenerateHBA(t *testing.T) {
	// minimal clusterdata with only the fields used by generateHBA
	cd := &cluster.ClusterData{
		Cluster: &cluster.Cluster{
			Spec:   &cluster.ClusterSpec{},
			Status: cluster.ClusterStatus{},
		},
		Keepers: cluster.Keepers{},
		DBs: cluster.DBs{
			"db1": &cluster.DB{
				UID: "db1",
				Spec: &cluster.DBSpec{
					Role: common.RoleMaster,
				},
				Status: cluster.DBStatus{
					ListenAddress: "192.168.0.1",
				},
			},
			"db2": &cluster.DB{
				UID: "db2",
				Spec: &cluster.DBSpec{
					Role: common.RoleStandby,
					FollowConfig: &cluster.FollowConfig{
						Type:  cluster.FollowTypeInternal,
						DBUID: "db1",
					},
				},
				Status: cluster.DBStatus{
					ListenAddress: "192.168.0.2",
				},
			},
			"db3": &cluster.DB{
				UID: "db3",
				Spec: &cluster.DBSpec{
					Role: common.RoleStandby,
					FollowConfig: &cluster.FollowConfig{
						Type:  cluster.FollowTypeInternal,
						DBUID: "db1",
					},
				},
				Status: cluster.DBStatus{
					ListenAddress: "192.168.0.3",
				},
			},
		},
		Proxy: &cluster.Proxy{},
	}

	tests := []struct {
		DefaultSUReplAccessMode cluster.SUReplAccessMode
		dbUID                   string
		pgHBA                   []string
		out                     []string
	}{
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessAll,
			dbUID:                   "db1",
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all superuser 0.0.0.0/0 md5",
				"host all superuser ::0/0 md5",
				"host replication repluser 0.0.0.0/0 md5",
				"host replication repluser ::0/0 md5",
				"host all all 0.0.0.0/0 md5",
				"host all all ::0/0 md5",
			},
		},
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessAll,
			dbUID:                   "db2",
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all superuser 0.0.0.0/0 md5",
				"host all superuser ::0/0 md5",
				"host replication repluser 0.0.0.0/0 md5",
				"host replication repluser ::0/0 md5",
				"host all all 0.0.0.0/0 md5",
				"host all all ::0/0 md5",
			},
		},
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessAll,
			dbUID:                   "db1",
			pgHBA: []string{
				"host all all 192.168.0.0/24 md5",
			},
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all superuser 0.0.0.0/0 md5",
				"host all superuser ::0/0 md5",
				"host replication repluser 0.0.0.0/0 md5",
				"host replication repluser ::0/0 md5",
				"host all all 192.168.0.0/24 md5",
			},
		},
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessAll,
			dbUID:                   "db2",
			pgHBA: []string{
				"host all all 192.168.0.0/24 md5",
			},
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all superuser 0.0.0.0/0 md5",
				"host all superuser ::0/0 md5",
				"host replication repluser 0.0.0.0/0 md5",
				"host replication repluser ::0/0 md5",
				"host all all 192.168.0.0/24 md5",
			},
		},
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessStrict,
			dbUID:                   "db1",
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all superuser 192.168.0.2/32 md5",
				"host replication repluser 192.168.0.2/32 md5",
				"host all superuser 192.168.0.3/32 md5",
				"host replication repluser 192.168.0.3/32 md5",
				"host all all 0.0.0.0/0 md5",
				"host all all ::0/0 md5",
			},
		},
		{
			DefaultSUReplAccessMode: cluster.SUReplAccessStrict,
			dbUID:                   "db2",
			out: []string{
				"local postgres superuser md5",
				"local replication repluser md5",
				"host all all 0.0.0.0/0 md5",
				"host all all ::0/0 md5",
			},
		},
	}

	for i, tt := range tests {
		p := &PostgresKeeper{
			pgSUAuthMethod:   "md5",
			pgSUUsername:     "superuser",
			pgReplAuthMethod: "md5",
			pgReplUsername:   "repluser",
		}

		cd.Cluster.Spec.DefaultSUReplAccessMode = &tt.DefaultSUReplAccessMode

		db := cd.DBs[tt.dbUID]
		db.Spec.PGHBA = tt.pgHBA

		out := p.generateHBA(cd, db)

		if !reflect.DeepEqual(out, tt.out) {
			var b bytes.Buffer
			b.WriteString(fmt.Sprintf("#%d: wrong output: got:\n", i))
			for _, o := range out {
				b.WriteString(fmt.Sprintf("%s\n", o))
			}
			b.WriteString(fmt.Sprintf("\nwant:\n"))
			for _, o := range tt.out {
				b.WriteString(fmt.Sprintf("%s\n", o))
			}
			t.Errorf(b.String())
		}
	}
}
