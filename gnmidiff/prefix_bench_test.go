// Copyright 2023 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnmidiff

import (
	"fmt"
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

func BenchmarkDiffSetRequest(b *testing.B) {
	reqA := &gpb.SetRequest{}
	reqB := &gpb.SetRequest{}

	for i := 0; i < 100; i++ {
		ethName := fmt.Sprintf("eth%d", i)
		pathStr := fmt.Sprintf("/interfaces/interface[name=%s]/config/description", ethName)
		reqA.Update = append(reqA.Update, &gpb.Update{
			Path: ygot.MustStringToPath(pathStr),
			Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: fmt.Sprintf("desc-%d", i)}},
		})
		reqB.Update = append(reqB.Update, &gpb.Update{
			Path: ygot.MustStringToPath(pathStr),
			Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: fmt.Sprintf("desc-%d", i)}},
		})
	}
	for i := 0; i < 50; i++ {
		ethName := fmt.Sprintf("eth%d", i)
		pathStr := fmt.Sprintf("/interfaces/interface[name=%s]", ethName)
		reqA.Delete = append(reqA.Delete, ygot.MustStringToPath(pathStr))
		reqB.Delete = append(reqB.Delete, ygot.MustStringToPath(pathStr))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := DiffSetRequest(reqA, reqB, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiffSetRequestToNotifications(b *testing.B) {
	req := &gpb.SetRequest{}
	for i := 0; i < 50; i++ {
		ethName := fmt.Sprintf("eth%d", i)
		pathStr := fmt.Sprintf("/interfaces/interface[name=%s]", ethName)
		req.Delete = append(req.Delete, ygot.MustStringToPath(pathStr))
	}

	notifs := []*gpb.Notification{
		{
			Update: []*gpb.Update{},
		},
	}
	for i := 0; i < 100; i++ {
		ethName := fmt.Sprintf("eth%d", i)
		for j := 0; j < 5; j++ {
			pathStr := fmt.Sprintf("/interfaces/interface[name=%s]/subinterfaces/subinterface[index=%d]/config/description", ethName, j)
			notifs[0].Update = append(notifs[0].Update, &gpb.Update{
				Path: ygot.MustStringToPath(pathStr),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: fmt.Sprintf("sub-desc-%d", j)}},
			})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := DiffSetRequestToNotifications(req, notifs, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
