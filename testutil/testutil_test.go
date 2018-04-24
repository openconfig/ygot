// Copyright 2017 Google Inc.
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

package testutil

import (
	"testing"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestNotificationSetEqual(t *testing.T) {
	tests := []struct {
		name string
		inA  []*gnmipb.Notification
		inB  []*gnmipb.Notification
		want bool
	}{{
		name: "equal sets, length one",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
		}},
		want: true,
	}, {
		name: "unequal sets, length one",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 84,
		}},
		want: false,
	}, {
		name: "equal sets: length two",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
		}, {
			Timestamp: 4242,
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
		}, {
			Timestamp: 4242,
		}},
		want: true,
	}, {
		name: "equal sets: length  different order",
		inA: []*gnmipb.Notification{{
			Timestamp: 4242,
		}, {
			Timestamp: 42,
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
		}, {
			Timestamp: 4242,
		}},
		want: true,
	}, {
		name: "integration example - same order",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}},
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}},
		}},
		want: true,
	}, {
		name: "integration example - different order",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}},
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
			}},
		}},
		want: true,
	}, {
		name: "equal: pathelem integration example",
		inA: []*gnmipb.Notification{{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "neighbors",
					}, {
						Name: "neighbor",
						Key:  map[string]string{"neighbor-address": "192.0.2.1"},
					}, {
						Name: "config",
					}, {
						Name: "neighbor-address",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "neighbors",
					}, {
						Name: "neighbor",
						Key:  map[string]string{"neighbor-address": "192.0.2.1"},
					}, {
						Name: "neighbor-address",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
			}},
		}},
		inB: []*gnmipb.Notification{{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "neighbors",
					}, {
						Name: "neighbor",
						Key:  map[string]string{"neighbor-address": "192.0.2.1"},
					}, {
						Name: "neighbor-address",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "neighbors",
					}, {
						Name: "neighbor",
						Key:  map[string]string{"neighbor-address": "192.0.2.1"},
					}, {
						Name: "config",
					}, {
						Name: "neighbor-address",
					}},
				},
				Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
			}},
		}},
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotificationSetEqual(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("NotificationSetEqual(%#v, %#v): did not get expected result, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}

func TestUpdateSetEqual(t *testing.T) {
	tests := []struct {
		name string
		inA  []*gnmipb.Update
		inB  []*gnmipb.Update
		want bool
	}{{
		name: "equal - length one",
		inA:  []*gnmipb.Update{{Duplicates: 42}},
		inB:  []*gnmipb.Update{{Duplicates: 42}},
		want: true,
	}, {
		name: "equal - length two",
		inA: []*gnmipb.Update{{
			Duplicates: 42,
		}, {
			Duplicates: 84,
		}},
		inB: []*gnmipb.Update{{
			Duplicates: 42,
		}, {
			Duplicates: 84,
		}},
		want: true,
	}, {
		name: "equal - length two, different order",
		inA: []*gnmipb.Update{{
			Duplicates: 84,
		}, {
			Duplicates: 42,
		}},
		inB: []*gnmipb.Update{{
			Duplicates: 42,
		}, {
			Duplicates: 84,
		}},
		want: true,
	}, {
		name: "unequal - length one",
		inA:  []*gnmipb.Update{{Duplicates: 42}},
		inB:  []*gnmipb.Update{{Duplicates: 128}},
		want: false,
	}, {
		name: "unequal - length two",
		inA: []*gnmipb.Update{{
			Duplicates: 42,
		}, {
			Duplicates: 84,
		}},
		inB: []*gnmipb.Update{{
			Duplicates: 42,
		}, {
			Duplicates: 96,
		}},
		want: false,
	}, {
		name: "equal: integration example",
		inA: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		inB: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		want: true,
	}, {
		name: "equal: integration example",
		inA: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		inB: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		want: true,
	}, {
		name: "equal: integration example - different order",
		inA: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		inB: []*gnmipb.Update{{
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}, {
			Path: &gnmipb.Path{Element: []string{"enum-list", "VAL_TWO", "config", "key"}},
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"VAL_TWO"}},
		}},
		want: true,
	}, {
		name: "equal: pathelem integration example",
		inA: []*gnmipb.Update{{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "neighbors",
				}, {
					Name: "neighbor",
					Key:  map[string]string{"neighbor-address": "192.0.2.1"},
				}, {
					Name: "config",
				}, {
					Name: "neighbor-address",
				}},
			},
			Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
		}, {
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "neighbors",
				}, {
					Name: "neighbor",
					Key:  map[string]string{"neighbor-address": "192.0.2.1"},
				}, {
					Name: "neighbor-address",
				}},
			},
			Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
		}},
		inB: []*gnmipb.Update{{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "neighbors",
				}, {
					Name: "neighbor",
					Key:  map[string]string{"neighbor-address": "192.0.2.1"},
				}, {
					Name: "neighbor-address",
				}},
			},
			Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
		}, {
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "neighbors",
				}, {
					Name: "neighbor",
					Key:  map[string]string{"neighbor-address": "192.0.2.1"},
				}, {
					Name: "config",
				}, {
					Name: "neighbor-address",
				}},
			},
			Val: &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"192.0.2.1"}},
		}},
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateSetEqual(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("UpdateSetEqual(%v, %v): did not get expected result, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}

func TestNotificationLess(t *testing.T) {
	tests := []struct {
		name string
		inA  *gnmipb.Notification
		inB  *gnmipb.Notification
		want bool
	}{{
		name: "equal",
		inA: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two",
					}},
				},
			}},
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "three",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two",
					}},
				},
			}},
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "three",
				}},
			}},
		},
		want: false,
	}, {
		name: "timestamp: a < b",
		inA: &gnmipb.Notification{
			Timestamp: 0,
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
		},
		want: true,
	}, {
		name: "timestamp: b < a",
		inA: &gnmipb.Notification{
			Timestamp: 42,
		},
		inB: &gnmipb.Notification{
			Timestamp: 0,
		},
		want: false,
	}, {
		name: "prefix: a < b",
		inA: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			},
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
		},
		want: true,
	}, {
		name: "prefix: b < a",
		inA: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "zzz",
				}},
			},
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
			Prefix: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "aaa",
				}},
			},
		},
		want: false,
	}, {
		name: "update: a < b length",
		inA: &gnmipb.Notification{
			Timestamp: 42,
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Duplicates: 0,
			}},
		},
		want: true,
	}, {
		name: "update: b < a length",
		inA: &gnmipb.Notification{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Duplicates: 0,
			}},
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
		},
		want: false,
	}, {
		name: "update: a < b multiple updates",
		inA: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "one-z",
					}},
				},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two-q",
					}},
				},
			}},
		},
		inB: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two-a",
					}},
				},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "one-z",
					}},
				},
			}},
		},
		want: true,
	}, {
		name: "update: a < b multiple updates, different order",
		inA: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "one-z",
					}},
				},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two-q",
					}},
				},
			}},
		},
		inB: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "one-z",
					}},
				},
			}, {
				Path: &gnmipb.Path{
					Elem: []*gnmipb.PathElem{{
						Name: "two-a",
					}},
				},
			}},
		},
		want: true,
	}, {
		name: "delete: a < b, length",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "two",
				}},
			}},
		},
		want: true,
	}, {
		name: "delete: b < a, length",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "two",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}},
		},
		want: false,
	}, {
		name: "delete: a < b, path",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}},
		},
		want: true,
	}, {
		name: "delete: b < a, path",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			}},
		},
		want: false,
	}, {
		name: "delete: a < b - multiple paths",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "three",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "three",
				}},
			}},
		},
		want: true,
	}, {
		name: "delete: b < a, multiple paths",
		inA: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			}},
		},
		inB: &gnmipb.Notification{
			Delete: []*gnmipb.Path{{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			}, {
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "three",
				}},
			}},
		},
		want: false,
	}, {
		name: "nil: both nil",
		want: false,
	}, {
		name: "nil: a nil, b not",
		inB:  &gnmipb.Notification{Timestamp: 42},
		want: true,
	}, {
		name: "nil: a not, b nil",
		inA:  &gnmipb.Notification{Timestamp: 42},
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotificationLess(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("NotificationLess(%#v, %#v): did not get expected result, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}

func TestUpdateLess(t *testing.T) {
	tests := []struct {
		name string
		inA  *gnmipb.Update
		inB  *gnmipb.Update
		want bool
	}{{
		name: "updates equal",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		want: false,
	}, {
		name: "path: a < b",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		want: true,
	}, {
		name: "path: b < a",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}, {
					Name: "two",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		want: false,
	}, {
		name: "typed value: a < b",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{24},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		want: true,
	}, {
		name: "typed value: b < a",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{0},
			},
			Duplicates: 42,
		},
		want: false,
	}, {
		name: "duplicates: a < b",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 84,
		},
		want: true,
	}, {
		name: "duplicates: b < a",
		inA: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 42,
		},
		inB: &gnmipb.Update{
			Path: &gnmipb.Path{
				Elem: []*gnmipb.PathElem{{
					Name: "one",
				}},
			},
			Val: &gnmipb.TypedValue{
				Value: &gnmipb.TypedValue_UintVal{42},
			},
			Duplicates: 0,
		},
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateLess(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("UpdateLess(%#v, %#v): did not get expected result, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}

func TestPathLess(t *testing.T) {
	tests := []struct {
		name string
		inA  *gnmipb.Path
		inB  *gnmipb.Path
		want bool
	}{{
		name: "nil a, non-nil b",
		inB:  &gnmipb.Path{},
		want: true,
	}, {
		name: "nil b, non-nil a",
		inA:  &gnmipb.Path{},
		want: false,
	}, {
		name: "both nil",
		want: false,
	}, {
		name: "equal - a < b",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "one",
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "one",
			}},
		},
		want: false,
	}, {
		name: "a < b due to path element name",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "b",
			}},
		},
		want: true,
	}, {
		name: "b < a due to path element name",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "b",
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}},
		},
		want: false,
	}, {
		name: "equal: a < b with path elem keys",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		want: false,
	}, {
		name: "a < b due to path elem key name",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"b": "a"},
			}},
		},
		want: true,
	}, {
		name: "b < a due to path elem key name",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"b": "a"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		want: false,
	}, {
		name: "a < b due to path elem key value",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "z"},
			}},
		},
		want: true,
	}, {
		name: "b < a due to path elem key value",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "z"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"a": "a"},
			}},
		},
		want: false,
	}, {
		name: "a < b due to more specific path",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}},
		},
		want: true,
	}, {
		name: "b < a due to more specific path",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
		},
		want: false,
	}, {
		name: "a < b due to number of keys",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"one": "1"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"one": "1", "two": "2"},
			}},
		},
		want: true,
	}, {
		name: "b < a due to number of keys",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"one": "1", "two": "2"},
			}},
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
				Key:  map[string]string{"one": "1"},
			}},
		},
		want: false,
	}, {
		name: "equal - a < b with origin",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "a",
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "a",
		},
		want: false,
	}, {
		name: "a < b due to origin",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "a",
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "z",
		},
		want: true,
	}, {
		name: "b < a due to origin",
		inA: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "z",
		},
		inB: &gnmipb.Path{
			Elem: []*gnmipb.PathElem{{
				Name: "a",
			}, {
				Name: "b",
			}},
			Origin: "a",
		},
		want: false,
	}, {
		name: "path element: a < b based on path value",
		inA: &gnmipb.Path{
			Element: []string{"a"},
		},
		inB: &gnmipb.Path{
			Element: []string{"z"},
		},
		want: true,
	}, {
		name: "path element: b < a based on path value",
		inA: &gnmipb.Path{
			Element: []string{"z"},
		},
		inB: &gnmipb.Path{
			Element: []string{"a"},
		},
		want: false,
	}, {
		name: "path element: a < b based on path length",
		inA: &gnmipb.Path{
			Element: []string{"a", "z"},
		},
		inB: &gnmipb.Path{
			Element: []string{"z"},
		},
		want: true,
	}, {
		name: "path element: b < a based on path length",
		inA: &gnmipb.Path{
			Element: []string{"a"},
		},
		inB: &gnmipb.Path{
			Element: []string{"z", "q"},
		},
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathLess(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("PathLess(%#v, %#v): did not get expected result, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}

func TestTypedValueLess(t *testing.T) {
	tests := []struct {
		name string
		inA  *gnmipb.TypedValue
		inB  *gnmipb.TypedValue
		want bool
	}{{
		name: "different types: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_UintVal{42},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"ab"},
		},
		want: true,
	}, {
		name: "different types: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"zzxx"},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_IntVal{42},
		},
		want: false,
	}, {
		name: "different types: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 4,
			}},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"forty-two"},
		},
		want: true,
	}, {
		name: "different types: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"forty-two"},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 4,
			}},
		},
		want: false,
	}, {
		name: "a and b nil: b < a",
		want: false,
	}, {
		name: "a nil, b non-nil: b < a",
		inB:  &gnmipb.TypedValue{},
		want: false,
	}, {
		name: "a non-nil, b nil: a < b",
		inA:  &gnmipb.TypedValue{},
		want: true,
	}, {
		name: "non-scalar: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_JsonVal{[]byte("json")},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_JsonVal{[]byte("zzz")},
		},
		want: true,
	}, {
		name: "non-scalar: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_JsonIetfVal{[]byte("aa")},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_JsonIetfVal{[]byte("zz")},
		},
		want: false,
	}, {
		name: "scalar string: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"a"},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"z"},
		},
		want: true,
	}, {
		name: "scalar string: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"z"},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_StringVal{"a"},
		},
		want: false,
	}, {
		name: "scalar float32: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 4,
			}},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 2,
			}},
		},
		want: true,
	}, {
		name: "scalar float32: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 0,
			}},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_DecimalVal{&gnmipb.Decimal64{
				Digits:    1234,
				Precision: 10,
			}},
		},
		want: false,
	}, {
		name: "scalar float64: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_FloatVal{42.42},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_FloatVal{84.84},
		},
		want: true,
	}, {
		name: "scalar float64: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_FloatVal{84.84},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_FloatVal{42.42},
		},
	}, {
		name: "scalar int64: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_IntVal{-42},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_IntVal{42},
		},
		want: true,
	}, {
		name: "scalar int64: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_IntVal{42},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_IntVal{-42},
		},
	}, {
		name: "scalar int64: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_UintVal{0},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_UintVal{42},
		},
		want: true,
	}, {
		name: "scalar int64: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_UintVal{42},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_UintVal{0},
		},
		want: false,
	}, {
		name: "scalar bool: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{false},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{true},
		},
		want: true,
	}, {
		name: "scalar bool: a < b but equal",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{true},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{true},
		},
		want: true,
	}, {
		name: "scalar bool: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{true},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_BoolVal{false},
		},
		want: false,
	}, {
		name: "non-scalar: a < b",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_LeaflistVal{&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"a"},
				}},
			}},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_LeaflistVal{&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"z"},
				}},
			}},
		},
		want: true,
	}, {
		name: "non-scalar: b < a",
		inA: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_LeaflistVal{&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"z"},
				}},
			}},
		},
		inB: &gnmipb.TypedValue{
			Value: &gnmipb.TypedValue_LeaflistVal{&gnmipb.ScalarArray{
				Element: []*gnmipb.TypedValue{{
					Value: &gnmipb.TypedValue_StringVal{"a"},
				}},
			}},
		},
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typedValueLess(tt.inA, tt.inB); got != tt.want {
				t.Fatalf("typedValueLess(%#v, %#v): did not get expected value, got: %v, want: %v", tt.inA, tt.inB, got, tt.want)
			}
		})
	}
}
