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

package gnmiparse

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/prototext"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// NotifsFromFile parses a sequence of Notifications from a textproto file.
//
// The notifications can either be output from a GetResponse or a stream of
// SubscribeResponses. Any non-Notification messages are discarded.
func NotifsFromFile(file string) ([]*gpb.Notification, error) {
	bs, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	protos, err := splitByEmptyline(bs)
	if err != nil {
		return nil, err
	}

	var notifs []*gpb.Notification
	for _, proto := range protos {
		resp := &gpb.SubscribeResponse{}
		if err := prototext.Unmarshal([]byte(proto), resp); err != nil {
			resp := &gpb.GetResponse{}
			// Check if it's a GetResponse
			if err := prototext.Unmarshal([]byte(proto), resp); err == nil {
				return resp.GetNotification(), err
			}
			return nil, fmt.Errorf("invalid Notification/SubscribeResponse from file %q: %v", file, err)
		}
		if notif := resp.GetUpdate(); notif != nil {
			notifs = append(notifs, notif)
		}
	}

	return notifs, nil
}

// splitByEmptyline splits the input by empty lines.
//
// If there are consecutive empty lines, then they're treated as a single empty
// line.
func splitByEmptyline(input []byte) ([][]byte, error) {
	var ret [][]byte
	scanner := bufio.NewScanner(bytes.NewReader(input))
	var nextLine []byte
	for scanner.Scan() {
		next := scanner.Bytes()
		if len(next) == 0 {
			if len(nextLine) != 0 {
				ret = append(ret, nextLine)
				nextLine = []byte{}
			}
			continue
		}
		// Add back the newline if it is not the first line.
		if len(nextLine) != 0 {
			nextLine = append(nextLine, '\n')
		}
		nextLine = append(nextLine, next...)
	}
	// Handle edge case
	if len(nextLine) != 0 {
		ret = append(ret, nextLine)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input bytes: %v", err)
	}
	return ret, nil
}
