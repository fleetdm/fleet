

go run get_uuid_list/getuuids.go <TOKEN> | grep uuid  |  awk '{ print  substr($2, 2, length($2)-3)  }'  > uuids_list.txt


