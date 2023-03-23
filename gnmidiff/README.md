## Usage

```bash
$ go install ./gnmidiff

$ gnmidiff setrequest cmd/demo/setrequest.textproto cmd/demo/setrequest2.textproto

SetRequestIntentDiff(-A, +B):
-------- deletes/replaces --------
+ /network-instances/network-instance[name=VrfBlue]: deleted or replaced only in B
-------- updates --------
m /system/config/hostname:
  - "violetsareblue"
  + "rosesarered"

$ gnmidiff set-to-notifs cmd/demo/setrequest.textproto cmd/demo/notifs.textproto

SetToNotifsDiff(-want/SetRequest, +got/Notifications):
- /lacp/interfaces/interface[name=Port-Channel9]/config/interval: "FAST"
- /lacp/interfaces/interface[name=Port-Channel9]/config/name: "Port-Channel9"
- /lacp/interfaces/interface[name=Port-Channel9]/name: "Port-Channel9"
- /network-instances/network-instance[name=VrfBlue]/config/name: "VrfBlue"
- /network-instances/network-instance[name=VrfBlue]/config/type: "openconfig-network-instance-types:L3VRF"
- /network-instances/network-instance[name=VrfBlue]/name: "VrfBlue"
m /system/config/hostname:
  - "violetsareblue"
  + "rosesarered"

$ gnmidiff set-to-notifs cmd/demo/setrequest.textproto cmd/demo/getresponse.textproto

SetToNotifsDiff(-want/SetRequest, +got/Notifications):
- /lacp/interfaces/interface[name=Port-Channel9]/config/interval: "FAST"
- /lacp/interfaces/interface[name=Port-Channel9]/config/name: "Port-Channel9"
- /lacp/interfaces/interface[name=Port-Channel9]/name: "Port-Channel9"
- /network-instances/network-instance[name=VrfBlue]/config/name: "VrfBlue"
- /network-instances/network-instance[name=VrfBlue]/config/type: "openconfig-network-instance-types:L3VRF"
- /network-instances/network-instance[name=VrfBlue]/name: "VrfBlue"
m /system/config/hostname:
  - "violetsareblue"
  + "rosesarered"
```
